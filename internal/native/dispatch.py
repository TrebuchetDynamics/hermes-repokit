"""Setup-only activation. Native Hermes owns config, claims, workers and review."""
import contextlib
import json
import os
import re
from pathlib import Path
import subprocess
import time


def dispatch_failure_diagnostic(error):
    # Only fixed stage names and line numbers cross the private native boundary.
    # Never format exception messages, source text, locals or arbitrary frames.
    stages={'dispatch_main','check_activation','switch_gateway','dispatcher_canary',
            'dispatch_fence','startup_since','log_cursor','create_dispatch_canary',
            'retry_dispatch_canary','suspend_failed_activation','release_dispatch'}
    kind='Exception'
    for cls in (RuntimeError,ValueError,OSError,subprocess.TimeoutExpired,subprocess.CalledProcessError):
        if isinstance(error,cls):
            kind=cls.__name__
            break
    frames=[]
    trace=error.__traceback__
    while trace is not None:
        name=trace.tb_frame.f_code.co_name
        if name in stages:
            frames.append(name+':'+str(trace.tb_lineno))
        trace=trace.tb_next
    return kind+'|'+'>'.join(frames[-8:] or ['unknown:0'])


def idle_gateway(observed):
    return (observed.get('state') == 'running' and observed.get('healthy') is True
            and observed.get('idle_verified') is True and type(observed.get('active')) is int
            and observed['active'] == 0)


def native_command(*args, capture=False):
    result = subprocess.run(['hermes', *args], check=True, timeout=120,
                            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE if capture else subprocess.DEVNULL,
                            stderr=subprocess.DEVNULL, text=True)
    if capture:
        if len(result.stdout) > 1048576: raise RuntimeError('oversized native result')
        return json.loads(result.stdout)


def native_worker_alive(pid, started):
    from hermes_cli.kanban_db_dispatch import _worker_alive
    return _worker_alive(pid, started)


@contextlib.contextmanager
def dispatch_fence(root):
    # The pinned dispatcher uses nonblocking per-board flocks. Holding those
    # through drain/restart prevents a new claim between the idle check and stop.
    directory = root/'kanban/boards'
    if directory.is_symlink() or (root/'kanban').is_symlink():
        raise RuntimeError('redirected board store')
    paths = [root/'kanban.db']
    entries = list(directory.iterdir()) if directory.exists() else []
    if len(entries) > 128: raise RuntimeError('too many boards')
    for entry in entries:
        if entry.name == '_archived': continue
        if entry.is_symlink() or not entry.is_dir(): raise RuntimeError('invalid board')
        paths.append(entry/'kanban.db')
    with contextlib.ExitStack() as stack:
        for path in sorted(paths):
            if path.is_symlink() or not path.is_file(): raise RuntimeError('invalid board')
            fd = os.open(str(path)+'.dispatch.lock', os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW|os.O_NONBLOCK, 0o600)
            handle = stack.enter_context(os.fdopen(fd, 'rb'))
            if not stat.S_ISREG(os.fstat(fd).st_mode): raise RuntimeError('invalid native board lock')
            fcntl.flock(handle, fcntl.LOCK_EX|fcntl.LOCK_NB)
        if not boards_quiescent(root, allow_queued=True, worker_alive=native_worker_alive): raise RuntimeError('active Kanban work')
        yield


def log_cursor(root):
    path=root/'logs/gateway.log'
    if (root/'logs').is_symlink() or path.is_symlink(): raise RuntimeError('redirected log')
    try:
        meta=path.stat()
        return meta.st_ino, meta.st_size
    except FileNotFoundError:
        return None, 0


def startup_since(root, cursor):
    path=root/'logs/gateway.log'
    fd=os.open(path,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK)
    with os.fdopen(fd,'rb') as stream:
        meta=os.fstat(stream.fileno())
        if not stat.S_ISREG(meta.st_mode): return None
        inode,offset=cursor
        if inode is not None and inode != meta.st_ino: return None  # retry after log rotation
        if meta.st_size < offset or meta.st_size-offset > 1048576: return None
        stream.seek(offset)
        return startup_settings(stream.read(1048577).decode('utf-8',errors='replace'))


def switch_gateway(root, roles, repo_id, enable, *, canary_only=False):
    with dispatch_fence(root):
        before=observe_gateway(root)
        if not idle_gateway(before): raise RuntimeError('gateway not healthy and idle')
        config=mapping(root/'config.yaml')
        if not manual_policy(config) and not operational_policy(config) and not canary_policy(config):
            raise RuntimeError('dispatch policy drift preserved')
        required=set(before.get('adapters', []))
        if not required: raise RuntimeError('no healthy default adapters')
        cursor=log_cursor(root)
        if enable:
            policy=dispatch_policy('default')
            if canary_only: policy['dispatch_profiles']=['researcher']
            for key,value in policy.items():
                if key != 'dispatch_in_gateway':
                    native_command('-p','default','config','set','kanban.'+key,value if isinstance(value,str) else json.dumps(value))
            native_command('-p','default','config','set','kanban.dispatch_interval_seconds','60')
        try:
            native_command('-p','default','config','set','kanban.dispatch_in_gateway',json.dumps(enable))
            try: digest=generation(root,roles,repo_id)
            except (OSError,ValueError):
                if enable: raise
                digest=None
            latest=observe_gateway(root)
            if not idle_gateway(latest) or latest['identity'] != before['identity']:
                raise RuntimeError('gateway became busy before restart')
            native_command('-p','default','gateway','restart')
            for _ in range(90):
                time.sleep(1)
                after=observe_gateway(root)
                if (after.get('state') != 'running' or not after.get('healthy') or
                        after.get('identity') == before['identity'] or
                        not required.issubset(after.get('adapters',[]))): continue
                owned=owns_dispatch_lock(root,after['identity']['pid'])
                evidence=startup_since(root,cursor) if enable else None
                if owned is not enable or (enable and not evidence): continue
                if digest is not None and generation(root,roles,repo_id) != digest: raise RuntimeError('configuration changed during restart')
                receipt={'schema':1,'generation':digest,'identity':after['identity'],
                         'adapters':sorted(required|set(after['adapters']))}
                if enable: receipt['dispatch']=evidence
                if digest is not None: publish_marker(root,receipt)
                return
            raise RuntimeError('replacement gateway/dispatcher startup unverified')
        except Exception:
            if enable:
                # All native claim fences remain held during failure recovery.
                # Fence hot-read claims before releasing locks, even if the
                # replacement process cannot prove its boot-only settings.
                native_command('-p','default','config','set','kanban.dispatch_profiles','[]')
                native_command('-p','default','config','set','kanban.dispatch_in_gateway','false')
                native_command('-p','default','gateway','restart')
            raise



PROVIDER_CHECK = r'''
import contextlib, os
from pathlib import Path
from hermes_cli.config import load_config
from agent.secret_scope import build_profile_secret_scope, set_secret_scope
set_secret_scope(build_profile_secret_scope(Path(os.environ['HERMES_HOME'])), profile_home=os.environ['HERMES_HOME'])
with open(os.devnull,'w') as sink, contextlib.redirect_stdout(sink), contextlib.redirect_stderr(sink):
    from hermes_cli.runtime_provider import resolve_runtime_provider
    model=load_config().get('model') or {}
    assert isinstance(model,dict) and model.get('default') and model.get('provider')
    runtime=resolve_runtime_provider(requested=model['provider'],target_model=model['default'])
    assert runtime.get('provider') and runtime.get('api_mode') and not runtime.get('auth_error')
    # Credentialless transports must be explicitly native local/external modes.
    from urllib.parse import urlparse
    local=urlparse(runtime.get('base_url') or '').hostname in {'127.0.0.1','localhost','::1'}
    assert runtime.get('api_key') or local or runtime.get('api_mode') in {'bedrock','vertex','codex_app_server'}
'''


def check_activation(root, payload):
    roles=payload['roles']
    names=[role['name'] for role in roles]
    if names != list(TEAM): raise RuntimeError('unexpected roster')
    for role in roles:
        home=root if role['name']=='default' else root/'profiles'/role['name']
        if not matches(home,role): raise RuntimeError('profile identity/capability drift')
        native_command('profile','show',role['name'])
    config=read_config(root)
    catalog=native_interactive_catalog()
    platforms=default_interactive_platforms(root,config,catalog)
    if default_native_tool_drift(config,catalog): raise RuntimeError('default tool parity incomplete')
    for platform in platforms:
        if not set(catalog[platform]['required']) <= native_platform_tools(config,platform):
            raise RuntimeError('effective default channel tools incomplete')
    # The native resolved gateway routes must keep default as the human interface.
    from gateway.config import load_gateway_config
    from agent.secret_scope import build_profile_secret_scope, set_secret_scope, reset_secret_scope
    token=set_secret_scope(build_profile_secret_scope(root),profile_home=str(root))
    try:
        with open(os.devnull,'w') as sink,contextlib.redirect_stdout(sink),contextlib.redirect_stderr(sink):
            gateway_config=load_gateway_config()
        if any(r.enabled and (r.bot_profile or 'default')=='default' and r.profile!='default'
               for r in gateway_config.profile_routes):
            raise RuntimeError('default channel routing differs')
    finally:
        reset_secret_scope(token)
    for name in names:
        home=root if name=='default' else root/'profiles'/name
        env=dict(os.environ,HERMES_HOME=str(home),HERMES_PROFILE=name,HERMES_PROFILE_NAME=name)
        subprocess.run([sys.executable,'-B','-c',PROVIDER_CHECK],env=env,check=True,timeout=90,
                       stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    native_command('-p','default','kanban','list','--json',capture=True)
    if (root/'ESTOP').exists(): raise RuntimeError('native emergency pause preserved')


LEGACY_CANARY_BODY = ('Read the repository README title. Do not modify files. Complete with the title in your summary '
                      'and structured metadata {"title": "the exact first Markdown heading without #", "changed_files": []}.')
V2_CANARY_BODY = ('Read /workspace/README.md without modifying files. Use the first nonempty heading on a line '
               'starting with "# ", removing that prefix and surrounding whitespace. If README.md is missing '
               'or has no such heading, use exactly NO_MARKDOWN_TITLE. Do not create or repair the README. '
               'Complete with that title in your summary and structured metadata '
               '{"title": "the title or NO_MARKDOWN_TITLE", "changed_files": []}.')

V3_CANARY_BODY = ('Read /workspace/README.md without modifying files. Ignore Markdown fenced code blocks '
               '(backticks or tildes, at least three; a closing fence must use the same character and '
               'at least the opening length). Use the first nonempty heading outside those blocks on a line '
               'starting with "# ", removing that prefix and surrounding whitespace. If README.md is missing '
               'or has no such heading, use exactly NO_MARKDOWN_TITLE. Do not create or repair the README. '
               'Complete with that title in your summary and structured metadata '
               '{"title": "the title or NO_MARKDOWN_TITLE", "changed_files": []}.')


CANARY_BODY = ('Use read_file to read line 1 of /workspace/README.md. Do not modify any files. '
               'Copy the first physical line exactly, excluding only its newline; preserve spaces, '
               'Markdown markers and HTML. Do not choose a heading or interpret the content. '
               'For an empty file or blank first line use an empty string. Only if the file does not '
               'exist use README_MISSING; permission or tool failures must block the task. '
               'Complete with a nonempty summary and structured metadata '
               '{"first_line": "the exact first line", "changed_files": []}.')


def canary_first_line(path):
    try:
        line=read_file(path).partition(b'\n')[0].removesuffix(b'\r')
    except FileNotFoundError:
        return 'README_MISSING'
    if len(line)>4096: raise RuntimeError('canary first line exceeds bounded evidence size')
    return line.decode('utf-8')


def canary_task_matches(task):
    if not isinstance(task,dict): return False
    body=CANARY_BODY
    if task.get('status') in ('done','blocked') and task.get('body') in (LEGACY_CANARY_BODY,V2_CANARY_BODY,V3_CANARY_BODY):
        body=task['body']  # Exact terminal history only; never resume old work.
    contract = dict(title='RepoKit dispatcher canary', assignee='researcher',
                    workspace_kind='dir', workspace_path='/workspace', created_by='default',
                    body=body, max_runtime_seconds=180, max_retries=1, priority=100,
                    skills=[], completion_contract='local-only')
    return (isinstance(task, dict) and all(task.get(key) == value for key,value in contract.items())
            and all(task.get(key) is None for key in ('model_override','provider_override',
                'project_id','branch_name','tenant','session_id',
                'workflow_template_id','current_step_key','result')))


def create_dispatch_canary(key='repokit-dispatcher-canary-v4'):
    task=native_command('-p','default','kanban','create','RepoKit dispatcher canary',
        '--assignee','researcher','--workspace','dir:/workspace',
        '--created-by','default','--idempotency-key',key,
        '--max-runtime','180','--max-retries','1','--priority','100',
        '--completion-contract','local-only','--body',CANARY_BODY,
        '--json',capture=True)
    task_id=task['id']
    if not isinstance(task_id,str) or not re.fullmatch(r'[A-Za-z0-9_-]{1,100}',task_id):
        raise RuntimeError('invalid native canary identity')
    if not canary_task_matches(task):
        raise RuntimeError('canary identity collision; owner card preserved')
    return task


def retry_dispatch_canary(root, task):
    # There is no native conditional archive operation. A show/archive pair can
    # race an owner edit/link even under the dispatcher flock. Never mutate the
    # old card: preserve its history and use a stable per-card retry key instead.
    with dispatch_fence(root):
        record=native_command('-p','default','kanban','show',task['id'],'--json',capture=True)
        current=record.get('task',{})
        runs=record.get('runs') or []
        if (current.get('id') != task['id'] or not canary_task_matches(current)
                or current.get('status') != task['status']
                or record.get('parents') != [] or record.get('children') != []):
            raise RuntimeError('canary changed; owner card preserved')
        if task['status']=='done':
            metadata=(runs[-1].get('metadata') or {}) if runs else {}
            if isinstance(metadata,str): metadata=json.loads(metadata)
            field='first_line' if current.get('body')==CANARY_BODY else 'title'
            previous_title=metadata.get(field) if isinstance(metadata,dict) else None
            # Preserve terminal evidence without comparing it to today's README.
            if not isinstance(previous_title,str) or not canary_complete(record,previous_title,field):
                raise RuntimeError('prior canary evidence differs')
        elif (task['status']!='blocked' or not runs or any(
                r.get('profile')!='researcher' or r.get('ended_at') is None
                or r.get('status') not in ('failed','blocked','reclaimed') for r in runs)):
            raise RuntimeError('failed canary ownership or quiescence unverified')
        key=hashlib.sha256(task['id'].encode()).hexdigest()
        return create_dispatch_canary('repokit-dispatcher-canary-v4-retry-'+key)


def dispatcher_canary(root, payload):
    digest=generation(root,list(TEAM),payload['repo_id'])
    observed=observe_gateway(root)
    marker=load_marker(root)
    config=mapping(root/'config.yaml')
    pid=(observed.get('identity') or {}).get('pid')
    if (not canary_policy(config) or not pid or
            dispatch_live(True,classify(digest,observed,marker),owns_dispatch_lock(root,pid),marker,(root/'ESTOP').exists())!='enabled'):
        raise RuntimeError('researcher-only dispatcher not live')
    title=canary_first_line(Path('/workspace/README.md'))
    # Reuse unfinished native work. Preserve a terminal prior attempt after an
    # interrupted setup and demand a new run from this replacement gateway.
    task=create_dispatch_canary()
    visited=set()
    for _ in range(32):
        if task.get('status') not in ('done','blocked'): break
        if task['id'] in visited: raise RuntimeError('canary retry identity collision')
        visited.add(task['id'])
        task=retry_dispatch_canary(root,task)
    if task.get('status') in ('done','blocked'):
        raise RuntimeError('canary retry history exceeds bounded recovery; preserved for inspection')
    task_id=task['id']
    deadline=time.monotonic()+300
    claim_deadline=time.monotonic()+130
    gateway_pid=observed['identity']['pid']
    gateway_spawn=False
    saw_running=False
    while time.monotonic()<deadline:
        record=native_command('-p','default','kanban','show',task_id,'--json',capture=True)
        if record.get('task',{}).get('status')=='running':
            saw_running=True
            worker=canary_worker_pid(record)
            if type(worker) is int and worker>0:
                try:
                    raw=Path('/proc',str(worker),'stat').read_text()
                    gateway_spawn |= int(raw[raw.rfind(')')+2:].split()[1])==gateway_pid
                except (OSError,ValueError,IndexError): pass
        if canary_complete(record,title,'first_line') and saw_running and gateway_spawn:
            current=observe_gateway(root)
            if current.get('identity') != observed['identity']: raise RuntimeError('gateway changed during canary')
            native_command('-p','default','kanban','archive',task_id)
            marker['canary']={'task_id':task_id,'profile':'researcher','gateway_spawn':True,'done':True}
            publish_marker(root,marker)
            print('REPOKIT_CANARY=researcher-done')
            return
        if record.get('task',{}).get('status') in ('blocked','done','archived'):
            raise RuntimeError('canary did not supply successful no-write researcher evidence')
        if not saw_running and time.monotonic()>claim_deadline:
            raise RuntimeError('gateway did not claim canary within two intervals')
        time.sleep(0.5)
    raise RuntimeError('canary timed out; native card preserved for inspection')


def release_dispatch(root, roles, repo_id):
    # Native completion precedes process exit. Wait for the proven worker to
    # finish, then fence claims while publishing the full operational policy.
    for _ in range(60):
        if not boards_quiescent(root,allow_queued=True,worker_alive=native_worker_alive):
            time.sleep(1)
            continue
        with dispatch_fence(root):
            observed=observe_gateway(root)
            marker=load_marker(root)
            digest=generation(root,roles,repo_id)
            evidence=(marker or {}).get('canary') or {}
            if (not idle_gateway(observed) or classify(digest,observed,marker)!='current'
                    or not canary_policy(mapping(root/'config.yaml'))
                    or evidence.get('profile')!='researcher' or evidence.get('done') is not True
                    or evidence.get('gateway_spawn') is not True
                    or not owns_dispatch_lock(root,observed['identity']['pid'])
                    or (root/'ESTOP').exists()):
                raise RuntimeError('canary release evidence unavailable or changed')
            # This allowlist is hot-read by the pinned native claim predicate.
            # Boot-only dispatch settings remain unchanged: no second restart.
            try:
                native_command('-p','default','config','set','kanban.dispatch_profiles',json.dumps(list(TEAM)))
                if not operational_policy(mapping(root/'config.yaml')):
                    raise RuntimeError('operational policy not restored')
                marker['generation']=generation(root,roles,repo_id)
                publish_marker(root,marker)
            except Exception:
                # Keep the native claim fences held through rollback: a failed
                # receipt must not expose the full team between ticks.
                native_command('-p','default','config','set','kanban.dispatch_profiles','[]')
                native_command('-p','default','config','set','kanban.dispatch_in_gateway','false')
                raise
            return
    raise RuntimeError('canary worker did not finish before release')


def suspend_failed_activation(root, roles, repo_id):
    # dispatch_profiles is reread by the pinned native claim predicate each tick.
    # Empty is its supported fail-closed fence: no new paid worker or reviewer
    # claims while an already running worker finishes. This is failure recovery,
    # never the operational allowlist. Do not kill or restart an active worker.
    native_command('-p','default','config','set','kanban.dispatch_profiles','[]')
    native_command('-p','default','config','set','kanban.dispatch_in_gateway','false')
    try:
        switch_gateway(root,roles,repo_id,False)
    except Exception:
        pass  # Remains configured off/claims fenced; verify reports live stale.


def dispatch_main(payload):
    root=Path('/opt/data')
    roles=[role['name'] for role in payload['roles']]
    config=mapping(root/'config.yaml')
    if payload['action']=='prepare':
        if config.get('kanban',{}).get('dispatch_in_gateway') is True:
            with dispatch_fence(root):
                observed=observe_gateway(root)
                stopped=observed.get('state')=='not-running'
                if stopped:
                    if not operational_policy(config) and not canary_policy(config): raise RuntimeError('owner dispatch policy drift')
                    native_command('-p','default','config','set','kanban.dispatch_in_gateway','false')
            if not stopped: switch_gateway(root,roles,payload['repo_id'],False)
        elif config.get('kanban',{}).get('dispatch_in_gateway') is not False:
            raise RuntimeError('ambiguous dispatch policy')
        else:
            with dispatch_fence(root):
                observed=observe_gateway(root)
                if observed.get('state') != 'not-running' and not idle_gateway(observed):
                    raise RuntimeError('gateway quiescence unverified')
            if (observed.get('identity') or {}).get('pid') and owns_dispatch_lock(root,observed['identity']['pid']):
                switch_gateway(root,roles,payload['repo_id'],False)
        print('REPOKIT_DISPATCH=prepared')
        return
    check_activation(root,payload)
    observed=observe_gateway(root)
    marker=load_marker(root)
    digest=generation(root,roles,payload['repo_id'])
    current=dispatch_observation(root,config,observed,classify(digest,observed,marker),marker)
    if current['live']=='enabled' and current['canary']:
        # Reconciliation of an unchanged operational installation is observation,
        # not another restart or paid canary. Required gates were still rechecked.
        print('REPOKIT_CANARY=researcher-done')
        print('REPOKIT_GATEWAY=current')
        return
    if observed.get('state') == 'not-running':
        with dispatch_fence(root):
            # Never start a saved operational allowlist before this boot is proven.
            native_command('-p','default','config','set','kanban.dispatch_in_gateway','false')
        native_command('-p','default','gateway','start')
        for _ in range(60):
            if idle_gateway(observe_gateway(root)): break
            time.sleep(1)
    try:
        switch_gateway(root,roles,payload['repo_id'],True,canary_only=True)
        dispatcher_canary(root,payload)
        release_dispatch(root,roles,payload['repo_id'])
    except Exception:
        suspend_failed_activation(root,roles,payload['repo_id'])
        raise
    print('REPOKIT_GATEWAY=current')

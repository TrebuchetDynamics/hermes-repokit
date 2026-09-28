"""One-shot generation diagnostics. Never imported by the running gateway."""
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import socket
import sqlite3
import stat
import time
import yaml

MARKER = '.repokit-gateway-generation.json'


def read_file(path, limit=262144):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as source:
        meta = os.fstat(source.fileno())
        if not stat.S_ISREG(meta.st_mode) or meta.st_size > limit:
            raise ValueError('invalid bounded native file')
        data = source.read(limit+1)
    if len(data) > limit:
        raise ValueError('oversized native file')
    return data


def mapping(path):
    value = yaml.safe_load(read_file(path))
    if not isinstance(value, dict):
        raise ValueError('invalid native mapping')
    return value


def generation(root, roles, repo_id):
    projection = {'schema':1, 'repository':repo_id, 'required_tools':['kanban','memory'], 'profiles':[]}
    for name in roles:
        home = root if name == 'default' else root/'profiles'/name
        if home.is_symlink() or (root/'profiles').is_symlink():
            raise ValueError('redirected profile')
        config = mapping(home/'config.yaml')
        memory = config.get('memory') or {}
        plugins = config.get('plugins') or {}
        # Only these public configuration fields enter the hash. Never read a
        # credential store, token or API-key field. Only model name/provider
        # identifiers are selected; credential fields never enter this projection.
        ov = memory.get('openviking') or {}
        item = {
            'name':name, 'soul':read_file(home/'SOUL.md',65536).decode(),
            'description':mapping(home/'profile.yaml').get('description'),
            'tools':config.get('toolsets'), 'platforms':config.get('platform_toolsets'),
            'disabled':(config.get('agent') or {}).get('disabled_toolsets'),
            'project_skills':{k:(config.get('skills') or {}).get(k) for k in ('project_discovery','trusted_project_dirs')},
            'kanban':{k:(config.get('kanban') or {}).get(k) for k in ('dispatch_in_gateway','auto_decompose','max_in_progress','orchestrator_profile','review_dispatch','dispatch_profiles','dispatch_interval_seconds','auto_subscribe_on_create','notify_in_gateway')},
            'memory':{k:memory.get(k) for k in ('provider','memory_enabled','user_profile_enabled')},
            'model':{k:(config.get('model') or {}).get(k) for k in ('default','provider')},
            'connection':{k:ov.get(k) for k in ('use_ovcli_config','ovcli_config_path','endpoint','account','user','agent','actor_peer_id')},
            'maintenance_enabled':'repokit_maintenance' in (plugins.get('enabled') or []),
            'maintenance_disabled':'repokit_maintenance' in (plugins.get('disabled') or []),
            'maintenance_revision':None,
        }
        if (home/'plugins').is_symlink():
            raise ValueError('redirected plugin store')
        metadata = home/'plugins/.install-metadata.json'
        if metadata.exists() or metadata.is_symlink():
            records = json.loads(read_file(metadata))
            item['maintenance_revision'] = (records.get('repokit_maintenance') or {}).get('revision')
        projection['profiles'].append(item)
    return hashlib.sha256(json.dumps(projection,sort_keys=True,separators=(',',':')).encode()).hexdigest()


def classify(digest, observed, marker):
    if observed.get('state') == 'not-running':
        return 'not-running'
    if observed.get('state') != 'running' or not observed.get('healthy'):
        return 'unknown'
    if not isinstance(marker,dict) or marker.get('schema') != 1:
        return 'unknown'
    required=marker.get('adapters')
    if not isinstance(required,list) or not required or not set(required).issubset(observed.get('adapters',[])):
        return 'unknown'
    if marker.get('identity') != observed.get('identity'):
        return 'unknown'
    return 'current' if marker.get('generation') == digest else 'stale'


def converge(digest, observe, quiescent, restart, publish, marker=None, attempts=90, pause=lambda:time.sleep(1)):
    before = digest()
    observed = observe()
    if observed.get('state') == 'not-running':
        return 'not-running'
    if observed.get('state') != 'running' or type(observed.get('active')) is not int or observed['active'] != 0 or observed.get('idle_verified') is not True or not quiescent():
        raise RuntimeError('gateway identity or quiescence unverified')
    if classify(before,observed,marker) == 'current':
        return 'current'
    required=set(observed.get('adapters',[]))
    if isinstance(marker,dict) and isinstance(marker.get('adapters'),list):
        required.update(marker['adapters'])
    if not required:
        raise RuntimeError('no observed gateway adapters to qualify')
    latest=observe()
    if latest.get('identity') != observed.get('identity') or latest.get('idle_verified') is not True or latest.get('active') != 0:
        raise RuntimeError('gateway changed or became busy before restart')
    restart()
    for _ in range(attempts):
        pause()
        after = observe()
        if after.get('state') == 'running' and after.get('healthy') and after.get('identity') != observed.get('identity'):
            if not required.issubset(after.get('adapters',[])):
                continue
            if before != digest():
                raise RuntimeError('managed state changed during restart')
            publish({'schema':1,'generation':before,'identity':after['identity'],'adapters':sorted(required|set(after['adapters']))})
            return 'current'
    raise RuntimeError('replacement gateway health unverified')


def process_identity(pid, proc=Path('/proc')):
    if type(pid) is not int or pid <= 0:
        raise ValueError('invalid gateway pid')
    raw = (proc/str(pid)/'stat').read_text()
    fields = raw[raw.rfind(')')+2:].split()
    if fields[0] in ('Z','X'):
        raise ValueError('dead gateway')
    return {'pid':pid,'start':int(fields[19]),
            'boot':(proc/'sys/kernel/random/boot_id').read_text().strip(),
            'namespace':os.readlink(proc/'self/ns/pid')}


def fresh(value, now):
    try:
        stamp = datetime.datetime.fromisoformat(value.replace('Z','+00:00')).timestamp()
        return -5 <= now-stamp <= 120
    except (ValueError,TypeError,AttributeError):
        return False


def loop_alive(root, pid):
    try:
        with socket.socket(socket.AF_UNIX,socket.SOCK_STREAM) as client:
            client.settimeout(1)
            client.connect(str(root/'state'/('gateway.loop-tick.%d.sock'%pid)))
            return client.recv(1) == b'1'
    except OSError:
        return False


def observe_gateway(root):
    try:
        records=[]
        for name in ('gateway.pid','gateway.lock','gateway_state.json'):
            try: records.append(json.loads(read_file(root/name)))
            except FileNotFoundError: records.append(None)
        if all(r is None for r in records):
            return {'state':'not-running'}
        pids = {r.get('pid') for r in records if isinstance(r,dict)}
        if len(pids) != 1:
            return {'state':'unknown'}
        pid=pids.pop()
        try: identity=process_identity(pid)
        except FileNotFoundError: return {'state':'not-running'}
        for r in records:
            if not isinstance(r,dict) or r.get('kind') != 'hermes-gateway' or r.get('start_time') != identity['start'] or r.get('hermes_home') != str(root):
                return {'state':'unknown'}
        with os.fdopen(os.open(root/'gateway.lock',os.O_RDONLY|os.O_NOFOLLOW),'rb') as lock:
            try:
                fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
            except BlockingIOError:
                pass
            else:
                fcntl.flock(lock,fcntl.LOCK_UN)
                return {'state':'unknown'}
        status=records[2]
        active=status.get('active_agents')
        if type(active) is not int or active < 0:
            return {'state':'unknown'}
        now=time.time()
        idle_verified=(status.get('gateway_state') in ('running','degraded') and status.get('restart_requested') is False and active==0 and fresh(status.get('updated_at'),now))
        healthy=(status.get('gateway_state')=='running' and status.get('restart_requested') is False
                 and (status.get('session_store') or {}).get('status')=='ok' and fresh(status.get('updated_at'),now))
        platforms=status.get('platforms')
        adapters=[]
        if not isinstance(platforms,dict):
            healthy=False
        else:
            for name,adapter in platforms.items():
                if not isinstance(adapter,dict):
                    healthy=False
                    continue
                if adapter.get('writer_pid')!=pid or adapter.get('writer_start_time')!=identity['start'] or adapter.get('state')=='disabled':
                    continue  # Native status preserves historical adapter rows.
                adapters.append(name)
                if adapter.get('state')!='connected' or adapter.get('needs_attention') is not False:
                    healthy=False
        if not adapters:
            healthy=False
        heartbeat=json.loads(read_file(root/'state/gateway.heartbeat'))
        live_loop=loop_alive(root,pid)
        healthy=healthy and heartbeat.get('pid')==pid and fresh(heartbeat.get('updated_at'),now) and live_loop
        if process_identity(pid) != identity:
            return {'state':'unknown'}
        return {'state':'running','identity':identity,'healthy':bool(healthy),'active':active,'adapters':sorted(adapters),'idle_verified':bool(idle_verified and live_loop and heartbeat.get('pid')==pid and fresh(heartbeat.get('updated_at'),now))}
    except (OSError,ValueError,TypeError,KeyError,IndexError,AttributeError):
        return {'state':'unknown'}


def load_marker(root):
    try:
        return json.loads(read_file(root/MARKER,4096))
    except FileNotFoundError:
        return None


def publish_marker(root, marker):
    import tempfile
    descriptor,name=tempfile.mkstemp(prefix='.repokit-gateway-',dir=root)
    try:
        with os.fdopen(descriptor,'w') as output:
            json.dump(marker,output,sort_keys=True)
            output.flush()
            os.fsync(output.fileno())
        destination=root/MARKER
        if destination.is_symlink():
            raise ValueError('redirected generation receipt')
        os.replace(name,destination)
    finally:
        if os.path.exists(name): os.unlink(name)


def manual_policy(config):
    policy=config.get('kanban') or {}
    return (isinstance(policy,dict) and policy.get('dispatch_in_gateway') is False
            and policy.get('auto_decompose') is False and policy.get('orchestrator_profile')=='default'
            and type(policy.get('max_in_progress')) is int and policy['max_in_progress']==1)


def boards_quiescent(root, allow_queued=False, worker_alive=None):
    """Fail closed across every native local board, including residual workers.

    A read snapshot is not a fence against a concurrent external dispatch. Keep
    embedded dispatch disabled and reobserve gateway activity before restart.
    """
    try:
        directory=root/'kanban/boards'
        if directory.is_symlink() or (root/'kanban').is_symlink():
            return False
        entries=list(directory.iterdir()) if directory.exists() else []
        if len(entries)>128:
            return False
        paths=[root/'kanban.db']
        for entry in entries:
            if entry.name=='_archived': continue
            if entry.is_symlink() or not entry.is_dir(): return False
            paths.append(entry/'kanban.db')
        for path in paths:
            if path.is_symlink() or not path.is_file(): return False
            with sqlite3.connect(path.resolve().as_uri()+'?mode=ro',uri=True,timeout=0.2) as conn:
                conn.execute('PRAGMA query_only=ON')
                statuses = "('running')" if allow_queued else "('ready','running','review')"
                if conn.execute("SELECT 1 FROM tasks WHERE status IN " + statuses + " OR worker_pid IS NOT NULL LIMIT 1").fetchone():
                    return False
                if worker_alive is not None:
                    # A terminal task can still have a finalizing subprocess.
                    # Historical PIDs are qualified by native spawn fingerprints.
                    for pid, started in conn.execute('SELECT worker_pid, worker_started_at FROM task_runs WHERE worker_pid IS NOT NULL'):
                        if worker_alive(pid, started): return False
        return True
    except (OSError,sqlite3.Error):
        return False


def main(payload):
    import subprocess
    import sys
    root=Path('/opt/data')
    roles=[role['name'] for role in payload['roles']]
    try:
        if payload['apply']:
            if not manual_policy(mapping(root/'config.yaml')):
                raise RuntimeError('manual dispatch policy drift preserved')
            def quiet():
                config=mapping(root/'config.yaml')
                if not manual_policy(config):
                    return False
                return boards_quiescent(root)
            def restart():
                subprocess.run(['hermes','-p','default','gateway','restart'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=60)
            status=converge(lambda:generation(root,roles,payload['repo_id']), lambda:observe_gateway(root),quiet,restart,lambda receipt:publish_marker(root,receipt),marker=load_marker(root))
            print('REPOKIT_GATEWAY='+status)
        else:
            result={'gateway':'unknown','identities':{},'descriptions':{}, 'channels':{'rows':[]}, 'pid':None, 'managed_generation':None, 'live_generation':None, 'restart_pending':'unknown', 'dispatch':{}}
            for role in payload['roles']:
                home=root if role['name']=='default' else root/'profiles'/role['name']
                try:
                    result['identities'][role['name']]=read_file(home/'SOUL.md',65536).decode()==role['soul']
                    result['descriptions'][role['name']]=mapping(home/'profile.yaml').get('description')==role['description']
                except Exception:
                    result['identities'][role['name']]=False
                    result['descriptions'][role['name']]=False
            try:
                digest=generation(root,roles,payload['repo_id'])
                observed=observe_gateway(root)
                marker=load_marker(root)
                result['gateway']=classify(digest,observed,marker)
                result['managed_generation']=digest
                result['pid']=(observed.get('identity') or {}).get('pid')
                if result['gateway']=='current':
                    result['live_generation']=digest
                state=mapping(root/'gateway_state.json') if observed.get('state')=='running' else {}
                # Discard historical adapter rows retained across restarts.
                identity=observed.get('identity') or {}
                if state and (state.get('pid')!=identity.get('pid') or state.get('start_time')!=identity.get('start') or state.get('hermes_home')!=str(root)):
                    raise ValueError('gateway changed during channel observation')
                if isinstance(state.get('platforms'),dict):
                    state['platforms']={k:v for k,v in state['platforms'].items() if isinstance(v,dict) and v.get('writer_pid')==identity.get('pid') and v.get('writer_start_time')==identity.get('start')}
                if type(state.get('restart_requested')) is bool:
                    result['restart_pending']='yes' if state['restart_requested'] else 'no'
                config=mapping(root/'config.yaml')
                result['dispatch']=dispatch_observation(root,config,observed,result['gateway'],marker)
                legacy=mapping(root/'gateway.json') if (root/'gateway.json').exists() else {}
                catalog=platform_catalog(read_file(Path('/opt/hermes/hermes_cli/platforms.py')).decode())
                result['channels']=project_channels(config,legacy,state,observed.get('healthy') is True,catalog)
            except Exception:
                pass
            print(json.dumps(result))
    except Exception:
        print('Gateway convergence incomplete; preserved native state. Inspect process health, active work and owner drift.',file=sys.stderr)
        sys.exit(1)

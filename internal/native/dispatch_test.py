import importlib.util
from pathlib import Path
import unittest
import contextlib
import io
import time
import tempfile
import json
import types
import sys
from unittest.mock import patch

spec=importlib.util.spec_from_file_location('activation',Path(__file__).with_name('dispatch.py'))
a=importlib.util.module_from_spec(spec)
try: spec.loader.exec_module(a)
except FileNotFoundError: pass

class GateTest(unittest.TestCase):
    def test_missing_any_required_gate_refuses_activation(self):
        gates=dict(provider=True,profiles=True,tools=True,board=True,routing=True,memory=True)
        self.assertTrue(a.activation_ready(gates))
        for key in gates:
            self.assertFalse(a.activation_ready(dict(gates,**{key:False})),key)
            self.assertFalse(a.activation_ready({k:v for k,v in gates.items() if k!=key}),key)
        self.assertFalse(a.activation_ready(dict(gates,provider='configured')))
    def test_queued_cards_do_not_block_but_active_or_unknown_gateway_does(self):
        self.assertTrue(a.idle_gateway({'state':'running','healthy':True,'idle_verified':True,'active':0}))
        for change in ({'active':1},{'idle_verified':False},{'state':'unknown'},{'healthy':False}):
            self.assertFalse(a.idle_gateway(dict(state='running',healthy=True,idle_verified=True,active=0,**{})|change))

class ActivationMemoryTest(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        self.names=('default','researcher','planner','executor','reviewer','steward')
        self.payload={'roles':[{'name':name} for name in self.names],
                      'repo_id':'repository-test','integrations_ready':True}
        self.checked=[]
        self.memory_status='active'
        def memory(config,env,root,repo,probe,cache):
            self.assertNotIn('plugins',config)
            self.assertEqual(root,self.root)
            self.assertEqual(repo,'repository-test')
            self.assertIs(probe,True)
            self.checked.append(env['home'])
            return self.memory_status
        scope=dict(vars(a), TEAM=self.names, matches=lambda *_:True,
            read_config=lambda _: {'memory':{'provider':'openviking'}},
            native_command=lambda *args,**kwargs:None,
            native_interactive_catalog=lambda: {'cli':{'required':['memory','kanban']}},
            default_interactive_platforms=lambda *_:['cli'],
            default_native_tool_drift=lambda *_:False,
            native_platform_tools=lambda *_:{'memory','kanban'},
            sys=sys, integration_scope={'env_for':lambda home,root:{'home':home}, 'memory_state':memory})
        exec(Path(__file__).with_name('dispatch.py').read_text(),scope)
        scope['native_command']=lambda *args,**kwargs:None
        self.check=scope['check_activation']
        modules={'gateway.config':types.SimpleNamespace(load_gateway_config=lambda:types.SimpleNamespace(profile_routes=[])),
                 'agent.secret_scope':types.SimpleNamespace(build_profile_secret_scope=lambda _:None,
                    set_secret_scope=lambda *args,**kwargs:None,reset_secret_scope=lambda _:None)}
        self.enterContext(patch.dict(sys.modules,modules))
        self.enterContext(patch.object(a.subprocess,'run'))
    def test_activation_checks_shared_memory_for_every_profile_without_supervision_plugins(self):
        self.check(self.root,self.payload)
        self.assertEqual(self.checked,[self.root if name=='default' else self.root/'profiles'/name for name in self.names])
    def test_unavailable_memory_still_blocks_activation(self):
        for status in ('configured','unavailable','drift'):
            self.memory_status=status
            with self.assertRaisesRegex(RuntimeError,'shared memory activation gate failed'):
                self.check(self.root,self.payload)
    def test_caller_readiness_does_not_replace_live_memory_probe(self):
        self.payload['integrations_ready']=False
        with self.assertRaisesRegex(RuntimeError,'required shared memory not ready'):
            self.check(self.root,self.payload)
        self.assertEqual(len(self.checked),6)

class NativeSwitchTest(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        sys.modules['yaml']=types.SimpleNamespace(safe_load=json.loads)
        scope={'__name__':'native_fixture'}
        for path in [Path(__file__).parents[1]/'gateway/state.py',Path(__file__).parents[1]/'gateway/dispatch.py',Path(__file__).with_name('dispatch.py')]:
            exec(path.read_text(),scope)
        self.scope=scope
        for name in scope['TEAM']:
            home=self.root if name=='default' else self.root/'profiles'/name
            home.mkdir(parents=True,exist_ok=True)
            config={'kanban':dict(scope['dispatch_policy'](name),dispatch_in_gateway=False)}
            (home/'config.yaml').write_text(json.dumps(config))
            (home/'SOUL.md').write_text(name)
            (home/'profile.yaml').write_text(json.dumps({'description':name}))
        with scope['sqlite3'].connect(self.root/'kanban.db') as db:
            db.execute('CREATE TABLE tasks (status TEXT, worker_pid INTEGER)')
            db.execute('CREATE TABLE task_runs(worker_pid INTEGER,worker_started_at TEXT)')
            db.execute("INSERT INTO tasks VALUES ('ready',NULL)")
        self.phase=0; self.calls=[]
        def observe(root):
            return dict(state='running',healthy=True,idle_verified=True,active=0,adapters=['telegram'],identity={'pid':10+self.phase})
        def native(*args,**kwargs):
            self.calls.append(args)
            if args[2:4]==('config','set'):
                path=self.root/'config.yaml'; config=json.loads(path.read_text())
                config['kanban'][args[4].split('.')[-1]]=args[5] if args[4]=='kanban.orchestrator_profile' else json.loads(args[5]); path.write_text(json.dumps(config))
            if args[2:]==('gateway','restart'):
                self.phase+=1
                (self.root/'logs').mkdir(exist_ok=True)
                with (self.root/'logs/gateway.log').open('a') as log:
                    log.write('kanban dispatcher: max_in_progress=1\nkanban dispatcher: embedded in gateway (interval=60.0s)\n')
        scope['native_command']=native; scope['observe_gateway']=observe
        scope['owns_dispatch_lock']=lambda root,pid: self.phase>0
        scope['time']=types.SimpleNamespace(sleep=lambda _:None)
        scope['native_worker_alive']=lambda pid,started:pid==42
    def test_ready_queue_survives_activation_and_specialists_remain_off(self):
        self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        config=json.loads((self.root/'config.yaml').read_text())
        self.assertTrue(self.scope['operational_policy'](config))
        self.assertIn(('-p','default','config','set','kanban.orchestrator_profile','default'),self.calls)
        marker=json.loads((self.root/self.scope['MARKER']).read_text())
        self.assertEqual(marker['identity'],{'pid':11})
        self.assertEqual(marker['dispatch']['max_in_progress'],1)
        self.assertEqual(sum(c[2:]==('gateway','restart') for c in self.calls),1)
        for role in self.scope['TEAM'][1:]:
            self.assertFalse(json.loads((self.root/'profiles'/role/'config.yaml').read_text())['kanban']['dispatch_in_gateway'])
        with self.scope['sqlite3'].connect(self.root/'kanban.db') as db:
            self.assertEqual(db.execute('SELECT status FROM tasks').fetchall(),[('ready',)])
    def test_active_worker_and_competing_tick_refuse_before_writes(self):
        with self.scope['sqlite3'].connect(self.root/'kanban.db') as db:
            db.execute("UPDATE tasks SET status='running',worker_pid=42")
        with self.assertRaises(RuntimeError):
            self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        self.assertEqual(self.calls,[])
        with (self.root/'kanban.db.dispatch.lock').open('rb') as lock:
            self.scope['fcntl'].flock(lock,self.scope['fcntl'].LOCK_EX|self.scope['fcntl'].LOCK_NB)
            with self.assertRaises(BlockingIOError):
                self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        self.assertEqual(self.calls,[])
    def test_terminal_run_with_live_pid_blocks_reconciliation(self):
        with self.scope['sqlite3'].connect(self.root/'kanban.db') as db:
            db.execute("UPDATE tasks SET status='done'")
            db.execute("INSERT INTO task_runs VALUES (42,'native-fingerprint')")
        with self.assertRaises(RuntimeError):
            self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        self.assertEqual(self.calls,[])
        # A recycled historical PID is not a live worker.
        self.scope['native_worker_alive']=lambda pid,started:False
        self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)

    def test_canary_failure_fences_future_claims_without_restarting_busy_worker(self):
        with self.scope['sqlite3'].connect(self.root/'kanban.db') as db:
            db.execute("UPDATE tasks SET status='running',worker_pid=42")
        self.scope['suspend_failed_activation'](self.root,list(self.scope['TEAM']),'repo')
        policy=json.loads((self.root/'config.yaml').read_text())['kanban']
        self.assertIs(policy['dispatch_in_gateway'],False)
        self.assertEqual(policy['dispatch_profiles'],[])
        self.assertFalse(any(c[2:]==('gateway','restart') for c in self.calls))

    def test_stopped_operational_gateway_can_prepare_without_restart(self):
        path=self.root/'config.yaml'; config=json.loads(path.read_text())
        config['kanban']['dispatch_in_gateway']=True; path.write_text(json.dumps(config))
        self.scope['observe_gateway']=lambda root:{'state':'not-running'}
        self.scope['Path']=lambda *parts:self.root if parts==('/opt/data',) else Path(*parts)
        with contextlib.redirect_stdout(io.StringIO()) as output:
            self.scope['dispatch_main']({'roles':[{'name':n} for n in self.scope['TEAM']], 'repo_id':'repo','action':'prepare'})
        self.assertIn('REPOKIT_DISPATCH=prepared',output.getvalue())
        self.assertFalse(json.loads(path.read_text())['kanban']['dispatch_in_gateway'])
        self.assertFalse(any(c[2:]==('gateway','restart') for c in self.calls))

    def test_canary_observes_native_run_pid_and_archives_without_dispatch_command(self):
        self.assert_canary_result('# RepoKit\n', 'RepoKit')

    def test_missing_readme_canary_still_requires_successful_worker_evidence(self):
        self.assert_canary_result(None, 'NO_MARKDOWN_TITLE')

    def test_setext_readme_canary_uses_explicit_no_markdown_title(self):
        self.assert_canary_result('RepoKit\n=======\n', 'NO_MARKDOWN_TITLE')

    def test_html_readme_canary_uses_explicit_no_markdown_title(self):
        self.assert_canary_result('<h1>RepoKit</h1>\n<p>Repository</p>\n', 'NO_MARKDOWN_TITLE')

    def test_canary_skips_empty_atx_heading(self):
        self.assert_canary_result('# \n# RepoKit\n', 'RepoKit')

    def test_fallback_canary_rejects_wrong_title_or_changed_files(self):
        for metadata in ({'title':'invented','changed_files':[]},
                         {'title':'NO_MARKDOWN_TITLE','changed_files':['README.md']}):
            with self.subTest(metadata=metadata):
                self.assert_canary_result(None, 'NO_MARKDOWN_TITLE', metadata, reject=True)

    def test_readme_safety_errors_are_not_mistaken_for_missing_title(self):
        self.assertTrue(callable(self.scope.get('canary_title')))
        readme=self.root/'README.md'
        readme.symlink_to(self.root/'absent')
        with self.assertRaises(OSError): self.scope['canary_title'](readme)
        readme.unlink()
        readme.write_bytes(b'x'*262145)
        with self.assertRaises(ValueError): self.scope['canary_title'](readme)
        readme.write_text('# RepoKit\n')
        with patch.object(self.scope['os'],'open',side_effect=PermissionError('fixture denied')):
            with self.assertRaises(PermissionError): self.scope['canary_title'](readme)

    def test_legacy_canary_body_is_accepted_only_for_terminal_history(self):
        task={'title':'RepoKit dispatcher canary','assignee':'researcher',
              'workspace_kind':'dir','workspace_path':'/workspace','created_by':'default',
              'body':'Read the repository README title. Do not modify files. Complete with the title in your summary '
                     'and structured metadata {"title": "the exact first Markdown heading without #", "changed_files": []}.',
              'max_runtime_seconds':180,'max_retries':1,'priority':100,'skills':[]}
        for status in ('ready','running'):
            self.assertFalse(self.scope['canary_task_matches'](dict(task,status=status)))
        for status in ('done','blocked'):
            self.assertTrue(self.scope['canary_task_matches'](dict(task,status=status)))

    def assert_canary_result(self, content, expected, metadata=None, reject=False):
        self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        native=self.scope['native_command']; records=[]
        run={'profile':'researcher','status':'running','worker_pid':123,'outcome':None,'ended_at':None}
        done=dict(run,status='done',outcome='completed',ended_at=12,summary=expected,
                  metadata=metadata if metadata is not None else {'title':expected,'changed_files':[]})
        pending=[{'task':{'status':'running'},'runs':[run]}, {'task':{'status':'done'},'runs':[done]}]
        def command(*args,**kwargs):
            records.append(args)
            if args[2:4]==('kanban','create'):
                self.assertNotEqual(args[args.index('--idempotency-key')+1],'repokit-dispatcher-canary')
                self.assertIn('NO_MARKDOWN_TITLE',args[args.index('--body')+1])
                return {'id':'t-canary','title':'RepoKit dispatcher canary','assignee':'researcher',
                        'workspace_kind':'dir','workspace_path':'/workspace','created_by':'default','status':'ready',
                        'body':args[args.index('--body')+1],
                        'max_runtime_seconds':180,'max_retries':1,'priority':100,'skills':[]}
            if args[2:4]==('kanban','show'): return pending.pop(0)
            if args[2:4]==('kanban','archive'): return None
            return native(*args,**kwargs)
        self.scope['native_command']=command
        self.scope['time']=types.SimpleNamespace(monotonic=time.monotonic,sleep=lambda _:None)
        read=self.scope['read_file']
        readme=self.root/'README.md'
        if content is not None: readme.write_text(content)
        self.scope['read_file']=lambda path,*args: read(readme if str(path)=='/workspace/README.md' else path,*args)
        original=Path.read_text
        def text(path,*args,**kwargs):
            return '123 (python) S 11 1 2' if str(path)=='/proc/123/stat' else original(path,*args,**kwargs)
        with patch.object(Path,'read_text',text),contextlib.redirect_stdout(io.StringIO()) as output:
            if reject:
                with self.assertRaisesRegex(RuntimeError,'successful no-write researcher evidence'):
                    self.scope['dispatcher_canary'](self.root,{'repo_id':'repo'})
            else:
                self.scope['dispatcher_canary'](self.root,{'repo_id':'repo'})
        if reject:
            self.assertFalse(any(args[2:4]==('kanban','archive') for args in records))
            self.assertNotIn('canary',json.loads((self.root/self.scope['MARKER']).read_text()))
            return
        self.assertIn('REPOKIT_CANARY=researcher-done',output.getvalue())
        self.assertTrue(any(args[2:4]==('kanban','archive') for args in records))
        self.assertFalse(any('dispatch' in args for args in records))
        marker=json.loads((self.root/self.scope['MARKER']).read_text())
        self.assertTrue(marker['canary']['gateway_spawn'])

    def test_current_verified_activation_does_not_restart_or_buy_another_canary(self):
        self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        marker=json.loads((self.root/self.scope['MARKER']).read_text())
        marker['canary']={'task_id':'prior','profile':'researcher','gateway_spawn':True,'done':True}
        self.scope['publish_marker'](self.root,marker)
        self.calls.clear()
        gates=[]
        self.scope['check_activation']=lambda root,payload:gates.append(True)
        self.scope['Path']=lambda *parts:self.root if parts==('/opt/data',) else Path(*parts)
        def unexpected(*args):
            self.fail('already proven live gateway restarted or canary repeated')
        self.scope['dispatcher_canary']=unexpected
        self.scope['switch_gateway']=unexpected
        with contextlib.redirect_stdout(io.StringIO()) as output:
            self.scope['dispatch_main']({'roles':[{'name':n} for n in self.scope['TEAM']], 'repo_id':'repo','action':'activate'})
        self.assertEqual(gates,[True])
        self.assertIn('REPOKIT_GATEWAY=current',output.getvalue())
        self.assertIn('REPOKIT_CANARY=researcher-done',output.getvalue())
        self.assertEqual(self.calls,[])

    def test_blocked_canary_history_is_preserved_during_retry(self):
        self.assert_canary_retry_chain(2)

    def test_retry_chain_accepts_ready_successor_at_exact_history_bound(self):
        self.assert_canary_retry_chain(32)

    def assert_canary_retry_chain(self, predecessor_count):
        self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        task={'id':'old','title':'RepoKit dispatcher canary','assignee':'researcher',
              'workspace_kind':'dir','workspace_path':'/workspace','created_by':'default','status':'blocked',
              'body':'Read the repository README title. Do not modify files. Complete with the title in your summary '
                     'and structured metadata {"title": "the exact first Markdown heading without #", "changed_files": []}.',
              'max_runtime_seconds':180,'max_retries':1,'priority':100,
              'model_override':None,'provider_override':None,'skills':[], 'completion_contract':None,
              'project_id':None,'branch_name':None,'tenant':None,'session_id':None,
              'workflow_template_id':None,'current_step_key':None}
        records=[]; archived=[]; keys=[]
        failed={'task':dict(task),'parents':[],'children':[],'comments':[],
                'runs':[{'profile':'researcher','status':'failed','outcome':'failed','ended_at':10}]}
        run={'profile':'researcher','status':'running','worker_pid':123,'outcome':None,'ended_at':None}
        done=dict(run,status='done',outcome='completed',ended_at=12,summary='RepoKit',metadata={'title':'RepoKit','changed_files':[]})
        pending=[{'task':{'status':'running'},'runs':[run]}, {'task':{'status':'done'},'runs':[done]}]
        def command(*args,**kwargs):
            records.append(args)
            if args[2:4]==('kanban','create'):
                key=args[args.index('--idempotency-key')+1]
                if key not in keys: keys.append(key)
                if len(keys)==1: return dict(task)
                if len(keys)<=predecessor_count: return dict(task,id='retry'+str(len(keys)),status='done')
                return dict(task,id='new',status='ready',body=args[args.index('--body')+1])
            if args[2:4]==('kanban','show'):
                if args[4]=='old': return failed
                if args[4].startswith('retry'):
                    return dict(failed,task=dict(task,id=args[4],status='done'),
                                runs=[dict(done,summary='Old heading',metadata={'title':'Old heading','changed_files':[]})])
                return pending.pop(0)
            if args[2:4]==('kanban','archive'):
                archived.append(args[4]); return
            self.fail('unexpected native command')
        self.scope['native_command']=command
        self.scope['time']=types.SimpleNamespace(monotonic=time.monotonic,sleep=lambda _:None)
        read=self.scope['read_file']
        self.scope['read_file']=lambda path,*args: b'# RepoKit\n' if str(path)=='/workspace/README.md' else read(path,*args)
        original=Path.read_text
        def text(path,*args,**kwargs):
            return '123 (python) S 11 1 2' if str(path)=='/proc/123/stat' else original(path,*args,**kwargs)
        with patch.object(Path,'read_text',text),contextlib.redirect_stdout(io.StringIO()) as output:
            self.scope['dispatcher_canary'](self.root,{'repo_id':'repo'})
        self.assertIn('REPOKIT_CANARY=researcher-done',output.getvalue())
        self.assertEqual(archived,['new'])
        self.assertEqual(failed['task']['status'],'blocked')
        self.assertFalse(any('dispatch' in args for args in records))
        self.assertEqual(json.loads((self.root/self.scope['MARKER']).read_text())['canary']['task_id'],'new')

    def test_retry_preserves_changed_linked_or_unfinished_canary(self):
        task={'id':'old','title':'RepoKit dispatcher canary','assignee':'researcher',
              'workspace_kind':'dir','workspace_path':'/workspace','created_by':'default','status':'blocked',
              'body':'Read the repository README title. Do not modify files. Complete with the title in your summary '
                     'and structured metadata {"title": "the exact first Markdown heading without #", "changed_files": []}.',
              'max_runtime_seconds':180,'max_retries':1,'priority':100,'skills':[]}
        for variant in ('body','children','parent','actor','active_run','active_worker','status','provider','result'):
            with self.subTest(variant=variant):
                record={'task':dict(task),'parents':[],'children':[],
                        'runs':[{'profile':'researcher','status':'failed','ended_at':10}]}
                if variant=='body': record['task']['body']='owner task'
                if variant=='children': record['children']=['owner-child']
                if variant=='parent': record['parents']=['owner-parent']
                if variant=='actor': record['runs'][0]['profile']='executor'
                if variant=='active_run': record['runs'][0]['ended_at']=None
                if variant=='status': record['task']['status']='running'
                if variant=='provider': record['task']['provider_override']='owner-provider'
                if variant=='result': record['task']['result']='Owner replacement result'
                with self.scope['sqlite3'].connect(self.root/'kanban.db') as db:
                    db.execute('DELETE FROM task_runs')
                    if variant=='active_worker': db.execute("INSERT INTO task_runs VALUES (42,'native-fingerprint')")
                def command(*args,**kwargs):
                    if args[2:4]==('kanban','show'): return record
                    self.fail('unsafe canary archive or mutation')
                self.scope['native_command']=command
                with self.assertRaises(RuntimeError):
                    self.scope['retry_dispatch_canary'](self.root,task)

    def test_retry_key_survives_gateway_replacement(self):
        task={'id':'old','title':'RepoKit dispatcher canary','assignee':'researcher',
              'workspace_kind':'dir','workspace_path':'/workspace','created_by':'default','status':'blocked',
              'body':'Read the repository README title. Do not modify files. Complete with the title in your summary '
                     'and structured metadata {"title": "the exact first Markdown heading without #", "changed_files": []}.',
              'max_runtime_seconds':180,'max_retries':1,'priority':100,'skills':[]}
        record={'task':task,'parents':[],'children':[],
                'runs':[{'profile':'researcher','status':'failed','ended_at':10}]}
        keys=[]
        def command(*args,**kwargs):
            if args[2:4]==('kanban','show'): return record
            if args[2:4]==('kanban','create'):
                keys.append(args[args.index('--idempotency-key')+1])
                self.assertFalse(keys[-1].startswith('repokit-dispatcher-canary-retry-'))
                return dict(task,id='queued-retry',status='ready',body=args[args.index('--body')+1])
            self.fail('retry must not mutate the original card')
        self.scope['native_command']=command
        for pid in (11,12):
            self.scope['observe_gateway']=lambda root:dict(identity={'pid':pid})
            self.scope['retry_dispatch_canary'](self.root,task)
        self.assertEqual(len(keys),2)
        self.assertEqual(keys[0],keys[1], 'a replacement gateway must reuse unfinished retry work')

    def test_current_canary_never_bypasses_failed_activation_gate(self):
        self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        marker=json.loads((self.root/self.scope['MARKER']).read_text())
        marker['canary']={'task_id':'prior','profile':'researcher','gateway_spawn':True,'done':True}
        self.scope['publish_marker'](self.root,marker)
        self.calls.clear()
        self.scope['Path']=lambda *parts:self.root if parts==('/opt/data',) else Path(*parts)
        def failed(*args): raise RuntimeError('required integration gate failed')
        self.scope['check_activation']=failed
        with self.assertRaisesRegex(RuntimeError,'required integration gate failed'),contextlib.redirect_stdout(io.StringIO()) as output:
            self.scope['dispatch_main']({'roles':[{'name':n} for n in self.scope['TEAM']], 'repo_id':'repo','action':'activate'})
        self.assertEqual(self.calls,[])
        self.assertNotIn('REPOKIT_GATEWAY=current',output.getvalue())

    def test_missing_startup_evidence_cannot_publish_current_receipt(self):
        self.scope['startup_since']=lambda *_:None
        with self.assertRaises(RuntimeError):
            self.scope['switch_gateway'](self.root,list(self.scope['TEAM']),'repo',True)
        self.assertFalse((self.root/self.scope['MARKER']).exists())
        self.assertIs(json.loads((self.root/'config.yaml').read_text())['kanban']['dispatch_in_gateway'],False)

if __name__=='__main__':unittest.main()

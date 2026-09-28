import importlib.util
import json
from pathlib import Path
import tempfile
import types
import sys
import unittest
from unittest.mock import patch
import fcntl
import time
import datetime
import sqlite3

sys.modules['yaml'] = types.SimpleNamespace(safe_load=json.loads)
spec = importlib.util.spec_from_file_location('state', Path(__file__).with_name('state.py'))
state = importlib.util.module_from_spec(spec)
if spec.loader:
    try:
        spec.loader.exec_module(state)
    except FileNotFoundError:
        pass

class GenerationTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.roles = ['default','researcher','planner','executor','reviewer','steward']
        for name in self.roles:
            home = self.root if name == 'default' else self.root/'profiles'/name
            home.mkdir(parents=True, exist_ok=True)
            (home/'SOUL.md').write_text('Identity for '+name)
            (home/'profile.yaml').write_text(json.dumps({'description':'Role '+name}))
            (home/'config.yaml').write_text(json.dumps({'toolsets':['kanban','memory'], 'kanban':{'dispatch_in_gateway':False}, 'model':{'api_key':'private-not-a-generation-input'}}))
    def digest(self):
        return state.generation(self.root, self.roles, 'repokit-fixture')
    def test_owner_plugins_do_not_change_managed_generation_or_get_modified(self):
        before=self.digest()
        path=self.root/'config.yaml'
        config=json.loads(path.read_text())
        config['plugins']={'enabled':['nerve','owner_plugin'], 'entries':{
            'nerve':{'settings':{'reflex_backend':'laya','private_key':'opaque'}},
            'owner_plugin':{'unknown':['untouched']}}}
        path.write_text(json.dumps(config))
        directory=self.root/'plugins'; directory.mkdir()
        metadata=directory/'.install-metadata.json'
        metadata.write_text(json.dumps({'nerve':{'revision':'owner-revision','source':'owner-source'}}))
        preserved={p:p.read_bytes() for p in (path,metadata)}
        self.assertEqual(before,self.digest())
        self.assertEqual(preserved,{p:p.read_bytes() for p in preserved})
        config['plugins']['enabled']=['repokit_maintenance']
        path.write_text(json.dumps(config))
        self.assertNotEqual(before,self.digest())

    def test_public_model_change_invalidates_live_generation(self):
        before=self.digest()
        path=self.root/'config.yaml'
        config=json.loads(path.read_text())
        config['model']['default']='different-model'
        path.write_text(json.dumps(config))
        self.assertNotEqual(before,self.digest())
    def test_all_boards_and_residual_worker_prevent_restart(self):
        def board(path):
            path.parent.mkdir(parents=True,exist_ok=True)
            with sqlite3.connect(path) as db:
                db.execute('CREATE TABLE tasks(status TEXT, worker_pid INTEGER)')
        board(self.root/'kanban.db')
        named=self.root/'kanban/boards/secondary/kanban.db'
        board(named)
        self.assertTrue(state.boards_quiescent(self.root))
        for status,pid in [('ready',None),('running',None),('review',None),('done',123)]:
            with sqlite3.connect(named) as db:
                db.execute('DELETE FROM tasks')
                db.execute('INSERT INTO tasks VALUES (?,?)',(status,pid))
            self.assertFalse(state.boards_quiescent(self.root),(status,pid))
        named.unlink()
        self.assertFalse(state.boards_quiescent(self.root))
    def test_managed_changes_invalidate_but_credentials_do_not_enter_generation(self):
        before = self.digest()
        p = self.root/'config.yaml'
        config=json.loads(p.read_text())
        config['model']['api_key']='changed-private-value'
        p.write_text(json.dumps(config))
        self.assertEqual(before,self.digest())
        config['platform_toolsets']={'telegram':['kanban']}
        p.write_text(json.dumps(config))
        self.assertNotEqual(before,self.digest())
        before=self.digest()
        (self.root/'profiles/researcher/SOUL.md').write_text('Updated managed identity')
        self.assertNotEqual(before,self.digest())
    def test_missing_or_redirected_input_cannot_get_generation(self):
        p=self.root/'profiles/reviewer/profile.yaml'
        p.unlink()
        with self.assertRaises(OSError): self.digest()
        p.symlink_to(self.root/'profile.yaml')
        with self.assertRaises(OSError): self.digest()
    def test_manual_policy_refuses_owner_dispatch_or_decomposition_drift(self):
        policy={'dispatch_in_gateway':False,'auto_decompose':False,'orchestrator_profile':'default','max_in_progress':1}
        self.assertTrue(state.manual_policy({'kanban':policy}))
        for key,value in [('dispatch_in_gateway',True),('auto_decompose',True),('orchestrator_profile','executor'),('max_in_progress',2),('max_in_progress',True)]:
            self.assertFalse(state.manual_policy({'kanban':dict(policy,**{key:value})}))
        self.assertFalse(state.manual_policy({}))

    def test_current_requires_matching_process_and_generation(self):
        running={'state':'running','identity':{'pid':42,'start':123,'boot':'boot-1','namespace':'pid:[1]'},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']}
        marker={'schema':1,'generation':'abc','identity':running['identity'],'adapters':['telegram']}
        self.assertEqual(state.classify('abc',running,marker),'current')
        self.assertEqual(state.classify('changed',running,marker),'stale')
        restarted=dict(running,identity=dict(running['identity'],start=456))
        self.assertEqual(state.classify('abc',restarted,marker),'unknown')
        self.assertEqual(state.classify('abc',dict(running,healthy=False),marker),'unknown')
        self.assertEqual(state.classify('abc',running,None),'unknown')
        self.assertEqual(state.classify('abc',dict(running,adapters=['discord']),marker),'unknown')
        self.assertEqual(state.classify('abc',{'state':'not-running'},marker),'not-running')
    def test_failed_restart_never_publishes_receipt_and_active_workers_prevent_restart(self):
        running={'state':'running','identity':{'pid':42},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']}
        writes=[]; restarts=[]
        def restart(): restarts.append(True)
        with self.assertRaises(RuntimeError):
            state.converge(lambda:'abc', lambda:running, lambda:False, restart, writes.append, attempts=2, pause=lambda:None)
        self.assertEqual(writes,[])
        self.assertEqual(restarts,[])
        with self.assertRaises(RuntimeError):
            state.converge(lambda:'abc', lambda:running, lambda:True, restart, writes.append, attempts=2, pause=lambda:None)
        self.assertEqual(len(restarts),1)
        self.assertEqual(writes,[])
    def test_single_restart_after_inputs_settle_and_idempotent_rerun(self):
        before={'state':'running','identity':{'pid':42},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']}
        after={'state':'running','identity':{'pid':43},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']}
        values=iter([before,before,after]); writes=[]; restarts=[]
        result=state.converge(lambda:'abc', lambda:next(values), lambda:True, lambda:restarts.append(True), writes.append, pause=lambda:None)
        self.assertEqual(result,'current')
        self.assertEqual(len(restarts),1)
        self.assertEqual(writes[0]['identity'],after['identity'])
        result=state.converge(lambda:'abc', lambda:after, lambda:True, lambda:restarts.append(True), writes.append, marker=writes[0], pause=lambda:None)
        self.assertEqual(result,'current')
        self.assertEqual(len(restarts),1)
    def test_stale_idle_counters_and_work_started_during_board_query_refuse_restart(self):
        def forbidden(): self.fail('unsafe restart')
        stale={'state':'running','identity':{'pid':1},'healthy':False,'active':0,'idle_verified':False}
        with self.assertRaises(RuntimeError):
            state.converge(lambda:'abc',lambda:stale,lambda:True,forbidden,lambda _:None)
        initial=dict(stale,healthy=True,idle_verified=True,adapters=["telegram"])
        busy=dict(initial,active=1,idle_verified=False)
        states=iter([initial,busy])
        with self.assertRaises(RuntimeError):
            state.converge(lambda:'abc',lambda:next(states),lambda:True,forbidden,lambda _:None)
        with self.assertRaises(StopIteration): next(states)

    def test_stopped_gateway_is_not_started_and_changed_inputs_refuse_receipt(self):
        def forbidden(): self.fail('must not restart/write')
        self.assertEqual(state.converge(lambda:'abc', lambda:{'state':'not-running'},lambda:True,forbidden,lambda _:forbidden()),'not-running')
        fingerprints=iter(['abc','changed'])
        states=iter([{'state':'running','identity':{'pid':1},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']},{'state':'running','identity':{'pid':1},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']},{'state':'running','identity':{'pid':2},'healthy':True,'active':0,'idle_verified':True,'adapters':['telegram']}])
        with self.assertRaises(RuntimeError):
            state.converge(lambda:next(fingerprints),lambda:next(states),lambda:True,lambda:None,lambda _:forbidden(),pause=lambda:None)

class NativeStatusTest(unittest.TestCase):
    def test_native_health_ignores_old_rows_but_rejects_current_failure_and_stale_idle(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory); (root/'state').mkdir()
            identity={'pid':42,'start':123,'boot':'boot','namespace':'pid:[1]'}
            record={'pid':42,'start_time':123,'kind':'hermes-gateway','hermes_home':str(root)}
            now=datetime.datetime.now(datetime.timezone.utc).isoformat()
            current=dict(record,gateway_state='running',restart_requested=False,active_agents=0,
                session_store={'status':'ok'},updated_at=now,
                platforms={'telegram':{'state':'connected','needs_attention':False,'writer_pid':42,'writer_start_time':123},
                'old':{'state':'failed','writer_pid':3,'writer_start_time':4},
                'disabled':{'state':'disabled','writer_pid':42,'writer_start_time':123}})
            for name in ('gateway.pid','gateway.lock'):
                (root/name).write_text(json.dumps(record))
            (root/'gateway_state.json').write_text(json.dumps(current))
            (root/'state/gateway.heartbeat').write_text(json.dumps({'pid':42,'updated_at':now}))
            with (root/'gateway.lock').open('rb') as lock, patch.object(state,'process_identity',return_value=identity), patch.object(state,'loop_alive',return_value=True):
                fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
                observed=state.observe_gateway(root)
                self.assertTrue(observed['healthy'])
                self.assertTrue(observed['idle_verified'])
                self.assertEqual(observed['adapters'],['telegram'])
                current['platforms']['telegram']['state']='retrying'
                (root/'gateway_state.json').write_text(json.dumps(current))
                self.assertFalse(state.observe_gateway(root)['healthy'])
                current['updated_at']='2000-01-01T00:00:00+00:00'
                (root/'gateway_state.json').write_text(json.dumps(current))
                self.assertFalse(state.observe_gateway(root)['idle_verified'])
                current['session_store']=['invalid']
                (root/'gateway_state.json').write_text(json.dumps(current))
                self.assertEqual(state.observe_gateway(root)['state'],'unknown')
            with patch.object(state,'process_identity',return_value=identity):
                self.assertEqual(state.observe_gateway(root)['state'],'unknown')

if __name__=='__main__': unittest.main()

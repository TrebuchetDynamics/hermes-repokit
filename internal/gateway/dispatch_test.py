import importlib.util
import unittest
import tempfile
import os
import fcntl
from pathlib import Path

spec=importlib.util.spec_from_file_location('dispatch',Path(__file__).with_name('dispatch.py'))
d=importlib.util.module_from_spec(spec)
if spec.loader:
    try: spec.loader.exec_module(d)
    except FileNotFoundError: pass

class DispatchTest(unittest.TestCase):
    def test_activation_policy_is_bounded_and_workers_never_own_dispatch(self):
        self.assertEqual(d.dispatch_policy('default'),dict(dispatch_in_gateway=True,auto_decompose=False,
            review_dispatch=True,max_in_progress=1,orchestrator_profile='default',dispatch_profiles=list(d.TEAM)))
        for role in d.TEAM[1:]:
            self.assertIs(d.dispatch_policy(role)['dispatch_in_gateway'],False)
        config={'kanban':d.dispatch_policy('default')}
        self.assertTrue(d.operational_policy(config))
        for key,value in [('dispatch_profiles',[]),('dispatch_profiles',list(d.TEAM)+['stranger']),
                          ('review_dispatch',False),('auto_decompose',True),('max_in_progress',True)]:
            self.assertFalse(d.operational_policy({'kanban':dict(config['kanban'],**{key:value})}))

    def test_live_requires_current_process_locked_and_startup_observed(self):
        receipt={'dispatch':{'max_in_progress':1,'interval':60}}
        self.assertEqual(d.dispatch_live(True,'current',True,receipt,False),'enabled')
        for generation,owned,marker,paused in [('stale',True,receipt,False),('current',False,receipt,False),
                ('current',True,{},False),('current',True,receipt,True)]:
            self.assertNotEqual(d.dispatch_live(True,generation,owned,marker,paused),'enabled')
        self.assertEqual(d.dispatch_live(False,'current',False,{},False),'disabled')
        self.assertEqual(d.dispatch_live(False,'current',True,receipt,False),'stale')

    def test_singleton_ownership_uses_kernel_lock_not_open_file(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory); (root/'kanban').mkdir()
            lock=root/'kanban/.dispatcher.lock'; lock.touch()
            with lock.open('rb') as handle:
                self.assertFalse(d.owns_dispatch_lock(root,os.getpid()))
                fcntl.flock(handle,fcntl.LOCK_EX|fcntl.LOCK_NB)
                self.assertTrue(d.owns_dispatch_lock(root,os.getpid()))
                self.assertFalse(d.owns_dispatch_lock(root,os.getpid()+100000000))
                fcntl.flock(handle,fcntl.LOCK_UN)
                self.assertFalse(d.owns_dispatch_lock(root,os.getpid()))

    def test_worker_pid_comes_from_native_run_not_task_projection(self):
        record={'task':{'status':'running'},'runs':[{'status':'running','profile':'researcher','worker_pid':123}]}
        self.assertEqual(d.canary_worker_pid(record),123)
        self.assertIsNone(d.canary_worker_pid({'task':{'status':'running','worker_pid':456},'runs':[]}))
        record['runs'][-1]['profile']='executor'
        self.assertIsNone(d.canary_worker_pid(record))

    def test_startup_cannot_accept_other_concurrency_or_missing_loop(self):
        good='kanban dispatcher: max_in_progress=1\nkanban dispatcher: embedded in gateway (interval=60.0s)\n'
        self.assertEqual(d.startup_settings(good),{'max_in_progress':1,'interval':60})
        for text in ['',good.replace('=1','=10'),good.splitlines()[0],good+'kanban dispatcher: max_in_progress=2\n']:
            self.assertIsNone(d.startup_settings(text))

    def test_canary_requires_successful_researcher_with_structured_no_write_evidence(self):
        run={'id':7,'profile':'researcher','status':'done','outcome':'completed','ended_at':123,'metadata':{'title':'RepoKit','changed_files':[]},'summary':'RepoKit'}
        record={'task':{'status':'done'},'runs':[run]}
        self.assertTrue(d.canary_complete(record,'RepoKit'))
        for key,value in [('profile','default'),('status','failed'),('metadata',{'title':'wrong','changed_files':[]}),('metadata',{'title':'RepoKit','changed_files':['README.md']})]:
            self.assertFalse(d.canary_complete(dict(record,runs=[dict(run,**{key:value})]),'RepoKit'))

if __name__=='__main__': unittest.main()

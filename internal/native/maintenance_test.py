import importlib.util
import json
from pathlib import Path
import shutil
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

sys.modules['yaml']=types.SimpleNamespace(safe_load=json.loads)
spec=importlib.util.spec_from_file_location('maintenance',Path(__file__).with_name('maintenance.py'))
maintenance=importlib.util.module_from_spec(spec)
spec.loader.exec_module(maintenance)

class MaintenanceTest(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        self.name='repokit_maintenance'
        self.files={'__init__.py':'# fixture\n','plugin.yaml':'name: repokit_maintenance\n'}
        self.config={'platform_toolsets':{'cli':['file'],'telegram':['web'],'api_server':['web']}}
        (self.root/'config.yaml').write_text(json.dumps(self.config))
        self.calls=[]
        self.addCleanup(patch.stopall)
        patch.object(maintenance,'native_interactive_catalog',return_value={'cli':{},'telegram':{}},create=True).start()
        patch.object(maintenance,'default_interactive_platforms',return_value=['cli','telegram'],create=True).start()
        patch.object(maintenance,'native_scan',return_value=True).start()
    def native_run(self,*args):
        self.calls.append(args)
        config=json.loads((self.root/'config.yaml').read_text())
        if args[2:4]==('plugins','install'):
            source=Path(args[4].removeprefix('file://'))
            target=self.root/'plugins'/self.name
            target.parent.mkdir(exist_ok=True)
            shutil.copytree(source,target)
        elif args[2:4]==('plugins','enable'):
            config.setdefault('plugins',{})['enabled']=[self.name]
        elif args[2:4]==('tools','enable'):
            config['platform_toolsets'][args[-1]].append(self.name)
        (self.root/'config.yaml').write_text(json.dumps(config))
    def apply(self):
        maintenance.provision_maintenance(self.root,self.files,self.name,self.native_run)
    def test_native_install_is_immutable_scanned_and_idempotent(self):
        self.apply()
        install=next(c for c in self.calls if c[2:4]==('plugins','install'))
        self.assertIn('--no-enable',install)
        self.assertNotIn('--force',install)
        self.assertRegex(install[install.index('--ref')+1],r'^[0-9a-f]{40}$')
        self.assertTrue(install[4].startswith(self.root.as_uri()+'/'))
        self.assertEqual(len(list(self.root.glob('.repokit-maintenance-*'))),0)
        self.assertTrue(all(c[1]=='default' for c in self.calls))
        self.assertEqual([c[-1] for c in self.calls if c[2:4]==('tools','enable')],['cli','telegram'])
        before=len(self.calls)
        self.apply()
        self.assertEqual(len(self.calls),before)
        self.assertEqual(maintenance.native_scan.call_count,2)
    def test_owner_source_drift_is_preserved_before_commands(self):
        self.apply()
        target=self.root/'plugins'/self.name/'__init__.py'
        target.write_text('# owner changed\n')
        self.calls.clear()
        with self.assertRaisesRegex(RuntimeError,'drift'):
            self.apply()
        self.assertEqual(self.calls,[])
        self.assertEqual(target.read_text(),'# owner changed\n')
    def test_disabled_scanner_and_nonadmission_are_fail_closed(self):
        for result in (False,None):
            with self.subTest(result=result),patch.object(maintenance,'native_scan',return_value=result):
                with self.assertRaisesRegex(RuntimeError,'scan'):
                    self.apply()
                self.assertEqual(self.calls,[])
    def test_native_command_zero_exit_without_install_is_rejected(self):
        with self.assertRaisesRegex(RuntimeError,'installation'):
            maintenance.provision_maintenance(self.root,self.files,self.name,lambda *args:None)
    def test_unexpected_source_and_symlinks_are_drift_but_bytecode_is_not(self):
        self.apply()
        target=self.root/'plugins'/self.name
        cache=target/'__pycache__'
        cache.mkdir()
        (cache/'__init__.cpython-312.pyc').write_bytes(b'cache')
        self.apply()
        (cache/'owner.py').write_text('# executable extra')
        with self.assertRaisesRegex(RuntimeError,'drift'):
            self.apply()
        (cache/'owner.py').unlink()
        (target/'extra.py').symlink_to(target/'__init__.py')
        with self.assertRaisesRegex(RuntimeError,'drift'):
            self.apply()
    def test_git_revision_ignores_owner_identity_and_timestamps(self):
        first=self.root/'first'; second=self.root/'second'
        first.mkdir(); second.mkdir()
        for path in (first,second):
            for name,body in self.files.items(): (path/name).write_text(body)
        with patch.dict(maintenance.os.environ,{'GIT_AUTHOR_NAME':'Owner','GIT_AUTHOR_EMAIL':'owner@example.invalid','GIT_AUTHOR_DATE':'now','GIT_CONFIG_COUNT':'1','GIT_CONFIG_KEY_0':'core.hooksPath','GIT_CONFIG_VALUE_0':'/invalid'}):
            self.assertEqual(maintenance.stage_revision(first),maintenance.stage_revision(second))

class NativeScanTest(unittest.TestCase):
    def test_disabled_scanner_is_not_an_admission_receipt(self):
        native=types.ModuleType('hermes_cli.plugins_cmd')
        native._scan_on_install_enabled=lambda:False
        native._scan_plugin_tree=lambda *args,**kwargs:self.fail('disabled scanner called')
        with patch.dict(sys.modules,{'hermes_cli.plugins_cmd':native}):
            self.assertIs(maintenance.native_scan(Path('/fixture')),False)

    def test_native_scan_has_no_force_review_pin_or_prompt_override(self):
        native=types.ModuleType('hermes_cli.plugins_cmd')
        native._scan_on_install_enabled=lambda:True
        calls=[]
        def scan(*args,**kwargs):
            calls.append((args,kwargs))
            return types.SimpleNamespace(verdict='safe')
        native._scan_plugin_tree=scan
        with patch.dict(sys.modules,{'hermes_cli.plugins_cmd':native}):
            self.assertIs(maintenance.native_scan(Path('/fixture')),True)
        self.assertEqual(calls, [((Path('/fixture'),'RepoKit bundled maintenance'),{'force':False})])

if __name__=='__main__':unittest.main()

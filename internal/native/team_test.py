import importlib.util
import json
from pathlib import Path
import shutil
import sys
import tempfile
import types
import unittest

# Offline fixtures are JSON (a YAML subset); the real image test uses PyYAML.
sys.modules['yaml'] = types.SimpleNamespace(safe_load=json.loads, safe_dump=lambda x, **kw: json.dumps(x))
spec = importlib.util.spec_from_file_location('team', Path(__file__).with_name('team.py'))
team = importlib.util.module_from_spec(spec)
spec.loader.exec_module(team)

class ProvisionTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root / 'profiles').mkdir()
        (self.root / 'memories').mkdir()
        (self.root / 'memories/MEMORY.md').write_text('default private history')
        (self.root / 'memories/USER.md').write_text('default user history')
        (self.root / 'SOUL.md').write_text('native default')
        (self.root / '.env').write_text('TEST_API_KEY=fixture')
        (self.root / 'config.yaml').write_text(json.dumps({'model': {'default':'fixture', 'provider':'custom'}, 'kanban': {'dispatch_in_gateway':False}}))
        self.roles = [{'name':n, 'soul':'# Role: '+n, 'description':'description '+n, 'toolsets':['memory']} for n in ['default','researcher','planner','executor','reviewer','steward']]
        config=json.loads((self.root/'config.yaml').read_text())
        for key,value in team.expected(self.roles[0]).items():
            obj=config
            parts=key.split('.')
            for part in parts[:-1]: obj=obj.setdefault(part,{})
            obj[parts[-1]]=value
        (self.root/'config.yaml').write_text(json.dumps(config))
        self.calls = []
        self.fail_config = False
    def native_run(self, *args):
        self.calls.append(args)
        if args[:2] == ('profile','create'):
            self.assertIn('--clone', args)
            self.assertNotIn('--clone-all', args)
            home = self.root/'profiles'/args[2]
            home.mkdir()
            for p in ['config.yaml','.env','SOUL.md']:
                shutil.copyfile(self.root/p, home/p)
            shutil.copytree(self.root/'memories', home/'memories')
            (home/'profile.yaml').write_text(json.dumps({'description':args[-1]}))
        elif args[0] == '-p':
            home = self.root if args[1]=='default' else self.root/'profiles'/args[1]
            if self.fail_config: raise RuntimeError('secret failure')
            config = json.loads((home/'config.yaml').read_text())
            parts=args[4].split('.')
            obj=config
            for part in parts[:-1]: obj=obj.setdefault(part,{})
            obj[parts[-1]]=json.loads(args[5]) if args[5].startswith(('[','{')) or args[5] in ('false','true','1') else args[5]
            (home/'config.yaml').write_text(json.dumps(config))
        elif args[:2] == ('profile','describe'):
            home=self.root if args[2]=='default' else self.root/'profiles'/args[2]
            (home/'profile.yaml').write_text(json.dumps({'description':args[-1]}))
    def apply(self):
        return team.provision(self.root, self.roles, self.native_run, native_default_soul='native default')
    def test_native_clone_replaces_identity_and_preserves_credentials(self):
        self.assertEqual(self.apply(), [])
        self.assertTrue(any(c[-2:] == ('terminal.cwd', '/workspace') for c in self.calls))
        for role in self.roles[1:]:
            p=self.root/'profiles'/role['name']
            self.assertEqual((p/'SOUL.md').read_text(), role['soul'])
            self.assertEqual((p/'.env').read_text(), 'TEST_API_KEY=fixture')
            self.assertFalse((p/'memories/MEMORY.md').exists())
            self.assertFalse((p/'memories/USER.md').exists())
        self.assertEqual((self.root/'memories/MEMORY.md').read_text(),'default private history')
    def test_rerun_preserves_role_memory_and_user_drift_and_unknown_profiles(self):
        self.apply()
        p=self.root/'profiles/executor'
        (p/'SOUL.md').write_text('owner soul')
        (p/'memories/MEMORY.md').write_text('role lesson')
        other=self.root/'profiles/builder'
        other.mkdir()
        (other/'marker').write_text('owner')
        before={str(x.relative_to(p)):x.read_bytes() for x in p.rglob('*') if x.is_file()}
        self.assertIn('executor',self.apply())
        self.assertEqual(before,{str(x.relative_to(p)):x.read_bytes() for x in p.rglob('*') if x.is_file()})
        self.assertEqual((other/'marker').read_text(),'owner')
    def test_failure_preserves_new_native_profile_without_default_memories(self):
        self.fail_config=True
        with self.assertRaises(RuntimeError): self.apply()
        p=self.root/'profiles/researcher'
        self.assertEqual((p/'SOUL.md').read_text(),self.roles[1]['soul'])
        self.assertFalse((p/'memories/MEMORY.md').exists())
        self.assertEqual([x.name for x in (self.root/'profiles').iterdir()],['researcher'])
    def test_unknown_default_soul_preserved(self):
        (self.root/'SOUL.md').write_text('owner custom identity')
        self.assertIn('default', self.apply())
        self.assertEqual((self.root/'SOUL.md').read_text(),'owner custom identity')
    def test_exact_native_docker_soul_newline_is_adopted(self):
        (self.root/'SOUL.md').write_text('native default\n')
        self.assertEqual(self.apply(), [])
        self.assertEqual((self.root/'SOUL.md').read_text(), self.roles[0]['soul'])
    def test_existing_default_config_and_description_are_not_overwritten(self):
        config=json.loads((self.root/'config.yaml').read_text())
        config['terminal']['cwd']='/owner'
        (self.root/'config.yaml').write_text(json.dumps(config))
        (self.root/'profile.yaml').write_text(json.dumps({'description':'owner routing'}))
        before=(self.root/'config.yaml').read_bytes()
        self.assertIn('default',self.apply())
        self.assertEqual((self.root/'config.yaml').read_bytes(),before)
        self.assertEqual(json.loads((self.root/'profile.yaml').read_text())['description'],'owner routing')
    def test_live_dispatch_refused_before_cloning(self):
        (self.root/'config.yaml').write_text(json.dumps({'kanban':{'dispatch_in_gateway':True}}))
        with self.assertRaises(RuntimeError): self.apply()
        self.assertEqual(self.calls,[])

if __name__=='__main__': unittest.main()

import importlib.util
import json
from pathlib import Path
import sys
import types
import unittest
sys.modules['yaml']=types.SimpleNamespace(safe_load=json.loads)
spec=importlib.util.spec_from_file_location('probe',Path(__file__).with_name('development_probe.py'))
probe=importlib.util.module_from_spec(spec);spec.loader.exec_module(probe)
class DevelopmentProbeTest(unittest.TestCase):
    def test_versions_use_fixed_tools_clean_environment_and_never_run_project_code(self):
        calls=[]
        def run(command,**kwargs):
            calls.append((command,kwargs))
            return types.SimpleNamespace(returncode=0,stdout='fixture 1.2\n')
        result=probe.inspect_tools(True,run)
        self.assertTrue(all(row['ok'] for row in result.values()))
        self.assertIn('go',result)
        self.assertIn('compose',result)
        for command,kwargs in calls:
            self.assertTrue(command[0].startswith(('/usr/bin/','/usr/local/')))
            self.assertEqual(kwargs['cwd'],'/workspace')
            self.assertEqual(kwargs['env']['HOME'],'/nonexistent')
            self.assertFalse(any(k.endswith(('TOKEN','KEY')) for k in kwargs['env']))
            self.assertNotIn('hermes',command)
            self.assertNotIn('test',command)
    def test_missing_compiler_is_degraded_not_hidden_by_other_tools(self):
        def run(command,**kwargs):
            if command[0].endswith('/go'):raise FileNotFoundError('missing')
            return types.SimpleNamespace(returncode=0,stdout='fixture 1.2\n')
        result=probe.inspect_tools(True,run)
        self.assertFalse(result['go']['ok'])
        self.assertTrue(result['git']['ok'])
    def test_failed_or_unprintable_output_is_not_echoed(self):
        def run(command,**kwargs):return types.SimpleNamespace(returncode=1,stdout='token=private!\n')
        result=probe.inspect_tools(False,run)
        self.assertNotIn('go',result)
        self.assertTrue(all(not row['ok'] and row['version']=='unrecognized' for row in result.values()))
if __name__=='__main__':unittest.main()

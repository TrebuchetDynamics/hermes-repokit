import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import types
import unittest

sys.modules['yaml']=types.SimpleNamespace(safe_load=json.loads)
spec=importlib.util.spec_from_file_location('probe',Path(__file__).with_name('kanban_probe.py'))
probe=importlib.util.module_from_spec(spec)
exec((Path(__file__).parents[1]/'team/kanban.py').read_text(),probe.__dict__)
spec.loader.exec_module(probe)

class KanbanProbeTest(unittest.TestCase):
    def test_saved_platform_missing_opt_in_is_drift_and_observation_is_read_only(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            p=root/'config.yaml'
            data=json.dumps({'toolsets':['kanban'],'platform_toolsets':{'cli':['kanban'],'telegram':[]}})
            p.write_text(data)
            result=probe.observe(root)
            self.assertEqual(result['kanban']['telegram'],'missing')
            self.assertEqual(result['kanban']['cli'],'enabled')
            self.assertNotIn('discord', result['kanban'])
            self.assertEqual(p.read_text(),data)
            p.write_text(json.dumps({'toolsets':['kanban'],'agent':{'disabled_toolsets':['kanban']}}))
            self.assertEqual(probe.observe(root)['kanban']['fallback'],'missing')
    def test_required_tools_reconcile_every_existing_selection_and_disabled_list(self):
        config={'toolsets':['web'], 'platform_toolsets':{'telegram':['web'], 'discord':[], 'custom':"['file','memory']"}, 'agent':{'disabled_toolsets':'kanban,memory,terminal'}}
        updates=probe.default_kanban_updates(config)
        self.assertEqual(updates, {'toolsets':['web','kanban','memory'], 'platform_toolsets.telegram':['web','kanban','memory'], 'platform_toolsets.discord':['kanban','memory'], 'platform_toolsets.custom':['file','memory','kanban'], 'agent.disabled_toolsets':['terminal']})
        self.assertEqual(config['toolsets'],['web'])
        self.assertNotIn('platform_toolsets.cli',updates)
        for key,value in updates.items():
            if '.' in key:
                parent,child=key.split('.',1)
                config[parent][child]=value
            else: config[key]=value
        self.assertEqual(probe.default_kanban_updates(config),{})
        self.assertEqual(probe.worker_kanban_updates(config)['platform_toolsets.telegram'],['web','memory'])

    def test_memory_drift_and_manual_dispatch_are_observed_without_changes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            p=root/'config.yaml'
            config={'toolsets':['kanban','memory'], 'platform_toolsets':{'telegram':['kanban']}, 'kanban':{'dispatch_in_gateway':False}}
            for dispatch,want in [(False,'manual'),(True,'enabled'),('false','invalid'),(None,'invalid')]:
                config['kanban']['dispatch_in_gateway']=dispatch
                data=json.dumps(config)
                p.write_text(data)
                result=probe.observe(root)
                self.assertEqual(result['memory']['telegram'],'missing')
                self.assertNotIn('discord', result['memory'])
                self.assertEqual(result['memory']['cli'],'unknown')
                self.assertEqual(result['memory']['fallback'],'not-configured')
                self.assertEqual(result['kanban']['telegram'],'enabled')
                self.assertEqual(result['dispatch'],want)
                self.assertEqual(p.read_text(),data)
            config.pop('kanban')
            config['agent']={'disabled_toolsets':['memory']}
            p.write_text(json.dumps(config))
            self.assertEqual(probe.observe(root)['memory']['fallback'],'not-configured')
            self.assertEqual(probe.observe(root)['dispatch'],'missing')

    def test_symlinked_config_is_not_read(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            (root/'config.yaml').symlink_to('/dev/zero')
            with self.assertRaises(OSError): probe.observe(root)

    def test_discovery_merges_nested_declarations_and_does_not_enable_explicitly_disabled_optional_channels(self):
        catalog = dict.fromkeys(['cli', 'telegram', 'discord', 'slack', 'signal', 'api_server'])
        config = {'gateway': {'platforms': {'telegram': {'enabled': True}},
                              'discord': {'enabled': True}, 'slack': {'enabled': True}},
                  'platforms': {'telegram': {'enabled': False}},
                  'slack': {'enabled': False}, 'signal': {'enabled': 'true'},
                  'platform_toolsets': {'api_server': ['memory']}}
        self.assertEqual(probe.configured_interactive_platforms(config, catalog), ['cli', 'discord', 'signal'])
        result = probe.default_kanban_states(config)
        self.assertEqual(result['telegram'], 'not-configured')
        self.assertEqual(result['slack'], 'not-configured')
        self.assertNotIn('api_server', result)
        self.assertNotIn('sms', result)

    def test_memory_does_not_inherit_global_opt_in_on_configured_channel(self):
        config = {'toolsets': ['kanban', 'memory'], 'gateway': {'telegram': {'enabled': True}}}
        self.assertEqual(probe.default_kanban_states(config)['telegram'], 'fallback')
        self.assertEqual(probe.default_memory_states(config)['telegram'], 'unknown')
        config['platform_toolsets'] = {'telegram': ['hermes-telegram']}
        self.assertEqual(probe.default_memory_states(config)['telegram'], 'unknown')

if __name__=='__main__': unittest.main()

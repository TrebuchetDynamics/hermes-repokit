import importlib.util
import contextlib
import io
import logging
from enum import Enum
import json
from pathlib import Path
import shutil
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

# Offline fixtures are JSON (a YAML subset); the real image test uses PyYAML.
sys.modules['yaml'] = types.SimpleNamespace(safe_load=json.loads, safe_dump=lambda x, **kw: json.dumps(x))
spec = importlib.util.spec_from_file_location('team', Path(__file__).with_name('team.py'))
team = importlib.util.module_from_spec(spec)
exec((Path(__file__).parents[1]/'team/kanban.py').read_text(),team.__dict__)
spec.loader.exec_module(team)

class ProvisionTest(unittest.TestCase):
    def test_historical_missing_backend_and_executor_tools_upgrade_without_memory_reset(self):
        self.apply()
        home=self.root/'profiles/executor'
        (home/'memories/MEMORY.md').write_text('executor lesson')
        for role in self.roles:
            target=self.root if role['name']=='default' else self.root/'profiles'/role['name']
            config=json.loads((target/'config.yaml').read_text())
            config['terminal'].pop('backend',None)
            (target/'config.yaml').write_text(json.dumps(config))
        self.roles[3]['legacy_toolsets']=list(self.roles[3]['toolsets'])
        self.roles[3]['toolsets']=['memory','code_execution','skills']
        self.assertEqual(self.apply(),[])
        self.assertEqual((home/'memories/MEMORY.md').read_text(),'executor lesson')
        for role in self.roles:
            target=self.root if role['name']=='default' else self.root/'profiles'/role['name']
            self.assertEqual(team.read_config(target)['terminal']['backend'],'local')
        self.assertEqual(self.apply(),[])

    def test_owner_backend_is_preserved_as_drift(self):
        self.apply()
        home=self.root/'profiles/executor'
        config=json.loads((home/'config.yaml').read_text())
        config['terminal']['backend']='ssh'
        (home/'config.yaml').write_text(json.dumps(config))
        self.assertIn('executor',self.apply())
        self.assertEqual(team.read_config(home)['terminal']['backend'],'ssh')

    def test_default_completion_delivery_reconciled_natively(self):
        config=json.loads((self.root/'config.yaml').read_text())
        config['kanban']['auto_subscribe_on_create']=False
        config['kanban']['notify_in_gateway']=False
        (self.root/'config.yaml').write_text(json.dumps(config))
        team.reconcile_default_kanban(self.root,self.native_run)
        changed=json.loads((self.root/'config.yaml').read_text())
        self.assertIs(changed['kanban']['auto_subscribe_on_create'],True)
        self.assertIs(changed['kanban']['notify_in_gateway'],True)

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
        self.roles[0]['toolsets']=['kanban','memory']
        config=json.loads((self.root/'config.yaml').read_text())
        for key,value in team.expected(self.roles[0]).items():
            obj=config
            parts=key.split('.')
            for part in parts[:-1]: obj=obj.setdefault(part,{})
            obj[parts[-1]]=value
        (self.root/'config.yaml').write_text(json.dumps(config))
        self.catalog = {p: {'preset': 'hermes-'+p, 'required': sorted(['file','terminal','web','kanban','memory'] + (['discord'] if p == 'discord' else []))} for p in ('cli','telegram','discord','slack')}
        self.addCleanup(patch.stopall)
        patch.object(team, 'native_interactive_catalog', return_value=self.catalog, create=True).start()
        patch.object(team, 'native_platform_tools', side_effect=self.resolved_tools, create=True).start()
        patch.object(team, 'native_default_enabled_platforms', return_value=[], create=True).start()
        config=json.loads((self.root/'config.yaml').read_text())
        config['platform_toolsets']['cli']=self.catalog['cli']['required']
        (self.root/'config.yaml').write_text(json.dumps(config))
        self.calls = []
        self.fail_config = False
    def resolved_tools(self, config, platform):
        selection=team.kanban_selections(config).get(platform, [self.catalog[platform]['preset']])
        enabled=set(selection)
        for row in self.catalog.values():
            if row['preset'] in enabled:
                enabled.remove(row['preset'])
                enabled.update(set(row['required']) - {'kanban'})
        return enabled - set(team.kanban_disabled(config))

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
            if args[2:4] == ('tools','enable'):
                platform=args[-1]
                enabled=self.resolved_tools(config,platform) | set(args[4:-2])
                config.setdefault('platform_toolsets',{})[platform]=sorted(enabled)
                config.setdefault('known_builtin_toolsets',{})[platform]=['fixture-native-bookkeeping']
                if 'agent' in config:
                    config['agent']['disabled_toolsets']=[v for v in team.kanban_disabled(config) if v not in enabled]
                (home/'config.yaml').write_text(json.dumps(config))
                return
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

    def test_default_preset_parity_preserves_additions_and_uses_native_tools(self):
        config=json.loads((self.root/'config.yaml').read_text())
        config['toolsets']=['memory','web']
        config['platform_toolsets']={'cli':['memory'], 'telegram':['web','spotify','owner_mcp'], 'discord':[], 'custom':"['file']", 'api_server':['web'], 'cron':['file'], 'webhook':['web']}
        config['agent']={'disabled_toolsets':['kanban','memory','terminal','video']}
        (self.root/'config.yaml').write_text(json.dumps(config))
        team.reconcile_default_kanban(self.root,self.native_run)
        result=json.loads((self.root/'config.yaml').read_text())
        self.assertEqual(result['toolsets'],['memory','web','kanban'])
        for platform in ('cli','telegram','discord'):
            self.assertTrue(set(self.catalog[platform]['required']) <= set(result['platform_toolsets'][platform]))
        self.assertTrue({'spotify','owner_mcp'} <= set(result['platform_toolsets']['telegram']))
        for platform in ('custom','api_server','cron','webhook'):
            self.assertEqual(result['platform_toolsets'][platform],config['platform_toolsets'][platform])
        self.assertEqual(result['agent']['disabled_toolsets'],['video'])
        self.assertFalse(any(c[2:4] == ('config','set') and c[4].startswith(('platform_toolsets','agent.disabled','known_')) for c in self.calls))
        before=len(self.calls)
        team.reconcile_default_kanban(self.root,self.native_run)
        self.assertEqual(len(self.calls),before)

    def test_native_enabled_env_or_legacy_channel_gets_preset_without_saved_list(self):
        with patch.object(team,'native_default_enabled_platforms',return_value=['telegram']):
            team.reconcile_default_kanban(self.root,self.native_run)
        config=json.loads((self.root/'config.yaml').read_text())
        self.assertEqual(config['platform_toolsets']['telegram'],self.catalog['telegram']['required'])
        self.assertNotIn('discord',config['platform_toolsets'])

    def test_unqualified_native_enabled_plugin_fails_before_any_mutation(self):
        before=(self.root/'config.yaml').read_bytes()
        with patch.object(team,'native_default_enabled_platforms',return_value=['custom_plugin']):
            with self.assertRaisesRegex(RuntimeError,'unqualified.*custom_plugin'):
                team.reconcile_default_kanban(self.root,self.native_run)
        self.assertEqual(self.calls,[])
        self.assertEqual((self.root/'config.yaml').read_bytes(),before)

    def test_native_discovery_failure_is_not_silently_omitted(self):
        with patch.object(team,'native_default_enabled_platforms',side_effect=RuntimeError('discovery unqualified')):
            with self.assertRaisesRegex(RuntimeError,'unqualified'):
                team.reconcile_default_kanban(self.root,self.native_run)
        self.assertEqual(self.calls,[])

    def test_enabled_unsaved_channel_is_reconciled_and_absent_channels_are_not_frozen(self):
        config=json.loads((self.root/'config.yaml').read_text())
        config['platforms']={'telegram':{'enabled':True}, 'discord':{'enabled':False}}
        (self.root/'config.yaml').write_text(json.dumps(config))
        team.reconcile_default_kanban(self.root,self.native_run)
        result=json.loads((self.root/'config.yaml').read_text())
        self.assertIn('telegram',result['platform_toolsets'])
        self.assertNotIn('discord',result['platform_toolsets'])
        self.assertNotIn('slack',result['platform_toolsets'])

    def test_native_enable_zero_exit_without_effect_is_rejected(self):
        config=json.loads((self.root/'config.yaml').read_text())
        config['platform_toolsets']['telegram']=[]
        (self.root/'config.yaml').write_text(json.dumps(config))
        with self.assertRaisesRegex(RuntimeError,'native platform tools'):
            team.reconcile_default_kanban(self.root,lambda *args: None)

    def test_new_workers_do_not_inherit_default_preset_on_any_saved_human_channel(self):
        config=json.loads((self.root/'config.yaml').read_text())
        config['platform_toolsets']['telegram']=['web','kanban']
        config['platform_toolsets']['discord']=['hermes-discord']
        (self.root/'config.yaml').write_text(json.dumps(config))
        self.assertEqual(self.apply(),[])
        for role in self.roles[1:]:
            config=json.loads((self.root/'profiles'/role['name']/'config.yaml').read_text())
            for platform in ('cli','telegram','discord'):
                self.assertEqual(config['platform_toolsets'][platform],role['toolsets'])

    def historical_team(self):
        self.assertEqual(self.apply(), [])
        for role in self.roles:
            role['legacy_soul'] = role['soul']
            role['soul'] = '# Repository identity\n' + role['soul']

    def test_exact_historical_souls_upgrade_without_changing_other_state(self):
        self.historical_team()
        for role in self.roles:
            home = self.root if role['name'] == 'default' else self.root/'profiles'/role['name']
            (home/'memories/MEMORY.md').write_text('preserved lesson '+role['name'])
        before = {str(p.relative_to(self.root)): p.read_bytes() for p in self.root.rglob('*') if p.is_file() and p.name != 'SOUL.md'}
        self.calls.clear()
        self.assertEqual(self.apply(), [])
        for role in self.roles:
            home = self.root if role['name'] == 'default' else self.root/'profiles'/role['name']
            self.assertEqual((home/'SOUL.md').read_text(), role['soul'])
            self.assertEqual((home/'SOUL.md').stat().st_mode & 0o777, 0o600)
        self.assertEqual(before, {str(p.relative_to(self.root)): p.read_bytes() for p in self.root.rglob('*') if p.is_file() and p.name != 'SOUL.md'})
        self.assertFalse(any(c[:2] in [('profile','create'), ('profile','describe')] for c in self.calls))
        self.assertEqual(self.apply(), [])

    def test_exact_previous_managed_policies_upgrade_but_owner_edit_is_preserved(self):
        self.assertEqual(self.apply(),[])
        for role in self.roles:
            role['previous_managed_souls']=[role['soul']]
            role['soul']='# Current core policy\n'+role['soul']
        steward=self.root/'profiles/steward/SOUL.md'
        owner=steward.read_bytes()+b'\nOwner instruction'
        steward.write_bytes(owner)
        self.assertEqual(self.apply(),['steward'])
        self.assertEqual((self.root/'SOUL.md').read_text(),self.roles[0]['soul'])
        self.assertEqual(steward.read_bytes(),owner)

    def test_historical_upgrade_preserves_owner_identity_and_config_drift(self):
        self.historical_team()
        (self.root/'SOUL.md').write_text(self.roles[0]['legacy_soul']+'\nowner changes')
        researcher = self.root/'profiles/researcher'
        (researcher/'profile.yaml').write_text(json.dumps({'description':'owner routing'}))
        planner = self.root/'profiles/planner'
        config = json.loads((planner/'config.yaml').read_text())
        config['terminal']['cwd'] = '/owner'
        (planner/'config.yaml').write_text(json.dumps(config))
        executor = self.root/'profiles/executor'
        (executor/'SOUL.md').write_text(self.roles[3]['legacy_soul']+'\n')
        reviewer = self.root/'profiles/reviewer'
        (reviewer/'SOUL.md').write_bytes(self.roles[4]['legacy_soul'].encode('utf-8')+b'\r\n')
        before = {str(p):p.read_bytes() for p in [self.root/'SOUL.md', researcher/'SOUL.md', researcher/'profile.yaml', planner/'SOUL.md', planner/'config.yaml', executor/'SOUL.md', reviewer/'SOUL.md']}
        self.assertEqual(self.apply(), ['default','researcher','planner','executor','reviewer'])
        self.assertEqual(before, {p:Path(p).read_bytes() for p in before})

    def test_rerun_recognizes_historical_team_without_requiring_setup_again(self):
        self.historical_team()
        native = types.ModuleType('hermes_cli.default_soul')
        native.DEFAULT_SOUL_MD = 'native default'
        with patch.dict(sys.modules, {'hermes_cli.default_soul':native}), patch.dict(team.os.environ, {'HERMES_HOME':str(self.root)}), patch.object(team, 'reconcile_default_kanban'), patch.object(team, 'provision', return_value=[]) as provision:
            team.main({'roles':self.roles, 'after_setup':False})
        provision.assert_called_once()

class NativeCatalogTest(unittest.TestCase):
    def test_catalog_uses_native_metadata_and_resolver_not_name_construction(self):
        platforms=types.ModuleType('hermes_cli.platforms')
        platforms.PLATFORMS={
            'cli':types.SimpleNamespace(default_toolset='hermes-cli'),
            'whatsapp_cloud':types.SimpleNamespace(default_toolset='hermes-whatsapp'),
            'api_server':types.SimpleNamespace(default_toolset='hermes-api-server'),
            'cron':types.SimpleNamespace(default_toolset='hermes-cron'),
            'webhook':types.SimpleNamespace(default_toolset='hermes-webhook')}
        native=types.ModuleType('hermes_cli.tools_config')
        native.CONFIGURABLE_TOOLSETS=[(key,key,'') for key in ('file','terminal','kanban','memory')]
        resolved=[]
        def resolve(config,platform,include_default_mcp_servers):
            self.assertFalse(include_default_mcp_servers)
            resolved.append((platform,config))
            return {'file','terminal','kanban','memory','unrelated_plugin'}
        native._get_platform_tools=resolve
        with patch.dict(sys.modules,{'hermes_cli.platforms':platforms,'hermes_cli.tools_config':native}):
            catalog=team.native_interactive_catalog()
        self.assertEqual(set(catalog),{'cli','whatsapp_cloud'})
        self.assertEqual(catalog['whatsapp_cloud']['preset'],'hermes-whatsapp')
        self.assertEqual(catalog['whatsapp_cloud']['required'],['file','kanban','memory','terminal'])
        self.assertEqual(resolved[1][1],{'platform_toolsets':{'whatsapp_cloud':['hermes-whatsapp','kanban','memory']}})

    def test_missing_required_native_toolset_fails_before_commands(self):
        platforms=types.ModuleType('hermes_cli.platforms')
        platforms.PLATFORMS={'cli':types.SimpleNamespace(default_toolset='hermes-cli')}
        native=types.ModuleType('hermes_cli.tools_config')
        native.CONFIGURABLE_TOOLSETS=[('memory','memory','')]
        native._get_platform_tools=lambda *args,**kwargs: {'memory'}
        with patch.dict(sys.modules,{'hermes_cli.platforms':platforms,'hermes_cli.tools_config':native}):
            with self.assertRaisesRegex(RuntimeError,'required opt-ins'):
                team.native_interactive_catalog()

class NativeDiscoveryTest(unittest.TestCase):
    def modules(self,loader):
        native=types.ModuleType('gateway.config')
        native.load_gateway_config=loader
        scope=types.ModuleType('agent.secret_scope')
        self.events=[]
        opaque=object()
        scope.build_profile_secret_scope=lambda root:self.events.append(('build',root)) or opaque
        def bind(value,profile_home):
            self.assertIs(value,opaque)
            self.events.append(('bind',profile_home))
            return 'native-reset-token'
        scope.set_secret_scope=bind
        scope.reset_secret_scope=lambda token:self.events.append(('reset',token))
        return {'gateway.config':native,'agent.secret_scope':scope}

    def test_enabled_identifiers_only_with_native_scope_and_no_connected_probe(self):
        class Platform(Enum):
            TELEGRAM='telegram'
            DISCORD='discord'
        class Settings:
            def __init__(self,enabled):self.enabled=enabled
            @property
            def token(self):raise AssertionError('RepoKit must not inspect credentials')
        config=types.SimpleNamespace(platforms={Platform.TELEGRAM:Settings(True),Platform.DISCORD:Settings(False)},
                                    get_connected_platforms=lambda:self.fail('connected checkers must not run'))
        with patch.dict(sys.modules,self.modules(lambda:config)):
            self.assertEqual(team.native_default_enabled_platforms(Path('/fixture')),['telegram'])
        self.assertEqual(self.events,[('build',Path('/fixture')),('bind','/fixture'),('reset','native-reset-token')])

    def test_native_resolution_exception_is_sanitized_and_scope_reset(self):
        def loader():
            print('private-native-detail')
            raise ValueError('private-credential-marker')
        output=io.StringIO()
        with patch.dict(sys.modules,self.modules(loader)),contextlib.redirect_stdout(output):
            with self.assertRaisesRegex(RuntimeError,'discovery unqualified') as caught:
                team.native_default_enabled_platforms(Path('/fixture'))
        self.assertNotIn('private-credential-marker',str(caught.exception))
        self.assertEqual(output.getvalue(),'')
        self.assertEqual(self.events[-1],('reset','native-reset-token'))

    def test_native_loader_warning_fallback_cannot_report_success(self):
        def loader():
            logging.getLogger('gateway.config').warning('private-native-detail')
            return types.SimpleNamespace(platforms={})
        logger=logging.getLogger('gateway.config')
        before=logger.handlers,logger.propagate,logger.level
        output=io.StringIO()
        with patch.dict(sys.modules,self.modules(loader)),contextlib.redirect_stderr(output):
            with self.assertRaisesRegex(RuntimeError,'unqualified'):
                team.native_default_enabled_platforms(Path('/fixture'))
        self.assertEqual(output.getvalue(),'')
        self.assertEqual((logger.handlers,logger.propagate,logger.level),before)

if __name__=='__main__': unittest.main()

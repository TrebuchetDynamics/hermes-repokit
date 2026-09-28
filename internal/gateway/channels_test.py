import copy
import importlib.util
import json
from pathlib import Path
import unittest


spec = importlib.util.spec_from_file_location('channels', Path(__file__).with_name('channels.py'))
channels = importlib.util.module_from_spec(spec)
if Path(spec.origin).exists():
    spec.loader.exec_module(channels)

CATALOG = {name: {'preset': 'hermes-' + name, 'required': ['terminal', 'kanban', 'memory']}
           for name in ('cli', 'telegram', 'discord', 'slack', 'api_server', 'webhook')}


class ProjectionTest(unittest.TestCase):
    def project(self, config=None, legacy=None, state=None, verified=True):
        self.assertTrue(hasattr(channels, 'project_channels'), 'channel projection is implemented')
        return channels.project_channels(config or {}, legacy or {}, state or {}, verified, CATALOG)

    def row(self, result, platform):
        return next(row for row in result['rows'] if row['platform'] == platform)

    def test_declared_saved_and_recorded_human_platforms_are_included_without_programmatic_surfaces(self):
        result = self.project({'platforms': {'telegram': {'enabled': True}, 'api_server': {'enabled': True}},
                               'platform_toolsets': {'discord': ['hermes-discord']}},
                              state={'platforms': {'slack': {'state': 'connected'}, 'webhook': {'state': 'connected'}}})
        self.assertEqual([row['platform'] for row in result['rows']], ['cli', 'discord', 'slack', 'telegram'])
        self.assertEqual(self.row(result, 'telegram')['declared_enabled'], 'yes')
        self.assertEqual(self.row(result, 'slack')['recorded_adapter_state'], 'connected')

    def test_scoped_other_route_is_conditional_and_identifiers_never_escape(self):
        config = {'platforms': {'telegram': {'enabled': True}}, 'gateway': {'profile_routes': [
            {'platform': 'telegram', 'profile': 'private-profile-name', 'chat_id': 'secret-chat',
             'name': 'secret-route', 'user_id': 'secret-user'}]}}
        result = self.project(config, state={'served_profiles': ['default', 'private-profile-name']})
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'conditional_other')
        encoded = json.dumps(result)
        for secret in ('private-profile-name', 'secret-chat', 'secret-route', 'secret-user'):
            self.assertNotIn(secret, encoded)

    def test_empty_yaml_routes_override_legacy_and_secondary_bot_routes_do_not_rehome_default(self):
        route = {'platform': 'telegram', 'profile': 'other'}
        state = {'served_profiles': ['default', 'other']}
        result = self.project({'platforms': {'telegram': {}}, 'profile_routes': []},
                              {'profile_routes': [route]}, state)
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'default')
        result = self.project({'platforms': {'telegram': {}}, 'profile_routes': [dict(route, bot_profile='other')]}, state=state)
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'default')

    def test_unserved_unresolved_or_unverified_routes_stay_unknown(self):
        base = {'platforms': {'telegram': {}}, 'profile_routes': [{'platform': 'telegram', 'profile': 'other'}]}
        for state, verified in [({}, True), ({'served_profiles': ['default']}, True),
                                ({'served_profiles': ['default', 'other']}, False)]:
            with self.subTest(state=state, verified=verified):
                self.assertEqual(self.row(self.project(base, state=state, verified=verified), 'telegram')['default_routing'], 'unknown')
        for field in ('platform', 'profile', 'chat_id', 'bot_profile'):
            config = copy.deepcopy(base)
            config['profile_routes'][0][field] = '${PRIVATE_VALUE}'
            self.assertEqual(self.row(self.project(config, state={'served_profiles': ['default', 'other']}), 'telegram')['default_routing'], 'unknown')

    def test_enablement_precedence_and_explicit_disable_are_only_declarations(self):
        result = self.project({'gateway': {'platforms': {'telegram': {'enabled': True}},
                                         'telegram': {'enabled': True}},
                               'platforms': {'telegram': {'enabled': True}}, 'telegram': {'enabled': False}},
                              {'platforms': {'telegram': {'enabled': True}}})
        self.assertEqual(self.row(result, 'telegram')['declared_enabled'], 'no')
        self.assertIn('projection', result['qualifier'].lower())

    def test_unverified_or_secondary_adapter_cannot_prove_default_ownership(self):
        record = {'platforms': {'telegram': {'state': 'connected', 'error_message': 'secret-error'},
                                'private-profile:discord': {'state': 'connected'}}}
        result = self.project(state=record, verified=False)
        self.assertEqual(self.row(result, 'telegram')['owner'], 'unknown')
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'unknown')
        result = self.project(state=record)
        self.assertEqual(self.row(result, 'discord')['owner'], 'other')
        self.assertEqual(self.row(result, 'discord')['default_routing'], 'unknown')
        self.assertNotIn('private-profile', json.dumps(result))
        self.assertNotIn('secret-error', json.dumps(result))

    def test_truthy_text_is_not_verified_ownership(self):
        result = self.project(state={'platforms': {'telegram': {'state': 'connected'}}}, verified='yes')
        self.assertEqual(self.row(result, 'telegram')['owner'], 'unknown')
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'unknown')

    def test_core_selection_expands_known_presets_and_subtracts_disabled_categories(self):
        config = {'toolsets': ['terminal', 'kanban', 'memory'],
                  'platform_toolsets': {'telegram': ['hermes-telegram', 'kanban'], 'discord': ['terminal', 'kanban']}}
        result = self.project(config)
        self.assertEqual(self.row(result, 'cli')['core_selection'], 'complete')
        self.assertEqual(self.row(result, 'telegram')['core_selection'], 'complete')
        self.assertEqual(self.row(result, 'discord')['core_selection'], 'incomplete')
        config['agent'] = {'disabled_toolsets': ['memory']}
        self.assertEqual(self.row(self.project(config), 'telegram')['core_selection'], 'incomplete')
        config['agent']['disabled_toolsets'] = ['${DISABLED}']
        self.assertEqual(self.row(self.project(config), 'telegram')['core_selection'], 'unknown')

    def test_native_platform_preset_does_not_imply_opt_in_kanban(self):
        result = self.project({'platform_toolsets': {'telegram': ['hermes-telegram']}})
        self.assertEqual(self.row(result, 'telegram')['core_selection'], 'incomplete')
        catalog = copy.deepcopy(CATALOG)
        catalog['telegram']['native_categories'] = ['terminal']
        result = channels.project_channels({'platform_toolsets': {'telegram': ['hermes-telegram', 'kanban']}},
                                           {}, {}, True, catalog)
        self.assertEqual(self.row(result, 'telegram')['core_selection'], 'incomplete')

    def test_missing_unknown_alias_and_unresolved_overlay_do_not_claim_complete_selection(self):
        config = {'platforms': {'telegram': {}}, 'platform_toolsets': {'discord': ['custom-preset']}}
        result = self.project(config)
        self.assertEqual(self.row(result, 'telegram')['core_selection'], 'unknown')
        self.assertEqual(self.row(result, 'discord')['core_selection'], 'unknown')
        config.update(_repokit_projection_unknown=True, toolsets=['hermes-cli'])
        result = self.project(config)
        self.assertEqual(self.row(result, 'cli')['core_selection'], 'unknown')
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'unknown')

    def test_inputs_are_not_mutated_and_bad_shapes_do_not_crash_or_leak(self):
        config = {'platforms': {'telegram': {'token': 'secret-token'}}, 'profile_routes': 'secret-bad-routes'}
        saved = copy.deepcopy(config)
        result = self.project(config)
        self.assertEqual(config, saved)
        self.assertEqual(self.row(result, 'telegram')['default_routing'], 'unknown')
        self.assertNotIn('secret-', json.dumps(result))

    def test_authorization_allowlist_is_only_declared_evidence_and_never_emits_principals(self):
        config = {'telegram': {'enabled': True, 'allow_from': ['private-user-id'],
                               'group_allow_from': 'private-group-user', 'token': 'private-bot-token'}}
        result = self.project(config)
        row = self.row(result, 'telegram')
        self.assertEqual(row['authorization'], 'restricted')
        self.assertEqual(row['authorization_effective'], 'unknown')
        self.assertEqual(row['authorization_scope'], 'declared')
        for secret in ('private-user-id', 'private-group-user', 'private-bot-token'):
            self.assertNotIn(secret, json.dumps(result))

    def test_authorization_absence_or_allow_all_false_never_proves_restricted_access(self):
        for block in ({}, {'allow_all_users': False}, {'dm_policy': 'open'}, {'allow_from': '${USERS}'}):
            with self.subTest(block=block):
                row = self.row(self.project({'telegram': block}), 'telegram')
                self.assertEqual(row['authorization'], 'unknown')
                self.assertEqual(row['authorization_effective'], 'unknown')

    def test_declared_open_grants_are_reported_without_proving_live_access(self):
        for config in ({'allow_all_users': True, 'telegram': {}},
                       {'gateway': {'allow_all_users': 'yes', 'telegram': {}}},
                       {'telegram': {'allow_from': ['*']}},
                       {'telegram': {'extra': {'group_allow_from': '["*"]'}}},
                       {'discord': {'allow_all_users': True}}):
            with self.subTest(config=config):
                rows = [row for row in self.project(config)['rows'] if row['platform'] != 'cli']
                self.assertEqual(rows[0]['authorization'], 'open')
                self.assertEqual(rows[0]['authorization_effective'], 'unknown')

    def test_secondary_only_state_cannot_certify_default_selection_or_authorization(self):
        config = {'platform_toolsets': {'telegram': ['hermes-telegram', 'kanban']},
                  'telegram': {'allow_from': ['private-default-user']}}
        result = self.project(config, state={'platforms': {'private-profile:telegram': {'state': 'connected'}}})
        row = self.row(result, 'telegram')
        self.assertEqual(row['core_selection'], 'unknown')
        self.assertEqual(row['authorization'], 'unknown')

    def test_unresolved_or_conflicting_auth_declarations_remain_unknown(self):
        for config in ({'_repokit_projection_unknown': True, 'telegram': {'allow_from': ['private-user']}},
                       {'platforms': {'telegram': {'allow_from': ['private-user']}},
                        'telegram': {'allow_from': ['*']}},
                       {'telegram': {'extra': {'allow_from': ['private-user']}, 'allow_from': ['*']}}):
            with self.subTest(config=config):
                self.assertEqual(self.row(self.project(config), 'telegram')['authorization'], 'unknown')

    def test_outbound_home_channel_is_not_an_inbound_profile_route(self):
        config = {'telegram': {'home_channel': {'chat_id': 'private-target'}, 'enabled': True}}
        row = self.row(self.project(config), 'telegram')
        self.assertEqual(row['default_routing'], 'default')
        self.assertEqual(row.get('authorization'), 'unknown')


if __name__ == '__main__':
    unittest.main()

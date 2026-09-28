import importlib.util
import json
from pathlib import Path
import sys
import subprocess
import tempfile
import types
import unittest

sys.modules['yaml'] = types.SimpleNamespace(safe_load=json.loads)
spec = importlib.util.spec_from_file_location('supervision', Path(__file__).with_name('supervision.py'))
supervision = importlib.util.module_from_spec(spec)
spec.loader.exec_module(supervision)
exec(Path(__file__).parent.parent.joinpath('supervision/checkout.py').read_text(), supervision.__dict__)


class SupervisionTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.roles = ['default', 'reviewer']
        self.events = []
        self.settings = {'nerve_profile': 'lean', 'reflex_backend': 'laya',
                         'reflex_laya_base_url': 'http://127.0.0.1:8765',
                         'reflex_laya_model': '/model', 'reflex_laya_timeout_seconds': 120}
        for role in self.roles:
            home = self.home(role)
            home.mkdir(parents=True, exist_ok=True)
            self.save(role, {'model': {'default': 'owner-model'}, 'plugins': {'enabled': ['owner']}})

    def home(self, role):
        return self.root if role == 'default' else self.root / 'profiles' / role

    def read(self, role):
        return json.loads((self.home(role) / 'config.yaml').read_text())

    def save(self, role, config):
        (self.home(role) / 'config.yaml').write_text(json.dumps(config))

    def native(self, *args):
        self.events.append(args)
        _, role, command, action, *rest = args
        config = self.read(role)
        if command == 'config':
            self.assertEqual(action, 'set')
            key, value = rest
            target = config
            parts = key.split('.')
            for part in parts[:-1]:
                target = target.setdefault(part, {})
            target[parts[-1]] = json.loads(value)
        elif action == 'install':
            self.assertEqual(rest, ['nerve', '--ref', 'pinned', '--no-enable'])
            self.assertNotIn('nerve', config['plugins']['enabled'])
            (self.home(role) / 'plugins' / 'nerve').mkdir(parents=True)
        elif action == 'enable':
            self.assertEqual(config['plugins']['entries']['nerve']['settings'], self.settings)
            self.assertIn(('health',), self.events)
            config['plugins']['enabled'].append('nerve')
        elif action == 'disable':
            if not (self.home(role) / 'plugins/nerve').exists():
                raise RuntimeError('native cannot disable an undiscovered plugin')
            config['plugins']['enabled'].remove('nerve')
        else:
            self.fail(args)
        self.save(role, config)

    def configure(self, native=None, probe=None):
        supervision.configure(self.root, self.roles, self.settings, 'pinned',
                              native or self.native,
                              lambda: self.events.append(('health',)),
                              lambda home, revision: None,
                              probe or (lambda role: self.events.append(('probe', role))))

    def test_disabled_install_configure_enable_and_preserve_rerun(self):
        self.configure()
        for role in self.roles:
            c = self.read(role)
            self.assertEqual(c['model']['default'], 'owner-model')
            self.assertEqual(c['plugins']['enabled'], ['owner', 'nerve'])
            self.assertEqual(c['plugins']['entries']['nerve']['settings'], self.settings)
        before = [self.read(role) for role in self.roles]
        self.events.clear()
        self.configure()
        self.assertEqual(before, [self.read(role) for role in self.roles])
        self.assertFalse(any('install' in event or 'enable' in event for event in self.events))

    def test_last_profile_drift_refuses_before_any_mutation(self):
        c = self.read('reviewer')
        c['plugins']['entries'] = {'nerve': {'settings': {'reflex_backend': 'jev'}}}
        self.save('reviewer', c)
        with self.assertRaises(RuntimeError):
            self.configure()
        self.assertEqual(self.events, [])

    def test_local_inference_failure_prevents_install(self):
        def unavailable():
            raise RuntimeError('local unavailable')
        with self.assertRaises(RuntimeError):
            supervision.configure(self.root, self.roles, self.settings, 'pinned',
                                  self.native, unavailable, lambda *_: None, lambda *_: None)
        self.assertEqual(self.events, [])

    def test_admission_refusal_never_enables(self):
        def rejected(*args):
            if 'install' in args:
                raise RuntimeError('scanner refused')
            self.native(*args)
        with self.assertRaises(RuntimeError):
            self.configure(native=rejected)
        for role in self.roles:
            self.assertNotIn('nerve', self.read(role)['plugins']['enabled'])

    def test_failed_registry_probe_disables_only_new_activation(self):
        def rejected(role):
            raise RuntimeError('local backend failed')
        with self.assertRaises(RuntimeError):
            self.configure(probe=rejected)
        self.assertEqual(self.read('default')['plugins']['enabled'], ['owner'])
        self.assertEqual(self.read('default')['plugins']['entries']['nerve']['settings']['reflex_backend'], 'laya')

    def test_clone_enabled_config_is_disabled_before_install(self):
        c = self.read('reviewer')
        c['plugins']['enabled'].append('nerve')
        c['plugins']['entries'] = {'nerve': {'settings': self.settings}}
        self.save('reviewer', c)
        self.configure()
        self.assertIn('nerve', self.read('reviewer')['plugins']['enabled'])


class EnvironmentTest(unittest.TestCase):
    def test_refuses_hosted_or_proxied_override_before_activation(self):
        for env in ({'HERMES_REFLEX_BACKEND':'jev'}, {'HTTP_PROXY':'http://proxy.invalid'},
                    {'HTTP_PROXY':'http://proxy.invalid', 'NO_PROXY':'127.0.0.1', 'no_proxy':''}):
            with self.assertRaises(RuntimeError): supervision.check_environment(env)
        supervision.check_environment({'HTTP_PROXY':'http://proxy.invalid', 'NO_PROXY':'127.0.0.1,localhost'})

    def test_op_env_and_external_sources_refused_before_native_calls(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'config.yaml').write_text('{}')
            (root / '.op.env').write_text('HTTP_PROXY=http://proxy.invalid\n')
            with self.assertRaises(RuntimeError):
                supervision.preflight_environment(root, ['default'], {})
            (root / '.op.env').unlink()
            (root / 'config.yaml').write_text(json.dumps({'secrets': {'onepassword': {'enabled': True}}}))
            with self.assertRaises(RuntimeError):
                supervision.preflight_environment(root, ['default'], {})


class NativeProvenanceTest(unittest.TestCase):
    def test_native_sidecar_allowed_but_extra_source_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            plugin = home / 'plugins/nerve'
            plugin.mkdir(parents=True)
            def git(*args):
                return subprocess.check_output(['git', '-C', str(plugin), *args], stderr=subprocess.DEVNULL, text=True).strip()
            git('init', '--quiet')
            (plugin / 'plugin.yaml').write_text('name: nerve')
            git('add', 'plugin.yaml')
            git('-c', 'user.name=fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '--quiet', '-m', 'fixture')
            revision = git('rev-parse', 'HEAD')
            (home / 'plugins/.install-metadata.json').write_text(json.dumps({'nerve': {
                'source': 'https://github.com/keeltrace/hermes-nerve', 'pinned': True, 'revision': revision}}))
            (plugin / '.hermes-catalog.json').write_text('{}')
            supervision.verify_native_checkout(home, revision)
            (plugin / 'extra.py').write_text('raise Exception()')
            with self.assertRaises(RuntimeError):
                supervision.verify_native_checkout(home, revision)

if __name__ == '__main__':
    unittest.main()

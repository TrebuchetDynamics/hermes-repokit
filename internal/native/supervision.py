"""One-shot native Nerve provisioning. Never a replacement plugin or runtime hook."""
import json
import os
from pathlib import Path
import subprocess
import sys
import urllib.request
import yaml


def configure(root, roles, settings, revision, native, health, verify_plugin, probe):
    pending = []
    # Inspect the whole roster before mutation, including the last profile.
    for role in roles:
        home = root if role == 'default' else root / 'profiles' / role
        config = yaml.safe_load((home / 'config.yaml').read_text()) or {}
        plugins = config.get('plugins') or {}
        if not isinstance(plugins, dict):
            raise RuntimeError('owner plugin configuration differs')
        if not isinstance(plugins.get('enabled', []), list) or not isinstance(plugins.get('disabled', []), list):
            raise RuntimeError('owner plugin enablement differs')
        entries = plugins.get('entries') or {}
        if not isinstance(entries, dict) or (home / 'nerve/profile.json').exists():
            raise RuntimeError('owner native Nerve profile differs')
        current = (entries.get('nerve') or {}).get('settings')
        enabled = 'nerve' in (plugins.get('enabled') or []) and 'nerve' not in (plugins.get('disabled') or [])
        if current is not None and current != settings:
            raise RuntimeError('owner Nerve settings differ')
        if enabled and current != settings:
            raise RuntimeError('enabled Nerve has no qualified local configuration')
        if 'hermes-nerve' in entries or (home / 'plugins/hermes-nerve').exists():
            raise RuntimeError('legacy Nerve identity requires native migration')
        path = home / 'plugins/nerve'
        if path.is_symlink():
            raise RuntimeError('plugin path is a symlink')
        installed = path.exists()
        if installed:
            verify_plugin(home, revision)
        pending.append((role, home, installed, enabled, current))
    # Real local typed decision before any plugin install/enable; failure stops here.
    health()
    for role, home, installed, enabled, current in pending:
        newly_enabled = False
        try:
            if not installed:
                # Native config cloning may inherit enablement without plugin files.
                if enabled:
                    # Native plugins disable needs a discovered manifest. A new
                    # clone has only the allow-list; use native config to remove
                    # that inherited entry before admitting any package.
                    config = yaml.safe_load((home / 'config.yaml').read_text()) or {}
                    allow = config.get('plugins', {}).get('enabled', [])
                    native('-p', role, 'config', 'set', 'plugins.enabled',
                           json.dumps([name for name in allow if name != 'nerve']))
                    enabled = False
                native('-p', role, 'plugins', 'install', 'nerve', '--ref', revision, '--no-enable')
                verify_plugin(home, revision)
            if current is None:
                native('-p', role, 'config', 'set', 'plugins.entries.nerve.settings', json.dumps(settings))
            actual = yaml.safe_load((home / 'config.yaml').read_text()) or {}
            if actual.get('plugins', {}).get('entries', {}).get('nerve', {}).get('settings') != settings:
                raise RuntimeError('native local settings did not persist')
            if not enabled:
                native('-p', role, 'plugins', 'enable', 'nerve')
                newly_enabled = True
            probe(role)
        except Exception:
            if newly_enabled:
                native('-p', role, 'plugins', 'disable', 'nerve')
            raise


# Run in a NEW profile-scoped interpreter. This invokes upstream plugin hooks and
# a harmless local semantic decision; it is setup, never a verify implementation.
PROBE = r'''
import json
from hermes_cli.plugins import discover_plugins, get_plugin_manager
from tools.registry import registry
discover_plugins()
manager = get_plugin_manager()
loaded = manager._plugins.get('nerve')
assert loaded and loaded.enabled and not loaded.error
required = {'pre_tool_call','post_tool_call','pre_llm_call','transform_tool_result',
            'pre_verify','post_api_request','api_request_error','post_llm_call','on_session_end'}
assert required <= set(loaded.hooks_registered)
result = registry.dispatch('nerve_decide', {
    'state': 'Synthetic installation check: a customer requests an invoice refund.',
    'instructions': 'Choose the responsible department.',
    'choices': ['billing','technical'],
}, scope=manager.scope_key)
if isinstance(result, str): result = json.loads(result)
assert result.get('provider') == 'Laya' and result.get('model') == '/model'
assert result.get('provenance_status') == 'LOCAL_ONLY'
assert result.get('execution', {}).get('live_provider_call') is False
assert result.get('receipt_id') and result.get('value') in {'billing','technical'}
'''



def verify_native_checkout(home, revision):
    path = home / 'plugins/nerve'
    record = json.loads((home / 'plugins/.install-metadata.json').read_text()).get('nerve', {})
    if record.get('revision') != revision or record.get('pinned') is not True or record.get('source', '').removesuffix('.git') != 'https://github.com/keeltrace/hermes-nerve':
        raise RuntimeError('native Nerve install provenance differs')
    if not verify_checkout(path, home, revision):
        raise RuntimeError('installed Nerve checkout differs')


def check_environment(environment):
    if any(key.startswith(('HERMES_NERVE_', 'HERMES_REFLEX_', 'NERVE_')) for key in environment):
        raise RuntimeError('native supervision environment override requires inspection')
    if environment.get('HERMES_MANAGED_DIR'):
        raise RuntimeError('managed supervision scope requires inspection')
    proxies = {}
    for key, value in environment.items():
        if not key.lower().endswith('_proxy'):
            continue
        normalized = key.lower()
        value = value or ''
        if normalized in proxies and proxies[normalized] != value:
            raise RuntimeError('conflicting proxy environment requires inspection')
        proxies[normalized] = value
    if any('$' in value for value in proxies.values()):
        raise RuntimeError('interpolated proxy configuration requires inspection')
    if proxies.get('http_proxy') or proxies.get('all_proxy'):
        bypass = {part.strip().lower() for part in proxies.get('no_proxy', '').split(',')}
        if not ({'*', '127.0.0.1'} & bypass):
            raise RuntimeError('local supervision must bypass environment proxies')


def preflight_environment(root, roles, inherited):
    from dotenv import dotenv_values
    if (Path('/etc/hermes').is_symlink() or
            any((Path('/etc/hermes') / name).exists() or (Path('/etc/hermes') / name).is_symlink()
                for name in ('config.yaml', '.env'))):
        raise RuntimeError('managed supervision scope requires inspection')
    check_environment(inherited)
    for role in roles:
        home = root if role == 'default' else root / 'profiles' / role
        config = yaml.safe_load((home / 'config.yaml').read_text()) or {}
        if config.get('secrets'):
            raise RuntimeError('external supervision secret sources require inspection')
        scoped = dict(inherited)
        for name in ('.op.env', '.env'):
            path = home / name
            if path.exists():
                values = dotenv_values(path, interpolate=False)
                # Inspect each source, including dormant overrides, before native
                # profile commands can hydrate it and install/enable anything.
                check_environment(dict(scoped, **values))
                scoped.update(values)
        check_environment(scoped)


def main(payload):
    root = Path(os.environ['HERMES_HOME'])
    preflight_environment(root, payload['roles'], os.environ)
    config = yaml.safe_load((root / 'config.yaml').read_text()) or {}
    if config.get('kanban', {}).get('dispatch_in_gateway') is not False:
        raise RuntimeError('disable automatic dispatch before supervision setup')
    # Native lifecycle checks are allowed during setup, never from verify.
    for state in ('running', 'review'):
        active = json.loads(subprocess.check_output(['hermes', 'kanban', 'list', '--status', state, '--json'],
                            timeout=30, stderr=subprocess.DEVNULL, text=True))
        if active:
            raise RuntimeError('finish or park active Kanban work before supervision setup')

    def command(args, **kwargs):
        subprocess.run(args, check=True, timeout=300, stdout=subprocess.DEVNULL,
                       stderr=subprocess.DEVNULL, **kwargs)

    def health():
        # Disable environment proxies and redirects for this fixed local transport.
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, *args):
                return None
        client = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        url = payload['settings']['reflex_laya_base_url']
        with client.open(url + '/healthz', timeout=5) as response:
            status = json.load(response)
        if status != {'ok': True, 'provider': 'Laya', 'model': '/model', 'transport': 'laya-local-http'}:
            raise RuntimeError('local sidecar identity differs')
        request = urllib.request.Request(url + '/v1/systemone', headers={'Content-Type': 'application/json'},
            data=json.dumps({'state': 'Synthetic installation check: invoice refund', 'questions': {
                'department': {'type': 'choice', 'instructions': 'Choose the responsible department.',
                               'criteria': {'billing': 'Invoices', 'technical': 'Software bugs'}}}}).encode())
        with client.open(request, timeout=120) as response:
            result = json.load(response)
        if result.get('provider') != 'Laya' or result.get('model') != '/model' or result.get('answers', {}).get('department', {}).get('choice') not in ('billing', 'technical'):
            raise RuntimeError('local typed decision failed')

    def native(*args):
        command(['hermes', *args])

    def probe(role):
        home = root if role == 'default' else root / 'profiles' / role
        env = dict(os.environ, HERMES_HOME=str(home), HERMES_PROFILE=role,
                   HERMES_PROFILE_NAME=role, PYTHONDONTWRITEBYTECODE='1')
        command([sys.executable, '-c', PROBE], env=env)

    configure(root, payload['roles'], payload['settings'], payload['revision'],
              native, health, verify_native_checkout, probe)
    print('REPOKIT_SUPERVISION all profiles loaded native hooks and LOCAL_ONLY decisions')

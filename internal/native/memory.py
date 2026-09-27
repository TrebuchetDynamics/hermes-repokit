"""One-shot native connection linking. Never a runtime memory provider."""
import copy
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import yaml


def configure(root, roles, check, run):
    root = root.resolve()
    default = yaml.safe_load((root / 'config.yaml').read_text()) or {}
    connection = default.get('memory', {}).get('openviking', {})
    if not connection.get('use_ovcli_config') or not connection.get('ovcli_config_path'):
        raise RuntimeError('native shared connection required')
    link = Path(connection['ovcli_config_path'])
    if not link.is_absolute() or link.is_symlink() or not link.resolve().is_relative_to(root / '.openviking'):
        raise RuntimeError('connection is outside the native shared store')
    info = link.stat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise RuntimeError('native shared connection is not private')
    pending = []
    # Check every candidate under its own native secret scope before any write.
    for name in roles:
        home = root if name == 'default' else root / 'profiles' / name
        config = yaml.safe_load((home / 'config.yaml').read_text()) or {}
        memory = config.get('memory') or {}
        if not isinstance(memory, dict) or memory.get('provider') not in (None, '', 'builtin', 'openviking'):
            raise RuntimeError('owner memory provider differs')
        prior = memory.get('openviking') or {}
        if not isinstance(prior, dict):
            raise RuntimeError('owner connection differs')
        if memory.get('provider') == 'openviking' and (prior.get('ovcli_config_path') != str(link) or prior.get('agent')):
            raise RuntimeError('owner connection differs')
        candidate = copy.deepcopy(memory)
        candidate.update(provider='openviking', memory_enabled=True, user_profile_enabled=True)
        candidate['openviking'] = dict(prior, use_ovcli_config=True, ovcli_config_path=str(link))
        check(home, candidate)
        pending.append((name, home, candidate))
    for name, home, candidate in pending:
        for key in ('openviking', 'memory_enabled', 'user_profile_enabled', 'provider'):
            value = candidate[key]
            run('-p', name, 'config', 'set', 'memory.' + key,
                value if isinstance(value, str) else json.dumps(value))
        actual = (yaml.safe_load((home / 'config.yaml').read_text()) or {}).get('memory', {})
        if actual != candidate:
            raise RuntimeError('native memory settings did not persist')
        check(home, actual)


def main(payload):
    from agent.secret_scope import build_profile_secret_scope, set_secret_scope, reset_secret_scope
    from plugins.memory import openviking as ov
    root = Path(os.environ['HERMES_HOME'])

    def check(home, memory):
        token = set_secret_scope(build_profile_secret_scope(home), profile_home=str(home))
        try:
            settings = ov._resolve_connection_settings(memory['openviking'])
            if settings['endpoint'] != 'http://openviking:1933' or settings['agent'] or not settings['api_key']:
                raise RuntimeError('effective native connection differs')
            client = ov._VikingClient(settings['endpoint'], settings['api_key'],
                                      account=settings['account'], user=settings['user'], agent=settings['agent'])
            # /health with the native key verifies server-derived identity. Never
            # use /ready here: that endpoint performs an embedding request.
            health = client.get('/health', timeout=5)
            if (health.get('auth_mode'), health.get('role'), health.get('account_id'), health.get('user_id')) != (
                    'api_key', 'user', 'repokit', payload['repo_id']):
                raise RuntimeError('native user key has unexpected authority or repository identity')
            status = client.validate_auth()
            if not status.get('result', {}).get('initialized') or status['result'].get('user') != payload['repo_id']:
                raise RuntimeError('native authenticated connection is unavailable')
        finally:
            reset_secret_scope(token)

    def run(*args):
        result = subprocess.run(['hermes', *args], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if result.returncode:
            raise RuntimeError('native configuration failed')

    configure(root, payload['roles'], check, run)
    print('Shared native OpenViking connection configured; live memory acceptance remains unproven.')

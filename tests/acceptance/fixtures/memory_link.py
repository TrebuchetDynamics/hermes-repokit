# Native resolver/configuration contract only. No service response, credentials,
# memory recall or authenticated inference is represented by this fixture.
from agent.secret_scope import build_profile_secret_scope, set_secret_scope, reset_secret_scope
from plugins.memory import openviking as ov

root = Path('/opt/data')
roles = ['default', 'researcher', 'planner', 'executor', 'reviewer', 'steward']
link = root / '.openviking/ovcli.conf.fixture'
link.parent.mkdir(mode=0o700, exist_ok=True)
link.write_text(json.dumps(ov._ovcli_data_from_connection_values({
    'endpoint': 'http://127.0.0.1:1933', 'api_key': 'NONSECRET_OFFLINE_FIXTURE',
    'account': 'repokit', 'user': 'fixture-repository'})))
link.chmod(0o600)

def native_set(*args):
    subprocess.run(['hermes', *args], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

native_set('-p', 'default', 'config', 'set', 'memory.openviking',
           json.dumps({'use_ovcli_config': True, 'ovcli_config_path': str(link)}))
native_set('-p', 'default', 'config', 'set', 'memory.provider', 'openviking')

def check_native_resolver(home, candidate):
    token = set_secret_scope(build_profile_secret_scope(home), profile_home=str(home))
    try:
        settings = resolve_connection(ov, candidate['openviking'], 'fixture-repository')
        assert settings['endpoint'] == 'http://127.0.0.1:1933'
        # Native user keys intentionally suppress account/user assertion headers:
        # the server derives identity from the key. Production checks /health.
        assert settings['account'] == ''
        assert settings['user'] == ''
        assert settings['agent'] == ''
        assert settings['api_key'] == 'NONSECRET_OFFLINE_FIXTURE'
    finally:
        reset_secret_scope(token)

configure(root, roles, check_native_resolver, native_set)
for name in roles:
    home = root if name == 'default' else root / 'profiles' / name
    stored = yaml.safe_load((home / 'config.yaml').read_text())['memory']
    check_native_resolver(home, stored)
    assert stored['memory_enabled'] and stored['user_profile_enabled']
    assert 'NONSECRET_OFFLINE_FIXTURE' not in (home / 'config.yaml').read_text()

# Exercise effective per-profile secrets through the real native resolver. The
# mismatches must refuse before a config setter can modify any profile.
reviewer = root / 'profiles' / 'reviewer'
env_path = reviewer / '.env'
original_env = env_path.read_bytes() if env_path.exists() else None
original_link = link.read_bytes()
try:
    for key, value in [('OPENVIKING_AGENT', 'fixture-peer'),
                       ('OPENVIKING_API_KEY', 'OTHER_NONSECRET_FIXTURE'),
                       ('OPENVIKING_ENDPOINT', 'http://different:1933'),
                       ('OPENVIKING_ACCOUNT', 'other-account'),
                       ('OPENVIKING_USER', 'other-repository')]:
        env_path.write_bytes((original_env or b'') + f'\n{key}={value}\n'.encode())
        env_path.chmod(0o600)
        try:
            configure(root, roles, check_native_resolver,
                      lambda *args: (_ for _ in ()).throw(AssertionError('wrote after scope drift')))
        except RuntimeError:
            pass
        else:
            raise AssertionError('accepted specialist secret-scope drift')
    if original_env is None:
        env_path.unlink()
    else:
        env_path.write_bytes(original_env)
    data = json.loads(original_link)
    data['actor_peer_id'] = 'fixture-linked-peer'
    link.write_text(json.dumps(data))
    try:
        configure(root, roles, check_native_resolver,
                  lambda *args: (_ for _ in ()).throw(AssertionError('wrote after linked peer drift')))
    except RuntimeError:
        pass
    else:
        raise AssertionError('accepted shared linked peer')
finally:
    link.write_bytes(original_link)
    if original_env is None:
        env_path.unlink(missing_ok=True)
    else:
        env_path.write_bytes(original_env)
print('Native shared-link configuration and resolver contract passed; no live memory acceptance.')

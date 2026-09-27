"""One-shot installer, executed by the pinned Hermes Python; not a runtime hook."""
import json
import os
from pathlib import Path
import subprocess
import sys
import yaml


def read_config(home):
    value = yaml.safe_load((home / 'config.yaml').read_text()) or {}
    if not isinstance(value, dict):
        raise RuntimeError('invalid native configuration')
    return value


def expected(role):
    return {
        'kanban.dispatch_in_gateway': False,
        'kanban.auto_decompose': False,
        'kanban.orchestrator_profile': 'default',
        'kanban.max_in_progress': 1,
        'toolsets': role['toolsets'],
        'platform_toolsets.cli': role['toolsets'],
        'terminal.cwd': '/workspace',
    }


def get(config, key):
    for part in key.split('.'):
        if not isinstance(config, dict): return None
        config = config.get(part)
    return config


def matches(home, role):
    try:
        if (home / 'SOUL.md').read_text() != role['soul']: return False
        meta = yaml.safe_load((home / 'profile.yaml').read_text()) or {}
        if meta.get('description') != role['description']: return False
        config = read_config(home)
        return all(get(config, k) == v for k, v in expected(role).items())
    except (OSError, ValueError):
        return False


def configure(home, name, role, run):
    # The native CLI remains responsible for config semantics and metadata.
    for key, value in expected(role).items():
        run('-p', name, 'config', 'set', key, value if isinstance(value, str) else json.dumps(value))
    run('profile', 'describe', name, '--text', role['description'])
    (home / 'SOUL.md').write_text(role['soul'])
    (home / 'SOUL.md').chmod(0o600)


def provision(root, roles, run, native_default_soul):
    if get(read_config(root), 'kanban.dispatch_in_gateway') is not False:
        raise RuntimeError('disable dispatch before provisioning')
    profiles = root / 'profiles'
    profiles.mkdir(mode=0o700, exist_ok=True)
    drift = []
    for role in roles:
        name = role['name']
        home = root if name == 'default' else profiles / name
        if home.exists():
            if matches(home, role):
                continue  # Never erase memories on a rerun.
            soul = home / 'SOUL.md'
            # Compare every managed field, not merely SOUL, before adoption.
            config = read_config(home)
            meta_path = home / 'profile.yaml'
            meta = yaml.safe_load(meta_path.read_text()) if meta_path.exists() else {}
            pristine_fields = all(get(config, k) == v for k, v in expected(role).items())
            pristine_meta = not meta or meta.get('description') == role['description']
            pristine_soul = not soul.exists() or soul.read_text() in (native_default_soul, native_default_soul + '\n')
            if name == 'default' and pristine_fields and pristine_meta and pristine_soul:
                run('profile', 'describe', name, '--text', role['description'])
                soul.write_text(role['soul'])
                soul.chmod(0o600)
            else:
                drift.append(name)
            continue
        # Use the final native name: create also registers native services and
        # routing. Renaming its directory would bypass that lifecycle machinery.
        run('profile', 'create', name, '--clone', '--clone-from', 'default',
            '--no-alias', '--description', role['description'])
        (home / 'SOUL.md').write_text(role['soul'])
        for rel in ('memories/MEMORY.md', 'memories/USER.md', 'MEMORY.md', 'USER.md'):
            (home / rel).unlink(missing_ok=True)
        configure(home, name, role, run)
        if not matches(home, role):
            raise RuntimeError('native profile configuration did not resolve')
        # Failure preserves native state; no automatic profile deletion or retry
        # memory erasure. The next run reports incomplete managed fields as drift.
    run('profile', 'list')
    for role in roles:
        run('profile', 'show', role['name'])
    return drift


def main(payload):
    root = Path(os.environ['HERMES_HOME'])
    roles = payload['roles']
    config = read_config(root)
    # Native setup can exit zero after printing noninteractive guidance. An
    # explicit saved model is required as well; this is not an auth/inference probe.
    model = config.get('model')
    configured = isinstance(model, dict) and bool(model.get('default'))
    managed = any(((root if r['name']=='default' else root/'profiles'/r['name'])/'SOUL.md').is_file()
                  and ((root if r['name']=='default' else root/'profiles'/r['name'])/'SOUL.md').read_text()==r['soul'] for r in roles)
    if not configured or not (payload['after_setup'] or managed):
        print('REPOKIT_TEAM=' + json.dumps({'status':'pending-setup','drift':[]}))
        return
    def run(*args):
        result = subprocess.run(['hermes', *args], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if result.returncode: raise RuntimeError('native operation failed')
    from hermes_cli.default_soul import DEFAULT_SOUL_MD
    drift = provision(root, roles, run, DEFAULT_SOUL_MD)
    print('REPOKIT_TEAM=' + json.dumps({'status':'drift' if drift else 'configured', 'drift':drift}))

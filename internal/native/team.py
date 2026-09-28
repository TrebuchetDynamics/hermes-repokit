"""One-shot installer, executed by the pinned Hermes Python; not a runtime hook."""
import contextlib
import json
import logging
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
    fields = {
        'kanban.dispatch_in_gateway': False,
        'kanban.auto_decompose': False,
        'kanban.orchestrator_profile': 'default',
        'kanban.max_in_progress': 1,
        'toolsets': role['toolsets'],
        'platform_toolsets.cli': role['toolsets'],
        'terminal.cwd': '/workspace',
        'terminal.backend': 'local',
    }
    if role['name'] == 'default':
        fields.update({'kanban.auto_subscribe_on_create': True, 'kanban.notify_in_gateway': True})
    return fields


def get(config, key):
    for part in key.split('.'):
        if not isinstance(config, dict): return None
        config = config.get(part)
    return config


def managed_souls(role):
    return [value for value in (
        role['soul'], role.get('legacy_soul'), role.get('previous_soul'),
        role.get('previous_repository_soul'), role.get('previous_repository_original_soul'),
        *role.get('previous_managed_souls', [])) if value]


def matches(home, role, soul=None):
    try:
        if (home / 'SOUL.md').read_bytes() != (role['soul'] if soul is None else soul).encode('utf-8'): return False
        meta = yaml.safe_load((home / 'profile.yaml').read_text()) or {}
        if meta.get('description') != role['description']: return False
        config = read_config(home)
        return managed_fields_match(config, role)
    except (OSError, ValueError):
        return False


def managed_fields_match(config, role):
    fields = expected(role)
    if role['name'] == 'default':
        # The owner selects useful tools; only orchestrator Kanban is mandatory.
        fields = {k:v for k,v in fields.items() if k not in ('toolsets','platform_toolsets.cli')}
        if default_native_tool_drift(config): return False
        if get(config,'kanban.dispatch_in_gateway') is True:
            if not managed_operational_dispatch(config): return False
            fields['kanban.dispatch_in_gateway'] = True
    else:
        catalog = native_interactive_catalog()
        selections = kanban_selections(config)
        if any(set(kanban_list(selections[p])) != set(role['toolsets']) for p in configured_interactive_platforms(config, catalog) if p in selections):
            return False
        if worker_kanban_updates(config): return False
        if set(role['toolsets']) & set(kanban_disabled(config)): return False
    for key,value in fields.items():
        observed=get(config,key)
        if key=='terminal.backend' and observed is None: observed='local'
        if key in ('toolsets','platform_toolsets.cli'):
            if set(kanban_list(observed or [])) != set(value): return False
        elif observed != value: return False
    return True


def native_platform_tools(config, platform):
    from hermes_cli.tools_config import _get_platform_tools
    return _get_platform_tools(config, platform, include_default_mcp_servers=False)


def native_interactive_catalog():
    from hermes_cli.platforms import PLATFORMS
    from hermes_cli.tools_config import CONFIGURABLE_TOOLSETS
    # tools enable accepts configurable category names, not composite presets.
    # Resolve the pinned native preset and let its CLI own all save bookkeeping.
    configurable = {key for key, *_ in CONFIGURABLE_TOOLSETS}
    catalog = {}
    for platform, info in PLATFORMS.items():
        if platform in NONINTERACTIVE_PLATFORMS:
            continue
        baseline = {'platform_toolsets': {platform: [info.default_toolset, *DEFAULT_REQUIRED_TOOLSETS]}}
        required = native_platform_tools(baseline, platform) & configurable
        if not set(DEFAULT_REQUIRED_TOOLSETS) <= required:
            raise RuntimeError('native platform tools cannot supply required opt-ins')
        catalog[platform] = {'preset': info.default_toolset, 'required': sorted(required)}
    return catalog


def native_default_enabled_platforms(root):
    # Mutating setup only: use the native config/hook trust boundary, never the
    # passive verifier. Scope loading reads through native APIs without dotenv
    # sanitization, external-source hydration or credential access in RepoKit.
    from agent.secret_scope import build_profile_secret_scope, set_secret_scope, reset_secret_scope
    from gateway.config import load_gateway_config
    class DiscoveryWarnings(logging.Handler):
        warned = False
        def emit(self, record):
            if record.levelno >= logging.WARNING:
                self.warned = True  # Never format or retain credential-bearing diagnostics.
    capture = DiscoveryWarnings()
    logger = logging.getLogger('gateway.config')
    previous = logger.handlers, logger.propagate, logger.level
    token = None
    try:
        logger.handlers, logger.propagate, logger.level = [capture], False, logging.WARNING
        with open(os.devnull, 'w') as sink, contextlib.redirect_stdout(sink), contextlib.redirect_stderr(sink):
            token = set_secret_scope(build_profile_secret_scope(root), profile_home=str(root))
            config = load_gateway_config()
            if capture.warned:
                raise RuntimeError('native channel discovery unqualified')
            names = []
            for platform, settings in config.platforms.items():
                if settings.enabled is True:
                    name = platform.value
                    if not isinstance(name, str) or not re.fullmatch(r'[a-zA-Z0-9_-]{1,64}', name):
                        raise RuntimeError('native channel discovery unqualified')
                    names.append(name)
            return sorted(set(names))
    except Exception:
        raise RuntimeError('native channel discovery unqualified; inspect native gateway configuration privately') from None
    finally:
        if token is not None:
            reset_secret_scope(token)
        logger.handlers, logger.propagate, logger.level = previous


def default_interactive_platforms(root, config, catalog):
    enabled = set(native_default_enabled_platforms(root)) - NONINTERACTIVE_PLATFORMS
    unknown = enabled - set(catalog)
    if unknown:
        raise RuntimeError('native enabled platforms unqualified: ' + ', '.join(sorted(unknown)))
    return sorted(set(configured_interactive_platforms(config, catalog)) | enabled)


def default_native_tool_drift(config, catalog=None):
    if not set(DEFAULT_REQUIRED_TOOLSETS) <= set(kanban_list(config.get('toolsets') or [])):
        return True
    catalog = native_interactive_catalog() if catalog is None else catalog
    selections = kanban_selections(config)
    for platform in configured_interactive_platforms(config, catalog):
        if not set(DEFAULT_REQUIRED_TOOLSETS) <= set(selections.get(platform, [])):
            return True
        if not set(catalog[platform]['required']) <= native_platform_tools(config, platform):
            return True
    return False


def reconcile_default_kanban(root, run):
    config = read_config(root)
    # Validate first. The global legacy fallback has no tools enable equivalent.
    kanban_selections(config)
    fallback = kanban_list(config.get('toolsets') or [])
    kanban_disabled(config)
    catalog = native_interactive_catalog()
    platforms = default_interactive_platforms(root, config, catalog)
    for key in ('kanban.auto_subscribe_on_create', 'kanban.notify_in_gateway'):
        if get(config, key) is not True:
            run('-p', 'default', 'config', 'set', key, 'true')
    if not set(DEFAULT_REQUIRED_TOOLSETS) <= set(fallback):
        fallback += [tool for tool in DEFAULT_REQUIRED_TOOLSETS if tool not in fallback]
        run('-p', 'default', 'config', 'set', 'toolsets', json.dumps(fallback))
    for platform in platforms:
        config = read_config(root)
        required = set(catalog[platform]['required'])
        saved = set(kanban_selections(config).get(platform, []))
        if required <= native_platform_tools(config, platform) and set(DEFAULT_REQUIRED_TOOLSETS) <= saved:
            continue
        run('-p', 'default', 'tools', 'enable', *sorted(required), '--platform', platform)
        # Native commands can print a rejected name and still exit zero.
        config = read_config(root)
        if not required <= native_platform_tools(config, platform) or not set(DEFAULT_REQUIRED_TOOLSETS) <= set(kanban_selections(config).get(platform, [])):
            raise RuntimeError('native platform tools did not resolve after enable')


def project_skill_settings(home):
    skills = read_config(home).get('skills')
    if skills is None: skills = {}
    if not isinstance(skills, dict):
        raise RuntimeError('invalid native project skill settings')
    trusted = skills.get('trusted_project_dirs')
    if trusted is None: trusted = []
    if isinstance(trusted, str): trusted = [trusted]
    if not isinstance(trusted, list) or any(not isinstance(path, str) or not path for path in trusted):
        raise RuntimeError('invalid native project skill trust')
    return skills, trusted


def reconcile_project_skills(home, name, run):
    skills, trusted = project_skill_settings(home)
    # Preserve an explicit owner opt-out and unrelated skills settings. Native
    # trust adds only this repository; normal scan-time quarantine still applies.
    if skills.get('project_discovery') is False:
        return
    if any(Path(path).expanduser().resolve() == Path('/workspace') for path in trusted):
        return
    run('-p', name, 'skills', 'trust', '/workspace')
    _, trusted = project_skill_settings(home)
    if not any(Path(path).expanduser().resolve() == Path('/workspace') for path in trusted):
        raise RuntimeError('native project skill trust did not persist')


def configure(home, name, role, run):
    # The native CLI remains responsible for config semantics and metadata.
    for key, value in expected(role).items():
        run('-p', name, 'config', 'set', key, value if isinstance(value, str) else json.dumps(value))
    if name != 'default':
        config = read_config(home)
        updates = worker_kanban_updates(config)
        # Cloning the orchestrator also clones its broad per-channel selections.
        # New specialists get their existing role boundary on every human channel.
        for platform in configured_interactive_platforms(config, native_interactive_catalog()):
            if platform in kanban_selections(config):
                updates['platform_toolsets.'+platform] = role['toolsets']
        for key,value in updates.items():
            run('-p', name, 'config', 'set', key, json.dumps(value))
        for platform in configured_interactive_platforms(read_config(home),native_interactive_catalog()):
            if platform in kanban_selections(read_config(home)):
                run('-p',name,'tools','enable',*role['toolsets'],'--platform',platform)
    run('profile', 'describe', name, '--text', role['description'])
    (home / 'SOUL.md').write_text(role['soul'])
    (home / 'SOUL.md').chmod(0o600)
    reconcile_project_skills(home, name, run)


def reconcile_coding_profile(home, role, run):
    # Upgrade only a complete exact earlier managed contract; no owner tool
    # selection or profile identity may be overwritten merely by role name.
    old_tools=role.get('legacy_toolsets')
    if not old_tools:return
    prior={**role,'toolsets':old_tools}
    if not any(matches(home,prior,soul) for soul in managed_souls(role)):return
    config=read_config(home)
    disabled=kanban_disabled(config)
    if any(tool in disabled for tool in role['toolsets']):return
    platforms=configured_interactive_platforms(config,native_interactive_catalog())
    run('-p',role['name'],'config','set','toolsets',json.dumps(role['toolsets']))
    for platform in platforms:
        if platform in kanban_selections(config):
            run('-p',role['name'],'tools','enable','code_execution','skills','--platform',platform)


def provision(root, roles, run, native_default_soul):
    if get(read_config(root), 'kanban.dispatch_in_gateway') is not False:
        raise RuntimeError('disable dispatch before provisioning')
    reconcile_default_kanban(root, run)
    profiles = root / 'profiles'
    profiles.mkdir(mode=0o700, exist_ok=True)
    drift = []
    for role in roles:
        name = role['name']
        home = root if name == 'default' else profiles / name
        if home.exists():
            reconcile_coding_profile(home,role,run)
            if matches(home, role):
                if get(read_config(home),'terminal.backend') is None:
                    run('-p',name,'config','set','terminal.backend','local')
                reconcile_project_skills(home, name, run)
                continue  # Never erase memories on a rerun.
            soul = home / 'SOUL.md'
            # Only an exact historical contract with matching managed metadata
            # and configuration is ours to upgrade. Preserve all other state.
            if any(matches(home,role,prior) for prior in managed_souls(role)):
                if get(read_config(home),'terminal.backend') is None:
                    run('-p',name,'config','set','terminal.backend','local')
                soul.write_text(role['soul'])
                soul.chmod(0o600)
                reconcile_project_skills(home, name, run)
                continue
            # Compare every managed field, not merely SOUL, before adoption.
            config = read_config(home)
            meta_path = home / 'profile.yaml'
            meta = yaml.safe_load(meta_path.read_text()) if meta_path.exists() else {}
            pristine_fields = managed_fields_match(config, role)
            pristine_meta = not meta or meta.get('description') == role['description']
            pristine_soul = not soul.exists() or soul.read_text() in (native_default_soul, native_default_soul + '\n')
            if name == 'default' and pristine_fields and pristine_meta and pristine_soul:
                if get(config,'terminal.backend') is None:
                    run('-p',name,'config','set','terminal.backend','local')
                run('profile', 'describe', name, '--text', role['description'])
                soul.write_text(role['soul'])
                soul.chmod(0o600)
                reconcile_project_skills(home, name, run)
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
    def run(*args):
        result = subprocess.run(['hermes', *args], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if result.returncode: raise RuntimeError('native operation failed')
    if get(read_config(root),'kanban.dispatch_in_gateway') is True:
        # Install reruns observe an operational team without racing live workers.
        drift=[r['name'] for r in roles if not matches(root if r['name']=='default' else root/'profiles'/r['name'],r)]
        print('REPOKIT_TEAM=' + json.dumps({'status':'drift' if drift else 'configured','drift':drift}))
        return
    # Also repair channel selections after a direct native setup, before the
    # separate interactive team-provisioning gate. Never copy credentials here.
    reconcile_default_kanban(root, run)
    reconcile_project_skills(root, 'default', run)
    config = read_config(root)
    # Native setup can exit zero after printing noninteractive guidance. An
    # explicit saved model is required as well; this is not an auth/inference probe.
    model = config.get('model')
    configured = isinstance(model, dict) and bool(model.get('default'))
    managed = any(((root if r['name']=='default' else root/'profiles'/r['name'])/'SOUL.md').is_file()
                  and ((root if r['name']=='default' else root/'profiles'/r['name'])/'SOUL.md').read_bytes()
                  in [value.encode('utf-8') for value in managed_souls(r)] for r in roles)
    if not configured or not (payload['after_setup'] or managed):
        print('REPOKIT_TEAM=' + json.dumps({'status':'pending-setup','drift':[]}))
        return
    from hermes_cli.default_soul import DEFAULT_SOUL_MD
    drift = provision(root, roles, run, DEFAULT_SOUL_MD)
    print('REPOKIT_TEAM=' + json.dumps({'status':'drift' if drift else 'configured', 'drift':drift}))

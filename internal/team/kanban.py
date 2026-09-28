"""Pure policy for the default orchestrator's required native tools."""
import ast
import re

DEFAULT_REQUIRED_TOOLSETS = ('kanban', 'memory')
NONINTERACTIVE_PLATFORMS = frozenset({'acp', 'api_server', 'cron', 'webhook'})
# These names qualify top-level declaration blocks only. Absent channels never
# become observation rows. Native installation uses its installed catalog instead.
_DECLARED_INTERACTIVE_NAMES = frozenset({
    'cli', 'telegram', 'discord', 'slack', 'whatsapp', 'whatsapp_cloud', 'signal',
    'bluebubbles', 'email', 'homeassistant', 'mattermost', 'matrix', 'dingtalk',
    'feishu', 'wecom', 'wecom_callback', 'weixin', 'qqbot', 'yuanbao',
})


def declared_interactive_platforms(config, catalog=None):
    """Project YAML declaration precedence without resolving credentials/plugins.

    Saved declarations cannot discover env-only, legacy-JSON-only or managed-overlay
    channels. Callers must not claim this is the complete effective gateway roster.
    Values are explicit enabled true/false or None (unspecified/unresolved).
    """
    gateway = config.get('gateway') or {}
    platforms = config.get('platforms') or {}
    if not isinstance(gateway, dict) or not isinstance(platforms, dict):
        raise ValueError('invalid native platform declarations')
    nested = gateway.get('platforms') or {}
    if not isinstance(nested, dict):
        raise ValueError('invalid nested native platforms')
    names = set(catalog) if catalog is not None else (
        _DECLARED_INTERACTIVE_NAMES | set(platforms) | set(nested))
    declared = {}
    for source in (nested, platforms, gateway, config):
        for name, value in source.items():
            if name not in names or name in NONINTERACTIVE_PLATFORMS or not isinstance(value, dict):
                continue
            declared.setdefault(name, None)
            if 'enabled' in value:
                token = str(value['enabled']).strip().lower()
                declared[name] = (True if token in {'true', '1', 'yes', 'on'} else
                                  False if token in {'false', '0', 'no', 'off'} else None)
    return declared


def configured_interactive_platforms(config, catalog):
    # Unknown/plugin surfaces need qualification; never infer a broad preset.
    configured = set(kanban_selections(config)) | {'cli'}
    configured.update(p for p, enabled in declared_interactive_platforms(config, catalog).items()
                      if enabled is not False)
    return sorted(configured & set(catalog) - NONINTERACTIVE_PLATFORMS)



def kanban_list(value):
    if isinstance(value, str):
        value = ast.literal_eval(value)
    if not isinstance(value, list) or any(not isinstance(v, str) for v in value):
        raise ValueError('invalid native tool selection')
    return value


def kanban_selections(config):
    platforms = config.get('platform_toolsets') or {}
    if not isinstance(platforms, dict):
        raise ValueError('invalid platform selections')
    result = {}
    for platform, value in platforms.items():
        if not isinstance(platform, str) or not re.fullmatch(r'[a-zA-Z0-9_-]{1,64}', platform):
            raise ValueError('invalid platform name')
        result[platform] = kanban_list(value)
    return result


def kanban_disabled(config):
    raw = (config.get('agent') or {}).get('disabled_toolsets') or []
    if isinstance(raw, str) and not raw.strip().startswith('['):
        return [v.strip() for v in raw.split(',') if v.strip()]
    return kanban_list(raw)


def default_kanban_updates(config):
    # Validate every selection before producing any mutation.
    lists = {'toolsets': kanban_list(config.get('toolsets') or [])}
    lists.update({'platform_toolsets.'+p: v for p,v in kanban_selections(config).items()})
    disabled = kanban_disabled(config)
    updates = {key: value+[tool for tool in DEFAULT_REQUIRED_TOOLSETS if tool not in value]
               for key,value in lists.items() if any(tool not in value for tool in DEFAULT_REQUIRED_TOOLSETS)}
    if any(tool in disabled for tool in DEFAULT_REQUIRED_TOOLSETS):
        updates['agent.disabled_toolsets'] = [v for v in disabled if v not in DEFAULT_REQUIRED_TOOLSETS]
    return updates


def worker_kanban_updates(config):
    return {'platform_toolsets.'+p: [v for v in values if v != 'kanban']
            for p,values in kanban_selections(config).items() if 'kanban' in values}


def default_toolset_states(config, toolset):
    selections = kanban_selections(config)
    fallback = toolset in kanban_list(config.get('toolsets') or [])
    disabled = toolset in kanban_disabled(config)
    # Only Kanban has the native global opt-in fallback. A saved global memory
    # token cannot certify any channel's effective memory selection.
    states = {'fallback': ('enabled' if fallback and not disabled else 'missing')
              if toolset == 'kanban' else 'not-configured'}
    declared = declared_interactive_platforms(config)
    for platform in sorted((set(selections) | set(declared) | {'cli'}) - NONINTERACTIVE_PLATFORMS):
        if platform in selections:
            states[platform] = 'enabled' if toolset in selections[platform] and not disabled else 'missing'
            if not disabled and toolset != 'kanban' and toolset not in selections[platform] and any(
                    value.startswith('hermes-') for value in selections[platform]):
                states[platform] = 'unknown'
        elif declared.get(platform) is False:
            states[platform] = 'not-configured'
        elif toolset == 'kanban':
            states[platform] = 'fallback' if fallback and not disabled else 'missing'
        else:
            states[platform] = 'missing' if disabled else 'unknown'
    return states


def default_kanban_states(config):
    return default_toolset_states(config, 'kanban')


def default_memory_states(config):
    return default_toolset_states(config, 'memory')


def managed_operational_dispatch(config):
    policy=config.get('kanban') or {}
    return (policy.get('dispatch_in_gateway') is True and policy.get('auto_decompose') is False
            and policy.get('review_dispatch') is True and type(policy.get('max_in_progress')) is int
            and policy['max_in_progress']==1 and policy.get('orchestrator_profile')=='default'
            and policy.get('dispatch_profiles')==['default','researcher','planner','executor','reviewer','steward'])

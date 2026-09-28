"""Bounded, read-only projection of channel declarations and runtime evidence.

This module deliberately does not import Hermes or read files/environment. Callers
must qualify ownership of the DEFAULT gateway before passing owner_verified=True.
It does not resolve credentials, plugins, environment expansion or managed policy.
Set config['_repokit_projection_unknown'] when a known unresolved overlay applies.
Routing labels describe declarations against the recorded served set, not a live
message trace. A recorded adapter state is never a connection probe.
"""

import json
import re


_NAME = re.compile(r'[a-z][a-z0-9_-]{0,79}\Z')
_PROGRAMMATIC = frozenset({'api_server', 'webhook', 'relay', 'cron', 'mcp', 'rpc', 'acp', 'a2a'})
_STATES = frozenset({'connected', 'connecting', 'retrying', 'disconnected', 'disabled',
                     'fatal', 'stopped', 'starting', 'degraded'})
_QUALIFIER = ('Declared configuration projection and recorded adapter state only; '
              'environment credentials, plugin hooks, managed overlays and live message routing '
              'are not resolved. Default routing means no declared applicable diversion was found; '
              'home channels are outbound destinations, not inbound profile-routing proof. '
              'Authorization labels describe declarations only; effective grants, pairing and '
              'environment allowlists are unknown, so restricted never certifies safe access.')

_AUTH_LIST_FIELDS = frozenset({'allow_from', 'group_allow_from', 'allowed_users'})
_AUTH_POLICY_FIELDS = frozenset({'dm_policy', 'group_policy'})


def _mapping(value):
    return value if isinstance(value, dict) else {}


def _name(value):
    return isinstance(value, str) and bool(_NAME.fullmatch(value))


def _unresolved(value):
    if isinstance(value, str):
        return '${' in value
    if isinstance(value, dict):
        return any(_unresolved(key) or _unresolved(item) for key, item in value.items())
    if isinstance(value, (list, tuple)):
        return any(_unresolved(item) for item in value)
    return False


def _boolean(value):
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        token = value.strip().lower()
        if token in {'true', '1', 'yes', 'on'}:
            return True
        if token in {'false', '0', 'no', 'off'}:
            return False
    return None


def _route_rows(config, legacy):
    gateway = _mapping(config.get('gateway'))
    routes = config.get('profile_routes')
    if routes is None:
        routes = gateway.get('profile_routes')
    if routes is None:
        routes = legacy.get('profile_routes', [])
    return routes if isinstance(routes, list) else None


def _routing(platform, routes, served, verified, uncertain):
    if platform == 'cli':
        return 'unknown' if uncertain else 'default'
    if not verified or uncertain or routes is None:
        return 'unknown'
    diversion = False
    for route in routes:
        if not isinstance(route, dict):
            return 'unknown'
        route_platform = route.get('platform')
        if not isinstance(route_platform, str) or _unresolved(route_platform):
            return 'unknown'
        if route_platform != platform:
            continue
        # Native route enabled uses truthiness, unlike platform bool coercion.
        # Only actual booleans are qualified here; strings are deliberately unknown.
        enabled = route.get('enabled', True)
        if enabled is False:
            continue
        if enabled is not True or _unresolved(route):
            return 'unknown'
        bot = route.get('bot_profile')
        if bot is not None and not isinstance(bot, str):
            return 'unknown'
        if bot and bot.strip() not in {'', 'default'}:
            continue
        target = route.get('profile')
        if not isinstance(target, str) or not target.strip():
            return 'unknown'
        for key in ('guild_id', 'chat_id', 'thread_id', 'user_id'):
            value = route.get(key)
            if value is not None and (isinstance(value, bool) or not isinstance(value, (str, int))):
                return 'unknown'
        if 'user_id' in route and (route['user_id'] is None or not str(route['user_id']).strip()):
            return 'unknown'
        # Config mode flags cannot prove the boot-time multiplex verdict.
        if not isinstance(served, list) or 'default' not in served or target not in served:
            return 'unknown'
        if target != 'default':
            diversion = True
    return 'conditional_other' if diversion else 'default'


def _selection(platform, config, catalog, uncertain):
    if uncertain:
        return 'unknown'
    entry = catalog[platform]
    required = entry.get('required')
    if not isinstance(required, list) or not required or not all(_name(item) for item in required):
        return 'unknown'
    selections = _mapping(config.get('platform_toolsets'))
    selected = selections.get(platform)
    if platform not in selections and platform == 'cli':
        selected = config.get('toolsets')
    if selected is None and entry.get('default_qualified') is True:
        selected = [entry.get('preset')]
    agent = config.get('agent', {})
    if not isinstance(agent, dict):
        return 'unknown'
    disabled = agent.get('disabled_toolsets', [])
    if not all(isinstance(value, list) and all(_name(item) for item in value)
               for value in (selected, disabled)):
        return 'unknown'
    aliases, known = {}, set()
    for item in catalog.values():
        categories = item.get('required')
        if not isinstance(categories, list) or not all(_name(name) for name in categories):
            continue
        known.update(categories)
        # Platform presets do not opt into Kanban. Required policy categories
        # must never be mistaken for the categories a native preset supplies.
        native = item.get('native_categories', [name for name in categories if name != 'kanban'])
        if _name(item.get('preset')) and isinstance(native, list) and all(_name(name) for name in native):
            aliases[item['preset']] = set(native)
            known.update(native)
    enabled_categories = set()
    unknown = False
    for item in selected:
        enabled_categories.update(aliases.get(item, {item}))
        unknown |= item not in aliases and item not in known
    disabled_categories = set()
    for item in disabled:
        disabled_categories.update(aliases.get(item, {item}))
        if item not in aliases and item not in known:
            return 'unknown'
    enabled_categories -= disabled_categories
    if set(required) <= enabled_categories:
        return 'complete'
    if set(required) & disabled_categories:
        return 'incomplete'
    return 'unknown' if unknown else 'incomplete'


def _authorization(platform, config, legacy, uncertain):
    """Return evidence of declared authorization, NEVER effective authorization.

    The pinned gateway unions env grants, pairing and adapter authorization.
    No raw YAML declaration can certify a restricted effective posture. Conflicting
    layers deliberately remain unknown rather than duplicating plugin bridges.
    A policy of 'open' alone is not a grant: central auth does not trust it.
    """
    if uncertain or platform in {'cli', 'homeassistant'}:
        return 'unknown'
    gateway = _mapping(config.get('gateway'))
    global_all = config.get('allow_all_users')
    if global_all is None:
        global_all = gateway.get('allow_all_users')
    if _unresolved(global_all):
        return 'unknown'
    # Native YAML bridge's truth tokens (not the more permissive platform enabled parser).
    open_grant = str(global_all).lower() in {'true', '1', 'yes'}
    values = {}
    fields = _AUTH_LIST_FIELDS | _AUTH_POLICY_FIELDS
    # Discord's YAML allow-all bridge is source-qualified. Other platform-specific
    # open flags are env/plugin surfaces and are not inferred here.
    if platform == 'discord':
        fields = fields | {'allow_all_users'}
    for source in (legacy.get('platforms'), gateway.get('platforms'), config.get('platforms'), gateway, config):
        block = _mapping(_mapping(source).get(platform))
        if 'extra' in block and not isinstance(block['extra'], dict):
            return 'unknown'
        for layer in (block, _mapping(block.get('extra'))):
            for key in fields & layer.keys():
                value = layer[key]
                if _unresolved(value) or (key in values and values[key] != value):
                    return 'unknown'
                values[key] = value
    restricted = False
    for key, raw in values.items():
        if key == 'allow_all_users':
            open_grant |= str(raw).lower() in {'true', '1', 'yes'}
        elif key in _AUTH_POLICY_FIELDS:
            if not isinstance(raw, str):
                return 'unknown'
            policy = raw.strip().lower()
            if policy not in {'open', 'allowlist', 'disabled', 'pairing'}:
                return 'unknown'
            restricted |= policy in {'allowlist', 'disabled', 'pairing'}
        else:
            if isinstance(raw, str):
                if raw.lstrip().startswith('['):
                    try:
                        raw = json.loads(raw)
                    except (ValueError, TypeError):
                        return 'unknown'
                else:
                    raw = raw.split(',')
            if not isinstance(raw, list) or any(isinstance(value, bool) or not isinstance(value, (str, int)) for value in raw):
                return 'unknown'
            # Values stay local to this pure projection; never returned, logged or counted.
            entries = {str(value).strip() for value in raw if str(value).strip()}
            open_grant |= '*' in entries
            restricted |= bool(entries) and '*' not in entries
    if open_grant:
        return 'open'
    return 'restricted' if restricted else 'unknown'


def project_channels(config, legacy, state, owner_verified, catalog):
    """Return safe rows and a fixed qualifier from already-loaded input mappings.

    Catalog is trusted pinned metadata: platform -> {preset, required: [categories],
    native_categories: [categories actually supplied by the preset]}.
    If native_categories is omitted, the bounded fallback excludes opt-in kanban.
    An optional default_qualified=True qualifies that entry's native default preset.
    No caller-supplied free text, route IDs, profile names or errors are returned.
    Unknown plugins absent from the catalog cannot be classified by this projection.
    """
    uncertain = not isinstance(config, dict) or not isinstance(legacy, dict)
    config, legacy, state = map(_mapping, (config, legacy, state))
    owner_verified = owner_verified is True
    catalog = {name: value for name, value in _mapping(catalog).items()
               if _name(name) and name not in _PROGRAMMATIC and isinstance(value, dict)}
    uncertain |= bool(config.get('_repokit_projection_unknown'))
    uncertain |= 'gateway' in config and not isinstance(config['gateway'], dict)
    gateway = _mapping(config.get('gateway'))
    declared = {}
    # Native order: legacy -> nested platforms -> root platforms -> gateway.<platform>
    # -> root <platform> enabled bridge. Never copy tokens or adapter extras.
    for source in (legacy.get('platforms'), gateway.get('platforms'), config.get('platforms'), gateway, config):
        for platform, block in _mapping(source).items():
            if platform not in catalog or not isinstance(block, dict):
                continue
            declared.setdefault(platform, None)
            if 'enabled' in block:
                declared[platform] = _boolean(block['enabled'])
    saved = set(_mapping(config.get('platform_toolsets'))) & set(catalog)
    recorded = {}
    for key, value in _mapping(state.get('platforms')).items():
        if not isinstance(key, str) or not isinstance(value, dict):
            continue
        owner, separator, platform = key.partition(':')
        platform = platform if separator else owner
        if platform in catalog:
            recorded.setdefault(platform, []).append((bool(separator), value.get('state')))
    platforms = set(declared) | saved | set(recorded)
    if 'cli' in catalog:
        platforms.add('cli')
    routes = _route_rows(config, legacy)
    rows = []
    for platform in sorted(platforms):
        entries = recorded.get(platform, [])
        default_entries = [status for secondary, status in entries if not secondary]
        status = default_entries[0] if default_entries else entries[0][1] if entries else None
        owner = 'unknown'
        if owner_verified and entries:
            owner = 'default' if default_entries else 'other'
        evidence = 'recorded' if entries else 'declared'
        if platform == 'cli':
            evidence, owner = 'cli', 'default'
        metadata = catalog[platform]
        rows.append({
            'platform': platform,
            'declared_enabled': {True: 'yes', False: 'no', None: 'unknown'}[declared.get(platform)],
            'evidence': evidence,
            'recorded_adapter_state': status if isinstance(status, str) and status in _STATES else 'unknown',
            'owner': owner,
            'default_routing': _routing(platform, routes, state.get('served_profiles'), owner_verified,
                                        uncertain or owner == 'other'),
            'preset': metadata.get('preset') if _name(metadata.get('preset')) else '',
            'required': [item for item in metadata.get('required', []) if _name(item)]
                        if isinstance(metadata.get('required'), list) else [],
            'core_selection': _selection(platform, config, catalog, uncertain or owner == 'other'),
            'authorization': _authorization(platform, config, legacy, uncertain or owner == 'other'),
            'authorization_scope': 'declared',
            'authorization_effective': 'unknown',
        })
    return {'rows': rows, 'qualifier': _QUALIFIER}

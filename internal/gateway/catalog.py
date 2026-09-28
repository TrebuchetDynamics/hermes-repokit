"""Read platform metadata as syntax, without importing Hermes or loading secrets."""
import ast


def platform_catalog(source):
    # This is the product's core contract, not a copied native preset. Provisioning
    # resolves the full installed preset through Hermes; this observer checks only
    # saved core selections and never claims credential-dependent tool availability.
    required = ['kanban', 'memory', 'file', 'terminal', 'web', 'skills', 'todo', 'code_execution']
    tree = ast.parse(source)
    catalog = {}
    for node in tree.body:
        if not isinstance(node, ast.AnnAssign) or not isinstance(node.target, ast.Name) or node.target.id != 'PLATFORMS':
            continue
        value = node.value
        if not isinstance(value, ast.Call) or not isinstance(value.func, ast.Name) or value.func.id != 'OrderedDict' or len(value.args) != 1 or not isinstance(value.args[0], ast.List):
            raise ValueError('unqualified platform registry shape')
        for entry in value.args[0].elts:
            if not isinstance(entry, ast.Tuple) or len(entry.elts) != 2:
                raise ValueError('unqualified platform entry')
            name = ast.literal_eval(entry.elts[0])
            info = entry.elts[1]
            if not isinstance(name, str) or not isinstance(info, ast.Call) or not isinstance(info.func, ast.Name) or info.func.id != 'PlatformInfo':
                raise ValueError('unqualified platform entry')
            preset = next(ast.literal_eval(k.value) for k in info.keywords if k.arg == 'default_toolset')
            if name not in ('acp', 'api_server', 'cron', 'webhook'):
                catalog[name] = {'preset': preset, 'required': required[:], 'default_qualified': False}
    if 'cli' not in catalog:
        raise ValueError('platform registry unavailable')
    return catalog

"""Network-disabled native resolver/schema acceptance, with synthetic home only."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

with tempfile.TemporaryDirectory(prefix='repokit-kanban-') as directory:
    root=Path(directory)
    os.environ['HOME']=directory
    os.environ['HERMES_HOME']=directory
    for key in ('HERMES_PROFILE','HERMES_PROFILE_NAME','HERMES_CONFIG','HERMES_ENV','HERMES_KANBAN_TASK'):
        os.environ.pop(key,None)
    sys.path.insert(0,'/opt/hermes')
    import yaml
    from hermes_cli.tools_config import _get_platform_tools
    from hermes_cli.platforms import PLATFORMS
    from model_tools import get_tool_definitions

    calls=[]
    def run(*args):
        calls.append(args)
        subprocess.run([sys.executable,'-m','hermes_cli.main',*args],cwd='/opt/hermes',check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)

    catalog=native_interactive_catalog()
    assert 'api_server' not in catalog and 'cron' not in catalog and 'webhook' not in catalog
    assert catalog['whatsapp_cloud']['preset']=='hermes-whatsapp'
    # Exercise every qualified saved human platform and leave one absent to
    # ensure installation does not freeze unconfigured native platform defaults.
    saved={p:[] for p in catalog if p!='matrix'}
    saved.update({'cli':['hermes-cli'], 'telegram':['web','spotify','owner_mcp'],
                  'slack':"['hermes-slack']", 'mattermost':"['memory']",
                  'custom': ['owner_mcp'], 'api_server':['web'], 'acp':['file'],
                  'cron':['memory'], 'webhook':['web']})
    config={'toolsets':['web'], 'platform_toolsets':saved,
            'agent':{'disabled_toolsets':['kanban','memory','terminal','video']},
            'kanban':{'dispatch_in_gateway':False}}
    (root/'config.yaml').write_text(yaml.safe_dump(config))
    assert 'kanban' not in _get_platform_tools(config,'telegram',include_default_mcp_servers=False)
    reconcile_default_kanban(root,run)
    observed=yaml.safe_load((root/'config.yaml').read_text())
    assert observed['toolsets']==['web','kanban','memory']
    assert observed['agent']['disabled_toolsets']==['video']
    assert observed['kanban']['dispatch_in_gateway'] is False
    assert 'matrix' not in observed['platform_toolsets'], 'must not freeze absent platform defaults'
    assert {'spotify','owner_mcp'} <= set(observed['platform_toolsets']['telegram'])
    for platform in ('custom','api_server','acp','cron','webhook'):
        assert observed['platform_toolsets'][platform]==config['platform_toolsets'][platform], platform
    assert not any(c[2:4]==('config','set') and c[4].startswith(('platform_toolsets','agent.disabled_toolsets','known_')) for c in calls)

    for platform, info in catalog.items():
        enabled=_get_platform_tools(observed,platform,include_default_mcp_servers=False)
        assert {'kanban','memory'} <= set(enabled), platform
        if platform=='matrix':
            continue
        baseline={'platform_toolsets':{platform:[info['preset'],'kanban','memory']}}
        native_enabled=_get_platform_tools(baseline,platform,include_default_mcp_servers=False)
        assert native_enabled <= enabled, (platform,native_enabled-enabled)
        # Native persistence owns both builtin bookkeeping and disabled-list
        # reconciliation; a synthetic dict-only unit fixture cannot prove this.
        assert observed['known_builtin_toolsets'][platform]
        assert {'kanban','memory'} <= set(observed['platform_toolsets'][platform])

    # CLI is the native selection shared by CLI, TUI and Desktop. New Telegram
    # conversations get the same file/terminal/tool inventory as their preset.
    for platform in ('cli','telegram'):
        enabled=_get_platform_tools(observed,platform,include_default_mcp_servers=False)
        baseline={'platform_toolsets':{platform:[catalog[platform]['preset'],'kanban','memory']}}
        native_enabled=_get_platform_tools(baseline,platform,include_default_mcp_servers=False)
        definitions=get_tool_definitions(enabled_toolsets=sorted(enabled),skip_tool_search_assembly=True)
        baseline_definitions=get_tool_definitions(enabled_toolsets=sorted(native_enabled),skip_tool_search_assembly=True)
        names={tool['function']['name'] for tool in definitions}
        baseline_names={tool['function']['name'] for tool in baseline_definitions}
        assert {'kanban_list','kanban_create'} <= names, (platform,names)
        assert baseline_names <= names, (platform,baseline_names-names)
        assert {'file','terminal','web','skills','memory'} <= enabled, (platform,enabled)

    # A new native specialist clone must lose the orchestrator's broad saved
    # selections. Exercise the actual configure path, using a synthetic profile.
    worker_home=root/'profiles'/'executor'
    worker_home.mkdir(parents=True)
    (worker_home/'config.yaml').write_text(yaml.safe_dump(copy.deepcopy(observed)))
    (worker_home/'profile.yaml').write_text(yaml.safe_dump({'description':'fixture executor'}))
    role={'name':'executor','toolsets':['file','memory'],'description':'fixture executor','soul':'fixture executor'}
    configure(worker_home,'executor',role,run)
    worker=yaml.safe_load((worker_home/'config.yaml').read_text())
    for platform in configured_interactive_platforms(worker,catalog):
        assert kanban_selections(worker)[platform]==role['toolsets'], platform
        enabled=_get_platform_tools(worker,platform,include_default_mcp_servers=False)
        assert not {'kanban','terminal','web','browser'} & enabled, (platform,enabled)
    assert worker['toolsets']==role['toolsets']

    before=(root/'config.yaml').read_bytes()
    count=len(calls)
    reconcile_default_kanban(root,run)
    assert (root/'config.yaml').read_bytes()==before, 'rerun rewrote matching configuration'
    assert len(calls)==count, 'rerun issued native mutations'
    print('NATIVE_KANBAN_MEMORY_CHANNELS_PASS')

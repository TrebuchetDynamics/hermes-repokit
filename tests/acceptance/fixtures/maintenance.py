"""Credential-free scanner/install/registry qualification in a disposable image.

No gateway is started and no restart may be requested. A CAUTION/DANGEROUS scan,
missing native installation receipt, or incompatible registry contract fails.
"""
import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from unittest.mock import patch


def maintenance_acceptance(package):
    with tempfile.TemporaryDirectory(prefix='repokit-maintenance-acceptance-') as temporary:
        root=Path(temporary)/'.hermes'
        root.mkdir()
        os.environ['HOME']=temporary
        os.environ['HERMES_HOME']=str(root)
        os.environ['PYTHONDONTWRITEBYTECODE']='1'
        for key in ('HERMES_PROFILE','HERMES_PROFILE_NAME','HERMES_CONFIG','HERMES_ENV',
                    'HERMES_KANBAN_TASK','HERMES_CRON_SESSION','HERMES_SESSION_PLATFORM'):
            os.environ.pop(key,None)
        sys.path.insert(0,'/opt/hermes')
        import yaml
        name=package['name']
        files=package['files']
        config={'model':{'default':'fixture'},
                'toolsets':['kanban','memory'],
                'platform_toolsets':{'cli':['file','memory'],'telegram':['web','memory'],
                                     'api_server':['web'],'cron':['file'],'webhook':['web']},
                'kanban':{'dispatch_in_gateway':False},
                'plugins':{'scan_on_install':True}}
        (root/'config.yaml').write_text(yaml.safe_dump(config))
        (root/'SOUL.md').write_text('Synthetic maintenance qualification only.\n')
        calls=[]
        def run(*args):
            calls.append(args)
            subprocess.run([sys.executable,'-B','-m','hermes_cli.main',*args],
                           cwd='/opt/hermes',env=maintenance_git_env(),check=True,
                           stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)

        # Admission uses the pinned scanner and native installer without force,
        # reviewed-pin claims, scanner switches, dependencies or live credentials.
        provision_maintenance(root,files,name,run)
        target=root/'plugins'/name
        assert exact_maintenance_tree(target,files), 'installed source differs from package'
        install=next(args for args in calls if args[2:4]==('plugins','install'))
        assert '--no-enable' in install and '--force' not in install
        metadata=json.loads((root/'plugins'/'.install-metadata.json').read_text())
        record=metadata[name]
        assert record['pinned'] is True
        assert record['revision']==install[install.index('--ref')+1]
        assert len(record['revision'])==40
        observed=yaml.safe_load((root/'config.yaml').read_text())
        assert maintenance_enabled(observed,name)
        for platform in ('cli','telegram'):
            assert name in observed['platform_toolsets'][platform]
        for platform in ('api_server','cron','webhook'):
            assert observed['platform_toolsets'][platform]==config['platform_toolsets'][platform]
        assert not (root/'profiles').exists(), 'provisioner created a specialist'
        assert not any(args[1]!='default' for args in calls)

        # Load actual PluginContext and scoped registry contracts. Availability
        # remains false in this unsupervised temporary home; inspect registration
        # schemas directly instead of faking a gateway or bypassing check_fn.
        from hermes_cli.plugins import discover_plugins
        from tools.registry import registry
        discover_plugins(force=True)
        names={'gateway_restart_after_turn','gateway_restart_status'}
        for tool in names:
            entry=registry.get_entry(tool)
            assert entry is not None, ('native plugin registration failed',tool)
            assert registry.get_toolset_for_tool(tool)==name
            schema=registry.get_schema(tool)
            assert schema['name']==tool
            assert schema['parameters']['type']=='object'
            assert schema['parameters']['properties']=={}
            assert schema['parameters']['additionalProperties'] is False
            assert entry.check_fn() is False
        assert registry.get_definitions(names,quiet=True)==[]
        with patch('gateway.control_socket.pause_gateway_for_update',side_effect=AssertionError('fixture must not restart')):
            request=json.loads(registry.dispatch('gateway_restart_after_turn',{}))
            status=json.loads(registry.dispatch('gateway_restart_status',{}))
        assert request['state']=='refused'
        assert status['state']=='unavailable'

        before_config=(root/'config.yaml').read_bytes()
        before_metadata=(root/'plugins'/'.install-metadata.json').read_bytes()
        count=len(calls)
        provision_maintenance(root,files,name,run)
        assert len(calls)==count, 'matching rerun issued native mutations'
        assert (root/'config.yaml').read_bytes()==before_config
        assert (root/'plugins'/'.install-metadata.json').read_bytes()==before_metadata
        assert exact_maintenance_tree(target,files)

        # Give a synthetic named profile the exact enabled plugin as a defensive
        # negative case. Real provisioning above only enabled the default profile.
        worker=root/'profiles'/'executor'
        worker.mkdir(parents=True)
        worker_config=copy.deepcopy(observed)
        worker_config['platform_toolsets']={'cli':[name]}
        (worker/'config.yaml').write_text(yaml.safe_dump(worker_config))
        (worker/'SOUL.md').write_text('Synthetic named worker.\n')
        (worker/'plugins').mkdir()
        shutil.copytree(target,worker/'plugins'/name)
        worker_env=maintenance_git_env()
        worker_env.update({'HERMES_HOME':str(worker),'HERMES_PROFILE_NAME':'executor',
                           'HERMES_SESSION_PLATFORM':'telegram'})
        worker_script='''
import json
import sys
from unittest.mock import patch
sys.path.insert(0,'/opt/hermes')
from hermes_cli.profiles import get_active_profile_name
from hermes_cli.plugins import discover_plugins
from tools.registry import registry
assert get_active_profile_name()=='executor'
discover_plugins(force=True)
names={'gateway_restart_after_turn','gateway_restart_status'}
for name in names:
    entry=registry.get_entry(name)
    assert entry is not None, name
    assert entry.check_fn() is False
assert registry.get_definitions(names,quiet=True)==[]
with patch('gateway.control_socket.pause_gateway_for_update',side_effect=AssertionError('named profile must not restart')):
    request=json.loads(registry.dispatch('gateway_restart_after_turn',{}))
    status=json.loads(registry.dispatch('gateway_restart_status',{}))
assert request['state']=='refused', request
assert request['reason']=='human_default_gateway_context_required', request
assert status['state']=='unavailable', status
print('NATIVE_MAINTENANCE_NAMED_PROFILE_REFUSED')
'''
        result=subprocess.run([sys.executable,'-B','-c',worker_script],cwd='/opt/hermes',
                              env=worker_env,check=True,stdin=subprocess.DEVNULL,
                              stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
        assert 'NATIVE_MAINTENANCE_NAMED_PROFILE_REFUSED' in result.stdout

        # A disabled scanner is never interpreted as an admission receipt, even
        # when exact installed source and enablement already exist.
        disabled=copy.deepcopy(observed)
        disabled['plugins']['scan_on_install']=False
        (root/'config.yaml').write_text(yaml.safe_dump(disabled))
        try:
            provision_maintenance(root,files,name,run)
        except RuntimeError as error:
            assert 'scan' in str(error)
        else:
            raise AssertionError('disabled scanner accepted')
        assert len(calls)==count
        print('NATIVE_MAINTENANCE_PACKAGE_PASS')

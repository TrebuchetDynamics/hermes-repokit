"""One-shot native plugin provisioning; never imported as a runtime controller."""
import contextlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import yaml


def maintenance_config(root):
    value=yaml.safe_load((root/'config.yaml').read_text()) or {}
    if not isinstance(value,dict):
        raise RuntimeError('invalid native maintenance configuration')
    return value


def maintenance_git_env():
    # Both staging and native cloning must ignore owner Git identity, hooks,
    # injected config and signing; no global Git configuration is written.
    env={key:value for key,value in os.environ.items() if not key.startswith('GIT_')}
    env.update({'GIT_CONFIG_NOSYSTEM':'1','GIT_CONFIG_SYSTEM':'/dev/null',
                'GIT_CONFIG_GLOBAL':'/dev/null','GIT_TERMINAL_PROMPT':'0',
                'GIT_ALLOW_PROTOCOL':'file','GIT_AUTHOR_NAME':'Hermes RepoKit',
                'GIT_AUTHOR_EMAIL':'repokit@localhost','GIT_COMMITTER_NAME':'Hermes RepoKit',
                'GIT_COMMITTER_EMAIL':'repokit@localhost',
                'GIT_AUTHOR_DATE':'2000-01-01T00:00:00+00:00',
                'GIT_COMMITTER_DATE':'2000-01-01T00:00:00+00:00',
                'PYTHONDONTWRITEBYTECODE':'1'})
    return env


def stage_revision(directory):
    command=['git','-c','core.hooksPath=/dev/null','-c','commit.gpgsign=false',
             '-c','init.defaultBranch=repokit','-C',str(directory)]
    env=maintenance_git_env()
    for args in (['init','--quiet'],['add','--','__init__.py','plugin.yaml'],
                 ['commit','--quiet','-m','Bundled native maintenance adapter']):
        subprocess.run([*command,*args],env=env,check=True,stdin=subprocess.DEVNULL,
                       stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    result=subprocess.run([*command,'rev-parse','HEAD'],env=env,check=True,
                          stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True)
    revision=result.stdout.strip()
    if not re.fullmatch(r'[0-9a-f]{40}',revision):
        raise RuntimeError('invalid maintenance source revision')
    return revision


def native_scan(directory):
    from hermes_cli.plugins_cmd import _scan_on_install_enabled, _scan_plugin_tree
    if not _scan_on_install_enabled():
        return False
    # Do not prompt, accept caution, invent a reviewed pin, or expose diagnostics.
    with open(os.devnull,'w') as sink,contextlib.redirect_stdout(sink),contextlib.redirect_stderr(sink):
        result=_scan_plugin_tree(directory,'RepoKit bundled maintenance',force=False)
    return result is not None


def exact_maintenance_tree(directory,files):
    if directory.is_symlink() or not directory.is_dir():
        return False
    found={}
    for path in directory.rglob('*'):
        relative=path.relative_to(directory)
        if relative.parts[0]=='.git':
            if relative==Path('.git') and path.is_symlink():
                return False
            continue
        if path.is_symlink():
            return False
        if path.is_dir():
            if str(relative)!='__pycache__':
                return False
            continue
        if len(relative.parts)==2 and relative.parts[0]=='__pycache__' and path.suffix=='.pyc':
            continue
        if str(relative) not in files:
            return False
        found[str(relative)]=path.read_bytes()
    return found=={name:body.encode('utf-8') for name,body in files.items()}


def maintenance_enabled(config,name):
    plugins=config.get('plugins') or {}
    return name in (plugins.get('enabled') or []) and name not in (plugins.get('disabled') or [])


def provision_maintenance(root,files,name,run):
    if set(files)!={'__init__.py','plugin.yaml'} or not re.fullmatch(r'[a-z][a-z0-9_]{1,63}',name):
        raise RuntimeError('invalid bundled maintenance package')
    catalog=native_interactive_catalog()
    platforms=default_interactive_platforms(root,maintenance_config(root),catalog)
    plugins=root/'plugins'
    if plugins.is_symlink():
        raise RuntimeError('maintenance plugin drift preserved')
    target=plugins/name
    present=target.exists() or target.is_symlink()
    if present and not exact_maintenance_tree(target,files):
        raise RuntimeError('maintenance plugin drift preserved')
    if present:
        if native_scan(target) is not True:
            raise RuntimeError('native maintenance scan did not admit package')
    else:
        with tempfile.TemporaryDirectory(prefix='.repokit-maintenance-',dir=root) as temporary:
            source=Path(temporary)
            for filename,body in files.items():
                (source/filename).write_text(body)
            if native_scan(source) is not True:
                raise RuntimeError('native maintenance scan did not admit package')
            revision=stage_revision(source)
            run('-p','default','plugins','install',source.as_uri(),'--ref',revision,'--no-enable')
        if not exact_maintenance_tree(target,files):
            raise RuntimeError('native maintenance installation did not resolve')
    config=maintenance_config(root)
    if not maintenance_enabled(config,name):
        run('-p','default','plugins','enable',name)
        if not maintenance_enabled(maintenance_config(root),name):
            raise RuntimeError('native maintenance plugin enable did not resolve')
    for platform in platforms:
        config=maintenance_config(root)
        selection=(config.get('platform_toolsets') or {}).get(platform,[])
        disabled=(config.get('agent') or {}).get('disabled_toolsets') or []
        if isinstance(selection,list) and name in selection and name not in disabled:
            continue
        run('-p','default','tools','enable',name,'--platform',platform)
        config=maintenance_config(root)
        selection=(config.get('platform_toolsets') or {}).get(platform,[])
        disabled=(config.get('agent') or {}).get('disabled_toolsets') or []
        if not isinstance(selection,list) or name not in selection or name in disabled:
            raise RuntimeError('native maintenance tool enable did not resolve')


def maintenance_main(payload):
    root=Path(os.environ['HERMES_HOME'])
    def run(*args):
        result=subprocess.run(['hermes',*args],env=maintenance_git_env(),stdin=subprocess.DEVNULL,
                              stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        if result.returncode:
            raise RuntimeError('native maintenance operation failed')
    provision_maintenance(root,payload['files'],payload['name'],run)
    print('REPOKIT_MAINTENANCE=configured')

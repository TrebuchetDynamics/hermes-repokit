"""Read-only development checks; no Hermes, provider, plugin or project imports."""
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import yaml

TOOLS={'git':('/usr/bin/git','--version'),'bash':('/usr/bin/bash','--version'),
 'curl':('/usr/bin/curl','--version'),'jq':('/usr/local/bin/jq','--version'),
 'rg':('/usr/bin/rg','--version'),'python':('/usr/bin/python3','--version'),
 'node':('/usr/local/bin/node','--version'),'npm':('/usr/local/bin/npm','--version'),
 'make':('/usr/bin/make','--version'),'gcc':('/usr/bin/gcc','--version'),
 'g++':('/usr/bin/g++','--version'),'docker':('/usr/bin/docker','--version'),
 'compose':('/usr/bin/docker','compose','version','--short'),'buildx':('/usr/bin/docker','buildx','version'),'go':('/usr/local/go/bin/go','version')}

def inspect_tools(go_required, run=subprocess.run):
    env={'PATH':'/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin','HOME':'/nonexistent',
         'GOTOOLCHAIN':'local','GOCACHE':'off','npm_config_userconfig':'/nonexistent/npm-user-config',
         'npm_config_globalconfig':'/nonexistent/npm-global-config','NODE_OPTIONS':'','LC_ALL':'C'}
    result={}
    for name,cmd in TOOLS.items():
        if name=='go' and not go_required: continue
        try:
            p=run(cmd,env=env,cwd='/workspace',stdin=subprocess.DEVNULL,
                  stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,timeout=10)
            # Selected version probes are bounded, known image binaries. Keep only
            # ordinary first-line version text; never echo arbitrary stderr.
            line=p.stdout.splitlines()[0] if p.stdout else ''
            result[name]={'ok':p.returncode==0 and bool(line),'version':line[:160] if re.fullmatch(r'[\w .()+,/:~+\-]+',line or '') else 'unrecognized'}
        except (OSError,subprocess.TimeoutExpired):result[name]={'ok':False,'version':'unavailable'}
    return result

def runtime_commands(go_required, root=Path('/')):
    # Observe immutable image links only; never source profile startup files in
    # passive verification. The Docker fixture exercises native shell snapshots.
    commands={'hermes':'opt/hermes/bin/hermes'}
    if go_required:
        commands.update({'go':'usr/local/go/bin/go','gofmt':'usr/local/go/bin/gofmt'})
    try:
        for name,target in commands.items():
            link=root/'usr/local/bin'/name
            expected=root/target
            if not link.is_symlink() or link.resolve(strict=True)!=expected.resolve(strict=True):
                return False
            if not expected.is_file() or not os.access(expected,os.X_OK):
                return False
        return True
    except (OSError,RuntimeError):
        return False


def native_terminals(root):
    result={}
    for name in ('default','researcher','planner','executor','reviewer','steward'):
        path=root if name=='default' else root/'profiles'/name
        try:
            if path.is_symlink() or (root/'profiles').is_symlink():raise ValueError('redirected')
            fd=os.open(path/'config.yaml',os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK)
            with os.fdopen(fd,'rb') as f:
                info=os.fstat(f.fileno())
                if not stat.S_ISREG(info.st_mode) or info.st_size>262144:raise ValueError('unsafe')
                data=f.read(262145)
            if len(data)>262144:raise ValueError('oversized')
            config=yaml.safe_load(data);terminal=config.get('terminal') or {}
            result[name]=terminal.get('cwd')=='/workspace' and terminal.get('backend','local')=='local'
        except Exception:result[name]=False
    return result

if __name__=='__main__':
    tools=inspect_tools(sys.argv[1]=='go')
    print(json.dumps({'tools':tools,'workspace':os.getcwd()=='/workspace' and os.access('/workspace',os.R_OK|os.W_OK|os.X_OK),
      'runtime_commands':runtime_commands(sys.argv[1]=='go'),'certificates':Path('/etc/ssl/certs/ca-certificates.crt').is_file(),'profiles':native_terminals(Path('/opt/data'))}))

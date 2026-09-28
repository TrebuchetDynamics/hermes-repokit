"""One-shot setup gates. Native doctor owns model/provider validation."""
import json
import os
from pathlib import Path
import subprocess
import sys
import time


def validate(config):
    # Explicit configuration prevents a future cwd change moving durable data.
    if config.get('storage', {}).get('workspace') != '/opt/data/openviking/data':
        raise RuntimeError('persistent workspace required')
    server = config.get('server', {})
    # Native init creates auth only in its Remote mode. The installed service
    # wrapper overrides the listener to loopback for both native config shapes.
    if server.get('host') not in ('127.0.0.1', '0.0.0.0') or server.get('port') != 1933 or not server.get('root_api_key'):
        raise RuntimeError('private API-key service required')
    if config.get('memory', {}).get('extraction_enabled', True) is not True:
        raise RuntimeError('native session extraction required')


def health():
    deadline = time.monotonic() + 120
    while time.monotonic() < deadline:
        try:
            result = subprocess.run(['repokit-openviking', 'health'],
                                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=5)
            if result.returncode == 0:
                return
        except subprocess.TimeoutExpired:
            pass
        time.sleep(1)
    raise RuntimeError('native health timeout')


if __name__ == '__main__':
    try:
        if sys.argv[1] == 'validate':
            validate(json.loads(os.path.expandvars(Path('/opt/data/openviking/ov.conf').read_text())))
        elif sys.argv[1] == 'health':
            health()
        else:
            raise RuntimeError('unknown setup gate')
    except Exception:
        print('OpenViking setup gate failed. Check native doctor, API-key binding, persistent workspace /opt/data/openviking/data, session extraction and service health before activation.', file=sys.stderr)
        sys.exit(1)

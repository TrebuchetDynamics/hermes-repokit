"""Observe only config selections; never load Hermes, plugins, or credentials."""
import json
import os
from pathlib import Path
import stat
import sys
import yaml


def observe(root):
    fd=os.open(root/'config.yaml', os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as source:
        meta=os.fstat(source.fileno())
        if not stat.S_ISREG(meta.st_mode) or meta.st_size > 262144:
            raise ValueError('invalid native config')
        data=source.read(262145)
    if len(data)>262144: raise ValueError('oversized native config')
    config=yaml.safe_load(data)
    if not isinstance(config,dict): raise ValueError('invalid native config')
    kanban=config.get('kanban', {})
    dispatch='invalid'
    if isinstance(kanban,dict):
        if 'dispatch_in_gateway' not in kanban:
            dispatch='missing'
        elif kanban['dispatch_in_gateway'] is False:
            dispatch='manual'
        elif kanban['dispatch_in_gateway'] is True:
            dispatch='enabled'
    notifications = 'unknown'
    if isinstance(kanban,dict):
        values=[kanban.get(key,True) for key in ('auto_subscribe_on_create','notify_in_gateway')]
        if all(value is True for value in values): notifications='enabled'
        elif any(value is False for value in values): notifications='disabled'
    return {'kanban': default_kanban_states(config),
            'memory': default_memory_states(config), 'dispatch': dispatch,
            'notifications':notifications}


if __name__=='__main__':
    try:
        print(json.dumps(observe(Path(sys.argv[1]))))
    except Exception:
        print('{"kanban":{"fallback":"invalid"},"memory":{"fallback":"invalid"},"dispatch":"invalid"}')

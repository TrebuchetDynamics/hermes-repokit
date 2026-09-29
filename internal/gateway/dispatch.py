"""RepoKit policy and passive evidence, not a second dispatcher or task store."""
import re
from pathlib import Path

TEAM = ('default', 'researcher', 'planner', 'executor', 'reviewer', 'steward')


def dispatch_policy(role):
    return dict(dispatch_in_gateway=role == 'default', auto_decompose=False,
                review_dispatch=True, max_in_progress=1, orchestrator_profile='default',
                dispatch_profiles=list(TEAM))


def matches_dispatch_policy(config, expected):
    policy = config.get('kanban') or {}
    return (isinstance(policy, dict) and type(policy.get('max_in_progress')) is int
            and all(type(policy.get(k)) is type(v) and policy[k] == v
                    for k, v in expected.items()))


def operational_policy(config):
    return matches_dispatch_policy(config, dispatch_policy('default'))


def canary_policy(config):
    return matches_dispatch_policy(config, dict(dispatch_policy('default'), dispatch_profiles=['researcher']))


def dispatch_live(configured, generation_state, owns_lock, marker, paused):
    if configured is False:
        return 'stale' if owns_lock else 'disabled'
    if configured is not True:
        return 'unknown'
    if paused:
        return 'paused'
    evidence = (marker or {}).get('dispatch') or {}
    if (generation_state != 'current' or not owns_lock or
            evidence.get('max_in_progress') != 1 or evidence.get('interval') != 60):
        return 'stale'
    return 'enabled'


def startup_settings(text):
    caps = re.findall(r'kanban dispatcher: max_in_progress=(\d+)\b', text)
    intervals = re.findall(r'kanban dispatcher: embedded in gateway \(interval=([\d.]+)s\)', text)
    if caps and all(v == '1' for v in caps) and intervals and all(float(v) == 60 for v in intervals):
        return {'max_in_progress': 1, 'interval': 60}
    return None


def owns_dispatch_lock(root, pid, proc=Path('/proc')):
    """Match the kernel's flock record to this process and exact native inode.

    An open FD or a locked file alone cannot establish which gateway owns it.
    fdinfo is scoped to the observed process; no global /proc scan or writes.
    """
    try:
        path = root/'kanban/.dispatcher.lock'
        if path.is_symlink() or (root/'kanban').is_symlink(): return False
        target = path.stat()
        for fd in (proc/str(pid)/'fd').iterdir():
            info = fd.stat()
            if (info.st_dev, info.st_ino) != (target.st_dev, target.st_ino): continue
            data = (proc/str(pid)/'fdinfo'/fd.name).read_text()
            if re.search(r'lock:\s+\d+: FLOCK\s+ADVISORY\s+WRITE\s+'+str(pid)+r'\s', data):
                return True
    except (OSError, ValueError):
        pass
    return False


def dispatch_observation(root, config, observed, generation_state, marker):
    policy = config.get('kanban') or {}
    pid = (observed.get('identity') or {}).get('pid')
    owns = bool(pid and owns_dispatch_lock(root, pid))
    configured = policy.get('dispatch_in_gateway')
    single = True
    for name in TEAM[1:]:
        try:
            single &= mapping(root/'profiles'/name/'config.yaml').get('kanban', {}).get('dispatch_in_gateway') is False
        except Exception:
            single = False
    live = dispatch_live(configured, generation_state, owns, marker, (root/'ESTOP').exists())
    if configured is True and (not operational_policy(config) or not single): live = 'stale'
    return {'configured': 'enabled' if configured is True else 'disabled' if configured is False else 'unknown',
            'live': live, 'owner': 'default' if owns and single else 'unknown',
            'max_in_progress': policy.get('max_in_progress'), 'auto_decompose': policy.get('auto_decompose'),
            'review_dispatch': policy.get('review_dispatch'), 'allowlist': policy.get('dispatch_profiles'),
            'policy': operational_policy(config) and single,
            'canary': generation_state=='current' and (marker or {}).get('canary',{}).get('profile')=='researcher'
                and (marker or {}).get('canary',{}).get('gateway_spawn') is True
                and (marker or {}).get('canary',{}).get('done') is True}


def canary_complete(record, title, field='title'):
    if record.get('task', {}).get('status') != 'done': return False
    runs = record.get('runs') or []
    if not runs: return False
    run = runs[-1]
    metadata = run.get('metadata') or {}
    if isinstance(metadata, str):
        import json
        try: metadata = json.loads(metadata)
        except ValueError: return False
    return (run.get('profile') == 'researcher' and run.get('status') == 'done'
            and run.get('outcome') == 'completed' and run.get('ended_at') is not None
            and isinstance(metadata, dict) and isinstance(metadata.get(field), str) and metadata[field] == title
            and metadata.get('changed_files') == [] and bool(run.get('summary')))


def canary_worker_pid(record):
    # Pinned `kanban show --json` excludes worker_pid from task, but exposes it
    # on each native run. Never infer an actor from the mutable assignee alone.
    runs=record.get('runs') or []
    if record.get('task',{}).get('status') != 'running' or not runs: return None
    run=runs[-1]
    pid=run.get('worker_pid')
    if run.get('profile')=='researcher' and run.get('status')=='running' and type(pid) is int and pid>0:
        return pid
    return None

#!/usr/bin/env python3
"""Opt-in real-model MANUAL dispatch gate. Not part of RepoKit's runtime.

Requires a privately configured, disposable native six-profile deployment.
Never copies credentials, edits a board DB, invents worker results or qualifies
later gateway/OpenViking gates. Raw model output stays in private logs.
"""
import argparse
from collections import Counter
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import uuid


def require(condition, message):
    if not condition:
        raise ValueError(message)


def validate_evidence(documents, marker, observations):
    require(len(documents) == 3, 'expected exactly three cards with same-card review')
    research, plan, execution = documents
    for doc in documents:
        require(doc['task'].get('created_by') == 'default', 'coordinator did not create card')
        require(doc['task']['status'] == 'done', 'card has not reached done')
        require(doc['runs'], 'missing run evidence')
        for run in doc['runs']:
            observed = observations.get((doc['task']['id'], run['id']), {})
            require(isinstance(observed.get('worker_pid'), int) and observed['worker_pid'] > 0
                    and observed.get('profile') == run['profile'],
                    'run lacks observed dispatcher-spawned worker PID/profile')
    require(plan['parents'] == [research['task']['id']], 'planner dependency differs')
    require(execution['parents'] == [plan['task']['id']], 'executor dependency differs')
    for doc, profile in [(research, 'researcher'), (plan, 'planner')]:
        runs = doc['runs']
        require(len(runs) == 1 and runs[0]['profile'] == profile and
                runs[0]['outcome'] == 'completed', 'parent worker did not complete cleanly')
        metadata = runs[0].get('metadata')
        if isinstance(metadata, str):
            metadata = json.loads(metadata)
        require(isinstance(metadata, dict) and marker in json.dumps(metadata)
                and 'fixtures/runtime' in json.dumps(metadata),
                'structured parent handoff lacks convention marker')
    runs = sorted(execution['runs'], key=lambda r: r['id'])
    require([(r['profile'], r['outcome']) for r in runs] == [
        ('executor', 'review_requested'), ('reviewer', 'changes_requested'),
        ('executor', 'review_requested'), ('reviewer', 'completed')],
        'expected executor/reviewer change cycle with distinct approval actor')


def coordinator_prompt(token):
    return f'''Create the disposable Kanban acceptance graph for {token}.
Use your native Kanban orchestration tools. Do not implement any artifact.
Create exactly three cards titled "{token} researcher", "{token} planner",
and "{token} executor". Assign those respective profiles. Link researcher
as planner's sole parent and planner as executor's sole parent. Every card
must use workspace_kind=dir and workspace_path=/workspace. Create them blocked,
then unblock them so only dependency-ready work can dispatch. No auto-decomposition.
No commits, pushes, network research or work outside this disposable repository.
Copy these precise contracts into the corresponding card bodies:

researcher: Read /workspace/convention.txt. Complete with structured metadata
containing the exact convention marker and fixture directory. No artifact edits.
planner: Read the researcher parent completion metadata. Complete with structured
metadata that repeats the exact marker and defines the execution acceptance:
produce /workspace/result.md with the marker and fixture directory from research.
executor: Read the planner parent completion metadata. Produce result.md as
specified. On the first pass omit a 'Reviewed: yes' footer, then request same-card
review by reviewer with metadata describing changed artifacts and verification.
On revision read the reviewer's requested change and add the footer. Resubmit
this SAME card to reviewer. Never declare independent acceptance yourself.
Reviewer contract included in executor's card: check the actual result.md and
parent handoffs. First review MUST request changes on the same card to add
'Reviewed: yes'. After executor revises, verify marker, fixture directory and
footer, then complete through native review lifecycle. Never edit artifacts.

The three cards are the entire graph: do not create a separate reviewer card.
When the graph is ready, report its IDs and exit. A human acceptance runner will
call native manual dispatch. Do not start gateway dispatch or spawn workers here.
'''


def run(args):
    root = args.repository.resolve(strict=True)
    require((root / '.repokit-disposable-acceptance').read_text().strip() ==
            'disposable-kanban', 'repository is not explicitly marked disposable')
    # RepoKit is used only for preflight; this gate makes no removal-first claim.
    # Its plan validates exact launcher bytes, symlinks, native state and context.
    report = json.loads(subprocess.check_output([str(args.repokit.resolve(strict=True)), 'plan'],
                                               cwd=root, timeout=30))
    target = report['target']
    require(not report['collisions'] and report['existing_state'] and
            target['Root'] == str(root), 'deployment/launcher routing not qualified')
    launcher = Path(target['Launcher'])
    docker = ['docker', '--context', report['docker_context']]
    # Select only nonsecret metadata; never inspect Docker Config.Env.
    metadata = json.loads(subprocess.check_output(docker + ['container', 'inspect', '--format',
        '{"running":{{json .State.Running}},"image":{{json .Config.Image}},"mounts":{{json .Mounts}}}',
        target['Container']], timeout=30))
    mounts = {m['Destination']: m['Source'] for m in metadata['mounts']}
    require(metadata['running'] and metadata['image'] ==
            report['candidate_images_not_release_qualified']['hermes'] and
            mounts.get('/workspace') == str(root) and mounts.get('/opt/data') == str(root / '.hermes'),
            'container does not match disposable repository')
    require(not any((root / name).exists() for name in ('convention.txt', 'result.md')),
            'acceptance artifacts already exist; use a fresh disposable repository')
    evidence = root / ('acceptance-' + uuid.uuid4().hex)
    evidence.mkdir(mode=0o700)
    token = 'KANBAN_TEST_' + uuid.uuid4().hex
    convention = 'CONVENTION_' + uuid.uuid4().hex
    count = 0

    def native(*argv, input_text=None, timeout=60):
        nonlocal count
        count += 1
        logfile = evidence / f'{count:04d}.log'
        with logfile.open('xb') as output:
            os.chmod(logfile, 0o600)
            result = subprocess.run([str(launcher), '-p', 'default', *argv],
                                    input=input_text.encode() if input_text else None,
                                    stdin=subprocess.DEVNULL if input_text is None else None,
                                    stdout=output, stderr=subprocess.STDOUT, timeout=timeout,
                                    cwd=root)
        require(result.returncode == 0, f'native command failed; private log: {logfile}')
        return logfile.read_text()

    # These are intentional native lifecycle operations, not read-only verify.
    require(json.loads(native('kanban', 'list', '--json')) == [],
            'disposable board must be empty before live acceptance')
    native('config', 'set', 'kanban.dispatch_in_gateway', 'false')
    native('config', 'set', 'kanban.auto_decompose', 'false')
    native('config', 'set', 'kanban.max_in_progress', '1')
    # Startup-sensitive config must apply BEFORE any ready card exists.
    subprocess.run(docker + ['restart', target['Container']], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=60)
    ready_deadline = time.monotonic() + 60
    while True:
        ready = subprocess.run(docker + ['exec', '--user', 'hermes', target['Container'],
                               'test', '-w', '/opt/data/kanban.db.init.lock'],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
        if ready.returncode == 0: break
        require(time.monotonic() < ready_deadline, 'restarted Hermes identity not ready')
        time.sleep(1)
    for profile in report['Profiles']:
        native('profile', 'show', profile)
    (root / 'convention.txt').write_text(f'{convention}: temporary fixtures use fixtures/runtime.\n')
    native('chat', '--oneshot', '--query-file', '-', '--max-turns', '30',
           input_text=coordinator_prompt(token), timeout=args.timeout)
    cards = json.loads(native('kanban', 'list', '--json'))
    require(len(cards) == 3, 'coordinator must create exactly three cards')
    by_title = {card['title']: card for card in cards}
    ids = [by_title[f'{token} {role}']['id'] for role in ('researcher','planner','executor')]
    deadline = time.monotonic() + args.timeout
    observations = {}
    spawns = []
    while time.monotonic() < deadline:
        dispatch = json.loads(native('kanban', 'dispatch', '--max', '1', '--json'))
        spawns.extend((spawn['task_id'], spawn['assignee']) for spawn in dispatch['spawned'])
        documents = [json.loads(native('kanban', 'show', card, '--json')) for card in ids]
        for doc in documents:
            for run in doc['runs']:
                if run.get('worker_pid'):
                    observations[(doc['task']['id'], run['id'])] = dict(run)
        require(sum(r.get('status') == 'running' for d in documents for r in d['runs']) <= 1,
                'observed concurrent worker runs despite cap one')
        require(not any(d['task']['status'] in ('blocked', 'triage') for d in documents),
                'worker blocked; inspect private native evidence before retrying')
        if all(d['task']['status'] == 'done' for d in documents):
            validate_evidence(documents, convention, observations)
            require(Counter(spawns) == Counter((doc['task']['id'], r['profile'])
                    for doc in documents for r in doc['runs']), 'run was not spawned by manual dispatch')
            result = (root / 'result.md').read_text()
            require(all(s in result for s in (convention, 'fixtures/runtime', 'Reviewed: yes')),
                    'result artifact does not satisfy the contract')
            (evidence / 'manual-acceptance.json').write_text(json.dumps({'cards': documents, 'spawns': spawns,
                'observed_runs': [{'task_id': key[0], 'run': value} for key, value in observations.items()]}, indent=2))
            print(f'Manual model-driven Kanban gate passed. Evidence: {evidence}')
            print('Gateway dispatch, recreation and integrated gates remain unqualified.')
            return
        time.sleep(3)
    raise ValueError(f'live dispatch deadline exceeded; native worker state preserved in {root}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repokit', required=True, type=Path, help='RepoKit binary for read-only preflight')
    parser.add_argument('--repository', required=True, type=Path)
    parser.add_argument('--timeout', type=int, default=1200)
    args = parser.parse_args()
    require(args.timeout > 0, 'timeout must be positive')
    os.umask(0o077)
    run(args)


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError, subprocess.TimeoutExpired, subprocess.CalledProcessError) as exc:
        print(f'Live acceptance incomplete: {exc}', file=sys.stderr)
        sys.exit(1)

"""Credential-free native lifecycle fixture, run in distinct profile processes."""
import json
import os
from pathlib import Path
import sys
from hermes_cli import kanban_db as k
from hermes_cli.kanban_db_connect import connect

root = Path('/opt/data')
refs = root / 'acceptance-card-ids.json'
action = sys.argv[1]
profile = os.environ.get('HERMES_PROFILE_NAME', 'default')
c = connect()
assert str(k.kanban_db_path()) == '/opt/data/kanban.db'
if action == 'create':
    assert profile == 'default'
    parent = k.create_task(c, title='Investigate fixture', assignee='researcher', created_by='default')
    plan = k.create_task(c, title='Plan fixture', assignee='planner', created_by='default', parents=[parent])
    card = k.create_task(c, title='Produce fixture', assignee='executor', created_by='default', parents=[plan])
    refs.write_text(json.dumps({'parent':parent,'plan':plan,'card':card}))
else:
    ids = json.loads(refs.read_text())
    parent, card = ids['parent'], ids['card']
    if action == 'research':
        assert profile == 'researcher'
        task=k.claim_task(c,parent,claimer=profile)
        assert task.assignee == profile
        assert k.complete_task(c,parent,summary='Verified parent evidence',metadata={'confirmed':['durable-repokit-fixture']},expected_run_id=task.current_run_id)
    elif action == 'plan':
        assert profile == 'planner'
        plan = ids['plan']
        assert 'durable-repokit-fixture' in k.build_worker_context(c,plan)
        task=k.claim_task(c,plan,claimer=profile)
        assert task.assignee == profile
        assert k.complete_task(c,plan,summary='Bounded fixture plan',metadata={'acceptance':['durable-repokit-fixture', 'planner-handoff-marker']},expected_run_id=task.current_run_id)
    elif action in ('execute','revise','fix'):
        assert profile == 'executor'
        context=k.build_worker_context(c,card)
        assert 'planner-handoff-marker' in context
        if action == 'revise':
            assert 'Add the missing regression test' in context
            task=k.claim_task(c,card,claimer=profile)
        elif action == 'fix':
            # Tester relayed reviewer-requested changes as a review assignment.
            task=k.claim_review_task(c,card,claimer=profile)
        else:
            task=k.claim_task(c,card,claimer=profile)
        assert task.assignee == profile
        assert k.request_review(c,card,summary='Fixture result ready',metadata={'verification':['native fixture']},reviewer='tester',expected_run_id=task.current_run_id)
        assert k.get_task(c,card).status == 'review' and k.get_task(c,card).assignee == 'tester'
    elif action == 'test-fail':
        assert profile == 'tester'
        task=k.claim_review_task(c,card,claimer=profile)
        assert task.assignee == profile
        assert k.request_changes(c,card,reason='Add the missing regression test',expected_run_id=task.current_run_id)==(True,'executor')
        assert k.get_task(c,card).assignee=='executor'
    elif action == 'test-pass':
        assert profile == 'tester'
        task=k.claim_review_task(c,card,claimer=profile)
        assert task.assignee == profile
        assert k.request_review(c,card,summary='Behavior verified',metadata={'verification':['fixture checks passed']},reviewer='reviewer',expected_run_id=task.current_run_id)
        assert k.get_task(c,card).assignee == 'reviewer'
    elif action == 'changes':
        assert profile == 'reviewer'
        task=k.claim_review_task(c,card,claimer=profile)
        assert task.assignee == profile
        # Hermes returns changes to the last profile that requested review.
        assert k.request_changes(c,card,reason='Include the missing fixture evidence',expected_run_id=task.current_run_id)==(True,'tester')
        assert k.get_task(c,card).assignee=='tester'
    elif action == 'relay':
        assert profile == 'tester'
        assert 'Include the missing fixture evidence' in k.build_worker_context(c,card)
        task=k.claim_task(c,card,claimer=profile)
        assert task.assignee == profile
        assert k.request_review(c,card,summary='Relaying reviewer changes: Include the missing fixture evidence',reviewer='executor',expected_run_id=task.current_run_id)
        assert k.get_task(c,card).assignee == 'executor'
    elif action == 'approve':
        assert profile == 'reviewer'
        task=k.claim_review_task(c,card,claimer=profile)
        assert task.assignee == profile
        assert k.complete_task(c,card,summary='Independently verified fixture',metadata={'verification':['reviewed native evidence']},expected_run_id=task.current_run_id)
        assert k.get_task(c,card).status=='done'
        runs=[(r['profile'],r['outcome']) for r in c.execute('SELECT profile, outcome FROM task_runs WHERE task_id=? ORDER BY id',(card,))]
        assert runs==[('executor','review_requested'),('tester','changes_requested'),('executor','review_requested'),('tester','review_requested'),('reviewer','changes_requested'),('tester','review_requested'),('executor','review_requested'),('tester','review_requested'),('reviewer','completed')], runs
    elif action == 'persist':
        assert k.get_task(c,card).status=='done'
        assert 'durable-repokit-fixture' in k.build_worker_context(c,card)
    else: raise AssertionError(action)
print('native lifecycle: '+action+' passed as '+profile)

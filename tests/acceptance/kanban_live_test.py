"""Acceptance evidence must not mistake a done card for independent review."""
import unittest
import argparse
import json
from pathlib import Path
import tempfile
from unittest.mock import patch
from kanban_live import validate_evidence, run


class EvidenceTests(unittest.TestCase):
    def setUp(self):
        self.docs = []
        for role, parent in [('researcher', []), ('planner', ['researcher']), ('executor', ['planner'])]:
            runs = [{'id': 1, 'profile': role, 'outcome': 'completed', 'worker_pid': 100,
                     'metadata': {'marker': 'OV_TEST_fixture', 'directory': 'fixtures/runtime'}}]
            if role == 'executor':
                runs = [dict(id=i, profile=p, outcome=o, worker_pid=100+i,
                             metadata={'marker': 'OV_TEST_fixture', 'directory': 'fixtures/runtime'}) for i,p,o in [
                    (1,'executor','review_requested'), (2,'reviewer','changes_requested'),
                    (3,'executor','review_requested'), (4,'reviewer','completed')]]
            self.docs.append({'task': {'id':role, 'status':'done', 'created_by':'default'},
                              'parents':parent, 'runs':runs})

    def observations(self):
        return {(doc['task']['id'], run['id']): dict(run)
                for doc in self.docs for run in doc['runs'] if run.get('worker_pid')}

    def check(self, observed=None):
        validate_evidence(self.docs, 'OV_TEST_fixture', self.observations() if observed is None else observed)

    def test_accepts_reaped_pid_with_prior_observation(self):
        observed = self.observations()
        self.docs[0]['runs'][0]['worker_pid'] = None
        self.check(observed)

    def test_accepts_observed_review_cycle(self):
        self.check()

    def test_rejects_self_approval(self):
        self.docs[2]['runs'][-1]['profile'] = 'executor'
        with self.assertRaises(ValueError): self.check()

    def test_rejects_synthetic_runs(self):
        self.docs[2]['runs'][1]['worker_pid'] = None
        with self.assertRaises(ValueError): self.check()

    def test_rejects_missing_parent_metadata(self):
        self.docs[1]['runs'][0]['metadata'] = {}
        with self.assertRaises(ValueError): self.check()

    def test_rejects_unobserved_actor(self):
        observed = self.observations()
        observed[('executor', 4)]['profile'] = 'executor'
        with self.assertRaises(ValueError): self.check(observed)

    def test_rejects_wrong_creator(self):
        self.docs[0]['task']['created_by'] = 'user'
        with self.assertRaises(ValueError): self.check()

    def test_rejects_missing_fixture_directory(self):
        del self.docs[0]['runs'][0]['metadata']['directory']
        with self.assertRaises(ValueError): self.check()

    def test_rejects_wrong_graph(self):
        self.docs[2]['parents'] = ['researcher']
        with self.assertRaises(ValueError): self.check()

    def test_rejects_missing_change_cycle(self):
        self.docs[2]['runs'] = self.docs[2]['runs'][-2:]
        with self.assertRaises(ValueError): self.check()

class PreflightTests(unittest.TestCase):
    def test_copied_launcher_refused_before_native_mutation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / '.repokit-disposable-acceptance').write_text('disposable-kanban')
            binary = root / 'repokit'
            binary.touch()
            report = {'target': {'Root': str(root)}, 'existing_state': True,
                      'collisions': ['existing launcher context cannot be verified']}
            with patch('kanban_live.subprocess.check_output', return_value=json.dumps(report).encode()) as inspect, \
                 patch('kanban_live.subprocess.run') as mutate:
                with self.assertRaisesRegex(ValueError, 'routing not qualified'):
                    run(argparse.Namespace(repository=root, repokit=binary, timeout=1))
                inspect.assert_called_once()
                mutate.assert_not_called()
                self.assertFalse((root / 'convention.txt').exists())

if __name__ == '__main__': unittest.main()

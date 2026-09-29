"""Exercise real provisioning with Go-generated current and historical roles."""
import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'native'))
from team_test import ProvisionTest

payload = json.load(sys.stdin)


class RuntimeMigrationTest(unittest.TestCase):
    def setUp(self):
        self.fixture = ProvisionTest()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.fixture.roles = copy.deepcopy(payload['previous'])
        self.assertEqual(self.fixture.apply(), [])

    def homes(self):
        return [self.fixture.root if role['name'] == 'default'
                else self.fixture.root / 'profiles' / role['name']
                for role in payload['current']]

    def snapshot(self):
        return {str(p): p.read_bytes() for p in self.fixture.root.rglob('*')
                if p.is_file() and p.name != 'SOUL.md'}

    def test_exact_previous_generation_upgrades_without_other_changes(self):
        before = self.snapshot()
        self.fixture.roles = copy.deepcopy(payload['current'])
        self.assertEqual(self.fixture.apply(), [])
        for home, role, old in zip(self.homes(), payload['current'], payload['previous']):
            self.assertNotEqual(role['soul'], old['soul'])
            self.assertEqual((home / 'SOUL.md').read_text(), role['soul'])
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(self.fixture.apply(), [])
        self.assertEqual(self.snapshot(), before)

    def test_owner_modified_previous_generation_is_preserved(self):
        before = self.snapshot()
        originals = {}
        for home in self.homes():
            path = home / 'SOUL.md'
            originals[path] = path.read_bytes() + b'\nOwner instruction\n'
            path.write_bytes(originals[path])
        self.fixture.roles = copy.deepcopy(payload['current'])
        self.assertEqual(self.fixture.apply(), [r['name'] for r in payload['current']])
        for path, content in originals.items():
            self.assertEqual(path.read_bytes(), content)
        self.assertEqual(self.snapshot(), before)


suite = unittest.defaultTestLoader.loadTestsFromTestCase(RuntimeMigrationTest)
sys.exit(not unittest.TextTestRunner(verbosity=2).run(suite).wasSuccessful())

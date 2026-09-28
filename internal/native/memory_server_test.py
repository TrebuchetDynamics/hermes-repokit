import copy
import importlib.util
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('memory_server', Path(__file__).with_name('memory_server.py'))
server = importlib.util.module_from_spec(spec)
spec.loader.exec_module(server)


class ServerGateTest(unittest.TestCase):
    def setUp(self):
        # Nonsecret config-shape input, never written to a service or used as auth.
        self.config = {'storage': {'workspace': '/app/.openviking/data'},
                       'server': {'host': '0.0.0.0', 'port': 1933, 'root_api_key': 'fixture'}}

    def test_native_extraction_default_is_preserved(self):
        before = copy.deepcopy(self.config)
        server.validate(self.config)
        self.assertEqual(self.config, before)

    def test_rejects_unpersisted_workspace_wrong_binding_and_disabled_extraction(self):
        for section, key, value in [('storage', 'workspace', './data'),
                                    ('storage', 'workspace', '/tmp/data'),
                                    ('server', 'host', '127.0.0.1'),
                                    ('server', 'port', 1934),
                                    ('server', 'root_api_key', ''),
                                    ('memory', 'extraction_enabled', False)]:
            with self.subTest(section=section, key=key):
                config = copy.deepcopy(self.config)
                config.setdefault(section, {})[key] = value
                with self.assertRaises(RuntimeError):
                    server.validate(config)

    def test_missing_workspace_is_not_accepted_as_durable(self):
        del self.config['storage']
        with self.assertRaises(RuntimeError):
            server.validate(self.config)

    def test_health_waits_through_pending_and_request_timeout(self):
        results = [subprocess.CompletedProcess([], 1),
                   subprocess.TimeoutExpired('health', 5),
                   subprocess.CompletedProcess([], 0)]
        with patch.object(server.subprocess, 'run', side_effect=results) as run, \
             patch.object(server.time, 'sleep'), patch.object(server.time, 'monotonic', return_value=0):
            server.health()
        self.assertEqual(run.call_count, 3)
        for call in run.call_args_list:
            self.assertEqual(call.args[0], ['openviking-entrypoint', '--healthcheck'])
            self.assertEqual(call.kwargs['timeout'], 5)
            self.assertEqual(call.kwargs['stdout'], subprocess.DEVNULL)

    def test_health_timeout_refuses_activation(self):
        with patch.object(server.time, 'monotonic', side_effect=[0, 121]):
            with self.assertRaises(RuntimeError):
                server.health()


if __name__ == '__main__':
    unittest.main()

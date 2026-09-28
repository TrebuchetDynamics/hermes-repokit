"""Offline contract tests: never import or contact a running Hermes gateway."""
import importlib.util
from datetime import datetime, timedelta, timezone
import json
from pathlib import Path
import sqlite3
import tempfile
import types
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("maintenance_bridge", Path(__file__).with_name("__init__.py"))
bridge = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(bridge)


class FakeNative:
    def __init__(self, home):
        self.home = home
        self.pid = 40
        self.start = 10
        self.healthy = True
        self.calls = 0
        self.reply = {"pausing": True, "already_stopping": False, "pid": 40}
        self.generation = "a" * 64
        self.current_generation = self.generation
        self.heartbeat_ok = True
        self.current_at_boot = False

    def identify(self):
        return {"pid": self.pid, "start_time": self.start, "hermes_home": str(self.home)}

    def status(self):
        return {"pid": self.pid, "answering_pid": self.pid, "start_time": self.start,
                "gateway_state": "running" if self.healthy else "starting", "restart_requested": False,
                "active_agents": 1,
                "session_store": {"status": "ok"}, "updated_at": datetime.now(timezone.utc).isoformat(),
                "platforms": {"telegram": {"state": "connected", "writer_pid": self.pid,
                                              "writer_start_time": self.start, "needs_attention": False}}}

    def generation_matches(self, generation, identity, *, require_current=True):
        return generation == self.generation and (not require_current or generation == self.current_generation)

    def heartbeat(self, identity):
        return self.heartbeat_ok

    def already_current(self, generation, identity):
        return self.current_at_boot and self.generation_matches(generation, identity)

    def request(self):
        self.calls += 1
        return self.reply


class BridgeTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name)
        self.native = FakeNative(self.home)
        self.service = bridge.Broker(self.home, self.native, now=lambda: 1000)

    def request(self, generation="a" * 64, busy=False):
        return self.service.request(generation, admitted=True, kanban_busy=busy)

    def test_acceptance_is_not_completion_and_duplicate_is_not_resent(self):
        self.assertEqual(self.request()["state"], "requested")
        self.assertEqual(self.request()["state"], "pending")
        self.assertEqual(self.native.calls, 1)

    def test_timeout_persists_uncertain_and_blocks_changed_generation(self):
        self.native.reply = None
        self.assertEqual(self.request()["state"], "uncertain")
        self.assertEqual(self.request("b" * 64)["state"], "pending")
        self.assertEqual(self.native.calls, 1)

    def test_success_needs_new_identity_and_healthy_native_state(self):
        self.request()
        self.assertEqual(self.service.status()["state"], "pending")
        self.native.pid, self.native.start = 41, 11
        self.native.healthy = False
        self.assertEqual(self.service.status()["state"], "successor_unverified")
        self.native.healthy = True
        before = self.service.marker.read_bytes()
        self.assertEqual(self.service.status()["state"], "verified")
        self.assertEqual(self.service.marker.read_bytes(), before, "status is read-only")
        self.assertEqual(self.request()["state"], "already_applied")
        self.assertEqual(self.native.calls, 1)

    def test_kanban_busy_or_wrong_caller_does_not_request_or_write_marker(self):
        self.assertEqual(self.request(busy=True)["state"], "deferred")
        self.assertEqual(self.service.request("a" * 64, admitted=False, kanban_busy=False)["state"], "refused")
        self.assertFalse(self.service.marker.exists())
        self.assertEqual(self.native.calls, 0)

    def test_corrupt_marker_fails_closed(self):
        self.service.marker.parent.mkdir(parents=True)
        self.service.marker.write_text("not-json")
        self.assertEqual(self.request()["state"], "refused")
        self.assertEqual(self.native.calls, 0)

    def test_endpoint_exception_is_uncertain_and_never_retried(self):
        def fail():
            self.native.calls += 1
            raise OSError("private detail")
        self.native.request = fail
        result = self.request()
        self.assertEqual(result["state"], "uncertain")
        self.assertNotIn("private detail", json.dumps(result))
        self.assertEqual(self.request()["state"], "pending")
        self.assertEqual(self.native.calls, 1)

    def test_identity_mismatch_in_ack_is_uncertain(self):
        self.native.reply["pid"] = 999
        self.assertEqual(self.request()["state"], "uncertain")

    def test_kanban_snapshot_uses_readonly_db_and_defers_on_ready_and_running(self):
        path = self.home / "kanban.db"
        with sqlite3.connect(path) as conn:
            conn.execute("CREATE TABLE tasks(status TEXT, worker_pid INTEGER)")
            conn.execute("INSERT INTO tasks VALUES ('running', 55)")
        self.assertTrue(bridge.kanban_busy(self.home))
        with sqlite3.connect(path) as conn:
            conn.execute("UPDATE tasks SET status='done', worker_pid=NULL")
        self.assertFalse(bridge.kanban_busy(self.home))

    def test_generation_ignores_secrets_but_tracks_runtime_values(self):
        one = {"model": {"default": "one", "api_key": "secret-a"}}
        two = {"model": {"default": "one", "api_key": "secret-b"}}
        self.assertEqual(bridge.generation_digest(one, "soul"), bridge.generation_digest(two, "soul"))
        two["model"]["default"] = "two"
        self.assertNotEqual(bridge.generation_digest(one, "soul"), bridge.generation_digest(two, "soul"))

    def test_successor_with_degraded_platform_is_not_verified(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        status = self.native.status()
        status["platforms"]["telegram"]["state"] = "retrying"
        self.native.status = lambda: status
        self.assertEqual(self.service.status()["state"], "successor_unverified")

    def test_new_generation_needs_cooldown_after_healthy_successor(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        self.assertEqual(self.request("b" * 64)["state"], "deferred")
        self.service.now = lambda: 1400
        self.native.current_generation = "b" * 64
        self.native.reply["pid"] = 41
        self.assertEqual(self.request("b" * 64)["state"], "requested")
        self.assertEqual(self.native.calls, 2)

    def test_generation_drift_or_missing_heartbeat_does_not_verify(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        self.native.current_generation = "b" * 64
        self.assertEqual(self.service.status()["state"], "successor_unverified")
        self.native.current_generation = "a" * 64
        self.native.heartbeat_ok = False
        self.assertEqual(self.service.status()["state"], "successor_unverified")

    def test_missing_board_or_review_card_defers(self):
        self.assertTrue(bridge.kanban_busy(self.home))
        with sqlite3.connect(self.home / "kanban.db") as conn:
            conn.execute("CREATE TABLE tasks(status TEXT, worker_pid INTEGER)")
            conn.execute("INSERT INTO tasks VALUES ('review', NULL)")
        self.assertTrue(bridge.kanban_busy(self.home))

    def test_unapproved_nested_config_cannot_affect_generation(self):
        self.assertEqual(bridge.generation_digest({"plugins": {"nested": {"innocent_key": "secret"}}}, "soul"),
                         bridge.generation_digest({}, "soul"))

    def test_symlink_receipt_refused_without_touching_target(self):
        self.service.marker.parent.mkdir(parents=True)
        target = self.home / "other"
        target.write_text("sensitive")
        self.service.marker.symlink_to(target)
        self.assertEqual(self.request()["state"], "refused")
        self.assertEqual(target.read_text(), "sensitive")

    def test_native_adapter_uses_only_existing_control_clients(self):
        calls = []
        module = types.ModuleType("gateway.control_socket")
        module.identify_gateway = lambda home, **kw: calls.append(("identify", home, kw))
        module.query_gateway_control = lambda home, verb, **kw: calls.append((verb, home, kw))
        module.pause_gateway_for_update = lambda home, **kw: calls.append(("pause-for-update", home, kw))
        with patch.dict("sys.modules", {"gateway.control_socket": module}):
            native = bridge.NativeControl(self.home)
            native.identify()
            native.status()
            native.request()
        self.assertEqual([c[0] for c in calls], ["identify", "status", "pause-for-update"])
        self.assertEqual(calls[-1][2], {"timeout": 6.0})

    def test_plugin_registers_two_parameterless_native_model_tools(self):
        tools = []
        ctx = types.SimpleNamespace(register_tool=lambda **kwargs: tools.append(kwargs))
        bridge.register(ctx)
        self.assertEqual([t["name"] for t in tools], ["gateway_restart_after_turn", "gateway_restart_status"])
        for tool in tools:
            self.assertEqual(tool["toolset"], "repokit_maintenance")
            self.assertEqual(tool["schema"]["parameters"]["properties"], {})
            self.assertFalse(tool["schema"]["parameters"]["additionalProperties"])

    def test_unattended_and_worker_contexts_are_rejected_at_call_time(self):
        session = types.ModuleType("gateway.session_context")
        process = types.ModuleType("tools.process_registry")
        platforms = types.ModuleType("hermes_cli.platforms")
        platforms.PLATFORMS = dict.fromkeys(["cli", "telegram", "mattermost", "email", "whatsapp_cloud", "cron", "api_server", "webhook"])
        variables = {"HERMES_SESSION_PLATFORM": "telegram", "HERMES_CRON_SESSION": ""}
        session.get_session_env = lambda name, default="": variables.get(name, default)
        process._is_supervised_gateway_process = lambda: True
        ctx = types.SimpleNamespace(profile_name="default")
        with patch.dict("sys.modules", {"gateway.session_context": session, "tools.process_registry": process, "hermes_cli.platforms": platforms}), patch.object(bridge, "_runtime", return_value=(self.home, self.service)):
            self.assertTrue(bridge._caller_admitted(ctx))
            for platform in ("mattermost", "email", "whatsapp_cloud"):
                variables["HERMES_SESSION_PLATFORM"] = platform
                self.assertTrue(bridge._caller_admitted(ctx))
            for platform in ("api_server", "webhook", "acp", "cron", "kanban", ""):
                variables["HERMES_SESSION_PLATFORM"] = platform
                self.assertFalse(bridge._caller_admitted(ctx))
            variables["HERMES_SESSION_PLATFORM"] = "telegram"
            variables["HERMES_CRON_SESSION"] = "1"
            self.assertFalse(bridge._caller_admitted(ctx))
            variables["HERMES_CRON_SESSION"] = ""
            process._is_supervised_gateway_process = lambda: False
            self.assertFalse(bridge._caller_admitted(ctx))

    def test_manual_dispatch_gate_requires_explicit_qualified_settings(self):
        good = {"kanban": {"dispatch_in_gateway": False, "auto_decompose": False, "max_in_progress": 1}}
        self.assertTrue(bridge.manual_dispatch_config(good))
        self.assertFalse(bridge.manual_dispatch_config({}))
        for key, bad in (("dispatch_in_gateway", True), ("dispatch_in_gateway", "false"),
                         ("auto_decompose", True), ("max_in_progress", 2), ("max_in_progress", True)):
            changed = {"kanban": {**good["kanban"], key: bad}}
            self.assertFalse(bridge.manual_dispatch_config(changed))
        for key in good["kanban"]:
            changed = {"kanban": {k: v for k, v in good["kanban"].items() if k != key}}
            self.assertFalse(bridge.manual_dispatch_config(changed))

    def test_request_context_checks_manual_policy_before_board_or_native_request(self):
        module = types.ModuleType("hermes_cli.kanban_db")
        module.kanban_home = lambda: self.home
        good = {"kanban": {"dispatch_in_gateway": False, "auto_decompose": False, "max_in_progress": 1}}
        ctx = types.SimpleNamespace(profile_name="default")
        with patch.dict("sys.modules", {"hermes_cli.kanban_db": module}), patch.object(bridge, "_caller_admitted", return_value=True), patch.object(bridge, "_runtime", return_value=(self.home, self.service)), patch.object(bridge, "kanban_busy", return_value=False) as busy:
            for config, loaded in (({}, {"manual_dispatch": True}),
                                   (good, {"manual_dispatch": False}), (good, None)):
                with patch.object(bridge, "_current_inputs", return_value=(config, "a" * 64)):
                    self.assertEqual(bridge._request_for_context(ctx, loaded)["state"], "deferred")
            busy.assert_not_called()
            self.assertEqual(self.native.calls, 0)
            with patch.object(bridge, "_current_inputs", return_value=(good, "a" * 64)):
                self.assertEqual(bridge._request_for_context(ctx, {"manual_dispatch": True})["state"], "requested")
            self.assertEqual(self.native.calls, 1)

    def test_unchanged_loaded_generation_does_not_restart(self):
        self.native.current_at_boot = True
        self.assertEqual(self.request()["state"], "already_current")
        self.assertEqual(self.native.calls, 0)
        self.assertFalse(self.service.marker.exists())

    def test_matching_loaded_generation_cannot_hide_pending_request(self):
        self.request()
        self.native.current_at_boot = True
        self.assertEqual(self.request()["state"], "pending")
        self.assertEqual(self.native.calls, 1)

    def test_generation_attestation_requires_loaded_process_and_current_inputs(self):
        native = bridge.NativeControl(self.home, {"pid": 41, "start_time": 11, "generation": "a" * 64})
        identity = {"pid": 41, "start_time": 11}
        with patch.object(bridge, "_current_generation", return_value="a" * 64):
            self.assertTrue(native.generation_matches("a" * 64, identity))
            self.assertFalse(native.generation_matches("a" * 64, {"pid": 42, "start_time": 12}))
        with patch.object(bridge, "_current_generation", return_value="b" * 64):
            self.assertFalse(native.generation_matches("a" * 64, identity))

    def test_stale_adapter_writer_identity_does_not_verify_successor(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        status = self.native.status()
        status["platforms"]["telegram"]["writer_pid"] = 40
        self.native.status = lambda: status
        self.assertEqual(self.service.status()["state"], "successor_unverified")

    def test_historical_stopped_platform_does_not_block_current_gateway_request(self):
        status = self.native.status()
        status["platforms"]["discord"] = {"state": "stopped", "writer_pid": 30, "writer_start_time": 5}
        self.native.status = lambda: status
        self.assertEqual(self.request()["state"], "requested")
        self.assertEqual(self.native.calls, 1)
        receipt = json.loads(self.service.marker.read_text())
        self.assertEqual(receipt["platforms"], ["telegram"])

    def test_unavailable_session_store_never_verifies_successor(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        status = self.native.status()
        status["session_store"] = {"status": "unavailable"}
        self.native.status = lambda: status
        self.assertEqual(self.service.status()["state"], "successor_unverified")

    def test_request_requires_exactly_one_native_active_work_unit(self):
        baseline = self.native.status()
        for count in (0, 2, None, True, "1"):
            self.native.status = lambda count=count: {**baseline, "active_agents": count}
            self.assertEqual(self.request()["state"], "deferred")
            self.assertEqual(self.native.calls, 0)
            self.assertFalse(self.service.marker.exists())
        self.native.status = lambda: {**baseline, "active_agents": 1}
        self.assertEqual(self.request()["state"], "requested")
        self.assertEqual(self.native.calls, 1)

    def test_successor_verification_does_not_require_requesting_active_turn(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        status = self.native.status()
        status["active_agents"] = 0
        self.native.status = lambda: status
        self.assertEqual(self.service.status()["state"], "verified")

    def test_adapter_attention_and_stale_runtime_status_never_verify(self):
        self.request()
        self.native.pid, self.native.start = 41, 11
        baseline = self.native.status()
        variants = []
        attention = json.loads(json.dumps(baseline))
        attention["platforms"]["telegram"]["needs_attention"] = True
        variants.append(attention)
        for timestamp in (None, "invalid", (datetime.now(timezone.utc) - timedelta(seconds=121)).isoformat()):
            variants.append({**baseline, "updated_at": timestamp})
        variants.append({**baseline, "session_store": {"status": "unknown"}})
        variants.append({**baseline, "session_store": None})
        for status in variants:
            self.native.status = lambda status=status: status
            self.assertEqual(self.service.status()["state"], "successor_unverified")

    def test_missing_named_board_and_symlink_config_fail_closed(self):
        with sqlite3.connect(self.home / "kanban.db") as conn:
            conn.execute("CREATE TABLE tasks(status TEXT, worker_pid INTEGER)")
        (self.home / "kanban" / "boards" / "pending").mkdir(parents=True)
        self.assertTrue(bridge.kanban_busy(self.home))
        target = self.home / "outside"
        target.write_text("private")
        config = self.home / "config.yaml"
        config.symlink_to(target)
        with self.assertRaises(OSError):
            bridge._read_bounded(config)


if __name__ == "__main__":
    unittest.main()

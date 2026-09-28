"""Narrow model tool adapter to Hermes' existing graceful restart control verb.

This plugin owns only a bounded request receipt. Hermes owns draining, exit,
supervision, recovery and health. No subprocess, signals, timers or retry loop.
"""
from __future__ import annotations

import fcntl
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import sqlite3
import stat
import tempfile
import time


_PUBLIC_FIELDS = (
    ("model", "default"), ("model", "provider"),
    ("agent", "max_turns"), ("agent", "restart_after_turn_timeout"),
    ("agent", "restart_drain_timeout"), ("agent", "cron_drain_timeout"),
    ("terminal", "backend"), ("memory", "provider"),
    ("gateway", "multiplex_profiles"), ("kanban", "dispatch_in_gateway"),
    ("kanban", "auto_decompose"), ("kanban", "max_in_progress"),
)
_HOME = Path("/opt/data")
_PROGRAMMATIC_PLATFORMS = frozenset({"cli", "cron", "api_server", "webhook", "msgraph_webhook", "acp"})
_GENERATION = re.compile(r"[0-9a-f]{64}\Z")
_MAX_BYTES = 1024 * 1024
_COOLDOWN_SECONDS = 300


def generation_digest(config: dict, soul: str) -> str:
    """Explicit benign scalar settings and SOUL; all other config fields excluded.

    Never reads .env, auth stores or session content. Only this opaque digest is
    persisted. Credential rotation is intentionally outside this restart trigger.
    """
    selected = {}
    for section, field in _PUBLIC_FIELDS:
        values = config.get(section)
        value = values.get(field) if isinstance(values, dict) else None
        if isinstance(value, (str, int, float, bool)):
            selected[f"{section}.{field}"] = value
    payload = json.dumps({"config": selected, "soul": soul}, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha256(payload.encode()).hexdigest()


def _read_bounded(path: Path, *, missing: str | None = None) -> str:
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
        with os.fdopen(fd, "rb") as stream:
            if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
                raise ValueError("regular file required")
            data = stream.read(_MAX_BYTES + 1)
    except FileNotFoundError:
        if missing is not None:
            return missing
        raise
    if len(data) > _MAX_BYTES:
        raise ValueError("input too large")
    return data.decode("utf-8")


def kanban_busy(root: Path) -> bool:
    """Read-only, fail-closed snapshot of native task rows on every local board.

    Native restart accounting omits Kanban subprocesses. Defer for ready work,
    running cards and workers still attached to terminal cards; do not manipulate
    leases, worker PIDs or dispatch. A later concurrent claim is not atomic with
    this snapshot, so this is admission evidence, not a Kanban drain guarantee.
    """
    try:
        boards = root / "kanban" / "boards"
        entries = list(boards.iterdir()) if boards.exists() else []
        if len(entries) > 128 or not (root / "kanban.db").is_file():
            return True
        paths = [root / "kanban.db"]
        for directory in entries:
            if directory.name == "_archived":
                continue
            if directory.is_symlink() or not directory.is_dir():
                return True
            paths.append(directory / "kanban.db")
        if len(paths) > 128:
            return True
        for path in paths:
            if path.is_symlink() or not path.is_file():
                return True
            with sqlite3.connect(path.resolve().as_uri() + "?mode=ro", uri=True, timeout=0.2) as conn:
                conn.execute("PRAGMA query_only=ON")
                if conn.execute("SELECT 1 FROM tasks WHERE status IN ('ready','running','review') OR worker_pid IS NOT NULL LIMIT 1").fetchone():
                    return True
        return False
    except (OSError, sqlite3.Error):
        return True


def manual_dispatch_config(config: dict) -> bool:
    """Require explicit pinned native manual-mode settings; absent is not false."""
    kanban = config.get("kanban")
    return (isinstance(kanban, dict) and kanban.get("dispatch_in_gateway") is False
            and kanban.get("auto_decompose") is False
            and type(kanban.get("max_in_progress")) is int
            and kanban["max_in_progress"] == 1)


def _identity(value, home):
    return (isinstance(value, dict) and isinstance(value.get("pid"), int)
            and value["pid"] > 1 and isinstance(value.get("start_time"), (int, float))
            and value["start_time"] > 0 and value.get("hermes_home") == str(home))


def _healthy(status, identity, platforms):
    if not isinstance(status, dict):
        return False
    try:
        updated = datetime.fromisoformat(status.get("updated_at", ""))
        if updated.tzinfo is None or not 0 <= (datetime.now(timezone.utc) - updated).total_seconds() <= 120:
            return False
    except (TypeError, ValueError):
        return False
    return (status.get("answering_pid") == identity["pid"]
            and status.get("pid") == identity["pid"]
            and status.get("start_time") == identity["start_time"]
            and status.get("gateway_state") == "running"
            and status.get("restart_requested") is False
            and isinstance(status.get("session_store"), dict)
            and status["session_store"].get("status") == "ok"
            and bool(platforms)
            and all(isinstance(status.get("platforms", {}).get(p), dict)
                    and status["platforms"][p].get("state") == "connected"
                    and status["platforms"][p].get("needs_attention") is False
                    and status["platforms"][p].get("writer_pid") == identity["pid"]
                    and status["platforms"][p].get("writer_start_time") == identity["start_time"] for p in platforms))


class Broker:
    """One native request per input generation; pending/uncertain blocks all retry."""
    def __init__(self, home, native, *, now=time.time):
        self.home = Path(home).resolve()
        self.native = native
        self.now = now
        self.marker = self.home / "state" / "repokit-maintenance.json"

    def _read(self):
        if self.marker.parent.is_symlink() or self.marker.is_symlink():
            raise ValueError("symlink state directory")
        if not self.marker.exists():
            return None
        record = json.loads(_read_bounded(self.marker))
        if (not isinstance(record, dict) or record.get("version") != 1
                or not _GENERATION.fullmatch(str(record.get("generation", "")))
                or not _identity(record.get("old"), self.home)
                or not isinstance(record.get("requested_at"), (int, float))
                or not isinstance(record.get("platforms"), list)
                or not all(isinstance(p, str) for p in record["platforms"])
                or not isinstance(record.get("history"), list) or len(record["history"]) > 16
                or not all(_GENERATION.fullmatch(str(g)) for g in record["history"])):
            raise ValueError("invalid receipt")
        return record

    def _verified(self, record, identity, status, *, require_current=True):
        return (_identity(identity, self.home)
                and identity["pid"] != record["old"]["pid"]
                and identity["start_time"] > record["old"]["start_time"]
                and _healthy(status, identity, record["platforms"])
                and self.native.generation_matches(record["generation"], identity, require_current=require_current)
                and self.native.heartbeat(identity))

    def status(self):
        """Read-only verification; no polling, marker writes or restart requests."""
        try:
            record = self._read()
            if record is None:
                return {"state": "not_requested"}
            identity, status = self.native.identify(), self.native.status()
            verified = self._verified(record, identity, status)
            successor = _identity(identity, self.home) and identity["pid"] != record["old"]["pid"]
            return {"state": "verified" if verified else ("successor_unverified" if successor else "pending"), "generation": record["generation"],
                    "previous_pid": record["old"]["pid"],
                    "successor_pid": identity["pid"] if verified else None}
        except Exception:
            return {"state": "unavailable", "reason": "receipt_or_native_status_unavailable"}

    def request(self, generation, *, admitted, kanban_busy):
        if not admitted or not _GENERATION.fullmatch(str(generation)):
            return {"state": "refused", "reason": "default_supervised_gateway_required"}
        if kanban_busy:
            return {"state": "deferred", "reason": "kanban_work_or_unreadable_board"}
        try:
            if self.marker.parent.is_symlink():
                raise ValueError("symlink state directory")
            self.marker.parent.mkdir(parents=True, exist_ok=True)
            # Cross-thread/process receipt serialization. No waiting behind a stuck requester.
            lock_fd = os.open(self.marker.with_suffix(".lock"), os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
            with os.fdopen(lock_fd, "a+b") as lock:
                try:
                    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                except BlockingIOError:
                    return {"state": "pending", "reason": "request_in_progress"}
                return self._request_locked(generation)
        except Exception:
            return {"state": "refused", "reason": "receipt_or_native_preflight_unavailable"}

    def _request_locked(self, generation):
        record = self._read()
        identity, status = self.native.identify(), self.native.status()
        history = []
        if record:
            if not self._verified(record, identity, status, require_current=False):
                return {"state": "pending", "generation": record["generation"]}
            history = [*record["history"], record["generation"]][-16:]
            if generation in history:
                return {"state": "already_applied", "generation": generation}
            if self.now() - record["requested_at"] < _COOLDOWN_SECONDS:
                return {"state": "deferred", "reason": "restart_cooldown"}
        if not _identity(identity, self.home):
            return {"state": "refused", "reason": "gateway_identity_unavailable"}
        if record is None and self.native.already_current(generation, identity):
            return {"state": "already_current", "generation": generation}
        platforms = sorted(p for p, state in (status or {}).get("platforms", {}).items()
                           if isinstance(state, dict) and state.get("state") != "disabled"
                           and state.get("writer_pid") == identity["pid"]
                           and state.get("writer_start_time") == identity["start_time"])
        if not _healthy(status, identity, platforms):
            return {"state": "deferred", "reason": "gateway_not_healthy_or_already_draining"}
        # Native active_agents is the aggregate of chat, cron, API and deferred
        # work. Require only this requesting turn before persisting/sending.
        if type(status.get("active_agents")) is not int or status["active_agents"] != 1:
            return {"state": "deferred", "reason": "requesting_turn_must_be_only_active_work"}
        record = {"version": 1, "generation": generation,
                  "old": {key: identity[key] for key in ("pid", "start_time", "hermes_home")},
                  "requested_at": self.now(), "history": history, "platforms": platforms}
        # Durable BEFORE send: timeout/crash after delivery never causes an automatic resend.
        fd, temp = tempfile.mkstemp(prefix=".repokit-maintenance-", dir=self.marker.parent)
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(record, stream, sort_keys=True)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp, self.marker)
        directory = os.open(self.marker.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
        try:
            ack = self.native.request()
        except Exception:
            ack = None
        accepted = (isinstance(ack, dict) and ack.get("pausing") is True
                    and ack.get("already_stopping") is False and ack.get("pid") == identity["pid"])
        return {"state": "requested" if accepted else "uncertain", "generation": generation,
                "previous_pid": identity["pid"], "verified": False,
                "message": "Finish this turn. Check gateway_restart_status after the supervisor relaunches; do not retry."}


class NativeControl:
    def __init__(self, home, loaded=None):
        self.home = home
        self.loaded = loaded

    def generation_matches(self, generation, identity, *, require_current=True):
        return (self.loaded is not None and self.loaded["pid"] == identity["pid"]
                and self.loaded["start_time"] == identity["start_time"]
                and self.loaded["generation"] == generation
                and (not require_current or generation == _current_generation(self.home)))

    def already_current(self, generation, identity):
        return self.generation_matches(generation, identity)

    def heartbeat(self, identity):
        from hermes_cli.gateway import probe_gateway_loop_liveness
        heartbeat_path = self.home / "state" / "gateway.heartbeat"
        payload = json.loads(_read_bounded(heartbeat_path))
        age = time.time() - heartbeat_path.stat().st_mtime
        return (payload.get("pid") == identity["pid"] and 0 <= age <= 90
                and probe_gateway_loop_liveness(identity["pid"], home=self.home,
                                                tick_timeout=1.0, tick_strikes=1) == "alive")

    def identify(self):
        from gateway.control_socket import identify_gateway
        return identify_gateway(self.home, timeout=2.0)

    def status(self):
        from gateway.control_socket import query_gateway_control
        return query_gateway_control(self.home, "status", timeout=2.0)

    def request(self):
        from gateway.control_socket import pause_gateway_for_update
        # Native handler waits at most 5s for main-loop acceptance. Do not use
        # its 2s default here, and NEVER invoke a legacy fallback on None.
        return pause_gateway_for_update(self.home, timeout=6.0)


def _current_inputs(home):
    import yaml  # existing Hermes dependency
    config = yaml.safe_load(_read_bounded(home / "config.yaml"))
    if not isinstance(config, dict):
        raise ValueError("invalid config")
    return config, generation_digest(config, _read_bounded(home / "SOUL.md", missing=""))


def _current_generation(home):
    return _current_inputs(home)[1]


def _runtime(ctx, loaded):
    from hermes_constants import get_hermes_home
    from gateway.status import _get_process_hermes_home
    home = _HOME
    if ctx.profile_name != "default" or Path(get_hermes_home()).resolve() != home or Path(_get_process_hermes_home()).resolve() != home:
        raise ValueError("default gateway home required")
    return home, Broker(home, NativeControl(home, loaded))


def _caller_admitted(ctx):
    try:
        from gateway.session_context import get_session_env
        from tools.process_registry import _is_supervised_gateway_process
        from hermes_cli.platforms import PLATFORMS
        _runtime(ctx, None)
        human_platforms = set(PLATFORMS) - _PROGRAMMATIC_PLATFORMS
        return (get_session_env("HERMES_SESSION_PLATFORM", "") in human_platforms
                and get_session_env("HERMES_CRON_SESSION", "").lower() not in {"1", "true", "yes", "on"}
                and _is_supervised_gateway_process())
    except Exception:
        return False


def _request_for_context(ctx, loaded):
    if not _caller_admitted(ctx):
        return {"state": "refused", "reason": "human_default_gateway_context_required"}
    from hermes_cli.kanban_db import kanban_home
    home, broker = _runtime(ctx, loaded)
    config, generation = _current_inputs(home)
    # The native dispatcher reads dispatch_in_gateway at boot. Checking only the
    # file would admit a still-running embedded dispatcher after a config edit.
    if not manual_dispatch_config(config) or not loaded or loaded.get("manual_dispatch") is not True:
        return {"state": "deferred", "reason": "manual_kanban_dispatch_required_at_boot_and_now"}
    return broker.request(generation, admitted=True, kanban_busy=kanban_busy(Path(kanban_home())))


def register(ctx):
    # Registration-time attestation is immutable for this loaded plugin instance.
    # A successor tool can verify its own startup digest; the old process cannot.
    try:
        from gateway.status import _get_process_start_time
        config, generation = _current_inputs(_HOME)
        loaded = {"pid": os.getpid(), "start_time": _get_process_start_time(os.getpid()),
                  "generation": generation, "manual_dispatch": manual_dispatch_config(config)}
    except Exception:
        loaded = None

    def available():
        return _caller_admitted(ctx)

    try:
        from tools.registry import no_cache_check_fn
        available = no_cache_check_fn(available)
    except ImportError:
        pass

    def request_handler(args, **kwargs):
        try:
            result = _request_for_context(ctx, loaded)
        except Exception:
            result = {"state": "refused", "reason": "native_preflight_unavailable"}
        return json.dumps(result)

    def status_handler(args, **kwargs):
        try:
            if not _caller_admitted(ctx):
                return json.dumps({"state": "unavailable", "reason": "human_default_gateway_context_required"})
            _, broker = _runtime(ctx, loaded)
            result = broker.status()
        except Exception:
            result = {"state": "unavailable", "reason": "default_gateway_context_required"}
        return json.dumps(result)

    for name, handler, description in (
        ("gateway_restart_after_turn", request_handler,
         "Request one graceful native restart after this turn to apply changed public runtime settings or SOUL. Finish your reply immediately after acceptance. A request is not proof of restart; never retry pending/uncertain requests. Defers while Kanban work is queued or active."),
        ("gateway_restart_status", status_handler,
         "Read-only verification of the requested restart: fresh successor PID/start time and healthy native platform state. Does not request or retry a restart."),
    ):
        ctx.register_tool(name=name, toolset="repokit_maintenance",
                          schema={"name": name, "description": description,
                                  "parameters": {"type": "object", "properties": {}, "additionalProperties": False}},
                          handler=handler, check_fn=available, description=description)

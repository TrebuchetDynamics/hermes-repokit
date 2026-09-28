# Native restart bridge qualification

Source inspected read-only in `hermes-hermes-repokit:/opt/hermes`, build
`749220ef`. No live restart, model message, credential file or log inspection was
used. The process-name-only snapshot showed Hermes under `s6-supervise`.

## Why a model tool adapter is necessary

There is no registered native restart model tool in this pinned source. The
native `/restart` slash command is a gateway message command, not a model tool.
`tools/terminal_tool.py:_pre_exec_block` runs the unconditional supervised
gateway lifecycle guard before executing commands. `tools/process_registry.py:
_is_supervised_gateway_process` requires launch markers AND ownership of the
live gateway PID; a child CLI is not the gateway itself.

`hermes_cli/gateway.py:_cmd_restart` checks its in-process guard, profile
lifecycle and multiplexer ownership, then dispatches to s6 BEFORE reaching any
external-supervisor restart fallback. `S6ServiceManager.restart` in
`hermes_cli/service_manager.py` calls `s6-svc -t` (SIGTERM). Therefore teaching
the model to invoke the CLI is neither a valid tool path nor the qualified
after-turn path. `_request_gateway_self_restart(pid)` sends SIGUSR1 only when
the target is an ancestor; it is not universally reached by CLI restart.

## Native contract reused

`gateway/control_socket.py` exposes `identify_gateway(home)`,
`query_gateway_control(home, "status")` and
`pause_gateway_for_update(home, timeout=...)`. The control server is local-only;
its Unix socket has mode 0600. A request is one JSON line with protocol 1 and
verb `pause-for-update`; the client unwraps a successful result object or
returns None. The ACK has `pausing`, `already_stopping`, `pid`, `drain_timeout`.

`gateway/run.py:_start_gateway_start_control_socket` marshals the pause request
onto the gateway event loop using `call_soon_threadsafe`, calls
`runner.request_restart(detached=False, via_service=True)` and waits up to five
seconds for acceptance. This plugin sets the client timeout to six seconds. A
timeout is uncertain: the queued callback may still run. No fallback or resend
is allowed. The SIGUSR1 handler reaches the same native request method, but the
plugin neither sends signals nor invokes a command.

`gateway/run_shutdown.py:request_restart` refuses a second restart task, marks
the runner draining, and keeps `_running` true so the requesting turn can
finish its final response. Its separate restart task first waits for active
work and then runs the native shutdown. Defaults in `config_defaults.py` are
1800 seconds after-turn, zero seconds chat stop drain, 30 seconds cron drain.
Wedged work can be excluded and the after-turn cap can expire; this is graceful
best effort, not a guarantee of unlimited drain. Shutdown can interrupt agents,
kill tool subprocesses, or finally exit through its watchdog. It marks
`resume_pending` before interruption and skips `.clean_shutdown` after a timed
out drain. Native planned-restart notification and restart-loop markers remain
owned by Hermes. Service restart exits 75; the s6 finish script permits its
respawn (0 and 78 stop respawning).

The native auto-resume breaker in `gateway/restart_loop_guard.py` suppresses
automatic recovery after three chained interrupted boots (default gap 300s).
It is not a per-generation model-request deduplicator. This adapter therefore
adds one durable, bounded receipt before sending the native request. Pending
or uncertain requests block every generation until a healthy successor is
observed. Sixteen recent generations plus a five-minute cooldown prevent
immediate repetition. There is no retry loop, scheduler or supervisor here.

## Kanban boundary

The native active-work aggregate covers chat, cron, API and deferred agent
builders; it does NOT count Kanban subprocesses. The embedded Kanban dispatcher
checks `_running` and the native pause state, not `_draining`, so it can still
dispatch during the native after-turn window. Requests require explicit native
manual-mode settings both at plugin registration and now:
`kanban.dispatch_in_gateway=false`, `kanban.auto_decompose=false`, and integer
`kanban.max_in_progress=1`. A current-file check alone would miss a dispatcher
still running with its boot settings, so either missing attestation defers.
This adapter then takes a read-only,
fail-closed SQLite snapshot of the shared default board and additional boards;
ready/running/review cards or any remaining worker PID defer the request. A
missing default board, missing named-board database or unreadable board also
defers. It never
updates cards or leases. This snapshot is not an atomic admission fence against
a concurrent external CLI claim or new ready card. Do not describe this as a
proven Kanban-wide drain. Qualification of such concurrent admission requires a
native gateway change or a separately authorized native pause workflow.
Before persisting or sending an actual request, the native status aggregate
`active_agents` must be exactly integer 1, representing the admitted requesting
turn. `gateway/run.py` publishes `_active_work_count()` into that field, so
other chat, cron, API or deferred work also defers the request. Zero, missing
and malformed counts fail closed. Successor health verification does not
require an active turn. This further limits local admission without making the
snapshot atomic against external CLI dispatch.

## Installation and verification boundary

Install this directory as a native local plugin using the pinned plugin
scanner/installer. The qualified default gateway home is fixed at `/opt/data`.
Enable the plugin and expose its `repokit_maintenance` toolset
only where intended. The request tool independently requires default-profile
context, a human messaging platform and native supervised gateway PID ownership.
Supported human surfaces come from pinned `hermes_cli.platforms.PLATFORMS`,
excluding CLI, cron and programmatic API/webhook/ACP surfaces. It does not load
the plugin platform registry or maintain a separate seven-platform subset.
Availability is uncached and handlers repeat the admission check. Workers, cron,
API/webhook sessions, named profiles
and arbitrary home/PID arguments are not admitted. No additional dependency is
required; PyYAML is already part of Hermes.

`gateway_restart_after_turn` derives a generation from selected public runtime
configuration fields and SOUL (an exact scalar allowlist; no recursive config
capture, `.env` or auth stores). The supported fields are model default/provider,
agent turn/drain budgets, terminal backend, memory provider, gateway multiplex
mode and Kanban dispatch/decomposition/concurrency settings. Other configuration changes do not trigger this
adapter. This is a configuration/SOUL trigger, not a source-code updater.
When the requested inputs already match this process's registration attestation
and no receipt is unresolved, the tool returns `already_current` without sending
a request. An actual request returns `requested` or `uncertain`, never success. The model must
finish its turn instead of polling its own successor or retrying.

`gateway_restart_status` performs one read-only observation. `verified` means a
different PID with later native start fingerprint and matching live socket
status, running gateway state, no pending restart, and every previously
enabled platform connected without an attention flag and with the successor's
writer identity. Native session-store status must be `ok`, and the runtime
snapshot must have a valid timezone-aware `updated_at` no older than the native
120-second staleness budget. The immutable
plugin-registration digest, current on-disk digest and requested digest must
agree; registration must belong to the successor's PID/start fingerprint. A
fresh native heartbeat for that PID and a successful native loop-liveness probe
are also required. Missing or mismatched evidence is `successor_unverified`.
This registration-time attestation is not an independent proof of the semantics
of arbitrary configuration values. It never clears or rewrites the
receipt. A malformed receipt fails closed; a dead/starting/degraded successor
does not pass. Repair of an unresolved receipt is an operator action, never an
automatic model retry.

Offline tests: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s
packaging/maintenance -p 'test_*.py'`. Fixtures mock native endpoint clients and
use temporary homes/SQLite only. They do not load the installed Hermes plugin,
contact a running gateway, or qualify a live restart.

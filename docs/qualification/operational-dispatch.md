# Operational dispatch — source validation, live acceptance pending

> Delivery follow-up: full offline and race suites now pass, including the
> Unix-socket cases blocked in the earlier record below. Live dispatch acceptance
> remains unqualified. See [delivery validation](../implementation-progress.md#git-delivery-validation-2026-09-28).

RepoKit now separates bootstrap dispatch-off from operational automatic dispatch.
Setup requires all six native identities, provider resolution, channel tool parity,
default routing, initialized Kanban and authenticated shared OpenViking.
It activates only the default gateway, with review dispatch,
concurrency one, no decomposition and an explicit six-profile allowlist.

The implementation uses the selected Hermes source revision
`749220ef0007f8d87bd1531f1c24b0fe93816385`:

- `gateway/kanban_watchers.py`: startup-sensitive embedded dispatcher and
  `kanban/.dispatcher.lock`.
- `gateway/kanban_watchers_common.py`: explicit concurrency startup log.
- `hermes_cli/kanban_db_dispatch.py`: hot-read, fail-closed profile allowlist and
  native worker fingerprint/liveness checks.
- `hermes_cli/kanban_output.py`: worker PID and actor belong to run history;
  task JSON does not expose the worker PID.
- `hermes_cli/config.py`: string config values must be passed without JSON quotes.

Setup fences native board claims and refuses active/finalizing workers. A stopped
previously operational gateway can recover. A failed canary saves dispatch-off
and an empty temporary allowlist to prevent new claims; it does not interrupt an
active worker. The next successful reconciliation restores the six-profile list.
Read-only verification checks configured versus live state, process identity,
singleton ownership, startup evidence and canary qualification separately.

## Evidence from this change

| Check | Result |
| --- | --- |
| Activation gate, restart/fence/recovery and native-shaped canary fixtures | PASS |
| Kernel singleton-lock ownership fixture | PASS |
| Targeted Go tests with race detector: cli, native, gateway, team, verify, acceptance | PASS |
| `go vet ./...` | PASS |
| `gofmt` and `git diff --check` | PASS |
| Full `go test ./...` / `go test -race ./...` | BLOCKED by `TestNativeGatewaySocketsRemainInspectable`: sandbox denies Unix-socket `setsockopt` |
| Deployed setup, real automatic researcher canary | BLOCKED: Docker socket permission denied |
| Fresh Telegram task → researcher → result notification | NOT TESTED |
| Real executor → independent same-card reviewer → Telegram result | NOT TESTED |

The native-command fixtures exercise source control flow and qualified data
shapes. They are not evidence of real model execution or Telegram delivery.
Independent source review identified and corrected the native PID projection,
string-config coercion, residual-worker and stopped-gateway recovery defects.

Apply from a Docker-enabled owner terminal using the new build's
`setup --team` after the required memory stage are configured.
No provider wizard is repeated by that command. Use a fresh Telegram `/new`
session after successful activation. Live canary and Telegram acceptance must
pass before calling this deployment operational; no manual dispatch is a substitute.

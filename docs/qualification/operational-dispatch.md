# Operational dispatch — source validation, live acceptance pending

> Delivery follow-up: full offline and race suites now pass, including the
> Unix-socket cases blocked in the earlier record below. Live dispatch acceptance
> remains unqualified. See [delivery validation](../implementation-progress.md#git-delivery-validation-2026-09-28).

RepoKit now separates bootstrap dispatch-off from operational automatic dispatch.
Setup requires all six native identities, provider resolution, channel tool parity,
default routing and initialized Kanban. Shared OpenViking readiness is independent
and does not block core dispatch. It activates only the default gateway, with
review dispatch, concurrency one and no decomposition. Activation initially admits
only researcher; the six-profile allowlist is released after the canary passes
and its worker exits.

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

## Canary repair (2026-09-29)

The v3 probe asked for an H1 Markdown title even when the README used an image
hero and H2 section headings. Two live researcher runs completed but returned
section headings (`Running and remaining gates`, then `Quickstart`), so the
required evidence comparison failed. Recovery correctly left the default config
at `dispatch_in_gateway: false` and `dispatch_profiles: []`; those settings explain
why the ready executor card was not claimed.

The v4 probe instead asks the researcher to read and return the exact first
physical README line in `metadata.first_line`, without Markdown interpretation.
An empty first line is valid; only an absent README uses `README_MISSING`.
Permission, symlink and oversized-input failures remain failures. The expected
line is not supplied in the card. Historical canaries retain their bodies and
run history; v4 has a distinct native idempotency key.

The executor/reviewer allowlist is not released while the canary is unqualified.
After native gateway-spawned completion, RepoKit waits for worker exit, reacquires
board claim fences, rechecks generation, gateway identity, singleton ownership
and emergency pause, then hot-updates the allowlist to the six managed profiles
and publishes the new generation. It does not run `kanban dispatch` or restart a
second time merely to change the native hot-read allowlist. Any canary/release
failure returns to dispatch-off/empty-list recovery.

Offline validation: full Go suite, vet and targeted race tests (`native`,
`gateway`, `cli`) passed. Live repair results are recorded separately after
reinstallation; this section alone is not a live acceptance claim.

## Historical evidence from the initial change

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
`setup --team` after the default provider is configured; optional memory setup
can be resumed separately with `setup --memory`.
No provider wizard is repeated by that command. Use a fresh Telegram `/new`
session after successful activation. Live canary and Telegram acceptance must
pass before calling this deployment operational; no manual dispatch is a substitute.

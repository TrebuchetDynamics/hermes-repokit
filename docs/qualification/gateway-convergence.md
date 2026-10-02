# Native gateway convergence

> Live follow-up: the repaired native setup passed automatic researcher execution
> on this repository. See the [2026-09-28 dogfood record](live-dogfood-2026-09-28.md)
> for the canary, remaining memory/channel limits and source compatibility fixes.

> Earlier delivery follow-up: full offline and race suites passed, including the
> Unix-socket cases blocked in the original record below. Live gateway acceptance
> was still pending at that stage. See [delivery validation](../implementation-progress.md#git-delivery-validation-2026-09-28).

Qualified source: Hermes image selected in `internal/qualification`, native build
`749220ef0007f8d87bd1531f1c24b0fe93816385`. Read-only inspection of the installed
`hermes_cli/gateway.py`, `hermes_cli/service_manager.py`, `gateway/status.py`,
`gateway/run_startup.py` and `gateway/shutdown_watchdog.py` established:

- Native restart returns after signalling; exit zero alone is not health evidence.
- PID/start/lock records, fresh heartbeat, loop witness and session-store health
  provide independent process observations.
- Only current adapter writer identities count; historical rows are insufficient.

RepoKit's operational setup finalizer holds the normal container writer lock and
native per-board dispatch locks. Ready/review queues are preserved; running tasks
and surviving terminal-run workers defer reconciliation. Native worker fingerprints
distinguish a finalizing subprocess from a recycled historical PID. Fresh zero
live gateway activity is checked again immediately before native restart. The
replacement must have healthy adapters, changed identity, unchanged input hash,
its own singleton dispatcher lock and native startup logs showing concurrency one.

Fresh/incomplete installs keep dispatch off. After provider/profile/tool/routing
gates pass, with optional memory reported independently, the default gateway
runs automatic dispatch and review with the six non-default worker profiles in
its allowlist (the permanent roster is default plus those six). Confirmed stopped
gateways can be prepared without restarting an absent process, then started through
native Hermes. `setup --team` requires no credential wizard. A real researcher
canary must be observed running as a child of this gateway, then complete with
structured title/no-write metadata. Completed canaries are archived natively.

A canary failure sets dispatch off and clears the native hot-read allowlist to
prevent new claims, then restarts only when workers are quiescent. A still-running
worker is preserved and live verification reports stale until reconciliation can
finish. Successful activation never retains an empty allowlist.

The private generation receipt contains only a hash, process/boot/namespace
identity, observed adapters, startup dispatch settings and bounded canary evidence.
It is diagnostic evidence, not another task database; native Kanban retains runs
and history. Inputs include managed identities, descriptions, tool/dispatch policy,
public memory settings, install provenance and model/provider
identifiers. Credentials, sessions and learned memory are excluded. Nothing in
the Hermes runtime imports RepoKit or depends on this receipt.

Verification recomputes the input generation and validates live evidence without
restarting, writing receipts, loading plugins or calling models. Current generation
does not prove every channel or a cached conversation schema. Start a fresh native
session after reconciliation; live channel acceptance remains separate.

Offline tests cover changed identity/tools, secret exclusion, missing/symlinked
inputs, stale counters, busy work, failed restart, PID reuse, changed inputs
during restart, idempotence and stopped gateways. Native metadata fixtures cover
held locks, old adapter rows, unhealthy current adapters and malformed state.
These fixtures are not live Telegram acceptance. The current restricted session
cannot run Docker subprocesses or the real Unix-socket test; live application,
Telegram `/new` identity/tools, researcher canary and memory acceptance remain
blocked/unperformed. No credentials were requested or runtime files hand-edited.

## Channel parity and agent-initiated restart

The follow-up directive broadens default's core capabilities to native
platform-preset-derived selections. Native `tools enable` accepts categories,
not composite preset names: it can reject `hermes-telegram` while exiting zero.
Provisioning therefore resolves the installed preset, uses native category
enablement and checks the effective result. Programmatic surfaces are excluded.
The observer parses `hermes_cli/platforms.py` as syntax without executing it;
it reports saved core categories and declared routes separately from live evidence.
Conditional profile routes are not reported as whole-channel reassignment.
No effective credential/plugin/managed-overlay resolver is invoked by verify.

The installer finalizer's external CLI `gateway restart` is an ordinary
operator-initiated restart; it is **not** the agent self-restart surface and does
not qualify an agent-initiated after-turn restart. The pinned gateway terminal
tool blocks lifecycle calls, and the s6 CLI path sends termination rather than an
after-turn request. No native registered model restart tool was found. The
permitted minimal broker in
`packaging/maintenance` calls native `pause-for-update` over the control socket.
That native handler schedules drain/restart on the serving loop and acknowledges
before the current turn finishes. Hermes owns drain, recovery and relaunch.

The broker checks default process ownership, native supervision, manual dispatch,
board activity and current adapter health. It records a bounded generation request
before sending; uncertain/pending outcomes cannot automatically repeat. Successor
status requires replacement PID/start, native health and adapter ownership,
registration-time loaded generation and unchanged current inputs. Source tests
use injected native observations; they do not qualify an actual Telegram restart.
`gateway_restart_after_turn` remains an unqualified/unknown maintenance capability
unless separately installed and accepted against the selected native release.
Scanner admission, native registry acceptance, channel reconnect and removal-first
agent-initiated restart remain explicit live gates. Ordinary CLI restart evidence
must not be used as evidence for that capability. There is no scanner bypass.

## Current repair validation (2026-09-28)

The full normal and race Go suites both ran. Every other package passed; the
same six `TestNativeGatewaySocketsRemainInspectable` subtests failed to create
Unix sockets with `setsockopt: operation not permitted`. Vet, formatting,
`git diff --check`, Docker-tagged acceptance compilation and a fresh build passed.
No socket test was weakened or treated as passing.

The review build at `/tmp/hermes-repokit-channel-parity` ran public read-only
`verify`. Compose, launcher and filesystem observations remain healthy; all six
historical managed SOULs await source-driven identity migration. Docker subprocess
metadata is unavailable in this sandbox; native memory configuration remains
pending. The build was not applied to the existing deployment. Native scanner and
registry fixtures, real remote coding, authorized channel delivery and self-restart
remain NOT TESTED here. Existing README and skill file hashes match the preserved
baseline; no commit, push, private setup or hand-edited runtime repair occurred.

Mutating setup now resolves configured adapters through native gateway config and
profile secret-scope APIs, extracting only enabled platform identifiers. It covers
native scoped environment and legacy JSON without manually opening secret stores
or printing values. It does not run native dotenv sanitization or fetch uncached
external-secret sources. Native installed-plugin config hooks are part of this
setup trust boundary; they are never used by the passive verifier. Resolution
failure or an enabled platform outside the qualified catalog stops provisioning.

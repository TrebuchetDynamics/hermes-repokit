> Historical source-repair plan. Its dispatch-off policy is superseded by [operational activation](2026-09-28-operational-dispatch.md). The process/identity safety work remains applicable; no live acceptance is implied.

# Live runtime convergence implementation plan

**Goal:** Apply the owner's source-only repair directive: repository-specific
profile identities, mandatory default interfaces, and honest gateway freshness.

**Architecture:** Keep the Go bootstrapper and native Hermes lifecycle. Render
identity from basename/stable project ID; upgrade only byte-exact historical
SOULs. Reconcile required tools additively through native config commands. Run
one native gateway restart after successful setup stages, never per profile.
Generation diagnostics are one-shot observations, not runtime dependencies.

**Policy:** Dispatch remains deliberately manual/off with auto-decomposition off
and maximum concurrency one. SOUL, setup output and verify must say so. Live
automatic orchestration is not claimed. Memory stays pending until private
native setup; a fresh plugin interpreter does not prove live gateway hooks.

**Constraints:** Preserve credentials, sessions, board, owner drift and all
existing uncommitted work. No commit/push. No manual runtime repairs. No model
calls in verify. Use the pinned Hermes native lifecycle and fail closed on
unknown process identity, active workers, failed restart, or stale generation.

## Work

- [x] Repository identity: `internal/team`, native payload/provisioning, exact
  historical migration, owner drift and cross-repository tests.
- [x] Default interfaces: required Kanban and memory fallback plus saved
  platforms; native config semantics, worker separation, read-only verification,
  synthetic Telegram schema acceptance.
- [x] Gateway: qualify pinned restart/status contracts, add projection hash
  and process-bound receipt, active-work refusal, one final restart, health
  confirmation and fresh-session instruction. Test stale/unknown/current and
  failed-restart paths without claiming mock output as live acceptance.
- [x] Integration: all setup variants and normal reconcile share finalization;
  remove redundant container recreation. Verify description,
  identity, roster, dispatch, interfaces and gateway independently.
- [ ] Validation: unit/race/vet/format/diff checks; build fresh temporary binary;
  use public RepoKit commands on existing installation when sandbox allows.
  Record Docker/socket/private-memory blockers and live Telegram gaps.

## Review focus

- User SOUL edit must survive historical migration.
- Telegram selection created after bootstrap must gain both required tools.
- Failed private setup must never certify a new gateway generation.
- An old receipt, reused PID, changed input or unhealthy gateway must not pass.
- An interrupted/repeated setup must preserve runtime state and remain resumable.

## Validation receipt

Focused tests and the offline suite excluding the separately reported real
Unix-socket test pass with the race detector. Vet, formatting and diff checks pass;
Docker-tagged acceptance compiles. The full suite still fails only the six
`TestNativeGatewaySocketsRemainInspectable` cases because the sandbox denies
socket creation. Do not weaken those tests or treat exclusion as full success.

A new temporary binary was built and public read-only `verify` run. Docker
subprocess metadata remains unavailable; existing historical identities await
`setup --team` migration. No live reconciliation or Telegram acceptance was
performed from a build whose mandatory offline gate remains blocked. Private
Memory acceptance remains blocked. No commit/push occurred.

## Follow-up channel parity and self-maintenance directive

- [x] Resolve installed native human platform presets, then use native category
  enablement with explicit Kanban/memory and preserve owner additions.
- [x] Give default a channel-independent identity and bounded non-secret
  self-maintenance contract; preserve steward's lifecycle authority.
- [x] Project configured/saved channels, core selections and declared routing
  without importing Hermes, loading plugins or exposing credential values.
- [x] Implement a minimal native drain/restart broker with bounded request
  receipts, active-work refusal and independent successor observations.
- [x] Install bundled broker through native scanner and pinned local Git install;
  preserve edited source and fail closed on scanner refusal.
- [ ] Native scanner/registry fixture, real model-initiated originating-channel
  restart, cross-channel identity/board/memory and removal-first acceptance.
- [ ] Full mandatory offline and Docker gates in an environment that permits
  Unix sockets and Docker subprocess access. Source fixtures are not acceptance.

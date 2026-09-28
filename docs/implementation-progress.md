# Implementation status

RepoKit's maintained runtime is one Hermes development container with six native
profiles, shared Kanban and embedded OpenViking. The optional Docker acceptance
daemon remains an explicit testing feature. Optional plugins are owner-managed.

The source implements private-state/ownership checks, a standalone launcher,
conservative profile reconciliation, shared-memory setup/linking, operational
activation and passive verification. Full v1 acceptance remains incomplete.

## Preserved historical evidence

Credential-free Docker foundation fixtures previously exercised native six-role
cloning, distinct SOULs, Kanban handoffs and same-card lifecycle transitions in
separate profile processes. The recorded final scaffold run took 120.56 seconds;
a later package run recorded 120.67 seconds. Those fixtures used synthetic provider
configuration, not live model workers. Ordinary Compose restart/recreation and
removal of copied installer artifacts preserved the board and profiles.

Native OpenViking configuration/linking fixtures also passed historically.
Configuration and pending-service observations do not qualify the current embedded
image, actual extraction/recall or cross-repository isolation. See [runtime
observations](qualification/runtime-observations.md), [team evidence](qualification/generic-team.md)
and [memory wiring](qualification/openviking-wiring.md).

The historical development self-install had six profiles and successful fresh
default/researcher model sessions. That narrower authentication/identity evidence
did not establish model-driven Kanban work, independent review or shared recall.
No new deployment evidence is asserted by the current source cleanup.

## Source changes and blockers recorded on 2026-09-28

Setup preflight checks private-state Git protection, generated artifacts and the
qualified running image/project/service/mounts before entering a native wizard.
Owner changes, foreign mounts/images and unknown runtime metadata refuse safely.
Native identity reconciliation uses repository-scoped SOULs and preserves drift.
Default's configured interactive channels receive native preset tools plus Kanban
and memory. A fresh native conversation is required to refresh cached tool schemas.

Operational setup separates dispatch-off bootstrap from automatic default gateway
execution/review. It checks providers, identities, tools, routing and authenticated
shared OpenViking; fences board claims; preserves active/finalizing workers; and
requires native singleton/startup evidence and a real no-write researcher canary.
Read-only verification never dispatches or runs inference.

The current full `go test ./...` and `go test -race ./...` attempts ran without
exclusions. All packages passed except `internal/target`: its six
`TestNativeGatewaySocketsRemainInspectable` cases failed with
`setsockopt: operation not permitted`. Required socket tests were neither weakened
nor counted as passing. Vet, formatting and diff-whitespace checks passed, and the
core executable built successfully at `/tmp/hermes-repokit-core`.

The full Docker acceptance suite was also attempted with both
`REPOKIT_DOCKER_TESTS=1` and `REPOKIT_DIND_TESTS=1`. All six fixtures failed at Docker
socket access or container enumeration; they were attempted, not skipped:
development runtime, foundation, isolated acceptance daemon, default Kanban channels,
native maintenance package and pending OpenViking. Local logs are
`/tmp/repokit-remove-supervision-test.log`,
`/tmp/repokit-remove-supervision-race.log` and
`/tmp/repokit-remove-supervision-docker.log`.

Live reconciliation, embedded-image builds, Telegram task/result delivery, canary,
self-restart and private-memory acceptance were not completed. No private setup was
repeated or runtime state repaired by hand. These are source-only changes, not proof
that a running deployment or published revision has converged.

## Git delivery validation, 2026-09-28

Fresh `go test -count=1 ./...` and `go test -race -count=1 ./...` passed every
package, including all previously blocked Unix-socket cases. Normal and
Docker-tagged vet, formatting, `git diff --check`, Docker acceptance compilation,
and `go build ./...` passed. Relative documentation links resolve. Python bytecode
caches are ignored and excluded from delivery.

Delivery review also caught activation failing on absent or heading-free READMEs,
including the new HTML-led README. The canary now requires `NO_MARKDOWN_TITLE`
for those cases while retaining worker ancestry and no-write checks. Regression
coverage preserves bounded-read failures and terminal legacy canary history;
versioned keys avoid resuming unfinished tasks with the older contract.

These offline results supersede the earlier socket blocker. Docker image builds,
live model work and private setup were not run during Git delivery and remain
separate acceptance gates.

## Remaining acceptance

1. Run the full required suite and current image build in an authorized environment.
2. Prove gateway-spawned researcher work and originating-channel notification.
3. Prove an actual executor/reviewer correction cycle with distinct same-card actors.
4. Prove cross-profile memory write/recall, restart persistence and repository denial.
5. Complete the integrated removal-first fixture, future-specialist qualification
   and bounded self-dogfood before release.
6. Resolve the recorded Superpowers scanner candidate separately if selected.

See [operational dispatch evidence](qualification/operational-dispatch.md),
[gateway evidence](qualification/gateway-convergence.md),
[development runtime](qualification/development-runtime.md) and [remaining work](../TODO.md).

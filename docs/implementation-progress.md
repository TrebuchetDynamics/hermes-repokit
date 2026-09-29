# Implementation status

Latest live repair: native setup completed, the gateway automatically ran and
archived a no-write researcher canary, and core dispatch remains enabled with
memory pending. See the [dogfood record](qualification/live-dogfood-2026-09-28.md).

RepoKit's maintained runtime is one Hermes development container with seven native
profiles, shared Kanban and user-managed memory. The optional Docker acceptance
daemon remains an explicit testing feature. Optional plugins are owner-managed.

The source implements private-state/ownership checks, a standalone launcher,
conservative profile reconciliation, operational
activation and passive verification. Full v1 acceptance remains incomplete.

## Preserved historical evidence

Credential-free Docker foundation fixtures previously exercised native six-role
cloning, distinct SOULs, Kanban handoffs and same-card lifecycle transitions in
separate profile processes. The recorded final scaffold run took 120.56 seconds;
a later package run recorded 120.67 seconds. Those fixtures used synthetic provider
configuration, not live model workers. Ordinary Compose restart/recreation and
removal of copied installer artifacts preserved the board and profiles.

Native memory configuration/linking fixtures also passed historically.
Configuration and pending-service observations do not qualify actual
extraction/recall or cross-repository isolation. See [runtime
observations](qualification/runtime-observations.md) and
[team evidence](qualification/generic-team.md).

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
execution/review. After checks pass, setup writes the native dispatch policy with
`hermes config set` and restarts the gateway once. `verify` is observational; it
does not dispatch or run inference. The explicit `verify --dispatch-check` is the
researcher proof of automatic gateway dispatch.

In a later Docker-capable session the full `go test ./...` and `go test -race ./...`
suites passed every package, including the six
`TestNativeGatewaySocketsRemainInspectable` cases previously blocked by
`setsockopt`. Vet, formatting and diff-whitespace checks passed.

The credential-free Docker fixtures also ran with `REPOKIT_DOCKER_TESTS=1`: the
development runtime, foundation, default Kanban
channels and native maintenance package all pass. The privileged nested-daemon
fixture (`REPOKIT_DIND_TESTS=1`) was not run.

This pass also repaired pre-existing defects in the uncommitted verification work:
the npm probe's duplicate `/dev/null` config, the standalone Kanban fixture's
missing `/workspace` mount and too-short timeout, its over-strict worker toolset
assertion, and the gateway probe passing a ~278 KB one-shot script through
`python -c` beyond the kernel per-argument limit (now streamed on stdin).

That fixture pass did not exercise live providers. The subsequent dogfood run
built the image, reconciled the live team and completed automatic researcher work.
Telegram round-trip delivery, agent self-restart and private-memory acceptance
remain open; no private setup was repeated.

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

These offline results supersede the earlier socket blocker. Subsequent Docker
and live researcher evidence is recorded in the dogfood record above; it does not
qualify private memory, independent review or human-facing channel delivery.

## Remaining acceptance

1. Prove originating-channel task/result delivery after the completed researcher run.
2. Prove an actual executor/reviewer correction cycle with distinct same-card actors.
3. Complete the integrated removal-first fixture, future-specialist qualification
   and bounded self-dogfood before release.
4. Resolve the recorded Superpowers scanner candidate separately if selected.

See [operational dispatch evidence](qualification/operational-dispatch.md),
[gateway evidence](qualification/gateway-convergence.md),
[development runtime](qualification/development-runtime.md) and [remaining work](../TODO.md).

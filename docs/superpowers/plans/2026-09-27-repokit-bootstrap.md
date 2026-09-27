# Hermes RepoKit final implementation plan

2026-09-27. Authoritative plan replacing the prior twelve tasks and all A–G runtime-manager plans. Implement the [approved bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) in order. User explicitly authorizes implementation and a separate commit per completed phase. Existing Go skeleton is reusable; historical source research is not release qualification.

Use TDD: write the behavior test, observe RED, implement minimally, observe GREEN and full offline suite, review, then commit. Keep each phase's evidence and remaining limitations in TODO.md. Do not proceed into a runtime integration using guessed upstream contracts. Resolve exact immutable source/image/plugin pins before admission; document unavailable evidence instead of inventing success. All live tests are gated; no mock proves inference/memory/review.

## Package boundaries

`cmd/hermes-repokit`, `internal/{cli,qualification,target,compose,process,locking,install,launcher,native,plugins,memory,verify}`, `testdata`, `tests/acceptance`, `docs/qualification`. Preserve the existing `qualification` package name rather than making a duplicate `qualify` package. Prefer standard library; no external dependency means no artificial go.sum. Generated launcher is POSIX shell; native configs are YAML/files. No host Python packaging or tests.

## Phase 0 — Correct authoritative documents

Replace prior plan; commit this phase alone before product edits.

## Phase 1 — Go CLI and qualification model

Reuse existing skeleton; correct module path; typed exact-version contracts reject absent/unsupported operations. Qualify image/chat/setup/profiles/descriptions/Kanban/toolsets/plugins/OpenViking/Nerve/Laya/exec identity. Source inspection and runtime evidence remain distinct.

## Phase 2 — Target identity and collision handling

Canonical paths, full normalized names and path-hashed Compose identity; preflight root Compose/state/links/ownership/permissions/tracked files/container/PATH collisions. Test fresh, ambiguous, same-basename, dangling-link and foreign-owner targets.

## Phase 3 — Compose renderer

Readable YAML, explicit project, pinned images, exact mounts and environment, isolated sidecars. Validate offline rendering and actual Compose parsing.

## Phase 4 — Atomic installer and locking

Private staging, flock and atomic publication, safe reruns preserving native edits/unknown files regardless of receipt. Test interruption, stale locks, retaining child, partial publication, pull failure and receipt loss/corruption.

## Phase 5 — Standalone launcher

POSIX exec Docker; absolute file, sanitized selectors, exact argv, terminal selection. Fake Docker tests hostile args, exit, streams, SIGINT and unrelated cwd. No RepoKit runtime dependency.

## Phase 6 — Setup delegation

Generated launcher setup with inherited streams; stopped native error plus start instruction, no automatic start or secret capture.

## Phase 7 — Native defaults, engineering profiles and Superpowers

Qualified native board initialization, dispatch/decomposition off, descriptions/toolsets and no credential cloning. Pin upstream Superpowers and obey exact scanner verdicts; fresh-session evidence.

## Phase 8 — Upstream Nerve and supported Laya

Qualify/pin/install/configure upstream Nerve; selected Nerve sidecar/checkpoint only. No custom plugin/server. Prove load, typed inference, model, consumption, restart and outage behavior.

## Phase 9 — Official OpenViking and native provider

Official digest, private persistent workspace, native init/doctor and memory setup handoffs. Accept/document native extraction; no credential collection. Prove write/restart/recall/isolation/degradation.

## Phase 10 — Same-card review qualification

Prove distinct builder/reviewer run actors under native Hermes/Nerve. Add minimal standalone policy only after demonstrating an actual enforcement gap.

## Phase 11 — Read-only plan and verify

Wire actual target/artifact/runtime inspection with bounded probes and independent statuses. No mutating native commands or receipt trust.

## Phase 12 — README and quickstart

Maintain README early; document the four commands, engineering, native setup, raw Compose operations, PATH opt-in and unfinished integrations truthfully.

## Phase 13 — Removal-first acceptance

Explicitly gated disposable-repo live fixture removes installer checkout/binary/receipt; proves independent native execution, raw Compose, actual same-card work and persistence. Mock results cannot qualify release.

## Phase 14 — RepoKit dogfood

Only after removal gate: installed native team makes bounded source improvement, distinct reviewer, tests and normal Git evidence.

## Phase 15 — Release binaries

Offline checks, patched toolchain qualification, reproducible linux/amd64 and linux/arm64 binaries and checksums. Cross-build is not runtime evidence; do not publish a v1 with incomplete gates.

## Completion contract

One Go binary bootstraps readable standard Compose, private native state, Kanban, Superpowers, optional engineering profiles, Nerve using local Laya and native OpenViking. After native setup, deleting RepoKit leaves `hermes-<repo>` and ordinary Docker/Hermes operational indefinitely. Complete means the real removal gate and dogfood pass, not just offline tests. No speculative memory abstraction, Nerve implementation, Laya API, runtime manager or policy framework belongs in this plan.

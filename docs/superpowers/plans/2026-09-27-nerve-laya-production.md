# Nerve/Laya production implementation plan

> Execute inline using the existing approved bootstrap design and Phase 8.

**Goal:** Package the qualified local tuple reproducibly, provision an explicitly
selected Laya sidecar, and configure native Nerve before enabling each profile.
**Spec:** ../specs/2026-09-27-repokit-bootstrap-design.md

## Constraints and decisions

Preserve the six roles, /workspace + /opt/data, manual dispatch, native Kanban,
native memory, existing profiles and ordinary Compose lifecycle. No custom
server/plugin, hosted fallback, public port or runtime RepoKit callback.
The existing OpenViking work is committed at d9f88be; private live memory inputs
are still missing. Independent packaging can proceed while those inputs wait.

Use the exact qualified Linux amd64 CPU tuple, dependency hash lock and five
model-file hashes. Build a self-contained image using ordinary Docker. Select
its verified content image ID explicitly for install; do not invent a registry
digest or silently publish. The image must load without network/cache mounts.
Laya shares the Hermes network namespace and binds 127.0.0.1:8765. Recreate both
services together when that namespace changes, and test this behavior.

## Tasks

1. Package `packaging/laya`: immutable base and upstream Nerve archive, hash-locked
   dependencies, verified fixed model revision, offline runtime and native
   healthcheck. Contract tests must fail before the recipe exists. Build it and
   run actual typed inference without external networking or host dependency
   mounts. A build/runtime failure blocks installer promotion of that image.
2. Add explicit `--laya-image` plan/install selection and generated Laya service.
   Accept full local content image IDs with pull disabled. Check the selected
   image's qualified recipe/platform identity. Extend conservative publication
   for exact recognized prior Compose, backing up old bytes and preserving native
   state. Test omitted/invalid/unknown image, owner edits, upgrade and rerun.
3. Add `setup --supervision`: verify exact runtime/topology/health, install pinned
   Nerve through native `--no-enable`, preserve owner settings, write local Laya
   settings, then native enable and registry-loaded LOCAL_ONLY probe. Apply to all
   six roles; native default clones inherit the same supervision settings. Test
   failure before enable, drift refusal, interruption/retry, restart and Laya loss.
4. Update evidence and docs; run unit/race/vet and disposable Docker acceptance
   after removing the copied installer. One fresh final review. Keep real memory
   and full model-driven dogfood explicitly pending private setup; never count
   synthetic local decision probes as authenticated main-model work.

## Review focus

- Unknown native plugins/settings must be preserved; scan failure never forced.
- No transient hosted-default enablement on partial setup or reruns.
- Mutable tags or wrong image architecture never count as the qualified image.
- Shared network namespace must survive coordinated Compose recreation.
- Model bytes/dependencies must be baked and checked, with no runtime download.

## Ledger

Baseline: d9f88be, clean isolated worktree on feat/nerve-laya-production.
Ruling: retain the accepted /opt/data and OpenViking design over generic installer
skill defaults; no Holographic migration or alternate layout. Cost if wrong:
requalify the selected integration rather than silently migrating owner state.
Ruling: ordinary local Docker build plus immutable image ID avoids an unapproved
registry publication. Cost: users build locally until a release image is qualified.

## Delivery checkpoint

The owner requested commit/push during implementation. This checkpoint ships
the image recipe and explicit sidecar selection only. Task 3, native Nerve
admission/configuration/activation, remains unimplemented; no
`setup --supervision` command is advertised by the CLI. Combined Compose
recreation and full-stack acceptance remain pending.

Validation: image build succeeded; the gated network-disabled image test loaded
the pinned dependencies and verified all five model hashes without host mounts.
Go unit and race suites, vet and whitespace checks passed.

The built image also passed actual upstream `/healthz` and choice/score/noul
inference under `--network none`, a read-only filesystem, nonroot UID, four CPUs
and 6 GiB memory, with no host mounts. This does not establish coordinated
Compose recreation or native Nerve admission across profiles.

Final delivery review found one command-path issue: whole-string replacement
could redirect the recreation command for an unusual supported ancestor path.
Regraded as important because lifecycle commands must preserve the selected
repository. `TestLayaRecreationCommandPreservesRepositoryPath` reproduced it;
replacing only the command suffix fixes it.

Review ruling: image labels are compatibility metadata under an explicitly
owner-trusted local build, not independent content attestation. The documented
image-content test and offline inference receipt qualify the tested build only.
Cost if that boundary is misunderstood: a modified local image could be selected;
no native Nerve activation is performed in this milestone. Combined namespace
recreation, all-role native activation and authenticated dogfood remain separate
acceptance gates; the review did not independently execute those tests.

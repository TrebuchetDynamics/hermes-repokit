# RepoKit Bootstrap Implementation Plan

> **For agentic workers:** Use superpowers:subagent-driven-development or superpowers:executing-plans. Follow the foundation-first execution order below, TDD and independent review. Commit each completed phase separately. The user's final directive authorizes implementation; do not reopen an architecture cycle.

**Goal:** One Go executable bootstraps independent native Hermes deployments and becomes unnecessary.
**Architecture:** Go inspects targets and publishes ordinary Compose/native configuration plus a standalone POSIX launcher. Upstream plugins/sidecars own runtime behavior. Receipts are optional information.
**Spec:** [authoritative bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md).
**Tech stack:** Go standard library, deterministic YAML/templates, POSIX shell, upstream Hermes/Superpowers/Nerve/OpenViking and Nerve-supported Laya. Host module `github.com/TrebuchetDynamics/hermes-repokit`.

## Corrections and current baseline

At the start of this amendment, the Go skeleton, qualification evaluator and README already existed (43f5a6a / 347d062). Their preservation and canonical module migration were the starting point; current progress is recorded below. This plan replaces the prior twelve-task implementation detail, not the bootstrap boundary. No custom Nerve plugin, custom Laya protocol/server, custom memory provider or speculative review-policy plugin. Remove the prior durable-only OpenViking gate: native sync/extraction is expected and documented. Older runtime-manager/A–G plans remain superseded.

The current implementation and evidence status is tracked in [TODO](../../../TODO.md) and [implementation progress](../../implementation-progress.md); unchecked steps below remain implementation/qualification obligations, not a claim that no foundation code exists.

## Execution order: finish the foundation first

The latest review explicitly requires a usable four-command foundation before
Nerve, Laya, OpenViking or multi-profile orchestration. Existing Go packages and
README are retained. The remaining foundation work executes inline:

1. Wire default `plan`/`install` to a Hermes-only artifact set using the recorded
   immutable Hermes image and existing Compose/launcher/publication primitives.
   `install` publishes files under the held lock; it does not pull images, start
   services, install plugins or run native initializers. Print the exact ordinary
   Compose start command. Recheck Git and container collisions under the lock.
   Preserve matching installations, native edits and captured Docker context;
   refuse foreign/stopped same-name containers. `--engineering` still refuses.
2. Make default `verify` report foundation artifacts and runtime metadata only.
   An intact generated Compose/launcher/config plus matching running container
   can pass without optional integrations. Unknown/edited artifacts and stopped
   containers remain non-success. Never imply authenticated chat or inference.
3. Exercise the real CLI in a disposable Git repository, preserve user config
   on rerun, and prove generated launchers operate from another cwd after the
   copied installer binary, source copy and receipt are removed. Run ordinary
   Compose restart against the pinned image in the gated Docker test. Keep this
   foundation removal gate distinct from the full v1 integration release gate.
4. Only after that gate passes, resume phases 7–10 for upstream integrations.
   Nerve 0.3.0 is a qualification candidate, pinned to an immutable upstream SHA
   before installation; its supported Laya sidecar supplies the backend.
   Same-card native review is tested before considering a tiny policy plugin.

Files: `internal/cli/{cli,install}.go`, `internal/cli/*_test.go`,
`internal/install/install.go`, `internal/verify/{verify,verify_test}.go`,
`tests/acceptance/{foundation,docker}_test.go`, README and quickstart.
Reuse `internal/qualification`; do not rename it or recreate the Go skeleton.

Tests must first fail for default install refusal and verification's unrelated
integration blockers. Add collision, failed under-lock inspection, rerun/context,
owner-edit and CLI removal regressions; retain existing publication/lock/TTY
coverage. Validate with offline `go test ./...`, `go test -race ./...`, `go vet
./...`, formatting, and separately gated credential-free Docker acceptance.

Full v1 remains subject to phases 13–15. Those release gates and the Superpowers
scanner decision do not block publication of the Hermes-only foundation.
The phase numbers below describe component scope, not permission to implement
integrations before this foundation gate.

## Global constraints

- Exactly four host commands: plan/install/setup/verify; no resident manager/dispatcher/chat proxy.
- One Hermes runtime per repository, standard independent Compose, exact full normalized `hermes-<name>` container/launcher names.
- Relative mounts from `.hermes/compose.yaml`: `..` → `/workspace`, `.` → `/opt/data`; explicit Docker context/file/env-file, no root Compose autodiscovery.
- Native files authoritative, no credential capture, no destructive reset, no `down -v`, no secret logs/receipts.
- Default and engineering presets both disable dispatch and auto_decompose; native operator owns enablement.
- External commands use argv arrays and context deadlines; setup streams inherited; bounded output for observations.
- Source qualification precedes native command use; immutable release pins, no guessed aliases, UID/HOME or model contracts.
- Ordinary tests offline (`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`); live tiers require explicit disposable scope and data/provider/budget authority.
- Keep source qualification, fake-runner evidence and actual runtime/model evidence distinct; pending live tiers block release completion, not independent implementation.

## Package interfaces and ownership

Create packages only as their phase needs them. Keep the existing `internal/qualification` name.

- `target`: `Identity{Root, Name, Project string}`; `Inspect(root string, inventory []Container) (Identity,error)` and injectable inventory records bind container labels to canonical roots. `CheckFilesystem` is read-only and refuses ambiguity.
- `compose`: typed `Config` with identity, digest-pinned selected services and native exec/start contract; `Render(Config) ([]byte,error)`, no secrets. YAML scalars escaped for both YAML and Compose interpolation.
- `process`: `Invocation{Program string; Args,Env []string; Dir string; In io.Reader; Out,Err io.Writer; ExtraFiles []*os.File}`; `Runner.Run(context.Context, Invocation) (int,error)`; normalized explicit environment, no shell strings. Bounded capture belongs to observations, not setup.
- `locking` and `install`: held Linux flock and private directory identity; `PublishNew` no-clobber; typed bootstrap request/result/receipt. Never use receipt to authorize overwrite/adoption.
- `launcher`: typed absolute Compose/context/native executable/exec options/noargs mapping; `Render` and explicit collision-checked shortcut creation.
- `native`, `plugins`, `memory`: typed source-qualified native argv/config and component selections; upstream packaging only. No secret DTOs or runtime control abstractions.
- `verify`: typed component observations and status, bounded read-only probes. No writes on missing state.
- `cli`: parses four command flag sets, wires one-shot calls; main only streams/context/exit/signal wiring.

Interface refinements must be documented with the phase, tested and reviewed; they cannot expand runtime scope.

## Review focus

1. Unknown ownership despite plausible receipt or matching basename must refuse before effects (phases 2/4).
2. Interrupted install and a surviving child must not allow another writer (phase 4).
3. Shell metacharacters/newlines, unrelated cwd and TTY/signals must remain transparent (phases 5/6).
4. Native scanner caution/failed probes never become implicit approval or successful verification (phases 7/11).
5. Source research and mock success cannot qualify native memory, inference or independent review (phases 8–10/13).

## Phase 0: amend authoritative docs before further code

- [x] Retain implemented Go skeleton and replace speculative integration scope in design/plan.
- [x] Commit the upstream scope correction separately before further code (`a90381b`); retain independent review findings and current status in TODO/qualification docs.

## Phase 1: Go module and version-specific qualification

Files: `go.mod`, `cmd/hermes-repokit/main.go`, `internal/cli`, `internal/qualification`, `docs/qualification`.

- [ ] RED: retain missing/unsupported/mismatched qualification rejection; add typed source-contract completeness tests for selected native operation facts, never a map of executable arbitrary strings from user input.
- [ ] Migrate module/imports to canonical GitHub path; retain four-command boundary. Add exact source revisions, image digest/platform correspondence and reviewed source references for native chat/setup, profile creation/descriptions, Kanban config/init, toolsets, plugin scanner, memory provider, Nerve/Laya and exec shim.
- [ ] Source-recorded contracts are not runtime qualification; missing upstream operations refuse affected enabling. Separate optional integration availability.
- [ ] GREEN: package tests, full tests/race/vet/format; independent review and commit. No fixture invents commands to pass.

## Phase 2: conservative target identity and collision refusal

Files: `internal/target/{identity,inspect}.go` and package-local tests; `testdata/` only for reusable fixtures.

- [ ] RED: temporary repositories cover aliases, spaces/control chars, full normalized names, same-basename foreign/stopped containers, root Compose variants, foreign `.hermes`, links/dangling links/hardlinks, unsafe ownership/modes and tracked ignored state.
- [ ] Implement canonical identity and deterministic Compose-project hash; use actual Git index/ignore and injected Docker inventory through bounded runners. No mutations during inspection. Refuse unknown ownership, not best-effort adoption. Existing launcher proof does not grant overwrite rights.
- [ ] GREEN: target tests including zero writes; full offline suite; review and commit.

## Phase 3: readable independent Compose

Files: `internal/compose/compose.go`, templates and tests.

- [ ] RED: deterministic bytes, correct mount bases, explicit project/container name, digests only, quotes/dollars, default service and selected sidecars, no public ports/socket/installer mounts, no secret interpolation.
- [ ] Render typed source-qualified service configurations. Hermes startup must keep the single intended container available for native exec without guessed startup command. OV scaffold persists `/app/.openviking`, bot disabled, config-pending handoff. Laya appears only with explicitly selected and qualified Nerve backend.
- [ ] GREEN: exact structural/golden fixtures and authorized `docker compose config` when available; no daemon/model required for unit tests; review and commit.

## Phase 4: atomic publication, held locks and reruns

Files: `internal/process`, `internal/locking`, `internal/install` plus package-local tests.

- [ ] RED: no-clobber and private modes, ancestor replacement/symlink races, interrupted publication, lock competition/stale file, parent exit with live child lock, partial effects, missing/corrupt receipt, owner edits/unknown native files and no-op rerun.
- [ ] Implement argv/context runner with explicit environment and exit/signal handling. Linux flock must remain held through authorized mutating children via shared file description, close instead of premature LOCK_UN. Pin private directory identities during safe writes; atomic no-replace publication and durability sync, never overwrite arbitrary native content.
- [ ] Installer creates new artifacts or preserves existing owner artifacts; receipt has version/identity/component pins/created paths/time only. Report effects from actual work, not intended state. Recheck inventory/preimages under lock. Failed pulls/scanners preserve state.
- [ ] GREEN: focused process/lock/publication tests and race suite; review and commit.

## Phase 5: standalone repository launcher

Files: `internal/launcher` template/render/shortcut and tests.

- [ ] RED: execute generated shell against fake Docker to capture exact argv, streams and exit status; spaces/quotes/dollars/semicolons/newlines/unknown command, noargs native mapping, all selector env cleared and unrelated cwd. Test TTY combinations and SIGINT/process group cleanup with Linux PTYs.
- [ ] Shell `exec docker --context ... compose --env-file /dev/null -f ABSOLUTE exec ...` with individually quoted literals and `"$@"`. No shell parse/eval/health/receipt/auto-start logic. TTY only when both input/output terminal. `sh -n` generated artifacts.
- [ ] Optional explicit shortcut: same verified link no-op, all other collisions refuse; no rc edits or invented alternative name.
- [ ] GREEN: launcher tests/race; review and commit.

## Phase 6: direct setup delegation

Files: `internal/cli/setup.go`, runner wiring and tests.

- [ ] RED: inherited streams and exact `[launcher, setup]`, return native exit, Ctrl-C and stopped-service result; no capture of credential prompts, no Docker start.
- [ ] Delegate through generated launcher; show explicit Compose start instruction on stopped failure without interpreting native commands or rerouting setup. No second Hermes container or credential schema.
- [ ] GREEN: injected runner and subprocess tests; review and commit.

## Phase 7: native defaults, engineering profiles and Superpowers

Files: `internal/native`, `internal/plugins`, immutable source contract records and tests.

- [ ] RED: default-only root home, Kanban persisted and dispatch/decomposition false, explicit engineering profiles/descriptions/role tools, existing profiles untouched, no implicit credential cloning. Scanner safe/caution/dangerous/failure/stale approval cases.
- [ ] Use only selected Hermes native init/config/profile/plugin APIs. Pin upstream Superpowers full SHA, honor exact scanner findings approvals, never force. Fresh native session verifies loaded plugin rather than stale schema.
- [ ] GREEN: offline argv/state tests; separately record authorized native evidence, not fake support. Review and commit.

## Phase 8: upstream Nerve and supported Laya backend

Files: `internal/plugins/nerve.go`, selected sidecar config, `docs/qualification/upstream-nerve.md`, tests/acceptance fixtures.

- [ ] RED: Nerve pin/profile config/canonical Kanban ownership, Laya selection/checkpoint/model compatibility, no local plugin/server implementation, no RepoKit runtime callbacks, sidecar omitted when unselected.
- [ ] Qualify and install upstream Nerve native artifact; use its exact Laya backend/sidecar configuration. Sidecar independent, no public port, necessary cache persistent. Health is not inference. Missing upstream pin/contract means unsupported for that selection, not a bespoke replacement.
- [ ] GREEN: offline configuration tests and explicitly authorized plugin load/actual typed decision/model identity/Nerve LOCAL_ONLY/restart/loss degradation evidence. Review and commit; record any unrun tier truthfully.

## Phase 9: official OpenViking and native Hermes provider

Files: `internal/memory` configuration/handoff, Compose sidecar inputs and tests.

- [ ] RED: durable workspace under mounted path, no credentials in config/receipts generated by RepoKit, isolated repo service/auth identity, no public port, native provider endpoint, setup-pending vs health status.
- [ ] Scaffold official service and native configuration; operator runs source-qualified `openviking-server init`/`doctor`, then native Hermes memory setup. Allow standard native automatic sync/extraction; document scope/egress. No durable-only gate, no custom provider or credential wizard.
- [ ] GREEN: offline preservation/config tests; authorized actual remember/restart/recall/cross-repo denial/outage tests. Review and commit.

## Phase 10: same-card review qualification

Files: `docs/qualification/review-path.md`, native acceptance fixtures; policy code only conditional on proven gap.

- [ ] Source-qualify selected Hermes/Nerve completion/review hooks and bind acceptance to actual same-card native history, distinct builder/reviewer runs and actors, candidate evidence and changes/re-review.
- [ ] RED: wrong/self/stale/nonexistent review must fail the acceptance predicate. Native fixture attempts bypass/error paths; no fake boolean fixture establishes enforcement.
- [ ] If evidence proves a builder/reviewer actor-independence enforcement gap, document it and implement only a minimal independent native policy with native state and fail-closed supervised-card behavior. Otherwise no policy plugin.
- [ ] GREEN: offline acceptance predicates plus actual qualified native enforcement evidence; review and commit, no invented controller.

## Phase 11: wire real read-only plan/verify and one-shot install

Files: `internal/cli`, `internal/verify`, `internal/install` orchestrator and tests.

- [ ] RED: execute real CLI with fake runner and temporary repo: plan snapshots unchanged, successful install/rerun, collision/failed pull/scanner effects, setup handoff, verify missing state no writes, independent component outcomes.
- [ ] Parse only four command flag sets including explicit engineering/context/target/shortcut selections. Wire qualified target inspection→admission→lock→publication→native safe initialization→handoff. No reconciliation loop. Preserve owner config and partial-effects output.
- [ ] Plan shows actual identity/collisions/state/pins/profiles/plugins/selection/defaults/proposals/unsupported facts. Verify actual state, not receipt; bounded time/output and no raw secret/native logs. No auth refresh/init/inference/dispatch in probes.
- [ ] GREEN: CLI tests, full tests/race/vet/format; review and commit.

## Phase 12: user README and native quickstart

Files: `README.md`, `docs/bootstrap-quickstart.md`, `docs/build.md`.

- [ ] Keep first screen simple: product/boundary/start/status; distinguish available implementation from qualified release evidence. Show transition from RepoKit to native launcher.
- [ ] Document ordinary Compose up/down/restart/logs/pull with explicit selectors; native Hermes and OpenViking setup without secret capture; selected engineering/plugin/Laya configuration and safe handoff.
- [ ] Verify local links and executable offline examples; live examples labeled by evidence. Review and commit.

## Phase 13: removal-first acceptance

Files: `tests/acceptance`, fixtures and machine-readable nonsecret evidence.

- [ ] Explicit environment/authorization gates for disposable root, selected artifacts, model/config references, spend/download/time and cleanup scope. Missing authority fails invoked suite, never skip-to-green release.
- [ ] Install in unrelated disposable repo; complete native setups; normal Compose start; verify; remove checkout/binary/receipt from all runtime access; unrelated cwd native chat/commands; raw Compose restart; actual bounded same-card builder/reviewer; second restart and persistence proof.
- [ ] Include native mount, sessions, Kanban, plugin/profile, actual memory and Nerve/Laya state. Prove cross-repo denial. No mocks counted as runtime/model evidence.
- [ ] Record outcomes and review. Unrun/failed selected tier means v1 incomplete. Commit fixtures/evidence only, no secrets or ephemeral runtime state.

## Phase 14: RepoKit dogfood

- [ ] Only after phase 13 passes, bootstrap this repository through the installed tool; installed native team performs one bounded improvement, distinct review, tests and normal Git evidence.
- [ ] No self-apply/promotion/runtime manager. Record independent execution evidence; review and commit.

## Phase 15: release binaries

Files: release build script/Makefile, `docs/build.md`, locally generated ignored `dist/`.

- [ ] Qualified patched Go, `CGO_ENABLED=0`, Linux amd64/arm64, `-trimpath`, reproducible build settings and SHA256SUMS. No invented digests; test actual architecture/runtime before qualifying it.
- [ ] Full offline checks plus all selected acceptance/dogfood tiers must pass for v1 completion. Produce local artifacts; publishing is separate authority. Review and commit build inputs, not generated binaries.

## Completion rule

Continue all independent implementation while live setup/budget answers are pending. Stop dependent live work until explicit authority/configuration exists. Do not call partial scaffolding or mock-only tests a complete v1. Track implemented, source-qualified, runtime-qualified and blocked tiers separately in TODO and evidence docs.

# Implementation progress — 2026-09-27

The latest six-role Universal Team Roster directive is the target; v1 is **not complete**.

Current generic-team work supersedes earlier engineering-profile policy:
[team model](team-model.md), [implementation plan](superpowers/plans/2026-09-27-repokit-generic-team.md),
[acceptance evidence](qualification/generic-team.md). Default is the primary UX;
steward owns profile evolution. Native config cloning follows private default
setup and resets only new identities. Shared OpenViking and local Nerve/Laya
production wiring is implemented; live memory and main-model work remain separate
acceptance gates. See the [current runtime record](qualification/runtime-integrations.md).

## Completed implementation foundations

- Separate plan correction commit `a90381b`: Go host, upstream Nerve/Laya, native OpenViking, native review first, automatic memory extraction accepted.
- Canonical Go module and typed version-bound qualification contracts.
- Canonical repository identity, exact public names, path-hashed Compose project; filesystem/PATH/root Compose checks. CLI additionally inspects Git tracking and Docker names.
- Digest-pinned readable Compose renderer, private atomic directory publication with Linux `RENAME_NOREPLACE`, fsync, kernel locking and conservative reruns.
- Standalone POSIX launcher with captured Docker context, absolute Compose path, selector clearing, exact argv/streams/exit behavior and terminal selection.
- Native setup delegation and context-correct stopped-service instruction.
- Read-only JSON `plan` and component `verify`: files/configuration, pinned runtime identity and bounded authenticated memory/Laya health checks; no receipt trust, native DB calls, plugin loading or inference.
- Default Hermes/OpenViking/Laya CLI `install`, embedded pinned Laya build inputs, persistent writable caches, locked inventory recheck, exact historical Compose upgrades/backups and ordinary Compose startup handoff.
- Plain `setup` sequences private default setup, six-role provisioning, native memory init/doctor/validation/restart/health/wizard/linking and native local Nerve installation/configuration/activation. Separate flags resume memory or supervision.
- All six Nerve installs use native admission, remain disabled until local configuration is verified, and check fresh native hooks plus actual `LOCAL_ONLY` decisions before and after coordinated Hermes/Laya recreation. No hosted fallback is configured.
- Exact plugin caution-admission predicate; no forced install.
- Offline unit/acceptance coverage including hostile argv, TTY matrix, SIGINT exit, child-held lock, stale lock file, owner edits, missing/corrupt receipts, failed preparation, concurrent publication, interrupted/partial-state refusal and process-group cleanup.
- Core Go packages and the executable entrypoint have tests; embedded packaging inputs are exercised through installation and real image qualification.

## Historical foundation evidence

Credential-free `TestDockerFoundation` passed: generated Compose parsed; a new
official Hermes container ran from an unrelated disposable repository; native
exec and Kanban initialization worked; raw Compose restart preserved the board.
Cleanup used ordinary Compose down, without `-v`.

Separately observed native profile creation/description and setup help on the
pinned image. Superpowers native installation failed closed at CAUTION with
229 findings for the exact recorded SHA. The official OpenViking image starts
its missing-config handoff; native init reports persistent workspace
`/app/.openviking/data` and cancels without required credentials. No inference,
actual memory write/recall or cross-repository denial was claimed.

Fresh independent code review identified five important issues. Regression
checks reproduced and fixed: no-clobber publication, incomplete reruns,
descendant cleanup, verify Docker context, and setup recovery context/selectors.
`Publish.prepare` remains restricted to immutable image retrieval. Implemented
native initialization holds its writer lock inside the container; Docker client
lifetime alone is not the locking boundary.

## Current runtime integration evidence

The combined foundation/native-memory-link fixture passed in about 163 seconds;
the OpenViking pending-service fixture passed in about 11 seconds. The real
six-role Nerve/Laya fixture passed in about 221 seconds, including native hooks,
`LOCAL_ONLY` decisions, offline operation, fail-closed outage, installer removal
and coordinated Compose recreation. The generated default build separately
passed offline image content qualification and persistent-cache recreation.
See [exact scope and commands](qualification/runtime-integrations.md).

These tests use synthetic main-model configuration or connection fixtures where
stated. They do not establish live main-model dispatch, memory recall/isolation,
independent model judgment or full integrated removal-first acceptance.

## Outstanding, in order

1. Supply private native main-model and embedding/extraction configuration for
   disposable acceptance. No host credentials have been borrowed.
2. Prove live model-driven manual and controlled gateway dispatch, including a
   genuine same-card executor/reviewer correction cycle.
3. Prove real cross-profile OpenViking write/recall, restart persistence and
   cross-repository denial. Configuration/linking and health are already wired.
4. Qualify future specialists, the full combined removal-first gate and full
   self-dogfood. Existing six-role local supervision evidence remains narrower.
5. Resolve exact Superpowers scanner admission independently; the recorded
   229 CAUTION findings remain open and are not waived by other integration work.

## Scope and evidence rulings

- Reused the existing `qualification` package instead of duplicating `qualify`; no external dependencies means no fabricated `go.sum`.
- Native readable child files beneath a 0700 state root are allowed; native Hermes itself creates 0755 profile directories. Group/world writable native children still refuse.
- Metadata observations can proceed while plugin/model admission is blocked. No later runtime integration is enabled based on guessed commands.
- Development binaries are not a v1 release. The local Go 1.26.1 toolchain needs patched-release qualification before shipping.
- Current ruling: normal `install` publishes all three services; startup belongs to ordinary Compose, and native configuration/credentials belong to native setup. Artifact publication does not certify live model work or full v1 acceptance.
- Concurrent edits to TODO, the bootstrap spec/plan and native-contract/upstream-Nerve notes were preserved separately from implementation commits.

## Foundation closeout validation

Fresh `go test ./... -count=1`, race tests, vet, formatting, diff checks and the
real Docker foundation fixture passed. Local links in the changed documentation
were checked. An independent review reproduced missing Git-ignore protection on
safe reruns and verification; regression tests failed before the fix and passed
after adding shared read-only index/ignore checks. Removed ignore files, ignore
exceptions and force-tracked native state now refuse without rewriting files.
The reviewer checked the fix and found no additional actionable issue.

## Phase 7 implementation rulings

- Ruling: preserve ordinary Compose startup. Initial publication reports pending;
  rerunning install on the running pinned deployment performs native initialization.
  An absent/stopped service never causes an implicit start.
- Current ruling: the six-role roster is the default; `--engineering` is a
  compatibility alias. Nerve/Laya deployment and native setup are implemented;
  Superpowers admission and live OpenViking acceptance remain deferred. Profile
  files alone do not establish active integrations.
- Ruling: acquire the writer lock inside Docker for native operations. A host
  lock inherited by the Docker client does not cover surviving daemon-owned exec
  work. Current team creation uses final native profile names to preserve native
  registration and routing semantics. Interrupted profiles are preserved for
  inspection; existing profiles remain owner state.
- Superpowers scanner decision was requested for the exact SHA/report and a
  disposable credential-free load test. Until approved, no plugin installation,
  enablement or affirmative scanner response is authorized. Phase 7 remains open.


## Universal roster closeout

Six native roles, their SOULs, default UX routing, steward policy, conservative
post-setup config cloning, drift preservation and explicit integration-gate
reporting are implemented. The final disposable native fixture passed in
120.56 seconds; unit/race/vet/format/link checks and Linux arm64 compilation
passed. Independent review findings about default adoption, setup targeting,
noninteractive success and native staging side effects were addressed with
regression checks. See the [current matrix](qualification/generic-team.md).

The initial single-worker local Laya experiment is now supplemented by default
pinned build packaging and six-role native Nerve acceptance, including offline
outage/removal/recreation. OpenViking still needs private embedding/extraction
configuration before live recall/isolation/persistence qualification. `verify`
reports native configuration and health independently; review remains explicitly
unqualified. Full team acceptance and release self-dogfood remain pending.

# Hermes RepoKit progress

## Current stage

The Go Hermes-only foundation is usable; **full v1 is not complete**. `plan`, `install`, native `setup` delegation and read-only metadata `verify` work. Installation publishes files and prints the ordinary Compose start command. The real CLI and generated deployment passed credential-free removal/restart acceptance. Native Kanban initialization and the six-role universal scaffold are implemented; see the [current acceptance matrix](docs/qualification/generic-team.md). Superpowers and optional sidecars remain unqualified.

The [design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md) and [phase plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md) follow the final upstream-first directive. The scope correction was committed separately as `a90381b`, before further implementation. Earlier runtime-manager/A–G plans must not execute. See [implementation evidence and remaining work](docs/implementation-progress.md).

## Phase tracking

A phase stays unchecked when its required qualification or runtime evidence is incomplete, even if implementation primitives exist.

- [x] 0: separately committed Go/upstream scope correction.
- [ ] 1: canonical module and typed contracts implemented; full upstream operation qualification pending.
- [x] 2: canonical identity and conservative filesystem, Git, PATH and container-name inspection.
- [x] 3: deterministic digest-pinned independent Compose renderer.
- [ ] 4: no-clobber publication, locks and safe rerun primitives implemented; Hermes-only CLI publication and under-lock collision rechecks implemented; later native initialization/failure matrix pending.
- [x] 5: standalone shell launcher, argv/TTY/signal tests; optional PATH-link installation deferred.
- [x] 6: native setup delegation with inherited streams and context-correct recovery guidance.
- [ ] 7: native Kanban/default config and six-role native clone scaffold implemented; full integration acceptance pending; exact Superpowers approval and fresh-session loading pending.
- [ ] 8: upstream Nerve and its supported Laya sidecar.
- [ ] 9: official OpenViking scaffold/native provider handoff and actual memory qualification.
- [ ] 10: same-card distinct executor/reviewer actor qualification; tiny policy only for a demonstrated actor-independence gap.
- [ ] 11: four-command foundation implemented; optional component initialization/verification incomplete.
- [x] 12: product README, native quickstart and development build documentation.
- [ ] 13: full removal-first acceptance in an unrelated disposable repository.
- [ ] 14: native-team dogfood after acceptance.
- [ ] 15: release qualification; amd64/arm64 development cross-builds exist, not a v1 release.

## Boundaries

Go host binary; standalone shell launcher; upstream plugins/services. No host Python/Node/Pi/Hermes requirement. Exactly one Hermes runtime per repository, `/workspace` and `/opt/data`, ordinary Compose and native commands. Native files authoritative; receipt optional/nonsecret. Default team dispatch and auto_decompose off. Preserve existing state, refuse ambiguity, no automatic rc edits or runtime manager.

No custom Nerve plugin, no speculative Laya API/server, no new memory provider. Native OpenViking synchronization/extraction is intended behavior; the durable-only block is removed. Credentials stay in native setup. Selected-version support and same-card actor independence require actual evidence.

## Evidence and remaining gates

Offline tests, race tests, vet and credential-free Docker foundation checks have passed. Native exec/profile/Kanban persistence and OpenViking's configuration handoff have limited recorded evidence; see [runtime observations](docs/qualification/runtime-observations.md). Actual local Laya typed inference and native Nerve consumption passed in a separate disposable fixture; memory write/recall/isolation, complete team removal-first acceptance and dogfood remain pending.

After the foundation milestone, Superpowers installation remains gated: it refused the exact candidate at CAUTION; the [229-finding scanner report](docs/qualification/superpowers-8ca22dba-scan.txt) requires explicit approval before plugin admission. Commit/merge approval does not waive that gate. Nerve/Laya transport, model identity and actor-independence gaps remain in [upstream qualification](docs/qualification/upstream-nerve.md). Live model work still needs configuration/data/download/spend/time scope. Source facts, mocks and development binaries do not make a release pass.

# Hermes RepoKit repository bootstrap

## Current stage

**Corrected design and replacement implementation plan are DRAFT, awaiting review.** The user requested both corrections, not execution. The previous 29-task A–G runtime-manager plan is obsolete and must not execute. No product code, tests, installs or runtime evidence exists.

- [x] Read existing design and immutable source findings.
- [x] Replace runtime-manager boundary with [bootstrap design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md).
- [x] Write bounded twelve-task [bootstrap plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md), with interfaces, tests/core snippets, red/green/commit steps and inline self-review.
- [x] Mark old September 27 spec and all eight A–G/master plan documents SUPERSEDED / DO NOT EXECUTE; preserve historical bodies.
- [x] Preserve already-selected **subagent-driven** execution method.
- [x] Amend twelve tasks to Go before code: one `hermes-repokit` binary, offline Go default tests; independently packaged native plugin/Laya Python remains container-only.
- [x] Parent-review corrected design/plan, including installed plugin discovery, explicit CLI bootstrap wiring, launcher transparency and separate build/runtime prerequisites.
- [ ] Review the Go draft before the first code task.
- [ ] Only after review/implementation authority, execute the replacement plan. Runtime, provider, download and inference permissions remain separately scoped.

## Boundary to preserve

Go builds one standalone `hermes-repokit` executable; compiler needed only for build/development. First supported release is Linux amd64/arm64 with local Docker, not unqualified cross-platform support. Bootstrap uses Git and qualified Docker/Compose; the generated launcher needs only shell and Docker/Compose. No host Python/venv/pip/Node/Hermes dependency. Embed templates, CGO-disabled reproducible builds and checksummed binary distribution; no required remote installer/curl-to-shell. Native plugin/Laya Python remains inside runtime images. Default `go test ./...` is offline, runtime suites require explicit tags AND scoped authority. Go version/module pins require implementation qualification; no builds/downloads occurred.

RepoKit generates readable `.hermes/compose.yaml` and persistent `.hermes/`, then is unnecessary. Standard Compose owns service lifecycle; native Hermes owns profiles, Kanban, plugins and execution. One Hermes runtime/project; OpenViking selected memory; optional Laya inference sidecar; separately installed self-contained native Nerve/review policy and Superpowers plugins. No resident manager, runtime DB, reconciliation or hidden plugin-to-installer dependency.

Core installer CLI: **plan/install/setup/verify** only; setup delegates directly, never a separate wizard. Default host command AND container are `hermes-<normalized-repo-name>`; `polymarket-mega-bot` becomes `hermes-polymarket-mega-bot`. Standalone POSIX shell launcher `.hermes/bin/hermes-<name>` opens native chat without args and passes ALL explicit args unchanged, preserving native errors. No host Hermes/Python/Node/Pi/RepoKit runtime dependency, health gate or chat supervisor. Optional authorized same-name PATH link refuses collisions; no suffix/rc/PATH edits. TTY only when stdin AND stdout terminal; otherwise `-T`, not help.

Lead with `hermes-<repo> setup`. If stopped, operator runs `docker --context CONTEXT compose --env-file /dev/null -f .hermes/compose.yaml up -d hermes` with selector environment cleared. Raw Compose up/down/restart/logs/pull remains supported with explicit `-f`; root autodiscovery is superseded. Relative mount base is `.hermes`: `..` → `/workspace`, `.` → `/opt/data`, no project-directory override or native `.env` interpolation. Exact full normalized container/command names collide deliberately; reject empty/long names, stopped-container and PATH/function/link conflicts. Official shim UID/GID/HOME and bare chat behavior require qualification, not guessed values.

- [x] Incorporate [precursor adopt/adapt/reject review](docs/research/2026-09-27-repokit-precursor-review.md) without importing its manager/chat adapter.

Fresh native default-only SAFE has dispatch false and auto_decompose false. Optional explicitly requested engineering preset also leaves dispatch false. Managed concurrency1/bounded fleet are native operator recipes, not RepoKit modes/admission receipts. Native downgrade preserves profiles/history. Same-card builder/reviewer must be distinct existing native runs with exact profiles and independently qualified fail-closed native policy; no goal mode default. Role runner qualification cannot invent a controller or claim universal adversarial enforcement over native administrators.

Owner config/Compose edits remain authoritative. Small optional `.hermes/repokit-install.json` is installer-only. Missing/corrupt receipt does not affect runtime; rerun refuses ambiguous ownership rather than adopting/resetting. Preserve unknown auth/history/config, native sessions/board/memories, sidecar data/cache and Nerve-derived state. No secret hashes, `down -v`, or RepoKit uninstall/apply/activate requirement.

## Unresolved qualification, not fabricated support

- Native OpenViking raw background uploads remain incompatible with durable-only privacy at the research pin. Refuse enabling this integration before active config writes/starts; leave existing state unchanged. Independent explicitly selected scaffold is allowed, not an alternate provider. A compatible native build must satisfy privacy with installer absent. No invented opt-out, pre-tool upload guard or silent upstream enhancement scope.
- Role/execution/review, root-key protection through both mount aliases, exact native command spelling, immutable component artifacts/scanner admission and optional Laya pin-aware bounded packaging require operation-specific qualification. Unsupported paths remain withheld; independent infrastructure need not stop.
- Laya outage is UNKNOWN/WATCH; memory outage degrades already-authorized native workflows, not a board kill switch. Health/offline mocks do not prove inference, privacy or memory recall.

## Release evidence still required

Authorized disposable Pi-absent host → qualify binary without Go/Python/Node host tools (Git/Docker/shell remain) → separate fresh target bootstrap → native setup/ordinary Compose → remove all installer checkout/binary/receipt artifacts from imports/subprocess/mount access → invoke generated launcher from unrelated cwd and raw native commands → restart → native administration and actual bounded same-card independent completion → restart again → addressable sessions/native logical history/actual selected memory/Nerve/cache persistence. PATH hiding alone is insufficient. Optional source-improvement dogfood is useful but no improved-CLI self-apply/promotion/host-release-root gate remains.

Separate offline, credentialless authorized runtime and scoped actual inference evidence; none is claimed now. Keep interrupted bootstrap, stale lock, collision, scanner/pull failure, owner-edit and state-preserving native down/restart/rerun coverage. Historical September 26 Task 1 remains paused/superseded; this correction does not revive it.

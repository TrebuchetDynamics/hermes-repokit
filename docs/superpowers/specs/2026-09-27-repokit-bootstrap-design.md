# Hermes RepoKit bootstrap design

2026-09-27 — authoritative final implementation directive.

Hermes RepoKit is an opinionated, short-lived repository bootstrapper that generates a normal Docker Compose + native Hermes environment. Once configured, RepoKit is not required. This document replaces earlier bootstrap requirements and all runtime-manager plans. The user's final directive authorizes implementation, phase commits, removal-first acceptance and subsequent dogfood.

## Product boundary

The Go module is `github.com/TrebuchetDynamics/hermes-repokit`. One executable exposes exactly `plan`, `install`, `setup`, `verify`; `--engineering` explicitly selects the engineering preset. No daemon, apply, activate, scheduler, task dispatcher, chat proxy or resident manager. Host Python, Node, Pi and Hermes are unnecessary. Prefer the standard library. Native plugins and ML runtimes retain their upstream implementations.

RepoKit qualifies, pins, installs, configures and verifies upstream components. Superpowers comes from obra/superpowers; Nerve is the upstream Hermes community plugin. Laya uses the selected Nerve release's supported backend, checkpoint and sidecar contract. OpenViking uses its official service and Hermes's native provider. Do not implement `plugins/nerve`, a Laya protocol/server, or a memory provider. A tiny independent native review-policy plugin is permitted only after selected-version qualification demonstrates an actor-independence enforcement gap; never a speculative policy framework or second approval/task database.

## Files and topology

One Hermes container per repository, plus OpenViking and optional engineering/Nerve-Laya sidecar. Generate readable `.hermes/compose.yaml`, `bin/hermes-<name>`, native `config.yaml`, `profiles/`, `plugins/`, native `kanban.db`, `openviking/`, `nerve/`, optional `laya/`, and optional informational `repokit-install.json`.

Mount the repository at `/workspace` and `.hermes` at `/opt/data`; set `HERMES_HOME=/opt/data` and `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`. Qualify the official image's exec UID/HOME behavior; do not derive an image just to change Unix HOME. Pin immutable release image digests. Compose has an explicit project name, no root autodiscovery, no sidecar public ports. Native state is private, persistent and ignored by Git.

Canonicalize the target path. Normalize the full basename to lowercase ASCII with nonalphanumeric runs replaced by hyphens: `My Project!!` becomes `my-project`. Container and launcher are exactly `hermes-<full-normalized-name>`. Refuse unsupported names and collisions, including stopped containers and same-basename repositories; never truncate, suffix or add hashes to these names. The internal Compose project may contain a canonical-path hash.

Inspect root Compose variants, existing state, symlinks including dangling links, ownership, permissions, tracked private files, containers and PATH links before writing. Never adopt ambiguous state. Lock installer operations, publish atomically, preserve owner edits and unknown native files. The receipt contains only version, identity, component versions, generated paths and timestamp; it is neither credentials nor authority. Missing/corrupt receipts do not disable runtime or authorize overwrites. Interrupted installation, stale locks, lock-retaining children, partial publication, pull/scanner failures and reruns must be covered. Never use `down -v` as recovery.

## Native launcher and setup

Owner-executable POSIX shell uses Docker directly with an absolute Compose path and `--env-file /dev/null`, clearing interfering `COMPOSE_*` selectors. It ultimately `exec`s Compose exec with native Hermes and exact `"$@"`. No args opens native chat. Explicit arguments, native errors, exit codes, streams and Ctrl-C pass through unchanged. Allocate TTY only when stdin and stdout are terminals. No eval, user-argument shell parsing, callbacks, receipts, health checks or auto-start. It works from unrelated directories without RepoKit. PATH links require explicit authorization and collision checks; never edit shell rc files.

`setup` directly delegates to the generated launcher's `setup` with inherited terminal streams. It never captures credentials, duplicates a wizard or starts a stopped service. Preserve the native error and show the explicit Compose start command when stopped.

## Native defaults and engineering

Fresh default profile only; native board initialized and persistent. Always start with `kanban.dispatch_in_gateway: false` and `kanban.auto_decompose: false`. Enable qualified Kanban toolsets as appropriate. Engineering may additionally create researcher, planner, builder, reviewer using qualified native `profile create` / `profile describe` commands and real descriptions; preserve existing profiles and never implicitly clone credentials. Operator enables dispatch.

Superpowers and Nerve use qualified immutable upstream revisions and native plugin installation/scanning. Safe may install; caution requires explicit approval of exact findings and SHA; dangerous, disabled or failed scanner refuses. Never silently force. Verify loading in a fresh session. Nerve uses Hermes Kanban as canonical lifecycle and its own supervision/DoD/evidence/trajectory state. Qualify installation, enablement, profile support, completion and restart persistence.

Laya stays in Nerve's supported independent sidecar, configured through qualified `reflex_backend`, `reflex_laya_base_url`, `reflex_laya_model` settings. Persist required cache beneath `.hermes/laya`. Model/checkpoint comes from Nerve compatibility evidence, not the latest standalone Laya. Health, actual typed result, reported model, restart and LOCAL_ONLY identity are live gates. Outage degrades supervision without corrupting Kanban. Mocks never prove inference.

## Native OpenViking

Mount `.hermes/openviking` at `/app/.openviking`, containing `ov.conf`, `ovcli.conf`, `data/`. Explicitly set durable storage workspace `/app/.openviking/data`; qualify the release. No public 1933 port; Hermes endpoint `http://openviking:1933` and `memory.provider: openviking`. Disable the bundled bot where required by the official image.

OpenViking requires embedding and VLM configuration. Hand operators to native `openviking-server init` and `openviking-server doctor` in the container, then native `hermes memory setup` or exact qualified configuration. This is separate from `hermes setup`. RepoKit never collects model/API credentials.

Accept native automatic session synchronization and extraction. The earlier durable-only privacy requirement is removed. Document that user/assistant turns and eligible tool inputs/results may be synchronized and model providers may receive content according to native configuration. No invented upload-disable semantics. Prove actual write, restart, recall, cross-repository denial and truthful outage degradation. Use independent per-repo services, networks, state and native authorization.

## Review, observation and release

Use native same-card builder request_review → distinct reviewer approve/request_changes → builder if changes requested, under Nerve supervision. Acceptance proves distinct native run actors. Only demonstrated upstream enforcement gaps justify a minimal native-only policy plugin usable without RepoKit.

`plan` is read-only and reports target, identity, collisions, existing state, qualified pins, profiles/plugins, board defaults, memory and Nerve/Laya selection, changes and unsupported features. `verify` reads actual artifacts/runtime, not receipts. Every probe has timeout/output bounds. Report healthy/degraded/pending-setup/unsupported/unknown independently. Never initialize DBs, refresh credentials, infer, dispatch, extract, migrate or write during verification.

Every Go package has offline unit tests. TDD throughout; injected runners/fake Docker test argv, hostile strings, TTY/non-TTY, SIGINT, unrelated cwd, publication, collisions, stopped services, owner edits and failures. Run `go test ./...`, `go test -race ./...`, `go vet ./...`, gofmt. Live fixtures require explicit environment gates and actual authorized execution, separate from offline evidence.

Mandatory release gate: disposable unrelated repo → install → native setup and selected component configuration → ordinary Compose start → verify → remove RepoKit binary AND checkout AND receipt → unrelated cwd → native launcher/chat/commands → raw Compose restart → actual bounded Kanban builder/distinct-reviewer work → restart → prove sessions/history/board/memory/Nerve/Laya persistence. No calls/imports/mounts of RepoKit may survive. Dogfood on this repository only after that gate, with real builder/reviewer/tests/Git evidence and no self-apply requirement.

Build CGO-disabled Linux amd64 and arm64 binaries where practical, include checksums and provenance, require no Go on end-user hosts. Do not claim release completion until live gates actually pass. The README explains product and quickstart; docs contain detailed architecture/qualification.

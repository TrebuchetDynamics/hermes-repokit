# Hermes RepoKit bootstrap design

Date: 2026-09-27. Authoritative amendment implementing the user's final directive.
This replaces earlier bootstrap assumptions and all obsolete runtime-manager plans.
The existing Go skeleton and README are retained; the repository is no longer documentation-only.

## Product

Hermes RepoKit is an opinionated, short-lived repository bootstrapper that generates a normal Docker Compose + native Hermes environment. Once configured, RepoKit is not required.

Exactly one Hermes container belongs to a repository. Standard Compose owns service lifecycle; native Hermes owns chat, profiles, Kanban, setup, plugins and memory integration. RepoKit has no daemon, apply loop, dispatcher, chat proxy, runtime database or second Hermes command hierarchy. Generated artifacts, plugins and sidecars must not import/call/mount RepoKit or depend on its optional receipt.

| Component | Implementation/owner |
| --- | --- |
| RepoKit CLI, configuration generation and installer tests | Go |
| Generated `hermes-<repo>` | Standalone POSIX shell |
| Compose | Readable deterministic YAML |
| Hermes configuration | Native YAML/files |
| Superpowers | Upstream `obra/superpowers` native Hermes plugin |
| Nerve | Upstream Hermes community plugin |
| Laya | Nerve-supported independent sidecar |
| OpenViking | Official service and Hermes's native provider |

Module: `github.com/TrebuchetDynamics/hermes-repokit`. Prefer the Go standard library; a pinned YAML library is acceptable if needed. There is no host Python/Node/Pi/Hermes prerequisite. Do not rewrite plugins or ML runtimes for language uniformity. Existing package names may remain where they express the same responsibilities; no empty scaffolding directories.

## First milestone

The initial default deployment contains only Hermes, native safe-default
configuration, Compose and the standalone launcher. `install` generates these
files with conservative locking/publication and prints the ordinary Compose
start command. Native setup follows after the operator starts Hermes. No plugin,
sidecar, multi-profile orchestration, model call or native initializer runs in
this milestone. Foundation verification covers artifacts and runtime identity;
it does not claim authenticated chat or model readiness.

Prove the four commands and launcher/Compose independence after removal of
RepoKit before starting upstream integrations. The later full v1 release gate
still covers the selected plugins, sidecars and distinct same-card review.

## Deployment and identity

```text
<repo>/.hermes/
  compose.yaml
  bin/hermes-<normalized-full-repo-name>
  config.yaml, profiles/, plugins/
  ... native Kanban, sessions, authentication and history ...
  openviking/{ov.conf,ovcli.conf,data/}
  nerve/
  laya/                         # only when selected
  repokit-install.json          # optional nonsecret information
```

Repository root mounts at `/workspace`; `.hermes` mounts at `/opt/data`.
Use `HERMES_HOME=/opt/data` and `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`.
Do not derive an image merely to change Unix HOME or guess exec UID/GID.
Qualify the official image and its exec shim. One Hermes runtime plus selected sidecars is the invariant.

Canonicalize target paths. Normalize the complete basename to lowercase ASCII, replace nonalphanumeric runs with hyphens and trim hyphens. Container and launcher are exactly `hermes-<name>`; never truncate, suffix or hash them. Empty/unsupported names refuse. A canonical-path hash may distinguish Compose projects; equal normalized basenames intentionally collide across projects, including stopped containers.

Before writing, inspect root Compose variants/overrides, existing `.hermes`, symlinks/dangling links, ownership/modes, tracked private state, running/stopped container identity, PATH and launcher-link collisions. Never adopt ambiguous deployments, silently switch Docker contexts, recursively chown source or overwrite foreign files. Existing owner configuration is authoritative. `.hermes` must remain private and ignored/unindexed.

Compose is independently usable with explicit `--env-file /dev/null -f .hermes/compose.yaml`; mount paths resolve relative to that file. Clear interfering Compose selectors, use the selected explicit Docker context, no root forwarding Compose, no project-directory override, no native secret `.env` interpolation. Released presets store immutable image digests and full plugin SHAs, never moving tags. No public sidecar ports, Docker socket, privileged workaround or second board/provider.

## Four host commands

- `plan`: bounded read-only target/collision/component inspection and proposed changes; never changes state.
- `install`: one-shot bootstrap and conservative safe rerun. Creates only new managed artifacts, installs source-qualified pinned components through native admission and initializes safe native defaults. Reports partial effects; never releases dispatch or captures credentials.
- `setup`: directly executes generated-launcher `setup` with inherited terminal streams. No wizard or credential capture. Stopped service preserves native failure and shows the explicit Compose start command; no automatic start.
- `verify`: bounded read-only observations of actual files/services/native artifacts, independent of receipt. Statuses: healthy, degraded, pending-setup, unsupported, unknown. Optional failure does not falsely fail unrelated components. No initialization, migration, credential refresh, inference, extraction, dispatch or write probes.

Explicit `--engineering` selects profiles and qualified Nerve/Laya capability. Selection details must appear in plan output; no silent fallback or enabling incompatible components. Separate native setup handoffs explain pending component configuration.

## Standalone launcher

`.hermes/bin/hermes-<repo>` is owner-executable POSIX shell plus Docker. No arguments opens native Hermes chat (qualify whether a zero-arg mapping is required). Every explicit argument is forwarded unchanged, including unknown native commands; native errors are preserved. Use an absolute Compose path and explicit context. Clear `COMPOSE_FILE`, `COMPOSE_PROJECT_NAME`, `COMPOSE_PROFILES`, `COMPOSE_ENV_FILES`. Use `exec`, not eval or `sh -c` around arguments. Preserve streams, exit status, Ctrl-C; allocate TTY only when stdin and stdout are terminals, otherwise use `-T`. Work from any directory.

No auto-start, health gate, receipt check or RepoKit callback. Optional same-name PATH shortcut requires explicit selection/authorization and collision checks; never edit shell rc/PATH automatically. Unknown parent-shell alias/function state leaves shortcut activation pending rather than claiming resolution is safe.

## Safe publication and process execution

Use `os/exec` argv arrays, `context.Context` deadlines, bounded read-only output and cancellation of process groups where appropriate. Native interactive setup inherits streams and signal behavior; never log raw credentials/arguments. Typed configuration/observation structs contain no credentials intended for receipts/logs.

Use private installer locks and atomic no-clobber publication. Verify filesystem ownership and symlink/ancestor identity, preserve dirty Git trees and unknown native state. Reruns compare actual artifacts and independent ownership evidence, not receipt desired state. Missing/corrupt receipt never resets configuration or blocks runtime. Interrupted installs retain truthful partial effects. Locks survive parent death while an authorized mutating child retains the lock descriptor. Never delete a held lock as stale-PID recovery; never use `down -v` for normal recovery.

## Native profiles and plugins

Fresh installation: native default profile only, persistent Kanban initialized through qualified native APIs, `kanban.dispatch_in_gateway: false`, `kanban.auto_decompose: false`. Enable the Kanban toolset where qualified.

Explicit engineering selection provisions researcher/planner/builder/reviewer with real native descriptions and qualified role-appropriate toolsets. Dispatch/decomposition remain off. Never clone credentials implicitly; preserve existing profiles/history. The operator enables autonomous dispatch through native Hermes, not a RepoKit mode.

Install full-SHA upstream Superpowers via native Hermes plugin installation. Scanner safe may install; caution needs approval for exact findings/SHA; dangerous/failed/disabled scanner refuses. Never force trust. Verify newly loaded schema in a fresh native session.

Install/configure pinned upstream Nerve through its supported native plugin/catalog route. Hermes Kanban remains canonical lifecycle; Nerve provides supervision, definition-of-done, evidence and trajectory. Qualify enablement, per-profile configuration, completion behavior and restart persistence. No local `plugins/nerve` implementation or parallel task store.

Nerve's selected release determines supported Laya protocol, checkpoint and sidecar inputs. Use its native backend/settings (exact names qualified, not inferred from example names) and upstream packaging. Do not create a custom Laya API/server without a proven upstream incompatibility and separately reviewed minimal response. Keep torch/transformers in the sidecar, no host port, necessary cache beneath `.hermes/laya`. Qualify actual typed result/model identity, Nerve LOCAL_ONLY/self-hosted interpretation, restart and degradation on loss. Health/mocks are not inference evidence.

## Native OpenViking

Scaffold the official digest-pinned service, persistent `.hermes/openviking` → `/app/.openviking`, no published 1933 port. Use endpoint `http://openviking:1933`, storage workspace `/app/.openviking/data`, disable bundled bot, and Hermes `memory.provider: openviking` through qualified native configuration. No replacement provider, memory abstraction or secret collection.

Delegate embedding/VLM configuration to native `openviking-server init` and `doctor` inside the selected container, then Hermes `memory setup` or exact qualified configuration. Do not fabricate credentials/model names. Startup pending configuration is pending-setup, not a successful initialized service. Document how native setup can run before the sidecar is healthy without creating a second Hermes runtime.

Accept native sync, automatic extraction and provider egress as intended semantics. The former durable-only/privacy exclusion is **removed**; raw turn/tool synchronization alone does not make the selected integration unsupported. Document synchronized/extracted data and require operator-authorized credentials/model use for live tests. Stronger privacy modes require an explicit future request and upstream support.

Use per-repository service/data/network and appropriate native authentication/identities; test cross-repository denial, durable remember/restart/recall and truthful degradation. Preserve profile-local native memory without pretending it is a replica. Keep credentials out of receipts/logs.

## Same-card independent review

Use native Hermes request-review/request-changes/completion and Nerve supervision first. Acceptance must demonstrate distinct builder and reviewer run actors on the same card, with current candidate evidence and renewed review after changes. No speculative custom review-policy framework.

Only a demonstrated builder/reviewer actor-independence enforcement gap in selected Hermes+Nerve may justify a small standalone native policy plugin. It may use only native state, must fail closed for supervised engineering cards, remain independent of RepoKit and create no second approval database. Record the gap and qualification evidence before implementing it. Do not claim universal enforcement over native administrators or arbitrary same-UID code.

## Testing, release and dogfood

Every substantive Go package has package-local unit tests; ordinary tests are offline and use temporary files/injected runners/fake Docker. Run `go test ./...`, `go test -race ./...`, `go vet ./...` and gofmt checks. Integration/acceptance fixtures stay separate and explicitly gated; absent authority is not a passing release tier.

Cover collisions, aliases, dangling links, ownership, tracked state, interruptions, child-held locks, safe reruns, owner edits, missing/corrupt receipts, failed pull/scanner, shell metacharacters/newlines, TTY/non-TTY and Ctrl-C, stopped service, profile preservation and conservative defaults. Live tiers prove mounts, native setup/chat/args/plugins/profiles/restart; actual Nerve/Laya and OpenViking memory must be measured, not mocked. Enforce operator-specified time/download/spend/data limits.

Mandatory removal-first gate uses an unrelated disposable repository: bootstrap and native component setup; start normal Compose; verify; make RepoKit checkout/binary and optional receipt inaccessible to imports/subprocesses/mounts; change cwd; native launcher and raw Docker/Compose remain usable; restart; actual bounded builder→distinct reviewer same-card task; restart again; sessions/history/memory/Nerve/Laya state persist. PATH hiding alone is insufficient. Unsupported selected integration or absent live evidence prevents v1 completion.

After this gate, dogfood on RepoKit itself with bounded builder implementation, distinct review, tests and normal Git evidence. No improved binary self-apply or runtime-manager promotion.

Release single CGO-disabled binaries for Linux amd64/arm64 with checksums, qualified patched toolchain and reproducible inputs; no host Go needed. Future download helper only verifies/places binaries. Publishing a release is separate from producing local artifacts.

See the [implementation plan](../plans/2026-09-27-repokit-bootstrap.md), [current progress](../../../TODO.md) and source/runtime evidence under `docs/qualification`.

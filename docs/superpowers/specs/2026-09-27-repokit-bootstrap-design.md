# Hermes RepoKit repository bootstrap

Date: 2026-09-27. **Corrected draft; documentation only, awaiting review.**

Replaces the [team-runtime design](2026-09-27-repokit-team-runtime-design.md)/A–G plans. The [bootstrap plan](../plans/2026-09-27-repokit-bootstrap.md) retains **subagent-driven** execution after review; no code/runtime evidence exists. [Integration findings](../../research/2026-09-27-repokit-integration-findings.md) remain source evidence, not approved deployment pins. The [precursor review](../../research/2026-09-27-repokit-precursor-review.md) records adopted patterns and rejected manager/chat-supervisor behavior.

## 1. Product and ownership

RepoKit is an opinionated Hermes-on-Docker **repository bootstrapper**, not a runtime manager. It generates readable `.hermes/compose.yaml` and persistent `.hermes/`. Standard Compose owns services; native Hermes owns profiles, Kanban, plugins and execution. OpenViking is the selected external memory service. Laya is optional inference. Nerve is a separately installed native advisory plugin; Superpowers is a native plugin.

RepoKit is not resident: no daemon, scheduler, supervisor, reconciler, installer command proxy, runtime database or mandatory periodic invocation. Removing its checkout, binary and optional installer receipt must not affect restart or execution. Artifacts cannot bind/import/symlink/call RepoKit or Pi. Independently packaged Nerve/review-policy plugins and selected Laya sidecar artifacts may remain only as self-contained native artifacts, configured by Hermes/Compose without installer APIs or receipt reads. Nerve-derived observations are plugin state, not RepoKit runtime state.

**Implementation:** Go builds one `hermes-repokit` executable; Go/compiler needed only for development/build, not installation/runtime. First release targets Linux amd64/arm64 with local Docker; cross-compilation does not qualify other OSes. Bootstrap host prerequisites include Git and qualified Docker/Compose; the generated launcher needs only POSIX shell and Docker/Compose. No host Python/venv/pip/Node requirement. Templates embed in the CGO-disabled reproducible binary; default tests are offline Go. Native plugin/Laya Python stays inside independent runtime artifacts, not ported to Go. Compose/native config stays YAML, optional receipt JSON, launcher shell+Docker. Qualify toolchain/module pins before building; prefer checksummed standalone binary, not required curl-to-shell. Runtime tests require explicit tags and scoped authority.

## 2. Physical contract

One Hermes runtime and explicitly named Compose project per repository; OpenViking and optional Laya are supporting sidecars. Official layout stays `/workspace` and `/opt/data`, with `HERMES_HOME=/opt/data` and `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`. No custom HOME image, second Hermes for setup/probes, Docker socket, remote control endpoint or privileged workaround.

```text
<target>/.hermes/
  compose.yaml
  bin/hermes-<normalized-repo-name>  # standalone POSIX shell launcher
  config.yaml, profiles/, plugins/, memories/
  ... native sessions, authentication, Kanban and logical history ...
  openviking/ov.conf, openviking/data/
  nerve/                 # native plugin-derived state
  laya/                  # optional qualified cache/artifacts
  repokit-install.json   # optional nonsecret installer-only receipt
```

Compose uses immutable images/native plugin versions and relative mounts based on `.hermes/compose.yaml`: `..` → `/workspace`, `.` → `/opt/data`; never override project-directory. Human-readable YAML embeds a stable canonical-path-hashed project name. The container AND default host command are exactly `hermes-<normalized-repo-name>`: `polymarket-mega-bot` → `hermes-polymarket-mega-bot`. Normalize the full basename to lowercase ASCII, replace nonalphanumeric runs with hyphens and trim hyphens; no hash/truncation/suffix in the container/command. Reject empty, invalid or unsupported-length names; identical basenames intentionally collide and must refuse, including stopped containers. Never silently switch context/adopt.

From the repo, use `docker --context CONTEXT compose --env-file /dev/null -f .hermes/compose.yaml up -d` (also down/restart/logs/pull), with selector environment cleared as below. This explicitly supersedes root autodiscovery: no root forwarding file/duplicate Compose. Existing root Compose variants/overrides still cause conservative refusal. Native `.hermes/.env` is NOT Compose interpolation input. Digest-pinned pull retrieves selected artifacts, not latest. No published sidecar ports, bundled VikingBot or second board/provider. Explicit remote-provider egress remains separately authorized.

Keep `.hermes` ignored and unindexed, private modes, canonical-path/link/ownership checks and no recursive source chown. Unknown native auth/history/config files survive. Both `/opt/data` and `/workspace/.hermes` expose the home: safe roots are not security boundaries. Root OpenViking authority must be demonstrably inaccessible to worker surfaces under actual UIDs and both aliases, not merely replaced with a client user key.

## 3. Native host launcher and installer conveniences

Generate executable owner-only `.hermes/bin/hermes-<normalized-name>` as standalone POSIX shell plus Docker only: no host Hermes/Python/Node/Pi/RepoKit dependency. If authorized, install a same-name PATH link after preflighting source/destination ownership, all PATH/alias/function collisions and dangling symlinks; identical verified link is a no-op. Never overwrite, suffix, edit rc/PATH automatically or chmod existing host paths. Without authorization use the absolute repo-local launcher and report PATH activation pending.

No arguments opens native chat. Qualify bare Hermes behavior; only if required by the selected native version map zero arguments to its explicit native chat command. ALL explicit arguments pass unchanged (`setup`, `kanban list`, `plugins list`, `profile list`, `gateway status` are forwarding examples, not assertions of supported aliases). Preserve native errors; no validation/reinterpretation/second hierarchy or noarg pipe-help fallback. Native Hermes owns session coordination; report unsupported guarantees, never import the precursor chat supervisor.

Launcher shell-quotes literals, clears `COMPOSE_FILE`, `COMPOSE_PROJECT_NAME`, `COMPOSE_PROFILES`, `COMPOSE_ENV_FILES`, uses explicit qualified Docker context/absolute Compose path and `--env-file /dev/null`, then `exec` with `"$@"`, no eval/container shell. Allocate TTY only when stdin AND stdout are terminals; use `-T` otherwise. Preserve inherited streams, exit status and Ctrl-C. Qualify exec-time UID/GID and Unix HOME against the official shim; no guessed 1000, forced HOME or service-user override breaking root bootstrap. Workdir is `/workspace`, HERMES_HOME `/opt/data`. No launcher auto-start/pull/restart, receipt check, health gate or session manager.

Installer CLI remains four conveniences:

- `plan`: bounded read-only inspection of actual target files/service identity and proposed bootstrap actions.
- `install`: explicitly requested bootstrap, immutable artifact/native plugin admission and safe initialization. A conservative rerun may repair proven owned missing artifacts; owner native config/Compose edits remain authoritative, never overwritten from a stale receipt. No dispatch, implicit credentials, downloads of weights or inference.
- `setup`: only direct delegation to the generated launcher's `setup`, never a separate wizard. Lead handoff with `hermes-<repo> setup`; native `hermes setup` runs in the existing container with private inherited I/O. If stopped, operator uses the explicit Compose command above with `up -d hermes`; later full up is idempotent. No secret capture/args/logs, cloned prompts or specialist staging schema. Direct Docker/Compose native invocation remains valid.
- `verify`: bounded read-only observations of actual artifacts, loaded components and freshness; unsupported probes report unknown. No initialization, migration, auth refresh, canary write or inference.

Status/repair/update helpers are deferred. There is no required mode/activate/apply/uninstall. Ordinary Compose down **without `-v`** preserves state. Owner edits native configuration and administers Hermes directly.

Receipt loss/corruption cannot break runtime. Rerun must independently prove ownership before writing, never adopt/reset foreign or ambiguous state. Transient installer locks, atomic publication and failure journals may protect bootstrap only; runtime never depends on them. Surviving lock-holding children defeat naive stale-PID cleanup. Interrupted install, failed pulls/scans and collisions preserve existing data and report partial effects without releasing dispatch. No secret hashes are published.

## 4. Native profiles, execution and review

Fresh installation has native root default only, SAFE/native Kanban dispatch false and `auto_decompose:false`. An explicitly requested bootstrap/rerun engineering preset may create researcher/planner/builder/reviewer with descriptions, without cloning credentials, and also leaves dispatch false. Native profile administration is equally valid. Qualify exact native subcommand spelling; do not invent `profile`/`profiles` aliases.

Document native operator admission recipes: managed concurrency 1, then separately bounded fleet. Native recipes, not RepoKit modes/TeamAdmission receipts. Safe native downgrade suspends new claims, preserves profiles/history and does not manufacture task completion or cancel existing runs.

Default coordinates; researcher investigates; planner plans; builder implements; reviewer independently inspects/tests. Effective schemas and scoped execution enforce supported role guarantees, not prompts. Qualify a self-contained fail-closed native policy path: lifecycle-tool auto-append, missing policy, caught hook errors, precheck/transition races, CLI/Python/SQLite bypass and protected-state writes must not bypass supported worker enforcement. Repository tests are not trusted. Engineer-qualify role runners within the topology; invent no controller. Unsupported surfaces withhold affected dispatch, not independent infrastructure. Native host-admin APIs remain usable; this is not universal adversarial enforcement against administrators/compromised plugins.

Use native request-review/request-changes/completion. Require exact `builderProfile=builder`, `reviewerProfile=reviewer`, distinct existing runs and actors on the **same card**, current candidate/evidence, and renewed review after changes. Builder cannot self-complete; default cannot approve for reviewer. Native history is authority; no parallel approval database. No `goal_mode` default until independently qualified.

## 5. Memory, advisory plugins and qualification

Native OpenViking uses `memory.provider: openviking`, endpoint `http://openviking:1933`, project-bound user key and shared authorized profile identity, peer/agent unset. Verify environment → linked ovcli → YAML precedence. Keep profile-local built-in `memories/`; they are not an external replica. Per-repo service/network/data/keys and negative cross-repo access tests are required. Disable the bundled bot. Deliberate knowledge excludes raw logs/secrets; curated ingestion is off by default.

**At the research pin, durable-only privacy is unsupported:** native background turn/tool upload has no verified opt-out; disabling extraction or hiding tools does not prevent disclosure. No invented toggle, filter or alternate provider. Independent safe scaffold/service observation can proceed, but incompatible memory-connected sessions cannot. Refuse enabling an incompatible selected integration before writing or starting active native provider configuration; report unsupported and leave existing state unchanged. A separately requested default-only/independent-component scaffold may proceed without silently selecting another provider. A compatible native build must itself satisfy privacy after ordinary restart with RepoKit absent; neither pre-tool guards, installer gates nor verify can prevent background sync. A compatible upstream enhancement outside this repository requires separate authorization. Memory outage degrades authorized workflows, not a board kill switch; no guaranteed replay/extraction/shared recall claims.

Nerve uses native hooks/tools, bounded nonblocking projection and independently owned native plugin consumer/state. Hooks may run under dispatch locks: no network, inference, subprocess startup or contention waits there. Bound ingress, retention, deduplication and read-only native-history reconciliation; missing/reordered observations remain unknown. Never transition cards or synchronously depend on memory. Its exact installed version and closure are separately admitted.

Optional Laya stays outside Hermes dependencies. Research release/source both report 0.3.20 but differ: release lacks immutable revision loading/bounded admission; newer stock server does not wire pin-aware configuration. Qualify a self-contained sidecar package with locked image/dependencies/checkpoint/tokenizer/config/weight hashes and bounded compute admission; no fabricated image or current support. Fixed typed rubric, sanitized bounded payloads, deadlines, finite typed answers and exact model provenance are required. Missing/timeout/malformed/low-confidence results mean UNKNOWN/WATCH, never board stop, second provider or routing profiles. No downloads/model calls are authorized now.

Superpowers uses exact approved full SHA and native scanner admission: safe allowed, caution requires current exact findings approval, dangerous/disabled/failed scanner refused. Never force trust or silently upgrade.

## 6. Release gate, not current evidence

On authorized disposable Linux amd64/arm64 hosts without Go/Python/Node/Pi (Git, shell and Docker/Compose remain), qualify the binary and bootstrap separate fresh targets, not the checkout. Use native setup and ordinary Compose up; verify selected services/plugins/profiles. Remove installer checkout/binary and optional receipt, making all installer artifacts inaccessible through subprocess/import/mount paths—not merely hidden from PATH. Only generated standalone launcher/Compose/native state and independent plugins/sidecars remain. From unrelated cwd invoke the launcher with no args and explicit native commands; prove raw Docker/Compose native commands also work. Restart using explicit Compose selectors; perform native administration without installer wrappers and an actual bounded task with independent same-card completion. Restart again and prove addressable sessions, native logical history, actual selected memory write/recall, Nerve history and selected Laya cache persistence. Protect credentials with private boolean comparisons, never published secret hashes.

Optional useful dogfood lets this independent team improve RepoKit source. No mandatory improved-CLI self-apply, promotion or host-release-root machinery; independence must hold before/after.

Separate offline mocks, credentialless authorized runtime, and scoped actual inference evidence. Full release cannot pass with unsupported selected memory or simulated completion. Actual inference needs explicit data/egress/retention/provider authority and ceilings for runs, correction cycles, time, tokens/spend (including extraction), downloads and compute. Unbounded fixtures remain blocked. No runtime evidence claimed.

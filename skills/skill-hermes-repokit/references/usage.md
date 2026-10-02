# Use and verify an installed repository

## Native commands

Use the host command, or the absolute `.hermes/bin/hermes-<repo>` launcher when
it is not on PATH. For a repository whose launcher is `hermes-design-notes`:

```sh
hermes-design-notes                     # native chat as default
hermes-design-notes -p researcher       # chat with another profile
hermes-design-notes profile list
hermes-design-notes kanban list
hermes-design-notes gateway status
hermes-design-notes plugins list
```

With no arguments the launcher selects `default`; arguments pass through to
native Hermes unchanged. `hermes-design-notes setup` is native setup only — use
`repokit setup` for team reconciliation. Native syntax is
version-sensitive: check scoped `--help` before dispatch, profile or config
changes.

## Working with the team

Talk to `default`. It owns the task graph and coordinates `researcher`,
`planner`, `executor`, `tester`, `reviewer` and `steward`; steward owns team
lifecycle (it prefers a task-scoped skill to a new specialist, and deleting a
profile needs explicit approval). Role boundaries in SOULs are advisory, not
operating-system isolation.

Owner edits to a roster profile's SOUL or description are preserved and reported
as `customized`. To deliberately return one profile to RepoKit's baseline, preview
with `repokit plan --reset-profile <role>` and, with the owner's consent, apply
with `repokit install --reset-profile <role>` (prior files are backed up).
Owner-created profiles are never reset.

## Project skills

Skills every profile should share belong in the repository's own
`.agents/skills/<name>/SKILL.md`; RepoKit trusts `/workspace` for native
project-skill discovery in all seven profiles, and Hermes still scans them and
honors disabled skills. A profile-specific skill goes under
`/opt/data/profiles/<profile>/skills/` (steward's job). Start a fresh
conversation after adding skills so the index refreshes.

Inside the container the repository's `.hermes` is masked by an empty read-only
mount at `/workspace/.hermes`; private state is reachable only at `/opt/data`.
An empty `/workspace/.hermes` is expected, not data loss.

Substantive work runs executor → tester → reviewer on the **same card**.
Reviewer-requested changes go back to tester, which relays them to executor; the
fix passes tester again. `done` alone does not prove independent review: inspect
card history for distinct executor, tester and reviewer runs. Never fabricate a
rejection to exercise the loop.

Before promising progress, check configured and live dispatch separately
(`hermes-<repo> config get kanban --json`, `gateway status`, `kanban list`). If
activation is incomplete, rerun `repokit setup`; do not bypass it
with one-shot dispatch, and preserve queued work. Do not commit or push work the
team produced unless asked.

Configured channels (Telegram and others) route to `default` with the same core
development and Kanban tools as the CLI. After reconciliation, start a fresh
conversation (`/new`) so the coordinator reloads its tools and skills. Actual
task/result delivery on the originating channel still needs a live test.

## Runtime recovery

Read the launcher for its captured Docker context and absolute Compose file;
refuse unknown routing. Then use ordinary Compose with those selectors:

```sh
unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES
docker [--context NAME] compose --env-file /dev/null -f /abs/repo/.hermes/compose.yaml ps
docker [--context NAME] compose --env-file /dev/null -f /abs/repo/.hermes/compose.yaml up -d
docker [--context NAME] compose --env-file /dev/null -f /abs/repo/.hermes/compose.yaml up -d --force-recreate hermes
docker [--context NAME] compose --env-file /dev/null -f /abs/repo/.hermes/compose.yaml logs --tail 200 hermes
```

Start a stopped service in its existing project; recreate the single Hermes
service from the generated pins when replacing the image. `down` preserves
mounted state — never use `down -v`, delete volumes, profiles, credentials or
board state as a shortcut, and never act on containers outside this project.
If `setup` or reconciliation is needed after the bootstrap was removed,
reinstall it as in [installation](installation.md#2-obtain-the-bootstrap-cli);
runtime state is untouched.

Memory is user-managed: it lives in the owner's own configuration and mounts,
never gates engineering work, and is no reason to restart the runtime.

## Troubleshooting

Start from `repokit verify` and fix the components named in `CORE_READY`'s detail;
when a probe's detail prints a command, prefer that one.

| Symptom or probe | Likely cause | Next action |
| --- | --- | --- |
| `compose`, `config`, `kanban` `pending-setup` | Install not finished | `repokit install` |
| `profile:<role>` degraded or missing | Team not reconciled, or drift | `repokit setup --team` (refuses while a card runs) |
| `profile:<role>` `customized` | Owner edit | Leave it; reset only on owner request (see above) |
| `kanban:dispatch-*` inactive/degraded | Setup incomplete or a card was running | `repokit setup`; never dispatch by hand |
| `gateway` degraded, cards stay `ready` | Gateway stopped or crashed | `hermes-<repo> gateway status`, Compose `logs`, then `repokit setup` |
| `channel:<name>` degraded, bot silent or tool-less | Default lacks the Kanban tool on that platform, or stale session | The native command in the probe detail (`hermes-<repo> -p default tools enable kanban --platform <name>`) or `repokit setup --team`, then `/new` in that chat |
| `development:<tool>` degraded | Image predates a new manifest, or edited recipe | `repokit install`; an edited recipe is refused — report to owner |
| `python-imports` degraded | Root module shadows a Hermes module (upstream Hermes defect) | Report; do not rename the owner's modules or patch Hermes |
| `git` degraded | `.hermes` tracked or not ignored | `repokit install` restores the local exclude; tracked files need the owner to untrack them |
| `selinux`/`access` degraded | Mounts lack private relabeling | `repokit install` (see [installation](installation.md#selinux-bind-mounts)) |
| `review:evidence` unqualified | No reviewed card yet | `repokit verify --dispatch-check` (paid) or real reviewed work |

A row that names Hermes behavior (model errors, provider auth, channel adapter
outages) is a Hermes problem: use native `hermes-<repo>` commands and logs, not
RepoKit changes.

## Readiness report

`repokit verify` prints a JSON array:

```json
[
  {"component": "CORE_READY", "status": "unqualified", "detail": "configured and running; no automatic executor/tester/reviewer loop observed yet"},
  {"component": "...", "status": "healthy", "detail": "..."}
]
```

- `CORE_READY: healthy` — every core probe healthy and a same-card
  executor→tester→reviewer completion observed.
- `CORE_READY: unqualified` — nothing broken; the review loop has not been seen.
  Exit status 0. Run `verify --dispatch-check` or real reviewed work next.
- `CORE_READY: degraded` — the detail names the failing components. Exit 1.
  Fix those before anything else.

`verify` reads files, metadata and bounded Docker/native observations only. It
never runs inference, dispatches work, migrates state or probes memory; do not
add any of those to make it pass. Summarize for the owner like this:

```text
bootstrap            installed (repokit, ~/.local/bin)
host command         hermes-my-project on PATH
core team            7 profiles healthy
dispatch             policy on; dispatch-check PASS
development runtime  go 1.26 in /workspace
channels             telegram → default, tools healthy
readiness            CORE_READY unqualified — no reviewed card yet
```

## Live acceptance gates

Passive checks cannot establish these; run them only within an authorized live
test and report each as PASS, FAIL, BLOCKED or NOT TESTED with evidence:

| Gate | Required evidence |
| --- | --- |
| Host/default chat | Fresh shell, unrelated cwd, default profile, `/workspace` identity, real chat |
| Team | Seven resolving profiles with distinct descriptions and SOULs; owner edits preserved |
| Automatic dispatch | Gateway claims a no-write researcher card with no manual dispatch |
| Development runtime | Target's manifest requirements match the in-container toolchain and mounts |
| Channels | Adapter routes to default with CLI core parity; task and result delivered on that channel |
| Same-card review | Real artifact with distinct executor, tester and reviewer runs |
| Runtime independence | After removing a test-owned bootstrap, launcher and raw Compose restart keep profiles and board |

Never delete a system-installed RepoKit binary to manufacture removal evidence.
Finally compare the repository with the baseline, run `git diff --check`, and
state exactly what changed, including host links and `.hermes` files. Partial
bootstrap success is not a passed full-stack acceptance.

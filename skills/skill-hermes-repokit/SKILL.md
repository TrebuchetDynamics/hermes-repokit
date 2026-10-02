---
name: skill-hermes-repokit
description: Use when installing, resuming, verifying or operating Hermes RepoKit in a repository — the repokit bootstrap CLI, the generated hermes-<repo> launcher, the seven-profile team, Kanban dispatch, the development runtime and channel parity. Not for unrelated Hermes installations, memory-provider setup, or developing RepoKit itself.
---

# Hermes RepoKit in repositories

RepoKit is a short-lived Go bootstrapper with four commands: `plan`, `install`,
`setup` and `verify`. It prepares one Docker Compose Hermes container per
repository, then gets out of the way: after bootstrap, the generated
`hermes-<repo>` launcher and ordinary Compose own the runtime, and RepoKit can be
removed.

**RepoKit prepares; Hermes operates; Docker contains; Kanban coordinates;
memory stays user-managed; Git stays under owner control.**

## Route the request

| Request | Read |
| --- | --- |
| Install or resume setup of a deployment | [installation](references/installation.md) |
| Chat, profiles, Kanban, recovery, acceptance checks | [usage](references/usage.md) |
| "Is it working?" / status only | Stay observational: `repokit verify` plus [usage](references/usage.md#readiness-report) |

Reading or installing this skill does not authorize deploying Hermes. An explicit
installation request authorizes ordinary non-destructive bootstrap and starting
the generated service for that one target repository. Private credential setup,
destructive cleanup, legacy-state deletion, source changes to RepoKit, commit and
push are each separately scoped.

## Ground rules

1. **Target vs. source.** The target repository is the one being prepared; a
   RepoKit source checkout is separate. Every RepoKit command runs from the
   **target root** — the CLI uses its current directory and has no path argument.
   Use the user's chosen repository, otherwise `git rev-parse --show-toplevel`.
2. **Baseline first.** Record branch, HEAD, dirty/untracked paths and a status
   hash before any mutation. Never reset, stash, clean, commit or discard owner work.
3. **Check the CLI you have.** Run `repokit --help` and follow the documentation
   of that revision (`README.md`, `docs/bootstrap-quickstart.md`). There is no
   `--version`, no release binary and no `migrate`, `chat` or `run` subcommand
   — do not invent them.
4. **Docker scope.** Touch only the target's own Compose project, through the
   exact command `install` printed (explicit context, absolute file,
   `--env-file /dev/null`). Never `docker kill`, `rm` or `prune` by name pattern,
   never `down -v`, and never touch other repositories' `hermes-*` containers.
   When other agents share the host, announce Docker test runs first.
5. **Secrets stay private.** `setup` runs in the owner's own terminal. Never
   capture credentials, drive the wizard through a PTY, borrow unrelated host
   auth, or pick providers/models on the owner's behalf.
6. **No bypasses.** Do not hand-edit `.hermes/compose.yaml`, managed SOULs,
   platform tool checkboxes or other `.hermes` state to make a check pass. Durable
   fixes belong in RepoKit source or native Hermes configuration.
7. **Memory is user-managed.** RepoKit neither configures nor verifies a memory
   provider, and memory never gates core readiness. Do not set one up as part of
   this workflow; report it as operator-owned.
8. **Autonomy is the owner's answer.** Before it creates a new team, `setup`
   states that workers run without approval prompts and asks to confirm. Let the
   owner answer in their own terminal; never answer it for them.

## Command names

| Intent | Command |
| --- | --- |
| Inspect a repository (writes nothing) | `repokit plan` |
| Build and start the deployment, host link, Kanban; converge on reruns | `repokit install` |
| Private provider setup, then team, dispatch and canary → `RepoKit ready.` | `repokit setup` |
| Recovery: reconcile the team without the wizard | `repokit setup --team` |
| Observe readiness | `repokit verify` |
| Paid proof of automatic dispatch (one no-write card) | `repokit verify --dispatch-check` |
| Add an isolated Docker test daemon | `repokit plan --docker-tests`, `repokit install --docker-tests` |
| Native default chat | `hermes-<repo>` (no arguments) |
| Other native Hermes commands | `hermes-<repo> kanban list`, `profile list`, `gateway status`, … |
| Stop or start the deployment (state kept) | `repokit stop`, `repokit start` |

`repokit` and `hermes-repokit` are the same bootstrap binary. Prefer `repokit`:
in a repository whose name normalizes to `repokit`, the generated launcher owns
`hermes-repokit`. `hermes-<repo> setup` is **native** Hermes setup only; it does
not reconcile the team.

## Lifecycle at a glance

```text
repokit plan                         inspect: names, context, collisions
repokit install                      publish .hermes/, launcher, ~/.local/bin link; build + start
                                     hermes-<repo>; native Kanban init (dispatch off)
repokit setup        (owner, private) provider/model → seven profiles → dispatch policy → gateway start
repokit verify                       CORE_READY report
repokit verify --dispatch-check      (explicit, paid) gateway claims a researcher card
real task                            executor → tester → reviewer on the same card
repokit stop | start                 stop / start the deployment; all state kept
repokit remove       (owner, typed)  delete the deployment and .hermes
```

## Deployment contract

- One container per repository, `hermes-<repo>` as printed by `plan`. Repository
  at `/workspace`; private `<repo>/.hermes` at `/opt/data` (`HERMES_HOME`).
- The owner's own Compose files and services coexist untouched; RepoKit uses
  `.hermes/compose.yaml` and its own project namespace.
- `install` links `~/.local/bin/hermes-<repo>` to the generated launcher,
  preserving conflicts and warning about a missing PATH entry. It never creates
  aliases or edits shell startup files.
- On SELinux hosts `install` adds private `Z` relabeling for the two repository
  mounts. Never disable SELinux or change global policy.
- Coding readiness needs the target's toolchain inside `/workspace` (the pinned
  development image); terminal/file tools alone are not enough.
- Seven permanent profiles: `default` (coordinator, the user's entry point),
  `researcher`, `planner`, `executor`, `tester`, `reviewer`, `steward` (team
  lifecycle). Configured channels such as Telegram route to `default` with the
  same core development and Kanban tools as the CLI.
- Kanban is canonical. Setup enables the single native gateway dispatcher
  (automatic review, concurrency one, no auto-decomposition) with
  `hermes config set` and one gateway restart. It refuses while a card is running
  and preserves an owner-changed policy. RepoKit never runs a worker itself.

## Readiness

`verify` prints a JSON array: `CORE_READY` first, then one probe per component.
Memory is not reported; it is a Hermes feature RepoKit does not own. `CORE_READY` is:

- `healthy` — configuration, runtime, toolchain, dispatch policy, gateway and
  channel Kanban tools are healthy **and** native card history shows same-card
  executor→tester→reviewer completion.
- `unqualified` — configured, but that review loop has not been observed yet.
- `degraded` — a core component is broken; the only nonzero exit.

Container health is not readiness, and passive `verify` proves no model work,
channel delivery, reviewer independence or removal-first acceptance. Report
bootstrap, host command, default chat, profiles, dispatch, toolchain and
channels separately, each with evidence and the next action.

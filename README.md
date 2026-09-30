<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="Hermes RepoKit — bootstrap one repository-specific Docker Hermes container with seven native profiles: default, researcher, planner, executor, tester, reviewer and steward.">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/runtime-Docker%20Compose-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white" alt="Docker Compose runtime">
  <img src="https://img.shields.io/badge/profiles-7-F0B24A?style=flat-square" alt="Seven native profiles">
  <img src="https://img.shields.io/badge/status-pre--v1-D08A2E?style=flat-square" alt="Status: pre-v1">
</p>

**Hermes RepoKit** prepares a repository for native Hermes work without becoming
part of the runtime. It handles the delicate host/bootstrap work—safe private
state, ownership and symlink checks, deterministic Compose, mounts and
installation—then hands control to Hermes for profiles, models, channels, Kanban
and agent execution. The result is one repository-specific container, seven native
profiles and a shared board. Memory is user-managed: the operator configures and
operates whatever provider the repository needs; RepoKit does not. After
bootstrap, ordinary Compose and the generated launcher work without RepoKit
installed.

The [v1 architecture rule](docs/architecture.md) is: **RepoKit configures
capabilities; it does not implement them.** Production bootstrap uses Go and
public Hermes interfaces. A capability that those interfaces cannot safely
establish remains unqualified.

It is for repository owners who want a persistent, role-based team to work on
code or other repository artifacts while keeping deployment state isolated from
the application's own Compose stack. Go is the bootstrap implementation choice;
the generated runtime is a pinned Hermes development image, not a Go service or
a RepoKit daemon.

> **Status · pre-v1.** Live dogfood proved the core loop: a Telegram request to
> `default` was executed by `executor`, approved by `reviewer` on the same card and
> reported back to the same chat, and the team kept working after `docker restart`.
> The same loop passed on a fresh clone of an unrelated Go repository. Memory is
> user-managed and outside RepoKit's scope. See
> [remaining gates](#running-and-remaining-gates).

## Quickstart

Install the bootstrap CLI from the latest release, [v0.2.3](https://github.com/TrebuchetDynamics/hermes-repokit/releases/tag/v0.2.3), with Go 1.26+:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.2.3/install.sh | REPOKIT_REF=v0.2.3 sh
```

The script builds that release's source; it downloads no release binary. This
README describes `main`, which can be ahead of the release; the release's own
[README](https://github.com/TrebuchetDynamics/hermes-repokit/blob/v0.2.3/README.md)
matches what it installs. To try unreleased changes, build `main` instead:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/main/install.sh | sh
```

With the `main` script, `REPOKIT_REF` selects any branch, tag or commit to build,
for example `… | REPOKIT_REF=v0.2.3 sh`. The script
publishes one binary under `hermes-repokit` and the short `repokit` alias,
updates a command it previously installed, and preserves any other existing
command. A generated `hermes-<repo>` host launcher can already own the
`hermes-repokit` name when a repository's name normalizes to `repokit`; the
script reports that name with a relocation command instead of replacing the
launcher. It also reports if `~/.local/bin` is not on PATH or another command
shadows the bootstrap. The commands below use `repokit`, which is always the
bootstrap. The installed CLI needs no host Go, Python, Node or Hermes. To
prepare a repository, you also need Docker Compose, Git and a POSIX shell. Run
these commands from the target repository:

```sh
cd my-project
repokit install           # build and start the container, prepare Kanban
repokit setup             # Hermes's private provider setup, then team, dispatch and canary
hermes-my-project         # talk to your team
```

That is the whole first-time path. `install` builds the development image,
starts the container, installs the host command and prepares everything that
needs no secrets. `setup` hands your terminal to Hermes's own private setup
(RepoKit never sees credentials), then creates the seven profiles, turns on
automatic dispatch and proves it with one canary card before printing
`RepoKit ready.` Running `setup` again skips the private wizard once default
has a model.

Everyday commands:

```sh
repokit install           # converge to the current baseline; owner customizations are kept
repokit verify            # observational CORE_READY report
repokit stop              # stop the deployment; all state is kept
repokit start             # start it again; Hermes restarts a gateway that was running
repokit plan              # preview what install would change; writes nothing
```

Recovery only: `repokit setup --team` reconciles the team without the private
wizard, and `setup --no-canary` skips the canary card (no model call).
The launcher forwards native arguments
unchanged (`.hermes/bin/hermes-my-project kanban list`). After install, ordinary
Docker Compose owns the runtime using the context recorded in the launcher — see
[runtime management](docs/bootstrap-quickstart.md#ordinary-runtime-management).
Install safely creates `~/.local/bin/hermes-<repo>` as a symlink to the generated
launcher, reuses matching links, and preserves conflicting entries. It reports
unusable host directories or missing PATH entries with an absolute command to use.
It does not create shell aliases or edit shell startup files; see the
[setup and recovery guide](docs/bootstrap-quickstart.md#host-command).

## The team

| Profile | Responsibility |
| --- | --- |
| `default` | Assistant, coordinator and decision owner |
| `researcher` | Resolve unknowns and gather evidence |
| `planner` | Define a bounded execution contract |
| `executor` | Produce the requested artifact |
| `tester` | Prove the behavior without modifying the repository |
| `reviewer` | Decide whether the verified change is accepted |
| `steward` | Maintain profile identities and the team lifecycle |

Substantive work passes two review stages on the same card:
executor → tester → reviewer. Every revision goes back through tester.

Steward prefers a task-scoped skill before creating a specialist, retirement
preserves history, and deletion needs explicit approval. See the
[team model](docs/team-model.md).

Profiles are persistent identities, not necessarily running workers. The native
gateway dispatcher is off on a fresh install; successful setup enables the
single default dispatcher through native configuration and one gateway restart.
RepoKit does not run as a supervisor or mediate ongoing agent work.

## What it leaves behind

<p align="center">
  <img src="./assets/readme/deployment.svg" width="100%" alt="RepoKit writes a private .hermes directory and one Compose-owned Hermes container running an s6 supervisor, a default gateway, seven profiles and a shared Kanban board.">
</p>

RepoKit writes a private `.hermes/` directory (Compose file, `bin/hermes-<repo>`
launcher, config, pinned development image and a `.gitignore`) and one
Compose-owned container. It mounts the repository at `/workspace` and all native
state at `/opt/data`. Memory is user-managed and is not configured by RepoKit.
Credentials stay in private setup,
and optional `install --docker-tests` adds an isolated acceptance daemon without
mounting the host Docker socket. On SELinux-enabled Linux hosts it requests
Docker's private `Z` relabeling for the repository and `.hermes` mounts as normal
installation compatibility; SELinux policy and unrelated host paths are never
changed. See
[development runtime](docs/qualification/development-runtime.md).

Existing repository Compose files and services stay in place. RepoKit uses its
own `.hermes/compose.yaml` and project namespace, with explicit file/context routing.

## Removing a deployment

```sh
cd my-project
repokit remove            # deletes the deployment and .hermes after typed confirmation
```

`remove` only acts on a deployment it can prove it generated, and asks you to type
the repository name in an interactive terminal. It then removes RepoKit's Compose
project (container, network, docker-test volumes), the generated image, the
`~/.local/bin/hermes-<repo>` symlink it created, the installer lock and its
exclude line, and **the private `.hermes` state: profiles, Kanban board, provider
logins, messaging tokens, sessions and memory.** That cannot be undone. Repository
files and Git history are not touched. If `.hermes` is already gone, `remove` clears only the
container, volumes, network, image and host link left by this repository's
earlier install, so `install` can start over.

## Running and remaining gates

Fresh installs keep dispatch off. After the seven profiles reconcile, setup writes
the native Kanban policy on `default` (automatic review, concurrency one, no
auto-decomposition, seven-profile allowlist) with `hermes config set` and restarts
the gateway once. It refuses while a card is running, preserves an owner-changed
policy and never claims a worker ran. Memory is user-managed and never blocks
core work.

`verify` is observational and reports `CORE_READY`: whether RepoKit produced the
team it promises. It does not report Hermes features RepoKit does not own, such
as memory. Core is `healthy` only when configuration, runtime, worker-shell
toolchain, dispatch policy, gateway and per-channel Kanban tools are healthy and native
card history shows same-card executor→tester→reviewer completion; otherwise
`unqualified` (not yet exercised) or `degraded`. It exits 0 unless core is
degraded. `verify --dispatch-check` is the explicit, paid proof: one no-write
researcher card must be claimed and completed by the gateway without manual
dispatch.

| Area | Remaining evidence |
| --- | --- |
| Team work | Reviewer request-changes correction cycle |
| Runtime independence | Removal-first acceptance with real work |
| Optional plugins | Native scanner admission and owner configuration |

Credential-free fixtures demonstrated native profiles, Kanban persistence and
launcher/Compose operation after removing a copied installer. Live dogfood also
completed a gateway-spawned researcher canary and the original queued research
task; see the [dogfood record](docs/qualification/live-dogfood-2026-09-28.md),
[runtime observations](docs/qualification/runtime-observations.md) and
[implementation status](docs/implementation-progress.md). The recorded Superpowers
candidate was refused with 229 CAUTION findings; its
[scanner report](docs/qualification/superpowers-8ca22dba-scan.txt) remains an explicit
admission gate.

## Agent skill

[skill-hermes-repokit](skills/skill-hermes-repokit/SKILL.md) teaches a coding
agent to install, resume, verify and operate RepoKit safely: it keeps the target
and source apart, records a Git baseline, uses only the commands `install`
prints, leaves credentials to your private terminal and reports each component
separately. Installing the skill does not start a deployment.

Install it for your agent (the skill is the `skills/skill-hermes-repokit` folder):

```sh
# Claude Code
mkdir -p ~/.claude/skills && curl -fsSL https://github.com/TrebuchetDynamics/hermes-repokit/archive/refs/heads/main.tar.gz \
  | tar -xz -C ~/.claude/skills --strip-components=2 hermes-repokit-main/skills/skill-hermes-repokit

# Codex, Pi and OpenCode
mkdir -p ~/.agents/skills && curl -fsSL https://github.com/TrebuchetDynamics/hermes-repokit/archive/refs/heads/main.tar.gz \
  | tar -xz -C ~/.agents/skills --strip-components=2 hermes-repokit-main/skills/skill-hermes-repokit
```

Rerun the same command to update it. Then, from the repository you want to
prepare, give your agent a prompt like this:

```text
Install the Hermes RepoKit agent skill if it is not already available: download
https://github.com/TrebuchetDynamics/hermes-repokit/archive/refs/heads/main.tar.gz
and extract only skills/skill-hermes-repokit into your skills directory
(~/.claude/skills for Claude Code, ~/.agents/skills for Codex, Pi or OpenCode).
Load the skill, then use it to set up Hermes RepoKit for this repository:
record a Git baseline, install the repokit CLI if missing, run plan, then run
install (it builds and starts the runtime itself). Stop there and give me the
exact `repokit setup` command to run in my
own terminal — never ask for or handle my credentials. After I confirm, run
repokit verify and report bootstrap, host command, team, dispatch, development
runtime and channels separately. Don't run --dispatch-check, commit, push or
delete anything without asking me.
```

For an existing deployment, a shorter prompt is enough: *"Use the
skill-hermes-repokit skill to check this repository's Hermes deployment and tell
me what is not ready."*

## Troubleshooting and deeper guides

- `setup` starts the deployment itself. If Docker cannot build or start it,
  the failure line names the Compose command to retry; fix Docker, then rerun
  `setup`.
- `setup` ended without `RepoKit ready.`? The last line says why (for example a
  failed canary); `repokit verify` reports what is not ready.
- Missing `hermes-<repo>` on `PATH`? Use `.hermes/bin/hermes-<repo>` directly;
  RepoKit does not edit shell startup files. See [host command and recovery](docs/bootstrap-quickstart.md#host-command).
- Memory is user-managed and separate from core team readiness; RepoKit does
  not configure or verify a provider.
- `verify` is passive and cannot qualify model work, memory recall or review.
  See [qualification boundaries](docs/bootstrap-quickstart.md#qualification-boundaries)
  and the [remaining release gates](TODO.md).

For contributors, see [build details](docs/build.md), [development runtime](docs/qualification/development-runtime.md),
[team behavior](docs/team-model.md), [implementation status](docs/implementation-progress.md)
and [remaining work](TODO.md).

## Development

Go 1.26+; host builds target Linux amd64 and arm64, the development image Linux
amd64. End-user binaries need Docker Compose, Git and a POSIX shell.
The bootstrap also builds from source with `CGO_ENABLED=0 go build ./cmd/hermes-repokit`;
no host C compiler is needed. The contributor race checks below require a C compiler.

```sh
go test ./...
go test -race ./...
go vet ./...
gofmt -l cmd internal tests
```

Ordinary tests are offline; Docker acceptance is opt-in:

```sh
REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerFoundation -v
```

The full offline and race suites, including the Unix-socket fixtures, pass.
Memory behavior is user-managed; independent review and channel round-trip
acceptance remain open. See the
[design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md),
[plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md) and
[remaining work](TODO.md).

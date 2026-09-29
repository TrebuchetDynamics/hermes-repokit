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

From a RepoKit source checkout, install the bootstrap CLI with Go 1.26+:

```sh
./install.sh              # installs ~/.local/bin/hermes-repokit and the repokit alias
```

The script builds locally; it does not download an unpublished release. It
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
repokit plan              # inspect; writes nothing
repokit install           # publish Compose, launcher, host command and private state
# run the printed Compose build/start command
repokit install           # initialize native Kanban after the runtime is running
repokit setup             # private provider, team, memory and activation
repokit verify            # observational CORE/MEMORY/FULL readiness report
hermes-my-project        # native Hermes CLI as default, when ~/.local/bin is on PATH
```

`setup` runs in your private terminal and never captures credentials; `--team` or
`--memory` resumes an interrupted stage. The launcher forwards native arguments
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

## Running and remaining gates

Fresh installs keep dispatch off. After the seven profiles reconcile, setup writes
the native Kanban policy on `default` (automatic review, concurrency one, no
auto-decomposition, seven-profile allowlist) with `hermes config set` and restarts
the gateway once. It refuses while a card is running, preserves an owner-changed
policy and never claims a worker ran. Memory is user-managed and never blocks
core work.

`verify` is observational and leads with `CORE_READY`; it reports memory as
user-managed and does not configure or verify any provider. Core is `healthy` only when configuration, runtime, worker-shell
toolchain, dispatch policy, gateway and per-channel tools are healthy and native
card history shows same-card executor→tester→reviewer completion; otherwise
`unqualified` (not yet exercised) or `degraded`. It exits 0 unless core is
degraded. `verify --dispatch-check` is the explicit, paid proof: one no-write
researcher card must be claimed and completed by the gateway without manual
dispatch.

| Area | Remaining evidence |
| --- | --- |
| User-managed memory | Operator-owned provider setup and behavior; RepoKit does not verify it |
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

[skill-hermes-repokit](skills/skill-hermes-repokit/SKILL.md) teaches an agent to
install, resume, verify and use RepoKit. Copy the folder into your agent's skills
directory — `~/.agents/skills/` for Codex, Pi and OpenCode, or `~/.claude/skills/`
for Claude Code. Installing it does not start a deployment.

## Troubleshooting and deeper guides

- Setup needs the generated Hermes service running; use the exact Compose command
  printed by `install` and rerun `install` for native Kanban initialization.
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

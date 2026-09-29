<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="Hermes RepoKit — bootstrap one repository-specific Docker Hermes container with six native profiles: default, researcher, planner, executor, reviewer and steward.">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/runtime-Docker%20Compose-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white" alt="Docker Compose runtime">
  <img src="https://img.shields.io/badge/profiles-6-F0B24A?style=flat-square" alt="Six native profiles">
  <img src="https://img.shields.io/badge/status-pre--v1-D08A2E?style=flat-square" alt="Status: pre-v1">
</p>

**Hermes RepoKit** prepares a repository for native Hermes work without becoming
part of the runtime. It handles the delicate host/bootstrap work—safe private
state, ownership and symlink checks, deterministic Compose, mounts and
installation—then hands control to Hermes for profiles, models, channels, Kanban
and agent execution, and to embedded OpenViking for memory. The result is one
repository-specific container, six native profiles, a shared board and private
memory state. After bootstrap, ordinary Compose and the generated launcher work
without RepoKit installed.

It is for repository owners who want a persistent, role-based team to work on
code or other repository artifacts while keeping deployment state isolated from
the application's own Compose stack. Go is the bootstrap implementation choice;
the generated runtime is a pinned Hermes development image, not a Go service or
a RepoKit daemon.

> **Status · pre-v1.** The six-role team, shared memory and gateway activation are
> implemented and tested offline; live model work, memory recall and independent
> same-card review remain unqualified. See [remaining gates](#running-and-remaining-gates).

## Quickstart

Needs Docker Compose, Git and a POSIX shell — no host Go, Python, Node or Hermes.
Run from the repository you want a team for:

```sh
cd my-project
hermes-repokit plan      # inspect; writes nothing
hermes-repokit install   # publish Compose, launcher, host command and private state
# run the printed Compose build/start command
hermes-repokit install   # initialize native Kanban after the runtime is running
hermes-repokit setup     # private provider, team, memory and activation
hermes-repokit verify    # read-only health and gate report
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
| `reviewer` | Independently verify the contract |
| `steward` | Maintain profile identities and the team lifecycle |

Steward prefers a task-scoped skill before creating a specialist, retirement
preserves history, and deletion needs explicit approval. See the
[team model](docs/team-model.md).

Profiles are persistent identities, not necessarily running workers. The native
gateway dispatcher is off on a fresh install; successful setup enables the
configured single default dispatcher only after readiness checks and a real
no-write researcher canary. RepoKit does not run as a supervisor or mediate
ongoing agent work.

## What it leaves behind

<p align="center">
  <img src="./assets/readme/deployment.svg" width="100%" alt="RepoKit writes a private .hermes directory and one Compose-owned Hermes container running an s6 supervisor, a default gateway, six profiles, a shared Kanban board and embedded memory on loopback at 127.0.0.1:1933.">
</p>

RepoKit writes a private `.hermes/` directory (Compose file, `bin/hermes-<repo>`
launcher, config, pinned development image, private `openviking/` data and a
`.gitignore`) and one Compose-owned container. It mounts the repository at
`/workspace` and all native state (including embedded memory) at `/opt/data`;
OpenViking runs inside under the native s6 supervisor on loopback with no published
host port and can extract memory automatically. Credentials stay in private setup,
and optional `install --docker-tests` adds an isolated acceptance daemon without
mounting the host Docker socket. On SELinux-enabled Linux hosts it requests
Docker's private `Z` relabeling for the repository and `.hermes` mounts as normal
installation compatibility; SELinux policy and unrelated host paths are never
changed. See
[development runtime](docs/qualification/development-runtime.md).

Existing repository Compose files and services stay in place. RepoKit uses its
own `.hermes/compose.yaml` and project namespace, with explicit file/context routing.

## Running and remaining gates

Fresh installs keep dispatch off. Setup enables one `default` gateway dispatcher
(automatic review, concurrency one, no auto-decomposition) only after native
core readiness and a real no-write researcher canary. OpenViking readiness is
reported separately and does not block core work. Live Telegram delivery and
same-card review remain separate acceptance gates.

`verify` is read-only: it observes artifacts, native configuration and bounded health
responses, and never runs models, dispatches tasks or writes memory. Memory `active`
means the six bindings authenticate, not that recall or extraction works.

`verify --memory-check` currently returns a structured `unsupported` report and
exit code 1 **before runtime access or writes**. It does not run a canary: the
pinned provider lacks a qualified isolated lifecycle with complete cleanup.
Write, extraction, both recall checks and cleanup remain individually
`unqualified`. See [memory self-check safety](docs/qualification/memory-self-check.md).

| Area | Remaining evidence |
| --- | --- |
| Embedded memory | Live write/recall, restart persistence and cross-repository denial |
| Team work | Real executor/reviewer correction cycle and channel delivery |
| Runtime independence | Removal-first acceptance with real work and memory |
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
- A pending/degraded memory service is separate from core team readiness. Resume
  with `setup --memory`; see [bootstrap and memory setup](docs/bootstrap-quickstart.md#openviking-configuration).
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
REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerOpenVikingPending -v
```

The full offline and race suites, including the Unix-socket fixtures, pass.
Live memory, independent review and channel round-trip acceptance remain open. See the
[design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md),
[plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md) and
[remaining work](TODO.md).

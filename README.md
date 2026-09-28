<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="Hermes RepoKit — bootstrap one repository-specific Docker Hermes container with six native profiles: default, researcher, planner, executor, reviewer and steward.">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/runtime-Docker%20Compose-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white" alt="Docker Compose runtime">
  <img src="https://img.shields.io/badge/profiles-6-F0B24A?style=flat-square" alt="Six native profiles">
  <img src="https://img.shields.io/badge/status-pre--v1-D08A2E?style=flat-square" alt="Status: pre-v1">
</p>

**Hermes RepoKit** turns a repository into a self-contained Hermes development
environment: one Docker container, six coordinated native profiles, a shared
Kanban board and embedded OpenViking memory. It generates ordinary Docker Compose,
private `.hermes` state and a standalone `hermes-<repo>` launcher — and the
installed runtime does not need RepoKit.

> **Status · pre-v1.** The six-role team, shared memory and gateway activation are
> implemented and tested offline; live model work, memory recall and independent
> same-card review remain unqualified. See [remaining gates](#running-and-remaining-gates).

## Quickstart

Needs Docker Compose, Git and a POSIX shell — no host Go, Python, Node or Hermes.
Run from the repository you want a team for:

```sh
cd my-project
hermes-repokit plan      # inspect; writes nothing
hermes-repokit install   # publish Compose, launcher and private state
# run the printed Compose build/start command
hermes-repokit setup     # private provider, team, memory and activation
hermes-repokit verify    # read-only health and gate report
.hermes/bin/hermes-my-project   # native chat as default
```

`setup` runs in your private terminal and never captures credentials; `--team` or
`--memory` resumes an interrupted stage. The launcher forwards native arguments
unchanged (`.hermes/bin/hermes-my-project kanban list`). After install, ordinary
Docker Compose owns the runtime using the context recorded in the launcher — see
[runtime management](docs/bootstrap-quickstart.md#ordinary-runtime-management). No
PATH link or shell change is automatic; see the
[setup and recovery guide](docs/bootstrap-quickstart.md).

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

## What it leaves behind

<p align="center">
  <img src="./assets/readme/deployment.svg" width="100%" alt="RepoKit writes a private .hermes directory and one Compose-owned Hermes container running an s6 supervisor, a default gateway, six profiles, a shared Kanban board and embedded memory on loopback at 127.0.0.1:1933.">
</p>

RepoKit writes a private `.hermes/` directory (Compose file, `bin/hermes-<repo>`
launcher, config, pinned development image, private `openviking/` data and a
`.gitignore`) and one Compose-owned container. It mounts the repository at
`/workspace`, native state at `/opt/data` and memory at `/app/.openviking`;
OpenViking runs inside under the native s6 supervisor on loopback with no published
host port and can extract memory automatically. Credentials stay in private setup,
and optional `install --docker-tests` adds an isolated acceptance daemon without
mounting the host Docker socket. See
[development runtime](docs/qualification/development-runtime.md).

## Running and remaining gates

Fresh installs keep dispatch off. Setup enables one `default` gateway dispatcher
(automatic review, concurrency one, no auto-decomposition) only after native
readiness, authenticated shared memory and a real no-write researcher canary; live
Telegram delivery and same-card review remain pending.

`verify` is read-only: it observes artifacts, native configuration and bounded health
responses, and never runs models, dispatches tasks or writes memory. Memory `active`
means the six bindings authenticate, not that recall or extraction works.

| Area | Remaining evidence |
| --- | --- |
| Embedded memory | Live write/recall, restart persistence and cross-repository denial |
| Team work | Real executor/reviewer correction cycle and channel delivery |
| Runtime independence | Removal-first acceptance with real work and memory |
| Optional plugins | Native scanner admission and owner configuration |

Credential-free fixtures demonstrated native profiles, Kanban persistence and
launcher/Compose operation after removing a copied installer, but do not qualify the
current image or model-driven work. See
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

## Development

Go 1.26+; host builds target Linux amd64 and arm64, the development image Linux
amd64. End-user binaries need Docker Compose, Git and a POSIX shell.

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

The full offline and race suites, including the Unix-socket fixtures, pass; live
Docker and model acceptance remain unqualified. See the
[design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md),
[plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md) and
[remaining work](TODO.md).

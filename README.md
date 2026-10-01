<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="Hermes RepoKit — one isolated Hermes environment for each repository.">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/runtime-Docker%20Compose-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white" alt="Docker Compose runtime">
  <img src="https://img.shields.io/badge/status-pre--v1-D08A2E?style=flat-square" alt="Status: pre-v1">
</p>

# A dedicated Hermes team for every repository

**RepoKit prepares the environment. Hermes runs the team.** Each repository gets
its own isolated Hermes container, profiles, conversations and Kanban board—ready
to work alongside your project without taking over its existing Docker Compose
stack.

- **Skip the hand-built glue:** RepoKit creates the container, launcher and native
  Hermes team with repeatable commands.
- **Keep projects apart:** each repository has its own Compose project and private
  `.hermes` state.
- **Stay in control:** credentials go directly into Hermes's private setup, never
  through RepoKit.
- **Use Hermes after bootstrap:** the generated launcher and ordinary Compose
  manage the environment; RepoKit does not need to run in the background.

> **Pre-v1:** the core loop has been live-tested, but broader platform support and
> release testing are still in progress. See [what is and isn't validated](#status-and-limits).

## One repository, one environment

<p align="center">
  <img src="./assets/readme/deployment.svg" width="100%" alt="RepoKit prepares a private .hermes deployment and a separate Compose-owned Hermes container for one repository; the repository's own Compose stack remains separate.">
</p>

RepoKit writes deployment files and private state under `.hermes/`. The Hermes
container mounts the project at `/workspace` and its private state at `/opt/data`.
The generated `.hermes/compose.yaml` has its own project namespace: your
application's Compose files and services stay separate and untouched. Memory
provider setup is user-managed and outside RepoKit's scope.

## Quickstart

**Supported host:** Linux amd64, with Go 1.26+, Docker Compose, Git and a POSIX
shell. macOS and Linux arm64 are not currently documented as supported hosts.
The installer builds RepoKit from source, so Go is needed for this step.

Install the bootstrap CLI from the latest release, [v0.2.3](https://github.com/TrebuchetDynamics/hermes-repokit/releases/tag/v0.2.3):

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.2.3/install.sh | REPOKIT_REF=v0.2.3 sh
```

From the repository you want to prepare, run the three-step path:

```sh
cd my-project
repokit install
repokit setup
hermes-my-project
```

`repokit` manages the environment: `install` builds and starts it, and `setup`
opens Hermes's private provider setup in your terminal before preparing the team.
Credentials are entered directly with Hermes; RepoKit does not read or store
them. `hermes-my-project` opens that repository's Hermes team. Replace the name
with the generated `hermes-<repo>` launcher for your repository.

For example, you might ask:

> “Research how this project handles authentication, then propose and implement
a safer token-refresh flow. Run the relevant tests and ask for an independent review.”

A typical request may flow through **research → plan → execute → test → review**.
The default profile coordinates and chooses the needed roles; a small question
may use fewer steps, and not every request uses every role. The seven-role roster
(default, researcher, planner, executor, tester, reviewer and steward) is RepoKit's
opinionated default, not a requirement for every team.

## Everyday commands

```sh
repokit plan       # preview changes without writing
repokit verify     # check whether the core environment is ready
repokit stop       # stop it; keep all state
repokit start      # start it again
```

`verify` is an observational readiness check; it does not send a test task.
`repokit verify --dispatch-check` asks the running Hermes gateway to complete one
no-write test card and may incur provider usage. See [runtime management and
recovery](docs/bootstrap-quickstart.md#ordinary-runtime-management).

After install, the generated Hermes launcher forwards native Hermes commands,
for example `hermes-my-project kanban list`. RepoKit does not create shell aliases
or edit shell startup files. See [host command and recovery](docs/bootstrap-quickstart.md#host-command).

## Removing a deployment

```sh
cd my-project
repokit remove
```

`remove` asks you to type the repository name, then removes the deployment it can
identify as RepoKit-generated, its container resources and image, and the host
launcher link it created. It also deletes **all `.hermes` state**, including any
user-configured provider data stored there, profiles, conversations and Kanban
history. This deletion cannot be undone. Repository files and Git history are
left alone. If `.hermes` is already gone, `remove` only cleans up the remaining
resources it can attribute to that repository.

## Status and limits

Live testing validated the core team workflow; it does not prove every host,
channel, provider or repository configuration. RepoKit is pre-v1, and Linux arm64
host support and broader release testing remain open. `verify` reports the
core environment, not memory-provider behavior or successful delivery to a human.

For implementation details, see the [architecture](docs/architecture.md),
[setup and recovery guide](docs/bootstrap-quickstart.md), [team model](docs/team-model.md),
[development runtime](docs/qualification/development-runtime.md) and
[remaining work](TODO.md).

## Agent skill

The [RepoKit agent skill](skills/skill-hermes-repokit/SKILL.md) guides agents through safe setup and operation; see its [installation instructions](skills/skill-hermes-repokit/references/installation.md).

## Contributing

See the [build and development guide](docs/build.md) for local checks and contributor setup.

<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="Hermes RepoKit — one isolated Hermes environment for each repository.">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/runtime-Docker%20Compose-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white" alt="Docker Compose runtime">
  <img src="https://img.shields.io/badge/status-pre--v1-D08A2E?style=flat-square" alt="Status: pre-v1">
</p>

# Give every repository its own Hermes developer

Run `repokit install` in a repository, then tell it what you want, from the
terminal or any Hermes channel you have set up. One agent researches, builds
the change on a Kanban card, and checks it in a separate fresh session (tests
run, no weakened tests, nothing out of scope) before it reports back.

**RepoKit sets up the environment. Hermes runs the agent.**

> **Pre-v1.** Linux amd64 is the documented host. Some behaviors, such as
> end-to-end remote coding over a messaging channel and memory recall, are not
> yet qualified. See [status and limits](#status-and-limits).

## 30 seconds

```sh
cd my-project
repokit install        # builds the environment, then Hermes's private setup and the team
hermes-my-project      # talk to this repository's agent
```

Then give it real work:

> "Find why the token refresh test is flaky, fix it, and tell me what changed."

```
You → default ─┬─ research: parallel read-only subagents
               └─ card: implement → fresh verification run → done → You
```

Small questions get a reply. Anything that changes the repository becomes a
Kanban card that survives restarts. The run that implemented a change never
approves it: a separate verification run, which never saw the implementer's
reasoning, tries to break it first.

## Install

**Documented deployment host:** Linux amd64, with Go 1.26+, Docker Compose, Git
and a POSIX shell. The RepoKit CLI can be built for Linux amd64 and arm64, but
the current development image is pinned to linux/amd64; Linux arm64 is not an
end-to-end supported deployment target yet. macOS is not a documented host.
The installer builds RepoKit from source, so Go is needed for this step.

Install the bootstrap CLI from the latest release, [v0.3.3](https://github.com/TrebuchetDynamics/hermes-repokit/releases/tag/v0.3.3):

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.3.3/install.sh | REPOKIT_REF=v0.3.3 sh
```

`repokit install` builds and starts the environment, then in a terminal goes
straight on into `repokit setup`: Hermes's own private provider setup (RepoKit
never sees your credentials), the team, automatic dispatch and one canary card.
`install --no-setup` stops after install. The launcher is named after your
repository: `hermes-<repo>`.

## How the agent works

RepoKit installs one Hermes profile, `default`, and it is the whole team:

- **It talks with you** and decides, in plain language, and stays quiet while
  work moves.
- **It researches** through parallel, read-only Hermes subagents that run in
  the background while the conversation continues.
- **It builds on a card.** Each repository change is a Kanban card it
  implements test-first, with the commands it ran as evidence.
- **It verifies in a fresh run.** It checks four things: each acceptance
  criterion is shown by command output; no test was deleted, weakened or
  special-cased; nothing outside the card's scope changed; new probes find no
  regression. Risky changes fan out up to four checking subagents. After two
  rounds of requested changes it asks you instead of looping.
- **It keeps your decisions** in Hermes's built-in memory, short itemized
  entries created only from your own messages.

Specialization comes from skills, not more profiles. Details:
[team model](docs/team-model.md).

## What RepoKit is, and is not

RepoKit is a small per-repository bootstrapper. It installs, configures and
leaves: nothing RepoKit-authored keeps running after install.

| RepoKit owns | Hermes owns |
| --- | --- |
| The repository's isolated Docker environment and development toolchains | Models and providers |
| The `default` profile's identity, grants and skills | Channels such as Telegram |
| Kanban readiness and the dispatch policy | Sessions, memory and plugins |
| Safe install, update, verify and remove | Running the agent |
| The host launcher `hermes-<repo>` | |

RepoKit configures no memory provider of its own; it only turns on Hermes's
built-in memory tool. The [oh-my-hermes](#oh-my-hermes) setup it runs adds
OMH's plugin and memory provider.

## oh-my-hermes

Every image ships [oh-my-hermes](https://github.com/rlaope/oh-my-hermes) (OMH)
3.0.0, pinned by checksum. Install runs OMH's own setup for `default` once per
OMH version:

- its core workflow skills, planner, handoff-guide, tracker and guide (about
  3.5k tokens of context per request; the full set adds about 50k);
- Hermes as the coding executor, so cards never stop to ask which one to use;
- its Hermes plugin, with OMH memory off.

OMH adds ways of working, not new systems: default plans, hands off and reports
with OMH's skills, while the board stays Kanban, subagents stay `delegate_task`
and memory stays Hermes memory. An earlier setup that made OMH default's memory
provider is undone; a provider you chose is kept. The gateway restarts to load
the plugin only while no card runs. RepoKit records its OMH setup in
`.hermes/.repokit-omh` and leaves OMH alone after that, so your own changes stay
until RepoKit's OMH setup changes. Run `omh setup --full` inside the
container for every OMH workflow, or `omh doctor` to check it.

## Everyday commands

```sh
repokit plan       # preview changes without writing
repokit verify     # check whether the core environment is ready
repokit stop       # stop it; keep all state
repokit start      # start it again
repokit update     # replace this binary with the latest release (--main: latest main)
repokit list       # every repository RepoKit installed into here, with live state (--json)
repokit github-login  # sign in to GitHub once so the team can push
```

Agents push to GitHub over HTTPS through the GitHub CLI's own login.
`repokit github-login` runs `gh auth login` inside the deployment, in your
terminal; RepoKit never sees the token, and gh keeps the login in `.hermes`
(never committed). Choose "Paste an authentication
token" with a fine-grained token limited to this repository for the least
privilege. Without it, agents finish their work and hand you `git push`.
GitHub SSH remotes resolve to HTTPS inside the container only. `verify` reports
the login as `github_push`.

`verify` is an observational readiness check; it does not send a test task.
`repokit verify --dispatch-check` asks the running Hermes gateway to complete one
no-write test card and may incur provider usage. See [runtime management and
recovery](docs/bootstrap-quickstart.md#ordinary-runtime-management).

After install, the generated Hermes launcher forwards native Hermes commands,
for example `hermes-my-project kanban list`. RepoKit does not create shell aliases
or edit shell startup files. See [host command and recovery](docs/bootstrap-quickstart.md#host-command).

## One repository, one isolated environment

<p align="center">
  <img src="./assets/readme/deployment.svg" width="100%" alt="RepoKit prepares a private .hermes deployment and a separate Compose-owned Hermes container for one repository; the repository's own Compose stack remains separate.">
</p>

RepoKit writes deployment files and private state under `.hermes/`. The Hermes
container mounts the project at `/workspace` and its private state at `/opt/data`.
The generated `.hermes/compose.yaml` has its own project namespace: your
application's Compose files and services stay separate and untouched.

## Lifecycle, safety and removal

RepoKit is a bootstrapper, not a persistent service: `repokit install` prepares
and starts the container, `repokit setup` runs Hermes's private provider setup
when needed and provisions the team, and the generated launcher plus ordinary
Compose manage it afterward. Credentials go directly into Hermes; RepoKit does
not read or store provider secrets. `repokit stop` and `repokit start` preserve
the deployment state. See [removing a deployment](#removing-a-deployment) below for
the destructive removal behavior and the [setup and recovery guide](docs/bootstrap-quickstart.md).

### Removing a deployment

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

RepoKit is pre-v1. Current qualification records establish selected bootstrap,
runtime and dispatch behaviors, not every repository, provider, host or channel.
In particular, Telegram delivery, fully model-driven team execution, memory
recall and cross-repository memory isolation are not established acceptance
claims. `verify` reports core environment readiness; it does not prove memory
provider behavior or successful delivery to a human. Review the linked evidence
before treating the deployment as qualified for a broader use case.

## Documentation

- [Setup, usage and recovery](docs/bootstrap-quickstart.md)
- [Architecture and ownership boundaries](docs/architecture.md)
- [Team model](docs/team-model.md)
- [Operational dispatch qualification](docs/qualification/operational-dispatch.md)
- [Generic-team acceptance status](docs/qualification/generic-team.md)
- [Development-runtime qualification](docs/qualification/development-runtime.md)
- [Remaining work](TODO.md)

## Agent skill

The [RepoKit agent skill](skills/skill-hermes-repokit/SKILL.md) guides agents
through safe setup and operation; see its [installation instructions](skills/skill-hermes-repokit/references/installation.md).

## Contributing

See the [build and development guide](docs/build.md) for local checks and
contributor setup.

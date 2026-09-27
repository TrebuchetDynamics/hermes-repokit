# Hermes RepoKit

Bootstrap a repository-specific, Docker-based Hermes Agent environment.

RepoKit is being built to configure Docker Compose, private `.hermes` state,
Kanban, plugins and a convenient `hermes-<repo>` launcher. After installation,
the environment should keep working without RepoKit.

**Early development — no usable installer yet.** The Go CLI skeleton,
qualification contracts and unit tests exist. All four actions currently report
`command not implemented`. The workflow and generated environment below are
planned. See [current progress](TODO.md).

## Why RepoKit?

Setting up Hermes separately for each repository means repeating Compose,
mounts, persistent state, plugin and launcher configuration. RepoKit aims to
make that setup reproducible.

**RepoKit is not a Hermes runtime or replacement CLI.** It is a bootstrap and
configuration helper. Standard Docker Compose owns the deployment; native
Hermes owns chat, profiles, Kanban and plugins. Removing RepoKit must not stop
either from working.

## Quick start — planned

The intended handoff moves from the installer to your repository's native
Hermes launcher:

```sh
cd my-project

hermes-repokit plan
hermes-repokit install
hermes-my-project setup
hermes-my-project
```

These are not working installation instructions yet. The generated service
must be running, and the short command requires an approved PATH shortcut;
otherwise use `.hermes/bin/hermes-my-project` directly.

The launcher will open native Hermes chat without arguments and pass explicit
arguments unchanged:

```sh
hermes-my-project                 # native chat
hermes-my-project setup           # native setup
hermes-my-project kanban list     # native Kanban command
```

Native command support must be checked against the selected Hermes version.
RepoKit does not implement those commands.

## What gets created — planned

```text
my-project/
├── source…
└── .hermes/                       private persistent Hermes state
    ├── compose.yaml
    ├── config.yaml
    ├── bin/hermes-my-project      standalone shell launcher
    ├── profiles/
    ├── plugins/
    └── …                         native Kanban, sessions and credentials

              │ RepoKit configures
              ▼
       Docker / Hermes
       ├── /workspace  ← my-project/
       └── /opt/data   ← my-project/.hermes/

Host command: hermes-my-project
```

The baseline will provide one Hermes container per repository, the native
default profile, persistent Kanban and the native `obra/superpowers` plugin.
Kanban starts conservatively: automatic dispatch and decomposition are off.
Sensitive `.hermes` state is intended to stay private and excluded from Git.

## RepoKit commands

All four are currently skeletons. Their planned responsibilities are:

| Command | Purpose |
| --- | --- |
| `hermes-repokit plan` | Show proposed configuration without changes. |
| `hermes-repokit install` | Bootstrap the environment; preserve owner edits on safe reruns. |
| `hermes-repokit setup` | Delegate directly to native Hermes setup. |
| `hermes-repokit verify` | Report read-only observations; leave unsupported checks unknown. |

## Defaults and safety

RepoKit favors Docker isolation per repository, native Hermes behavior,
persistent local state and immutable dependency pins. Safe installation is a
design requirement, not yet an implemented guarantee:

- Preserve dirty Git working trees, existing Hermes state and owner edits.
- Refuse ambiguous container ownership and conflicting names or paths.
- Keep credentials out of installer logs and receipts.
- Use atomic publication and conservative, safely repeatable installation.
- Leave readable Compose files and a standalone launcher behind, with no Pi
  dependency or requirement to keep RepoKit installed.

## Planned extensions

Fresh installs will begin with the native default profile. Optional
researcher, planner, builder and reviewer profiles will support multi-profile
Kanban workflows within the same deployment. Native workflow recipes are
planned; RepoKit will not become a resident team manager.

| Extension | Direction | Status |
| --- | --- | --- |
| OpenViking | Shared long-term project memory | Planned; native privacy compatibility remains unresolved. |
| Nerve | Native advisory plugin | Planned. |
| Laya | Optional inference sidecar for Nerve | Planned; bounded operation still needs qualification. |

None of these integrations, including the baseline Superpowers plugin, is
installed by the current CLI.

## Requirements — planned release

- Linux amd64 or arm64.
- Git, Docker, Docker Compose and a POSIX shell.

Released binaries are intended to require no Go, Python, Node or Pi on the
host. Hermes and Python-native plugins or sidecars retain their own runtimes
inside containers.

## Development

The current Go module requires Go 1.26.0 or newer. With a suitable toolchain
already installed, run the offline tests:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...
```

`cmd/hermes-repokit/` contains the entry point; `internal/` contains the CLI
and qualification model with package-local tests. Detailed design and future
integration work live in `docs/`:

- [Bootstrap design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md)
- [Implementation plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md)
- [Qualification evidence and development limits](docs/qualification/native-contract.md)
- [Progress and next tasks](TODO.md)

## Self-hosting goal

RepoKit should eventually bootstrap the Hermes environment used to develop
RepoKit itself. The release gate goes further: remove RepoKit, then prove real
Hermes work, restart and persistent state still function. Neither has been
demonstrated yet.

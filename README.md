# Hermes RepoKit

Bootstrap a repository-specific Docker Hermes environment.

RepoKit generates ordinary Docker Compose, private `.hermes` state and a
standalone `hermes-<repo>` launcher. After bootstrap, RepoKit is not required.

**The Hermes-only foundation works; v1 is not complete.** `plan`, `install`,
`setup` delegation and read-only `verify` are implemented. Installation generates
files; ordinary Compose starts Hermes. Plugins, sidecars and engineering profiles
are deferred until their upstream qualification is complete.

```sh
cd my-project
hermes-repokit plan
hermes-repokit install
# Run the exact Docker Compose start command printed by install.
hermes-repokit setup
hermes-repokit verify
.hermes/bin/hermes-my-project
```

`install --engineering` currently refuses without publishing artifacts.
No PATH link or shell configuration change is automatic.

Without a PATH link, the standalone command is
`.hermes/bin/hermes-my-project`. It opens native Hermes chat with no arguments
and forwards explicit arguments unchanged:

```sh
.hermes/bin/hermes-my-project setup
.hermes/bin/hermes-my-project kanban list
.hermes/bin/hermes-my-project plugins list
```

## What RepoKit leaves behind

```text
my-project/
└── .hermes/
    ├── compose.yaml
    ├── bin/hermes-my-project
    ├── config.yaml
    └── .gitignore              # ignores all native state, including credentials
```

Native setup creates authentication, sessions and other standard Hermes state.
Exactly one Hermes container mounts the repository at `/workspace` and native
state at `/opt/data`. Standard Compose owns the runtime. Native files remain
authoritative; receipts are not needed. Reruns preserve matching artifacts and
native configuration, and refuse ambiguous ownership or generated-file changes.
Dispatch and automatic decomposition start disabled.

## First milestone

The four commands have offline tests. A real Docker test installed through the
CLI, verified the container, safely reran installation, removed the copied
RepoKit source and binary, then used native commands and raw Compose restart
with persistent Kanban state. This establishes credential-free foundation
independence, not authenticated chat or full v1 integration qualification.

See [runtime evidence](docs/qualification/runtime-observations.md) and the
[implementation](internal/cli/install.go). `verify` checks core artifact and
runtime metadata; it does not validate credentials or native configuration
semantics. An edited Compose/launcher is preserved but reported unknown.

## Later integration qualification

| Component | Direction and evidence |
| --- | --- |
| Hermes | Official immutable image tested for Compose startup, native exec, profiles, Kanban initialization and restart persistence. Authenticated chat/setup remain pending. |
| Superpowers | Upstream `obra/superpowers`; exact candidate SHA received 229 CAUTION findings. Installation is blocked pending explicit approval of the [scanner report](docs/qualification/superpowers-8ca22dba-scan.txt). |
| Nerve/Laya | Upstream plugin and its supported sidecar only. Transport, checkpoint, scanner/loading and actual inference qualification remain pending. No custom implementation. |
| OpenViking | Official image and native Hermes provider. Native setup handoff observed; embedding/VLM configuration, write/recall and isolation evidence remain pending. |
| Same-card review | Native Hermes plus Nerve first. Distinct builder/reviewer actors must be proved before release; no speculative policy plugin. |

OpenViking is not installed by the foundation. Its planned native integration
can synchronize turns/tool results and extract memory automatically. Embedding/VLM configuration determines where model data
is processed. RepoKit delegates credentials to native setup and does not impose
invented durable-only memory semantics.

## Development

Go module: `github.com/TrebuchetDynamics/hermes-repokit`; Go 1.26 or newer.
No third-party Go modules are currently needed. Supported build targets are
Linux amd64 and arm64. End-user binaries will require Docker Compose, Git and
a POSIX shell, without host Go, Python, Node, Pi or Hermes.

```sh
go test ./...
go test -race ./...
go vet ./...
gofmt -l cmd internal tests
```

Ordinary tests are offline. The following explicitly opts into creating and
removing a disposable Docker project using the pinned official Hermes image:

```sh
REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerFoundation -v
```

That fixture proves only credential-free foundation behavior. It does not
prove real inference, memory, independent review or the full removal-first gate.

See the [quickstart and native handoff](docs/bootstrap-quickstart.md),
[design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md),
[implementation plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md),
[runtime evidence](docs/qualification/runtime-observations.md), and
[implementation progress](docs/implementation-progress.md).

# Hermes RepoKit

Bootstrap a repository-specific Docker Hermes environment.

RepoKit configures Docker Compose, private `.hermes` state, native Kanban,
plugins, optional engineering profiles, OpenViking memory and Nerve/Laya
supervision. After bootstrap, RepoKit is not required.

**In development; v1 is not complete.** `plan`, read-only `verify`, and delegation
of `setup` to an existing generated launcher work. Compose rendering, atomic
publication and the standalone launcher have offline tests and a credential-free
Docker acceptance test. **`install` currently refuses before writing:** no full
preset has passed upstream admission and the removal-first release gate.

```sh
cd my-project
hermes-repokit plan
hermes-repokit install          # currently reports qualification blockers
hermes-repokit setup            # requires a generated, running deployment
hermes-my-project              # requires an explicitly authorized PATH link
```

Engineering preset (not yet admitted):

```sh
hermes-repokit install --engineering
```

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
    ├── profiles/
    ├── plugins/
    ├── kanban.db
    ├── openviking/
    ├── nerve/
    ├── laya/                    # engineering/Nerve-Laya preset
    └── repokit-install.json     # optional, informational
```

Exactly one Hermes container mounts the repository at `/workspace` and native
state at `/opt/data`. OpenViking and optional Laya are separate services.
Standard Compose owns the runtime. Native files remain authoritative; missing
or corrupt receipts never authorize overwriting configuration.

Fresh deployments start with dispatch and automatic decomposition disabled.
Engineering adds researcher, planner, builder and reviewer profiles when
qualified; it does not enable autonomous work or clone credentials.

## First milestone

Finish `plan`, `install`, `setup` and `verify` for a Hermes-only deployment,
including target detection, naming, locking and a standalone launcher. Prove
that deployment survives RepoKit removal before adding plugins, Nerve/Laya,
OpenViking or engineering profiles. Full v1 integration qualification is a
separate gate; the implementation status above describes today's code.

## Integration status

| Component | Direction and evidence |
| --- | --- |
| Hermes | Official immutable image tested for Compose startup, native exec, profiles, Kanban initialization and restart persistence. Authenticated chat/setup remain pending. |
| Superpowers | Upstream `obra/superpowers`; exact candidate SHA received 229 CAUTION findings. Installation is blocked pending explicit approval of the [scanner report](docs/qualification/superpowers-8ca22dba-scan.txt). |
| Nerve/Laya | Upstream plugin and its supported sidecar only. Transport, checkpoint, scanner/loading and actual inference qualification remain pending. No custom implementation. |
| OpenViking | Official image and native Hermes provider. Native setup handoff observed; embedding/VLM configuration, write/recall and isolation evidence remain pending. |
| Same-card review | Native Hermes plus Nerve first. Distinct builder/reviewer actors must be proved before release; no speculative policy plugin. |

OpenViking's native integration can synchronize turns/tool results and extract
memory automatically. Embedding/VLM configuration determines where model data
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

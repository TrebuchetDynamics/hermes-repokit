# Hermes RepoKit

Bootstrap a repository-specific Docker Hermes environment.

RepoKit generates ordinary Docker Compose, private `.hermes` state and a
standalone `hermes-<repo>` launcher. After bootstrap, RepoKit is not required.

**The native six-role scaffold is implemented; full integration acceptance is
not complete.** RepoKit scaffolds a small repository organization, not a
collection of technology-specific bots.

| Profile | Purpose |
| --- | --- |
| default | Primary user-facing assistant and orchestrator |
| researcher | Evidence and unknowns |
| planner | Bounded execution contracts |
| executor | Artifact production |
| reviewer | Independent verification |
| steward | Team/profile lifecycle |

Talk to `default` normally. It answers lightweight questions directly and routes
substantive work. Steward prefers task skills before creating persistent
specialists; retirement preserves history, and deletion needs explicit approval.
See the [team model and boundaries](docs/team-model.md).

```sh
cd my-project
hermes-repokit plan
hermes-repokit install
# Run the exact Docker Compose start command printed by install.
hermes-repokit setup            # private default setup, then native team cloning
hermes-repokit verify
.hermes/bin/hermes-my-project   # primary default profile
```

Rerun `install` to initialize the shared board and inspect/reconcile an already
provisioned team. `--engineering` remains a compatibility alias; the universal
roster is now the default. Newly cloned profiles receive distinct SOULs and fresh
curated memory. Existing user changes and unknown profiles are preserved.
Installation also scaffolds the pinned official OpenViking sidecar. Its private
model/auth configuration and live memory qualification remain pending;
Nerve/Laya deployment remains separate integration work.
No PATH link or shell configuration change is automatic.

Without a PATH link, the standalone command is
`.hermes/bin/hermes-my-project`. It opens native Hermes chat as `default` with no arguments
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
    ├── openviking/             # native service configuration and durable data
    └── .gitignore              # ignores all native state, including credentials
```

Native setup creates authentication, sessions and other standard Hermes state.
Exactly one Hermes container mounts the repository at `/workspace` and native
state at `/opt/data`. Standard Compose owns the runtime. Native files remain
authoritative; receipts are not needed. Reruns preserve matching artifacts and
native configuration, and refuse ambiguous ownership or generated-file changes.
Dispatch and automatic decomposition start disabled, with one in-progress task.
New specialists are cloned only after default setup. Native lifecycle owns profile
publication; interrupted profiles are preserved for inspection. See the
[generic-team qualification](docs/qualification/generic-team.md).

## First milestone

The four commands have offline tests. A real Docker test installed through the
CLI, verified the container, safely reran installation, removed the copied
RepoKit source and binary, then used native commands and raw Compose restart
with persistent Kanban state. This establishes credential-free foundation
independence, not authenticated chat or full v1 integration qualification.

See [runtime evidence](docs/qualification/runtime-observations.md) and the
[implementation](internal/cli/install.go). `verify` checks core artifact and
runtime metadata and role SOULs. It explicitly reports memory/supervision
acceptance as unknown, so its exit status remains nonzero until those deployment
gates are implemented; it does not validate credentials. An edited Compose/launcher is preserved but reported unknown.

## Later integration qualification

| Component | Direction and evidence |
| --- | --- |
| Hermes | Official immutable image tested for Compose startup, native exec, profiles, Kanban initialization and restart persistence. Authenticated chat/setup remain pending. |
| Superpowers | Upstream `obra/superpowers`; exact candidate SHA received 229 CAUTION findings. Installation is blocked pending explicit approval of the [scanner report](docs/qualification/superpowers-8ca22dba-scan.txt). |
| Nerve/Laya | Upstream plugin and its supported sidecar only. Pinned local CPU inference, native plugin consumption and offline recreation passed in a disposable fixture; installer sidecar deployment and all-role supervision remain pending. See [evidence](docs/qualification/generic-team-laya.md). |
| OpenViking | Pinned official sidecar, private persistent directory, native `setup --memory` handoff and shared connection linking. Pending-mode/recreation qualified; embedding/VLM setup, live write/recall and isolation remain pending. See [wiring evidence](docs/qualification/openviking-wiring.md). |
| Same-card review | Native Hermes plus Nerve first. Distinct executor/reviewer actors must be proved before release; no speculative policy plugin. |

OpenViking is scaffolded without guessed models or credentials. Start its service
with the printed Compose command, then use `hermes-repokit setup --memory` in
your private terminal after default/team setup. Follow the
[native setup handoff](docs/bootstrap-quickstart.md#openviking-configuration).
Its native integration can synchronize turns/tool results and extract memory
automatically. Embedding/VLM configuration determines where model data
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
REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerOpenVikingPending -v
```

That fixture proves only credential-free foundation behavior. It does not
prove real inference, memory, independent review or the full removal-first gate.

See the [quickstart and native handoff](docs/bootstrap-quickstart.md),
[design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md),
[implementation plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md),
[runtime evidence](docs/qualification/runtime-observations.md), and
[implementation progress](docs/implementation-progress.md).

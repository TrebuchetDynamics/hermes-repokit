# Hermes RepoKit

Bootstrap a repository-specific Docker Hermes environment.

RepoKit generates ordinary Docker Compose, private `.hermes` state and a
standalone `hermes-<repo>` launcher. After bootstrap, RepoKit is not required.

**The six-role team, shared-memory setup and local Nerve/Laya wiring are
implemented; full integration acceptance and v1 are not complete.** RepoKit
scaffolds a small repository organization, not a collection of technology-specific
bots.

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
# Run the printed Compose build/start command for all three services.
hermes-repokit setup            # private default, team, memory, local supervision
hermes-repokit verify
.hermes/bin/hermes-my-project   # primary default profile
```

Rerun `install` to initialize the shared board and inspect/reconcile an already
provisioned team. `--engineering` remains a compatibility alias; the universal
roster is now the default. Newly cloned profiles receive distinct SOULs and fresh
curated memory. Existing user changes and unknown profiles are preserved.
Normal installation includes the pinned official OpenViking image and the
pinned local Laya build recipe. Plain `setup` runs native default setup, provisions
the team, configures shared OpenViking memory through private native wizards,
then installs/configures/enables upstream Nerve for all six profiles and checks
real `LOCAL_ONLY` decisions. Resume either integration with `setup --memory` or
`setup --supervision`. Live memory recall and model-driven task acceptance remain
pending. See the [native handoff](docs/bootstrap-quickstart.md) and
[Laya packaging guide](packaging/laya/README.md).
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
    ├── laya-image/             # standalone pinned image build inputs
    ├── laya/                   # persistent writable model-runtime caches
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
runtime metadata, role SOULs, native integration configuration and bounded health
responses. Memory `active` means all six bindings authenticate as the repository
user through read-only `/health`; it does not prove recall or extraction. Nerve
configuration and Laya health are reported separately without loading plugins or
running inference. `review` remains `unqualified`, so `verify` still exits nonzero.
An edited Compose/launcher is preserved but reported unknown. See the
[current evidence and limits](docs/qualification/runtime-integrations.md).

## Later integration qualification

| Component | Direction and evidence |
| --- | --- |
| Hermes | Official immutable image tested for Compose startup, native exec, profiles, Kanban initialization and restart persistence. Authenticated chat/setup remain pending. |
| Superpowers | Upstream `obra/superpowers`; exact candidate SHA received 229 CAUTION findings. Installation is blocked pending explicit approval of the [scanner report](docs/qualification/superpowers-8ca22dba-scan.txt). |
| Nerve/Laya | Upstream plugin and its supported sidecar only. Default pinned CPU build, six native plugin installs/hooks and `LOCAL_ONLY` decisions passed. Offline outage, installer removal and coordinated Compose recreation passed in a disposable stack; main-model supervised work remains unqualified. See [evidence](docs/qualification/runtime-integrations.md). |
| OpenViking | Pinned official sidecar, private persistent directory, native init/doctor/validation/restart/health/wizard setup and six-profile connection linking. Pending-mode/recreation and native linking fixtures passed; private model configuration, live write/recall and isolation remain pending. See [wiring evidence](docs/qualification/openviking-wiring.md). |
| Same-card review | Native Hermes plus Nerve first. Distinct executor/reviewer actors must be proved before release; no speculative policy plugin. |

OpenViking is scaffolded without guessed models or credentials. Start all services
with the printed Compose command, then run `hermes-repokit setup` in your private
terminal. Use `setup --memory` to resume memory setup after the team exists. Follow the
[native setup handoff](docs/bootstrap-quickstart.md#openviking-configuration).
Its native integration can synchronize turns/tool results and extract memory
automatically. Embedding/VLM configuration determines where model data
is processed. RepoKit delegates credentials to native setup and does not impose
invented durable-only memory semantics.

## Development

Go module: `github.com/TrebuchetDynamics/hermes-repokit`; Go 1.26 or newer.
No third-party Go modules are currently needed. Supported build targets are
Linux amd64 and arm64; the pinned local Laya CPU recipe is currently qualified
on Linux amd64. End-user binaries will require Docker Compose, Git and
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

Those fixtures prove credential-free foundation and pending memory-service
behavior. The separately gated `TestDockerSupervisionStack` performs real local
inference; its invocation and boundaries are in the
[runtime qualification record](docs/qualification/runtime-integrations.md).
None establishes live main-model work, memory recall, independent review or the
full integrated removal-first gate.

See the [quickstart and native handoff](docs/bootstrap-quickstart.md),
[design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md),
[implementation plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md),
[runtime evidence](docs/qualification/runtime-observations.md), and
[implementation progress](docs/implementation-progress.md).

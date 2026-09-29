# RepoKit architecture

> RepoKit configures capabilities; it does not implement them.

RepoKit is a Go bootstrapper for one repository. It identifies the repository,
protects owner and private state, renders the Docker/Compose inputs, invokes
public Hermes and OpenViking interfaces, installs a standalone launcher, and
verifies the resulting state. It exits after each command. Hermes owns profiles,
Kanban, channels, sessions, agents, and memory-provider integration. OpenViking
owns shared repository memory inside the same `hermes-<repo>` container.

The normal operational path has no RepoKit process, Python helper, plugin, or
sidecar. Removing the RepoKit executable does not stop Hermes, its gateway,
OpenViking, the launcher, or the repository team.

Production RepoKit behavior is Go plus generated Compose, image inputs, and a
standalone shell launcher. Setup may call supported public Hermes and
OpenViking CLIs inside the container. It must not import private modules from
either dependency, install a RepoKit-owned runtime extension, or fill a missing
Hermes capability with a second agent framework. No Python interpreter is
required on the host to install or run RepoKit. Python in the target
repository's development toolchain is independent of RepoKit.

Each readiness claim needs evidence from the owning component. Container health
is not Hermes, dispatch, toolchain, or memory readiness. A native configuration
value is not proof of a worker run or agent recall. If public interfaces cannot
safely configure or observe a capability, RepoKit reports it as unqualified and
preserves owner state. `CORE_READY` does not depend on OpenViking;
`MEMORY_READY` requires behavioral cross-profile proof; `FULL_READY` requires
both. Historical provider extraction and same-card review acceptance remain
separate from deterministic configuration checks.

Legacy RepoKit-generated state is preserved on an unrecognized migration path.
RepoKit does not hand-patch generated Compose or private `.hermes` state and
does not remove owner-installed plugins or private data automatically.

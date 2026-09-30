# RepoKit architecture

> RepoKit configures capabilities; it does not implement them.

RepoKit is a Go bootstrapper for one repository. It identifies the repository,
protects owner and private state, renders the Docker/Compose inputs, invokes
public Hermes interfaces, installs a standalone launcher, and verifies the
resulting state. It exits after each command. Hermes owns profiles, Kanban,
channels, sessions, agents, and memory-provider integration. Memory stays
user-managed: the operator configures and operates whatever provider the
repository needs.

The normal operational path has no RepoKit process, Python helper, plugin, or
sidecar. Removing the RepoKit executable does not stop Hermes, its gateway, the
launcher, or the repository team.

Production RepoKit behavior is Go plus generated Compose, image inputs, and a
standalone shell launcher. Setup may call supported public Hermes CLIs inside the
container. It must not import private modules from that dependency, install a
RepoKit-owned runtime extension, or fill a missing Hermes capability with a
second agent framework. No Python interpreter is required on the host to install
or run RepoKit. Python in the target repository's development toolchain is
independent of RepoKit.

Each readiness claim needs evidence from the owning component. Container health
is not Hermes, dispatch, or toolchain readiness. A native configuration value is
not proof of a worker run. If public interfaces cannot safely configure or
observe a capability, RepoKit reports it as unqualified and preserves owner
state. `CORE_READY` covers Hermes, tools and the reviewed work loop. Memory is
user-managed, so RepoKit neither configures nor certifies it. Historical
provider extraction and same-card review acceptance remain separate from
deterministic configuration checks.

RepoKit is authoritative over what it generated only while that state remains
unmodified. Once the owner changes managed team state through normal Hermes
usage, that state is owner-controlled: later installs may report it or offer
newer defaults, but never silently overwrite it.

RepoKit keeps no legacy support: it recognizes only its current generation and
the previous release, v0.2.0, which `install` upgrades in place (a Go
deployment gains the toolchain-cache volume; the old Compose and recipe are
backed up). Files from any earlier release, like edited or foreign ones, are
preserved and refused, never migrated; the owner starts over by stopping that deployment,
moving `.hermes` aside and installing again.
RepoKit does not hand-patch generated Compose or private `.hermes` state and
does not remove owner-installed plugins or private data automatically.

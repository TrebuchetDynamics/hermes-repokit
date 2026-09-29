# RepoKit architecture

> RepoKit prepares a repo-specific Hermes environment. Hermes handles Hermes features.

If standard Hermes already owns a capability, RepoKit does not build a second
version. See the [decision record](decisions/2026-09-29-hermes-owns-hermes-features.md).

RepoKit is a Go bootstrapper for one repository. It identifies the repository,
protects owner and private state, renders the Docker/Compose inputs for exactly
one `hermes-<repo>` container with `/workspace` and `.hermes` mounts, invokes
public Hermes interfaces, installs a standalone launcher, bootstraps the six
profiles and Kanban defaults, and verifies the resulting state. It exits after
each command. Hermes owns profiles, Kanban, channels, sessions, agents, models,
plugins and memory providers. RepoKit does not configure, run or verify a memory
provider; owners use native `hermes memory setup` / `hermes memory status`.

The normal operational path has no RepoKit process, Python helper, plugin, or
sidecar. Removing the RepoKit executable does not stop Hermes, its gateway,
the launcher, or the repository team.

Production RepoKit behavior is Go plus generated Compose, image inputs, and a
standalone shell launcher. Setup may call supported public Hermes CLIs inside
the container. It must not import private Hermes modules, install a
RepoKit-owned runtime extension, or fill a missing Hermes capability with a
second agent framework. No Python interpreter is required on the host to install
or run RepoKit. Python in the target repository's development toolchain is
independent of RepoKit.

Each readiness claim needs evidence from the owning component. Container health
is not Hermes, dispatch or toolchain readiness. A native configuration value is
not proof of a worker run. If public interfaces cannot safely configure or
observe a capability, RepoKit reports it as unqualified and preserves owner
state. `verify` summarizes `CORE_TEAM` (container, native state, profiles,
toolchain, Kanban, access) and `DISPATCH` (gateway, dispatch/notification
policy, channels); `DISPATCH` is `unqualified` until same-card independent review
is observed. Same-card review acceptance remains separate from deterministic
configuration checks.

Legacy RepoKit-generated state is preserved on an unrecognized migration path.
RepoKit does not hand-patch generated Compose or private `.hermes` state and
does not remove owner-installed plugins or private data automatically.

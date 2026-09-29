# Hermes owns Hermes features

- **Status:** accepted (owner decision, 2026-09-29)
- **Supersedes:** the embedded-OpenViking topology in the
  [single-container runtime design](../superpowers/specs/2026-09-28-single-container-runtime-design.md)
  and the RepoKit-managed OpenViking sidecar plan in
  [issue #2](https://github.com/TrebuchetDynamics/hermes-repokit/issues/2).

## Decision

> **RepoKit prepares a repo-specific Hermes environment. Hermes handles Hermes features.**

If standard Hermes already owns a capability, RepoKit does not build a second
version of it.

RepoKit owns:

- exactly one repository-specific `hermes-<repo>` container;
- the `/workspace` and `.hermes` (`/opt/data`) mounts;
- the generated launcher and host command;
- the seven-profile bootstrap: default, researcher, planner, executor, tester, reviewer, steward;
- Kanban defaults and the default gateway dispatch policy;
- the repository development toolchain;
- safe bootstrap (ownership, private state, conservative reruns) and observational verify.

RepoKit does not own:

- memory providers, including OpenViking: configuration, endpoints, credentials,
  behavior and verification;
- models and providers;
- plugins;
- messaging integrations;
- native Hermes setup flows.

RepoKit does not know OpenViking exists. To use a shared memory provider, the owner
uses native Hermes, for example `<launcher> -p default memory setup` and
`<launcher> memory status`.

## Removed

- OpenViking from the development image (build stage, s6 service,
  `repokit-openviking` helper, `REPOKIT_OPENVIKING`), from generated Compose, and
  from install (no `.hermes/openviking` directory).
- `setup --memory` and `verify --memory-check`. Plain `setup` has no memory stage.
  Commands are `plan`, `install`, `setup [--team]` and `verify [--dispatch-check]`,
  plus `--docker-tests` for `plan`/`install`.
- `CORE_READY`, `MEMORY_READY`, `FULL_READY`, `MEMORY` and every `openviking*` or
  `memory` probe. `verify` reports two summaries: `CORE_TEAM` (container, native
  state, profiles, toolchain, Kanban, filesystem/SELinux access) and `DISPATCH`
  (gateway, Kanban dispatch/notification policy, channels; `unqualified` until a
  same-card independent review is observed). It exits nonzero only when either is
  `degraded`.
- The plan JSON's OpenViking section, `proposed_memory_config_not_activated`, and
  the `openviking` candidate image.

## No migration

Deployments generated with embedded OpenViking (`REPOKIT_OPENVIKING` in
`.hermes/compose.yaml`), with the older OpenViking sidecar, or with the Laya stack
are no longer recognized. `install` refuses them like owner-edited Compose, and
also refuses a `.hermes/` whose `compose.yaml` was removed, so there is currently no
supported upgrade path. Such deployments keep running on the RepoKit version that
generated them until an upgrade path is decided (open follow-up).

Native state (config, profiles, `kanban.db`, `.env`) is untouched. Any
`.hermes/openviking` data stays on disk and RepoKit ignores it. Profiles that
Hermes already configured with an OpenViking provider keep that Hermes
configuration; manage it with native `hermes memory setup` / `hermes memory status`.

## Consequences

- Readiness no longer depends on, reports or qualifies memory.
- Memory qualification, provider pins and OpenViking deployment guidance leave
  RepoKit's documentation; earlier records under `docs/superpowers/` and dated
  qualification files remain history.
- Running OpenViking as a separate service, if wanted, is a Hermes/owner concern,
  not a RepoKit-managed Compose service.

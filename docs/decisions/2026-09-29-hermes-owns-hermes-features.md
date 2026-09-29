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
  `memory` probe. `verify` reports RepoKit's contract only: `CORE_TEAM` (container,
  native state, profiles, Kanban, filesystem/SELinux access), `DEVELOPMENT_RUNTIME`
  (worker toolchain, Python import safety), `HOST_LAUNCHER` and `DISPATCH`
  (gateway, Kanban dispatch/notification policy, channels; `unqualified` until a
  same-card independent review is observed). It exits nonzero only when a summary
  is `degraded`.
- Steward's ownership of models/providers, memory providers and OpenViking
  endpoint/account/user configuration. Steward owns the RepoKit team itself:
  roster, role definitions, managed SOULs, team skills and tool boundaries,
  specialist creation and retirement. The exact previous managed steward SOUL is
  upgraded by `install`; an owner-edited one is preserved and reported as drift.
- The plan JSON's OpenViking section, `proposed_memory_config_not_activated`, and
  the `openviking` candidate image.

## No migration

RepoKit is pre-v1 and embedded OpenViking is development history, not a
compatibility contract. `install` does not recognize Compose generated with
embedded OpenViking, the older OpenViking sidecar or the Laya stack, and refuses
it like owner-edited Compose. `install` is not changed to support a manual upgrade
procedure for that topology. The one existing dogfood deployment is migrated as
separate, one-off qualification work, recorded under `docs/qualification/`, not
as part of the supported product.

## Consequences

- Readiness no longer depends on, reports or qualifies memory.
- Memory qualification, provider pins and OpenViking deployment guidance leave
  RepoKit's documentation; earlier records under `docs/superpowers/` and dated
  qualification files remain history.
- Running OpenViking as a separate service, if wanted, is a Hermes/owner concern,
  not a RepoKit-managed Compose service.

---
name: skill-hermes-repokit
description: Use when installing, resuming, verifying, migrating, or operating Hermes RepoKit in a repository, including the generated launcher, six-profile team, Kanban dispatch, embedded OpenViking, channel parity, and repository development runtime. Not for unrelated Hermes installations or developing RepoKit features.
---

# Hermes RepoKit in repositories

RepoKit is a short-lived Go bootstrapper. Its commands are `plan`, `install`,
`setup`, and `verify`. Ordinary Docker Compose and the generated native Hermes
launcher own runtime usage after bootstrap; RepoKit can then be removed.

## Readiness model

Report capabilities independently; container health is not component readiness:

- **Core team:** Hermes, six profiles, shared Kanban, routing and dispatcher.
- **Development runtime:** repository mounted at `/workspace` with its required toolchain.
- **Memory:** embedded OpenViking, reported as healthy, degraded, pending or unknown.
- **Channels:** CLI and configured human-facing adapters route to `default`.

`CORE_READY` means the core team, automatic dispatch and required development
runtime work, the researcher canary passed, and configured channels meet their
readiness checks. `FULL_READY` adds authenticated OpenViking readiness. These are
reporting labels; do not assume the selected CLI emits them. Memory recall,
reviewer independence and end-to-end delivery still need behavioral evidence.

Core readiness must not depend on OpenViking availability. An observed working
project may be `CORE_READY` with memory degraded or pending. If the selected
version still blocks dispatch on memory, report the implementation gap and actual
dispatch failure; do not claim readiness or bypass activation. See [version and
failure handling](references/installation.md#readiness-and-version-gaps).

## Route the request

- Install, resume setup or migrate: read [installation](references/installation.md).
- Chat, profiles, Kanban, memory, service recovery, or acceptance checks:
  read [usage](references/usage.md).
- Status-only requests stay observational. Reading or installing this skill
  does not authorize deploying Hermes. An explicit installation request authorizes
  ordinary non-destructive bootstrap and generated-service startup for that target
  repository. Secret-bearing setup, destructive cleanup, legacy-state deletion,
  commit and push remain separately scoped.

Use the user's chosen repository, otherwise resolve the current Git root.
Keep the **target repository** separate from the **RepoKit source checkout**.
Run bootstrap commands from the target root: this CLI uses its current directory
as the target and has no repository-path argument. Record Git branch, HEAD,
dirty/untracked paths, and a status hash before mutation. Preserve owner work.

Read the target's instructions and the selected RepoKit revision's README and
bootstrap quickstart. Confirm the installed CLI with `--help` and use matching documentation when
versions differ. Do not assume `--version` or release binaries exist.

## Deployment contract

- Normal production topology is exactly one RepoKit runtime container per
  repository: `hermes-<repo>`, using the normalized name printed by `plan`.
  OpenViking is an internal supervised process, not a Compose sidecar.
- Repository root mounts at `/workspace`; private `<repo>/.hermes` mounts at
  `/opt/data`, with `HERMES_HOME=/opt/data`. Preserve this generated layout.
- Install safely exposes the generated launcher at `~/.local/bin/hermes-<repo>`
  through a symlink, preserving conflicts and reporting missing PATH. It creates
  no shell aliases and edits no shell startup files. See [host command handling](references/installation.md#automatic-host-command).
- Use generated pins, build inputs and Docker context. An explicitly selected
  isolated Docker acceptance daemon is test infrastructure, not another production runtime.
- Coding readiness requires the target repository's toolchain inside `/workspace`;
  terminal/file tools alone are insufficient.
- Permanent profiles: `default`, `researcher`, `planner`, `executor`, `reviewer`,
  `steward`. Default coordinates; steward manages team evolution.
- Configured human-facing channels such as Telegram route to `default` and retain
  core repository-development, Kanban and memory capabilities expected from CLI.
  Setup owns parity; do not require manual per-channel Kanban enablement. Memory
  capability can remain available while its service is reported degraded.
- Kanban is canonical. Successful setup gates operational dispatch on core
  Hermes/team readiness and a real researcher canary. OpenViking readiness is
  reported separately and does not block core team execution.

```text
install → dispatch off
setup/reconcile → core prerequisites ready
gateway claims no-write researcher canary → completion observed
operational → automatic execution and review on
```

Setup starts the gateway dispatcher to run its canary; operational status requires
observing that gateway claim and researcher completion without manual dispatch.
Concurrency stays one and automatic decomposition stays off.

## Repair and migration boundaries

Durable fixes belong in RepoKit source or native Hermes configuration. Do not
hand-edit generated Compose, managed SOULs, platform tool checkboxes or live
`.hermes` state merely to make acceptance pass. A source fix requires development
scope; this operating skill does not implicitly authorize a product rewrite.

Preserve owner state on legacy topology detection and use the current RepoKit
reconciliation path. Do not manually delete legacy containers before replacement
configuration is generated. If that topology is unsupported, report the exact
gap and required migration work; never invent a migration command or teardown.

## Distinguish bootstrap from native usage

| Intent | Interface |
| --- | --- |
| Inspect/publish repository deployment | `hermes-repokit plan` / `install` |
| Private default setup, team and memory | `hermes-repokit setup` |
| Resume an integration | `hermes-repokit setup --team` or `hermes-repokit setup --memory` |
| Observe installation | `hermes-repokit verify` |
| Open native default chat | Generated `hermes-<repo>` with no arguments |
| Native Hermes setup only | `hermes-<repo> setup` |
| Native operations | `hermes-<repo> kanban list`, `plugins list`, `profile list` |
| Start/restart services | Ordinary Compose using captured routing |

Private setup belongs in the owner's terminal. Never capture credentials,
borrow unrelated host authentication, or invent provider/model choices.
Existing state, scanner refusal, identity collisions, or owner drift must be
resolved on their evidence; do not erase state or force admission to proceed.

Report bootstrap, host launcher, default chat, profiles, Kanban, memory and
acceptance separately with evidence and the next action.
Healthy services do not prove live model work, shared recall, reviewer
independence, or removal-first acceptance. Independent review requires execution
history with different implementation and approval actors; `done` alone is
insufficient. Leave Git delivery to an explicit commit/push request.

RepoKit prepares; Hermes operates; Docker contains; Kanban coordinates;
OpenViking remembers; Git remains under owner control.

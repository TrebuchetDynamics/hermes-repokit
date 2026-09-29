---
name: skill-hermes-repokit
description: Use when installing, resuming, verifying, migrating, or operating Hermes RepoKit in a repository, including the generated launcher, seven-profile team, Kanban dispatch, channel parity, and repository development runtime. Not for unrelated Hermes installations or developing RepoKit features.
---

# Hermes RepoKit in repositories

RepoKit is a short-lived Go bootstrapper. Its commands are `plan`, `install`,
`setup [--team]`, and `verify [--dispatch-check]`. It prepares a repo-specific
Hermes environment; Hermes handles Hermes features such as memory providers,
models, plugins and messaging. Ordinary Docker Compose and the generated native Hermes
launcher own runtime usage after bootstrap; RepoKit can then be removed.

## Readiness model

Report capabilities independently; container health is not component readiness:

- **Core team:** Hermes, seven profiles, shared Kanban, routing and dispatcher.
- **Development runtime:** repository mounted at `/workspace` with its required toolchain.
- **Channels:** CLI and configured human-facing adapters route to `default`.

`verify` prints `CORE_TEAM` and `DISPATCH` first. `CORE_TEAM` covers the container,
native config, launcher, profiles, toolchain, Kanban and mount access. `DISPATCH`
covers the gateway, dispatch/notification policy and channel tools; it is
`healthy` only once native card history shows same-card executor→tester→reviewer
completion, and `unqualified` means configured but the loop is not yet observed.
`verify` exits nonzero only when either is `degraded`. End-to-end channel delivery
still needs behavioral evidence.

Memory is not RepoKit readiness. Shared memory providers are configured and
inspected with native Hermes (`hermes-<repo> -p default memory setup`,
`hermes memory status`). See [version and
failure handling](references/installation.md#readiness-and-version-gaps).

## Route the request

- Install, resume setup or migrate: read [installation](references/installation.md).
- Chat, profiles, Kanban, service recovery, or acceptance checks:
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
  RepoKit adds no memory service or sidecar.
- Repository root mounts at `/workspace`; private `<repo>/.hermes` mounts at
  `/opt/data`, with `HERMES_HOME=/opt/data`. Preserve this generated layout.
- Existing repository Compose files and services coexist with RepoKit's explicit
  `.hermes/compose.yaml` and separate project namespace; preserve the owner's stack.
  Source builds of the bootstrap use `CGO_ENABLED=0` and need no host C compiler.
- Install safely exposes the generated launcher at `~/.local/bin/hermes-<repo>`
  through a symlink, preserving conflicts and reporting missing PATH. It creates
  no shell aliases and edits no shell startup files. See [host command handling](references/installation.md#automatic-host-command).
- On SELinux-enabled Linux hosts, `install` requests Docker's private `Z`
  relabeling for the `<repo>` and `<repo>/.hermes` bind mounts. Treat this as
  normal installation compatibility; never disable SELinux, change global policy,
  or hand-edit generated Compose. See [SELinux bind mounts](references/installation.md#selinux-bind-mounts).
- Use generated pins, build inputs and Docker context. An explicitly selected
  isolated Docker acceptance daemon is test infrastructure, not another production runtime.
- Coding readiness requires the target repository's toolchain inside `/workspace`;
  terminal/file tools alone are insufficient.
- Permanent profiles: `default`, `researcher`, `planner`, `executor`, `tester`,
  `reviewer`, `steward`. Default coordinates; steward manages team evolution.
- Configured human-facing channels such as Telegram route to `default` and retain
  core repository-development, Kanban and memory capabilities expected from CLI.
  Setup owns parity; do not require manual per-channel Kanban enablement.
- Kanban is canonical. Setup enables native dispatch on `default` through
  `hermes config set` and one gateway restart after the team reconciles. It never
  runs a worker itself.

```text
install → dispatch off
setup → team reconciled → native dispatch policy set → gateway restarted
verify --dispatch-check (explicit, paid) → gateway claims no-write researcher card
real task → executor → same-card tester → reviewer → verify shows review evidence
```

Setup refuses while a card is running and preserves an owner-changed dispatch
policy. Concurrency stays one and automatic decomposition stays off.

## Repair and migration boundaries

Durable fixes belong in RepoKit source or native Hermes configuration. Do not
hand-edit generated Compose, managed SOULs, platform tool checkboxes or live
`.hermes` state merely to make acceptance pass. A source fix requires development
scope; this operating skill does not implicitly authorize a product rewrite.

Preserve owner state on legacy topology detection and use the current RepoKit
reconciliation path. Do not manually delete legacy containers before replacement
configuration is generated. If that topology is unsupported, report the exact
gap and required migration work; never invent a migration command or teardown.
Embedded-OpenViking, sidecar and Laya deployments are refused; follow the
documented manual move in [installation](references/installation.md#legacy-topology-migration).

## Distinguish bootstrap from native usage

| Intent | Interface |
| --- | --- |
| Inspect/publish repository deployment | `hermes-repokit plan` / `install` |
| Private default setup and team | `hermes-repokit setup` |
| Resume team provisioning | `hermes-repokit setup --team` |
| Observe installation | `hermes-repokit verify` |
| Open native default chat | Generated `hermes-<repo>` with no arguments |
| Native Hermes setup only | `hermes-<repo> setup` |
| Memory provider | `hermes-<repo> -p default memory setup` / `memory status` |
| Native operations | `hermes-<repo> kanban list`, `plugins list`, `profile list` |
| Start/restart services | Ordinary Compose using captured routing |

Private setup belongs in the owner's terminal. Never capture credentials,
borrow unrelated host authentication, or invent provider/model choices.
Existing state, scanner refusal, identity collisions, or owner drift must be
resolved on their evidence; do not erase state or force admission to proceed.

Report bootstrap, host launcher, default chat, profiles, Kanban and
acceptance separately with evidence and the next action.
Healthy services do not prove live model work, reviewer
independence, or removal-first acceptance. Independent review requires execution
history with different implementation and approval actors; `done` alone is
insufficient. Leave Git delivery to an explicit commit/push request.

RepoKit prepares; Hermes operates; Docker contains; Kanban coordinates;
Git remains under owner control.

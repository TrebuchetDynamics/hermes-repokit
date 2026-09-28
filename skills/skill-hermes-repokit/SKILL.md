---
name: skill-hermes-repokit
description: Use when installing, resuming, verifying, or using Hermes RepoKit in a repository, including its generated host launcher, native team setup, embedded OpenViking memory and the development toolchain. Not for unrelated Hermes installations or developing RepoKit features.
---

# Hermes RepoKit in repositories

RepoKit is a short-lived Go bootstrapper. Its commands are `plan`, `install`,
`setup`, and `verify`. Ordinary Docker Compose and the generated native Hermes
launcher own runtime usage after bootstrap; RepoKit can then be removed.

## Route the request

- Install or resume setup: read [installation](references/installation.md).
- Chat, profiles, Kanban, memory, service recovery, or acceptance checks:
  read [usage](references/usage.md).
- Status-only requests stay observational. Reading or installing this skill
  does not authorize deploying Hermes. An actual installation request covers
  ordinary bootstrap/start/setup preparation; reuse existing authorization.

Use the user's chosen repository, otherwise resolve the current Git root.
Keep the **target repository** separate from the **RepoKit source checkout**.
Run bootstrap commands from the target root: this CLI uses its current directory
as the target and has no repository-path argument. Record Git branch, HEAD,
dirty/untracked paths, and a status hash before mutation. Preserve owner work.

Read the target's instructions and the selected RepoKit revision's README and
bootstrap quickstart. Confirm the installed CLI with `--help` and use matching documentation when
versions differ. Do not assume `--version` or release binaries exist.

## Deployment contract

- One Hermes runtime per repo; use the container/launcher names printed by
  `plan`, normally `hermes-<normalized-repo-basename>`.
- Repository root mounts at `/workspace`; private `<repo>/.hermes` mounts at
  `/opt/data`, with `HERMES_HOME=/opt/data`. Preserve this generated layout.
- Normal runtime: one Hermes development container with official OpenViking
  embedded under native s6. Use the generated pins, build inputs and Docker context.
  An isolated Docker acceptance daemon is optional and explicitly selected.
- Permanent profiles: `default`, `researcher`, `planner`, `executor`, `reviewer`,
  `steward`. Default coordinates; steward manages team evolution.
- Kanban is canonical. Initial dispatch and auto-decomposition are off;
  `max_in_progress=1`. Successful setup gates operational dispatch on native
  readiness, shared memory and a real researcher canary.

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
independence, or removal-first acceptance. Leave Git delivery to an explicit
commit/push request.

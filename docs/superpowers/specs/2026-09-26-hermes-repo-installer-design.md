# Repository-local Hermes lifecycle toolkit

Date: 2026-09-26
Status: approved for implementation planning, incorporating the user's subsequent layout, manifest, plugin-security and ownership amendments. Implementation execution is not yet selected.

## 1. Goal and hard requirements

Build a standalone toolkit in `hermes-repo-multiprofiles-temporary-name/` that installs and maintains Docker-based Hermes for a repository without requiring Pi to assemble deployment artifacts.

1. **Exactly one Hermes runtime container per repository. This is a hard rule.** Preparation, setup, integration installation and normal operation use the same container. Replacement may temporarily leave zero containers, never two.
2. Start with the native **default profile and a persistent usable Kanban board**. No specialist roster is provisioned initially.
3. Install **obra/superpowers** through the native Hermes plugin integration, pinned to an approved full 40-character commit SHA and subject to Hermes's scanner.
4. **All persistent deployment state belongs to `<repo>/.hermes` on the host.** Container-side Unix HOME need not equal `/workspace/.hermes`.
5. Use exactly six public commands: **plan, install, setup, activate, status, apply**.
6. Reconcile explicit desired state while preserving credentials, sessions, memories, plugins, board history and unrelated repository files.

The user-approved direction supersedes the earlier spec's forced `/workspace/.hermes` container HOME, automatic stable-image upgrade policy, recreation-only `apply`, and implied need for a derived image. The reference skill remains useful for preservation and verification, but conflicting layout/upgrade policies do not override this spec. Do not change the sibling `pi-toolset` skill as part of this project.

## 2. Scope and environment

Initial supported hosts are Linux with a local Docker daemon, a regular Git checkout on a supported local filesystem, Git, Node.js 22 or newer, util-linux `flock`, and Docker Compose 2.30 or newer. No host Hermes or Python installation is required. Docker must already be usable; do not install it, enable services, add privileged groups or switch contexts automatically.

Use Node built-ins for orchestration/tests, a small held-lock wrapper, and native Hermes commands or revision-bound adapters inside the container. Do not build a generic policy/workflow engine or implement Hermes's plugin scanner, scheduler or credential system again.

V1 excludes public web/dashboard exposure, Docker socket access, host shell edits, additional skill packs, specialist provisioning, unattended jobs, model calls, real messages, uninstall and automatic legacy-state migration. Existing active teams or foreign/ambiguous installations are preserved and reported unsupported, not reset to fresh defaults.

Holographic memory remains the earlier reference-derived integration target, separate from Superpowers. It must use supported persistent installation and honest capability reporting; it is not permission to patch the image to satisfy an arbitrary path rule. Task-specific development readiness is reported separately from base runtime readiness.

Creating the toolkit is not authority to deploy it to a production repository. Runtime qualification uses explicitly authorized disposable fixture repositories, no production state or credentials.

## 3. Physical layout and process identity

```text
host repository/                       container
  source and .git/  ------------------> /workspace
  .hermes/         ------------------> /opt/data
    config.yaml
    .env, auth.json
    state.db, kanban.db
    plugins/, profiles/, memories/
    compose.yaml
    identity.json, setup-state.json
    toolkit/manifest.json
    toolkit/transactions/, backups/
```

Compose declares two absolute bind mounts with `create_host_path: false`: canonical repo → `/workspace`, canonical `<repo>/.hermes` → `/opt/data`. Explicitly mounting `/opt/data` prevents an anonymous image-declared volume from becoming the authoritative state store. No extra home volume, shared global state or host credential mount.

Use:

```text
HERMES_HOME=/opt/data
HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace
```

Let the official bootstrap manage Unix HOME, UID/GID dropping and supported tool subprocess homes, including `/opt/data/home` where the selected version uses it. Pass the actual nonroot host IDs through the supported image environment, not a Compose `user:` override that bypasses bootstrap. Keep `/opt/hermes` immutable. Verify effective gateway, CLI and tool contexts rather than forcing all three to have identical Unix HOME values.

The native `default` profile resolves to the base application home `/opt/data`, not `/opt/data/profiles/default`. Set the actual terminal backend/workspace to local `/workspace`; gateway process cwd is not proof of tool cwd.

Because `.hermes` is also visible through the repository bind at `/workspace/.hermes`, both paths expose the same underlying host files. Use `/opt/data` as the canonical state path for configuration, receipts, SQLite, plugins and board resolution. Avoid duplicate plugin discovery through the project alias; keep project-plugin opt-in disabled unless separately justified and qualified. This alias is not a second state store or a security boundary.

`HERMES_WRITE_SAFE_ROOT` is an application file-tool policy, not an OS sandbox. Verify actual write/edit enforcement, but do not claim it confines shell commands or native plugin code to those roots. No container administrator or same-UID hostile-process protection is promised.

## 4. Standalone implementation boundaries

Suggested modules:

```text
bin/hermes-repo.mjs
src/cli/                 six-command parsing and reports
src/identity/            canonical target and ownership
src/manifest/            validated desired state and drift
src/storage/             protection, locks, transactions and receipts
src/runtime/             Docker argv, metadata and qualified adapters
src/integrations/        Kanban, Superpowers, identity and memory
src/lifecycle/           ordered command operations
adapters/                exact reviewed source/image contracts
scripts/with-lock.sh
scripts/qualify-runtime.mjs
tests/
docs/
```

Reuse existing helpers only after license/provenance checks. Bundle any adaptations here; no runtime imports from the sibling `pi-toolset` checkout. Existing receipt formats are compatibility inputs, not automatic adoption authority. Keep known version-1 identity/setup fields and preserve unknown supported fields; store new desired-state/transaction schemas under `toolkit/`.

## 5. Desired state and immutable revisions

A release manifest supplied with the toolkit describes qualified defaults. A per-repo private `.hermes/toolkit/manifest.json` records that repository's selected desired state. It is declarative data, never executable adapter paths, shell snippets, credentials or permission to bypass safety checks.

Required fields:

- Schema version, canonical repository identity and selected Docker context.
- Hermes upstream tag, immutable image reference/digest, target platform, reported Hermes version, source revision when established, and bundled adapter ID/version.
- Superpowers canonical source `obra/superpowers`, exact 40-character commit SHA and intended enablement.
- Layout `/workspace` + `/opt/data`, native profile `default`, and owned workspace/write-policy settings.
- Manual Kanban mode and selected interface settings without secrets.
- Required integration choices and bounded readiness policies.

First install resolves and records the toolkit-approved image selection after source/platform verification. `plan` can fetch public metadata read-only, but cannot persist a manifest or promote a discovered tag. `install` accepts a desired manifest or chooses bundled qualified defaults for a fresh instance. Existing instances reuse their selected manifest. A newer toolkit version or moved `latest` tag does not change an installed repo's desired state.

`apply --manifest PATH` explicitly selects a changed desired manifest; plain `apply` reconciles the existing selection. Plan the same selection with `plan --manifest PATH`. Both image and plugin upgrades must appear as immutable changes in that selected manifest. A reference to an unknown/unqualified image or adapter blocks mutation. Never fabricate digest/source provenance from an OCI label alone.

Preserve the prior manifest, generated config and compatible recovery information before upgrades. State-changing upgrades require quiescence and consistent state backups, not live copies of WAL files. Retain the old image identity and source state. Rollback is not automatic: a new runtime may have migrated data, so restoring an old image alone is not a safe rollback claim. Unsupported migration/recovery blocks before disruption.

## 6. Six-command contract

All commands accept `--repo PATH`; omission resolves the current Git root. Outside Git, report the required path. Accept an explicit Docker context for discovery/fresh installation; installed context changes are conflicts, not automatic retargets. Reject unknown/conflicting arguments. Use absolute executable/Compose paths and literal argv, never shell interpolation of user data.

| Command | Behavior |
| --- | --- |
| `plan` | Strictly read-only repo/state/Docker discovery, manifest and image/plugin-pin resolution, ownership/drift analysis and exact proposed install/apply changes. No pulls, writes, plugin installation, DB initialization or recreation. |
| `install` | Acquire lock, protect state, create/reconcile owned infrastructure, pull/verify the selected image, establish the one maintenance container, initialize default and Kanban manual mode, provision approved integrations when safe. Leave credentials untouched; fresh setup ends with one private setup command. A second unchanged install is a no-op, not a re-pull, wizard or restart. |
| `setup` | TTY-only user-owned native setup in the same container/default home with verified IDs and environment. No secrets in toolkit arguments, logs or receipts. Coordinate wizard lifetime; reject active uncoordinated writers. |
| `activate` | Validate setup, integration and access gates, then start normal selected services. Missing credentials block cleanly. Retain `dispatch_in_gateway: false` and `auto_decompose: false`. No implicit board work or test messages. |
| `status` | Read-only health/drift inspection: container cardinality, immutable image, mounts, IDs, permissions, version, gateway, credential availability without values, plugin pin/enablement, board health, manual mode, write policy and receipt consistency. Unknown/unsupported checks remain pending. |
| `apply` | Idempotently reconcile an already-managed repo to its explicitly selected manifest. Preserve lifecycle intent: maintenance is not activation. Image/plugin changes are allowed only when declared, qualified and safely recoverable. Refuse missing/foreign ownership rather than creating a fresh instance. |

Every Docker command selects the verified context. Compose calls also specify `--env-file /dev/null`, exact project and absolute file; neutralize ambient Compose selectors. Do not use raw logs, full inspect output or expanded secret-bearing configuration as diagnostics.

CLI result codes: `0` command completed, `2` invalid usage, `3` blocked/unsupported, `4` lock busy, `5` failed/timed out, `10` expected owner action pending. `install` awaiting private setup or scanner approval returns `10`; `status` returns `0` only for its applicable required readiness gates, otherwise a specific nonzero result. Developer-task readiness is a separate verdict, not a reason to falsely mark a healthy runtime broken. Existing internal probe protocols require explicit translation.

Generate a verified repo-local native CLI launcher as a convenience, without adding public toolkit commands or host aliases. Only advertise interactive chat once a session-safe mapping exists. CLI-only mode can be ready without a configured channel; gateway/channel status then says not configured, not connected. No web service is enabled to manufacture readiness.

## 7. Ownership and single-container enforcement

Derive the full repo ID from the canonical checkout path using SHA-256. V1 repo ID and path hash may have the same value; neither is a secret or cryptographic authorization. A repo move requires explicit future migration rather than treating the copied receipt as local ownership.

Human-readable container name: `hermes-<normalized-full-repo-basename>`. Normalize lowercase ASCII alphanumeric/hyphen, trim hyphens, use `repo` when empty, never truncate or append a collision suffix. Compose project IDs remain hashed for internal isolation.

Required deterministic labels:

```text
io.hermes-repo-toolkit.managed=true
io.hermes-repo-toolkit.repo-id=<full canonical repo ID>
io.hermes-repo-toolkit.repo-path-hash=<full canonical path hash>
```

Labels are mandatory ownership selectors, but labels alone are not adoption authority. Compare them with the protected identity receipt, canonical host paths, actual mounts, selected context and recorded resource identity. Inspect all managed containers claiming the repo ID plus exact-name/mount conflicts, including stopped containers. Multiple claims block; a matching old container is not permission to create a new one beside it.

Lock the repository before mutation and recheck collisions immediately before Docker creation. The daemon's exact name reservation also closes same-name races across distinct repos. A second installer using the same repo must lose the lock or safely observe a completed no-op.

No `compose run`, sidecar setup container, second gateway container, scaling, temporary real-home probe container or blue/green replacement. If Compose recreation creates both old and replacement containers even briefly in the selected version, use a qualified stop/remove-owned-container/create sequence instead. Removal is container-only after quiescence; preserve state, volumes and networks needed for recovery. Recheck ownership before every destructive resource operation. Never prune, use `down -v`, remove orphans or stop another stack.

## 8. Private files, locking and crash recovery

`.hermes` directories are private (`0700`); credential/config/manifest/receipt files are `0600`, owned launchers `0700`. Verify host application ownership and file types; reject symlinks, unexpected hardlinks, shared/unsafe ancestors and foreign state. Do not recursively chown source files.

Check Git ignore rules and index separately. Default-ignore runtime state, but never blindly hide intentionally tracked maintained documents. Add narrow owned rules or report the exact conflict. Tracked/staged credentials block private provisioning; do not untrack files automatically.

Keep installer `bootstrap.env` separate from native `.env`/`auth.json`. Raw Compose env-file support applies only to bootstrap values. Never source/eval these files, duplicate provider secrets, copy host auth, emit secret values or store secret hashes.

Use a stable private lock inode and OS `flock`, nonblocking, held through child operations. No directory-lock fallback, age-based stealing or unlinking a live lock. Pre-lock bootstrap is limited to safely creating the protected directory/lock after ignore/ownership checks. A trustworthy bootstrap/transaction record distinguishes this installer's interrupted preparation from unrelated residue; unexplained residue blocks.

Atomicity is per file. Stage bounded data, verify expected preimages/modes, fsync, publish with no-clobber creation or verified replacement, and write transaction completion last. Keep private backups of changed owned artifacts. No-op reruns preserve inode/mtime. Incomplete transactions block dependent launchers; resume forward only when actual files and external effects match the journal. Re-observe Docker after crashes: do not replay create/recreate because a receipt update failed.

Lock availability does not prove the gateway, wizard, workers or external CLI are quiescent. Observe all writers before state changes and backups. Preserve valid credential/session files and unknown supported receipt fields. Release locks before handing control to the user; the `setup` command reacquires and holds its own lock for the wizard.

## 9. Kanban: usable board, manual execution

Use the pinned Hermes revision's actual schema and native initialization APIs. Initialization/migration is a disclosed installation write, not a status query. Canonical default board is expected at `/opt/data/kanban.db`; verify overrides and resolved paths instead of opening a second DB under the alias.

New-install baseline:

```yaml
kanban:
  orchestrator_profile: default
  dispatch_in_gateway: false
  auto_decompose: false
  dispatch_profiles: []
  auto_subscribe_on_create: false
  notify_in_gateway: false
  default_workdir: /workspace
```

The first two boolean controls are independent and mandatory. Enable Kanban tools for default's CLI and chosen platforms; preserve unrelated toolsets and report conflicts with explicit owner disables. Default can create/read/update cards, links, comments and history. It is the human-facing coordinator, not an automatically dispatched worker.

An empty specialist allowlist means the initial default-only install cannot execute worker cards even with a manual dispatcher invocation. Later, an owner can deliberately provision/authorize a specialist and explicit work scope in this same container, then invoke native manual dispatch without enabling continuous gateway dispatch. That later release is not an installer side effect. Do not promise manual dispatch of a worker card works before a worker exists.

Do not use `dispatch --dry-run` or ordinary DB `connect()` as read-only health probes without source checks; upstream can reclaim/promote/time out work or initialize schema. Status uses a qualified nonmutating observer or reports unknown. No live SQLite WAL copies or `immutable=1` reads that silently ignore active WAL content.

Tests must prove ready cards do not auto-run. A separate opt-in fixture can authorize a test-only specialist and local stub worker/provider to verify deliberate native manual dispatch; label stub evidence honestly, with actual model execution untested. No specialists are added to production by that test.

## 10. Superpowers pinning and security admission

Use the native integration with the manifest's approved SHA:

```sh
hermes plugins install obra/superpowers --ref "$APPROVED_SHA" --enable
```

`APPROVED_SHA` must be a full 40-character hexadecimal commit, never `main`, a tag or abbreviated SHA. Verify installed provenance and enablement after the command; listing a plugin name alone is insufficient. Record exact source/revision, scanner version/verdict and bounded nonsecret findings in the receipt. Pin selection is not blanket trust approval.

Scanner policy:

| Result | Toolkit behavior |
| --- | --- |
| safe | Continue within selected installation scope. |
| caution | Stop with `PLUGIN_APPROVAL_REQUIRED`, source, SHA and redacted findings. No automatic force/yes, no enablement. |
| dangerous | Refuse with `PLUGIN_BLOCKED`; no override, even after an earlier caution approval. |
| missing, disabled, failed or unparseable scanner | Block admission; never treat absence as safe or silently change owner security settings. |

For caution, the owner can explicitly rerun `install`/`apply` with `--approve-plugin-sha SHA` after reviewing the reported findings. The option is a narrow owner decision, not a generic `--force`. Require a recorded matching caution report for this repo/source/SHA and unchanged scanner/findings/candidate tree. A changed report requires fresh review. Record the decision only after checking it; unrelated revisions never inherit it. Current dangerous findings always win.

Upstream `--force` can both replace an existing plugin and accept caution. Therefore upgrades need admission checks on the exact final candidate tree before any replacement path that uses that flag. Use a source-qualified native admission transaction/adapter; do not patch the scanner or assume replacement permission equals trust permission. Unknown unsafe replacement semantics block upgrades. Owner-modified plugin source blocks automated replacement rather than being silently merged into a different reviewed tree.

Keep plugin backing under `/opt/data/plugins`. Verify skills, startup hook, tool mapping and persistence, including the `/workspace/.hermes` alias collision. Upstream warns that active sessions need restarting and Hermes currently lacks a post-compaction bootstrap hook; preserve conversations and document when a fresh session is needed. Do not claim post-compaction restoration.

Plugin skills do not authorize extra containers, host access, automatic Kanban work, model costs or third-party capability grants. Native plugin capability requests require their own explicit decisions; a scanner caution approval is not a capability grant.

## 11. Image qualification, memory and runtime evidence

Prefer the official immutable image with `/opt/data`. Qualify the exact source/platform/image rather than treating inspected main as a stable release. Record upstream tag/digest/platform/version/source evidence. Do not build a derived image solely to change HOME, and do not build one during planning. Only a demonstrated failure of an actual required behavior after official-layout qualification can justify a separately reviewed derived-image proposal.

Verify init, privilege drop, maintenance quiescence, gateway lifecycle, HOME/HERMES_HOME, canonical mounts, package activation, immutable `/opt/hermes`, CLI/tool workspace and native plugin/Kanban interfaces. Missing source/provenance or unknown read-only semantics block the dependent capability. No dummy health checks or invented maintenance CMD.

Holographic checks use its full source-backed setup contract adapted to `/opt/data`. Distinguish startup dependencies, functional HRR, existing vector coverage, canary cleanup/persistence, loaded integration and recreation durability. No paid extraction/model probe; preserve new-install `auto_extract: false`. Proven basic behavior is reported as **basic keyword mode—not full HRR capability**, not silently accepted as full readiness. Existing provider choices are preserved and incompatible requests reported, not overwritten.

Readiness is observed, not inferred from artifact generation, credential-file existence, historical receipts or container uptime. Keep CLI, gateway/channel, Kanban board/tools, dispatch release, plugin bootstrap, memory, development and manifest-alignment verdicts separate. No model-reply claim from channel connection. Use bounded timeouts and output limits for every probe.

## 12. Acceptance and delivery

Normal tests use Node's test runner, temporary owned repos, fake Docker and subprocess fixtures. No live Docker, models or credentials by default. Acceptance includes:

- Fresh install; unchanged second install no-op; manifest-owned drift repaired by apply.
- Name, mount and duplicate-label conflicts, including stopped containers; no duplicate runtime container during any operation.
- Concurrent installers and SIGKILL around every publication/external-effect boundary; safe resume without duplicated effects.
- Credential privacy, ignored/indexed state conflicts, hostile links, copied receipts and owner-edited artifacts.
- Recreation and approved image upgrade retain sessions, auth, memory, plugin pins, board/card history and repository modifications.
- Scanner safe/caution/dangerous/disabled/failure cases, stale approvals, changed findings and force-replacement admission.
- Missing credentials block activation; gateway activation leaves both automatic Kanban controls false.
- Ready card does not auto-run; opt-in deliberately authorized manual dispatch works in a disposable fixture with its own test-only worker.
- Dirty Git/submodule state remains unchanged; recovery never deletes user state. Uninstall remains unsupported rather than exposing an untested destructive command.
- Safe status probes never create files, mutate a DB, refresh auth, install dependencies or trigger work.

Integration qualification needs explicit disposable-container scope and safe cleanup confined to resources created by the test. Removal of test containers must preserve their state/evidence until reviewed cleanup; no daemon-wide actions. No adapter is production-qualified solely by fake-Docker tests. Release a working installer only after at least one real official image/pin combination meets its required runtime gates; otherwise report the precise incomplete milestone.

Output should be concise: actual readiness/alignment, changed versus preserved state, pending owner action or concrete blocker, and one next command. Full nonsecret evidence belongs in private receipts, not large terminal dumps.

## 13. Evidence and precedence

Planning inspection confirmed the official Docker `/opt/data` layout and full-SHA plugin support:

- [Hermes Docker guide](https://hermes-agent.nousresearch.com/docs/user-guide/docker/).
- [Hermes plugin guide](https://hermes-agent.nousresearch.com/docs/user-guide/features/plugins/).
- [Superpowers native Hermes instructions](https://github.com/obra/superpowers#hermes-agent).
- Source snapshot `28e6496a5e3adfea57bebfc9571b981bff378523`: [Dockerfile](https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/Dockerfile), [main wrapper](https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/docker/main-wrapper.sh), [stage2](https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/docker/stage2-hook.sh), [scanner admission](https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins_cmd.py#L174-L210), [plugin installation](https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins_cmd_install.py).

This source snapshot is not an image qualification or approved Superpowers revision. Exact safe-root enforcement, production adapter mapping and live integration behavior remain implementation checks. No image was pulled/built, no plugin installed and no Docker deployment changed during planning.

The earlier reference skill's preservation/ownership and Kanban warnings remain useful; this spec's explicit host-state topology and declarative upgrade policy win where they differ. The destination is not currently a Git repository: documents are saved, not committed. Self-review checks include scope, immutable desired state, scanner admission, alias handling, single-container replacement and the distinction between manual board use and authorized worker execution.

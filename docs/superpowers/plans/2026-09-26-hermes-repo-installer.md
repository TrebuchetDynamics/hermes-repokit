# Repository-local Hermes Toolkit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a standalone, declarative toolkit that safely installs and reconciles one official Hermes container per repository, with default-profile manual Kanban and an immutable, scanner-approved Superpowers plugin.

**Architecture:** Node.js CLI with six commands, pure desired-state/drift calculations, private crash-recoverable artifacts, explicit Docker argv, and a small revision-bound in-container adapter. Persist host `<repo>/.hermes` at container `/opt/data`, keep source at `/workspace`, and preserve native credentials and user state. Production mutation requires a source-reviewed and runtime-qualified official-image adapter; no agent-dependent assembly or automatic derived-image build.

**Tech Stack:** Node.js 22+ ES modules and built-in test runner; Linux util-linux `flock`; Git; Docker Compose 2.30+; official Hermes; native pinned plugin installation. Any Python bridge runs only through the image's verified interpreter, never host Python.

**Spec:** [2026-09-26-hermes-repo-installer-design.md](../specs/2026-09-26-hermes-repo-installer-design.md). Read it with this plan. The user's amended layout/upgrade policy supersedes conflicting sibling skill instructions.

## Global Constraints

- Exactly one Hermes runtime container per repository; zero during replacement is acceptable, two are not.
- Two absolute binds: canonical repo → `/workspace`; canonical `<repo>/.hermes` → `/opt/data`; both disable host-path auto-creation.
- `HERMES_HOME=/opt/data`; `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`; honor official Unix/tool HOME semantics and immutable `/opt/hermes`.
- Native profile `default`; no initial named profiles, worker cards or automatic work.
- `kanban.dispatch_in_gateway: false` and `kanban.auto_decompose: false` are both required; initial `dispatch_profiles: []`.
- Exactly six public commands: plan, install, setup, activate, status, apply.
- Image upgrades require an explicitly selected immutable desired manifest. A moving upstream tag or new toolkit default does not upgrade an installed repo.
- Superpowers source `obra/superpowers`, full 40-character commit SHA; never silently bypass caution, dangerous, missing or disabled scanning.
- Linux/local daemon/regular checkout/supported local filesystem; Git, Node.js 22+, flock and Compose 2.30+; no host Hermes/Python requirement.
- No `compose run`, sidecar, Docker socket, automatic derived image, root runtime bypass, source cleanup, model call, message, or production deployment during development.
- Runtime qualification requires separately authorized disposable fixtures. Offline passing tests are not live qualification.
- Keep all private state under host `.hermes`; credentials/config/receipts `0600`, private directories `0700`, owned launchers `0700`.
- Do not modify the sibling `pi-toolset` checkout. Copy helper code only after confirming its license and retaining provenance.

## Review Focus

1. **Double-mounted state alias:** `/workspace/.hermes` and `/opt/data` must not create two DB identities or double-load a plugin. Pin in Tasks 5, 6 and 12.
2. **Force reinstall changes trust semantics:** replacement must not accept a caution result automatically. Pin exact final-tree admission and stale approval rejection in Task 7.
3. **Docker committed but receipt did not:** SIGKILL after creation/replacement must not produce a second container or repeat an upgrade. Pin observed-effect reconciliation in Tasks 3, 9 and 10.
4. **Active WAL or status-import side effects:** a read-like native command may initialize/migrate state; read-only status must neither lose WAL data nor create files. Pin in Tasks 4 and 11.
5. **Repo relocation and ambiguous stopped containers:** a copied home or old managed container must block adoption even when the expected new name is free. Pin in Tasks 2 and 5.

---

## Execution boundary and delivery sequence

The directory currently contains only documents and is not a Git repository. Before coding, use the worktree/isolation skill: confirm that execution uses this intended new project, initialize its Git history if authorized by the implementation handoff, and preserve existing docs. Commit steps below assume that repository exists; do not accidentally stage the sibling checkout. No commits or runtime tests have been performed by writing this plan.

This is one lifecycle subsystem, delivered in ordered, individually testable slices. Tasks 1–5 establish safe read-only planning and runtime qualification interfaces. Tasks 6–11 implement integrations and lifecycle operations. Task 12 supplies real runtime qualification and the final delivery gate. A partial milestone is useful but is not a completed installer.

Do not invent image digests, plugin SHAs, stable-source bindings, maintenance commands or health endpoints. Task 4 researches and records exact source-backed operations. Task 12 tests them in an authorized fixture. If a necessary native interface is unsupported, retain a precise blocker and stop that feature; do not substitute a dummy adapter or derive an image to force the old HOME policy.

### Shared contracts

Create JSDoc contracts in `src/contracts.mjs` with these exact shapes. All code snippets below are intended implementation/test content, not commands already run.

```js
export const COMMANDS = Object.freeze(['plan', 'install', 'setup', 'activate', 'status', 'apply']);
export const EXIT = Object.freeze({ ok: 0, usage: 2, blocked: 3, busy: 4, failed: 5, pending: 10 });
export const HOME = '/opt/data';
export const WORKSPACE = '/workspace';
export class ToolkitError extends Error {
  constructor(code, message, exitCode = EXIT.blocked) {
    super(message); this.code = code; this.exitCode = exitCode;
  }
}
```

- `Identity`: `{version:1, repoPath, repoId, pathHash, profileName, projectName, containerName}`. `repoPath` is host realpath; `profileName` is displayed basename, not the native profile ID.
- `Manifest`: `{version:1, identity, context, image:{tag,ref,digest,platform,hermesVersion,sourceRevision,adapterId}, superpowers:{source,sha,enabled:true}, layout:{workspace:'/workspace',home:'/opt/data'}, kanban:{dispatchInGateway:false,autoDecompose:false,dispatchProfiles:[]}, interfaces:string[], memory:{provider:'holographic',autoExtract:false,plugin:null|{source,sha}}}`. `sourceRevision` can be null only if the adapter's provenance rule independently establishes support; no invented source value. Reject unknown security-sensitive keys, malformed paths and mutable image selectors. Preserve future receipt fields separately.
- `Observed`: `{identity, context, containers:Container[], generated, runtime, integrations, writers, gates}`. `Container`: `{id,name,labels,imageId,imageRef,mounts,state}`. Every field is nonsecret and bounded. Missing data is explicitly unknown, not a default success.
- `Gate`: `{state:'pass'|'pending'|'blocked'|'unsupported', code, evidence}`. Evidence contains whitelisted nonsecret scalars/arrays only, never raw config/env/log output.
- `Action`: `{id,kind,target,disruptive,expected,desired}` with a closed `kind` enum; no executable strings. Kinds: `protect-state`, `publish-artifacts`, `pull-image`, `create-maintenance`, `configure`, `initialize-board`, `install-plugin`, `verify-memory`, `replace-service`, `start-gateway`, `record-evidence`.
- `Plan`: `{version:1,command,identity,desired,actions:Action[],blockers:Gate[],nextAction}`. `install`/`apply` recompute under lock; a saved plan is never authorization or a replayable command list.
- `Driver`: `{collect, assertQuiescent, pullImage, createMaintenance, observeEffect, configure, initializeBoard, installPlugin, verifyMemory, runSetup, replaceService, startGateway, verify}`; each method takes `{manifest,identity}` plus a typed method-specific payload and returns a `Gate` or `Observed`. `collect`/`verify({readOnly:true})` cannot call mutators. Production driver methods come only from bundled qualified adapters.
- `Result`: `{exitCode,code,runtime,alignment,development,integrations,changes,preserved,nextAction,plan?}`. Optional `plan` is the validated nonsecret `Plan` for read-only planning output. No returned stdout/stderr blobs.

### File responsibilities

| Files | Responsibility |
| --- | --- |
| `src/contracts.mjs`, `src/manifest/validate.mjs` | Typed constants/errors and nonsecret desired-state validation. |
| `src/cli/args.mjs`, `src/cli/report.mjs`, `bin/hermes-repo.mjs` | Six commands, bounded flags, exit codes and output. |
| `src/identity/resolve.mjs`, `src/identity/ownership.mjs` | Canonical repo identity, resource checks and no-adoption rules. |
| `src/runtime/process.mjs`, `src/runtime/docker.mjs` | Bounded spawn, literal argv, sanitized selectors and safe Docker observations. |
| `src/storage/protect.mjs`, `lock.mjs`, `atomic.mjs`, `journal.mjs`, `receipts.mjs` | Private filesystem gates, held lock and crash recovery. |
| `src/manifest/compose.mjs`, `drift.mjs`, `select.mjs` | Two-mount Compose rendering and deterministic desired/observed delta. |
| `src/runtime/adapters.mjs`, `adapters/official-s6/` | Exact binding, reviewed source-specific operations, in-container bridge and qualification metadata. |
| `src/integrations/kanban.mjs`, `identity.mjs`, `superpowers.mjs`, `memory.mjs` | Narrow integration policies; no lifecycle ownership duplication. |
| `src/lifecycle/plan.mjs`, `install.mjs`, `setup.mjs`, `activate.mjs`, `apply.mjs`, `status.mjs` | Explicit lifecycle flows, not a generic command engine. |
| `tests/helpers/fixture.mjs`, `fake-driver.mjs`, `fake-docker.mjs` | Temporary repo/manifest builders and recorded-effect doubles. |
| `scripts/qualify-runtime.mjs`, `tests/runtime/` | Explicitly opted-in disposable-runtime qualification. |

## Task 1: Validated desired state and six-command interface

**Files:** Create `package.json`, `.gitignore`, `src/contracts.mjs`, `src/cli/args.mjs`, `src/cli/report.mjs`, `src/manifest/validate.mjs`, `bin/hermes-repo.mjs`, `tests/manifest.test.mjs`, `tests/cli.test.mjs`, `tests/helpers/fixture.mjs`.

**Interfaces:** `parseArgs(argv) -> {command,repo,context,manifestPath,json,approvePluginSha,help}` (`command` may be absent only with `--help`); `validateManifest(input) -> Manifest` throws `ToolkitError`; `renderResult(result,{json}) -> string`. `manifestFixture()` produces only clearly synthetic test pins and never a production release manifest. `temporaryRepo()` creates a temporary Git repo and exposes async cleanup.

- [ ] **1. Write failing schema/parser tests.** Include the exact six commands, duplicate/unknown options, missing flag values, controls/NUL, abbreviated plugin pins and mutable image refs.

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { parseArgs } from '../src/cli/args.mjs';
import { validateManifest } from '../src/manifest/validate.mjs';
import { manifestFixture } from './helpers/fixture.mjs';
test('only exact immutable plugin pins are accepted', () => {
  for (const sha of ['main', 'v1', 'abc123', 'a'.repeat(39), 'g'.repeat(40)]) {
    const input = manifestFixture(); input.superpowers.sha = sha;
    assert.throws(() => validateManifest(input), { code: 'INVALID_PLUGIN_PIN' });
  }
});
test('public command set is closed', () => {
  assert.equal(parseArgs(['plan', '--repo', '/tmp/project']).command, 'plan');
  assert.throws(() => parseArgs(['destroy']), { code: 'INVALID_COMMAND' });
  assert.throws(() => parseArgs(['status', '--force']), { code: 'UNKNOWN_OPTION' });
});
```

- [ ] **2. Run red:** `node --test tests/manifest.test.mjs tests/cli.test.mjs`; expect missing modules first, then explicit failed assertions if a rule is omitted.
- [ ] **3. Implement closed schemas and parsing.** Use `node:util` `parseArgs({strict:true,allowPositionals:true,...})` with explicit duplicate detection, per-command option allowlists, and `ToolkitError`. Regexes: SHA `/^[a-f0-9]{40}$/`; digest `/^sha256:[a-f0-9]{64}$/`; official ref `/^nousresearch\/hermes-agent@sha256:[a-f0-9]{64}$/`. Reject secret-bearing fields rather than storing an arbitrary manifest object. Approval flag accepted only on install/apply, never plan/status. Define `npm test` as `node --test tests/*.test.mjs` and a separate explicit runtime script; no npm runtime dependencies.

```js
export function assertPin(sha) {
  if (typeof sha !== 'string' || !/^[a-f0-9]{40}$/.test(sha))
    throw new ToolkitError('INVALID_PLUGIN_PIN', 'Use a full 40-character commit SHA.', EXIT.usage);
  return sha;
}
```

Until lifecycle modules are implemented, valid mutation commands return `ADAPTER_UNAVAILABLE`, not success. `--help` describes planned commands without implying readiness.
- [ ] **4. Run green:** `npm test`; test rendering with synthetic `secret-canary` in an unexpected field and prove it is rejected rather than printed.
- [ ] **5. Commit:** `git add package.json .gitignore bin src/contracts.mjs src/cli src/manifest/validate.mjs tests; git commit -m "feat: define validated Hermes toolkit interface"`.

## Task 2: Canonical identity and safe Docker observations

**Files:** Create `src/identity/resolve.mjs`, `src/identity/ownership.mjs`, `src/runtime/process.mjs`, `src/runtime/docker.mjs`, `tests/identity.test.mjs`, `tests/docker.test.mjs`, `tests/helpers/fake-docker.mjs`.

**Interfaces:** `deriveIdentity(canonicalRepo) -> Identity`; `resolveRepo({repo,cwd,run}) -> Promise<Identity>`; `assertOwnership({identity,receipt,containers,context}) -> void`; `runProcess(file,args,{env,cwd,timeoutMs,maxBytes,stdio}) -> Promise<{code,stdout,stderr}>` internal-only; `composeArgs({context,projectName,composePath},tail) -> string[]`; `collectDocker(selectors,identity) -> Promise<Container[]>` with no raw secret-bearing output crossing the module boundary.

- [ ] **1. Write failing tests for stopped duplicates, copied receipts, paths containing `$`, quotes/spaces/Unicode, nested cwd and linked worktrees with outside Git metadata.** Reject unsupported checkout topology without mounting broad parents.

```js
import { deriveIdentity, resolveRepo } from '../src/identity/resolve.mjs';
import { assertOwnership } from '../src/identity/ownership.mjs';
test('a stopped second claim still blocks', () => {
  const identity = deriveIdentity('/tmp/repo');
  const labels = {
    'io.hermes-repo-toolkit.managed': 'true',
    'io.hermes-repo-toolkit.repo-id': identity.repoId,
    'io.hermes-repo-toolkit.repo-path-hash': identity.pathHash,
  };
  const containers = ['a', 'b'].map(id => ({ id, labels, state: 'exited' }));
  assert.throws(() => assertOwnership({ identity, receipt: null, containers, context:'default' }),
    { code:'DUPLICATE_REPO_CLAIM' });
});
```

- [ ] **2. Run red:** `node --test tests/identity.test.mjs tests/docker.test.mjs`.
- [ ] **3. Implement deterministic identity and narrow observation.** Use `realpath`, `git rev-parse --show-toplevel`, and inspect `.git` indirection before acceptance. Full path hash for repo ID/path hash; existing name normalization from the spec. List exact label/name candidates plus target-mount conflicts, then inspect only necessary fields; include stopped containers. Compare protected receipt, full labels and both host mount sources. Missing ownership is not creation permission.

```js
export function composeArgs(s, tail) {
  return ['--context', s.context, 'compose', '--env-file', '/dev/null',
    '-p', s.projectName, '-f', s.composePath, ...tail];
}
```

All subprocesses use `spawn(...,{shell:false})`; bounded output, timeout/kill escalation and signal propagation. Whitelist inherited environment needed by Docker credentials/config, but reject/neutralize Docker remote-host and Compose override selectors that contradict selected context. Do not echo environment, argv containing credentials or child stderr blindly.
- [ ] **4. Run green:** targeted tests plus `npm test`. Assert fake Docker never receives `run`, `--remove-orphans`, `down -v`, `prune`, `--privileged`, socket mounts or undeclared selectors.
- [ ] **5. Commit:** `git add src/identity src/runtime tests; git commit -m "feat: verify repository and Docker ownership"`.

## Task 3: Private storage, held locks and interruption-safe receipts

**Files:** Create `src/storage/protect.mjs`, `src/storage/lock.mjs`, `src/storage/atomic.mjs`, `src/storage/journal.mjs`, `src/storage/receipts.mjs`, `scripts/with-lock.sh`, `tests/storage.test.mjs`, `tests/lock.test.mjs`, `tests/crash.test.mjs`, `tests/helpers/lock-worker.mjs`.

**Interfaces:** `inspectProtection(identity) -> Promise<Gate[]>` read-only; `preparePrivateState(identity) -> Promise<void>` mutating; `withRepoLock(identity,workerArgs) -> Promise<number>` supervises a worker process holding the lock; `publishArtifacts({root,files,expected,transactionId}) -> Promise<{changed,transactionId}>`; `recoverTransaction({root,observeEffect}) -> Promise<Gate>`; `mergeReceipt(previous,patch) -> object` preserves unknown valid v1 fields.

- [ ] **1. Write failing filesystem tests.** Cases: symlinked home/dangling target/hardlink, mode mismatch, tracked `.env`, preserved tracked documentation exception, failed fsync/ENOSPC, prior content edited between staging and publication, unknown journal version, and identical bytes preserving inode/mtime. Use injected filesystem operations for deterministic faults; use real child processes for locks.

```js
import { publishArtifacts } from '../src/storage/atomic.mjs';
test('unchanged generated files do not churn', async () => {
  const fixture = await temporaryRepo();
  try {
    const root = fixture.privateRoot;
    const files = [{ path:'toolkit/manifest.json', bytes:'{}\n', mode:0o600 }];
    await fixture.preparePrivateRoot();
    await publishArtifacts({ root, files, expected:{}, transactionId:'first' });
    const before = await fixture.statPrivate('toolkit/manifest.json');
    await publishArtifacts({ root, files,
      expected:{'toolkit/manifest.json':{bytes:'{}\n',mode:0o600}}, transactionId:'second' });
    const after = await fixture.statPrivate('toolkit/manifest.json');
    assert.equal(after.ino, before.ino); assert.equal(after.mtimeMs, before.mtimeMs);
  } finally { await fixture.cleanup(); }
});
```

Define the helper methods used above in `tests/helpers/fixture.mjs`: `privateRoot`, `preparePrivateRoot()`, `statPrivate(relative)`, `cleanup()`. They never target a real repo.
- [ ] **2. Run red:** `node --test tests/storage.test.mjs tests/lock.test.mjs tests/crash.test.mjs`.
- [ ] **3. Implement allowlisted artifact paths, held lock and transaction phases.** Protect/validate ancestors; allow only exclusive private-dir/lock bootstrap before lock acquisition, then revalidate. Use `O_NOFOLLOW|O_CREAT|O_EXCL`, same-directory staging, fsync file and containing directory, expected-preimage comparison and manifest completion last. New-file publication must not clobber a racing destination. Never unlink/replace the lock inode.

```sh
#!/bin/sh
# Invoked only after parent path/inode protection; worker revalidates before writing.
set -eu
lock=$1
shift
exec 9<>"$lock"
flock -n -E 4 9 || exit "$?"
exec "$@"
```

The supervised worker and relevant children inherit FD 9 until they finish; test SIGKILL and orphan-child lifetime. Metadata is diagnostic, not lock authority. Transaction phases: `staged`, `artifacts-published`, `effect-observed`, `complete`; uncertain external effects require observation, not replay. A protected bootstrap receipt explains this toolkit's interrupted directory creation without adopting unexplained files. Secret files are excluded from generated-artifact hashing/logging.
- [ ] **4. Run green:** targeted tests plus `npm test`. Real two-process test must yield one holder and one exit 4. Kill before/after each publication point; assert old/new parseability, preserved user files and no false completion.
- [ ] **5. Commit:** `git add src/storage scripts/with-lock.sh tests; git commit -m "feat: protect private state and recover interrupted writes"`.

## Task 4: Source-bound official runtime adapter and qualification harness

**Files:** Create `src/runtime/adapters.mjs`, `adapters/official-s6/contract.json`, `adapters/official-s6/bridge.py`, `adapters/official-s6/README.md`, `scripts/qualify-runtime.mjs`, `tests/adapters.test.mjs`, `tests/fixtures/runtime/`, `docs/runtime-qualification.md`.

**Interfaces:** `selectAdapter(manifest,{qualification}) -> Driver` rejects unqualified production bindings; `assertBinding(manifest,evidence) -> void`; `qualifyRuntime({fixtureRepo,manifest,authorization}) -> Promise<QualificationReport>`; bridge requests `{version:1,operation,payload}` from a closed operation enum, one bounded JSON response. Do not accept arbitrary Python/shell code from manifests.

- [ ] **1. Add failing binding/side-effect tests.** A matching version string but wrong digest/platform/source fails. Missing maintenance support fails. Unknown native result fails. Read-only operation routing must reject mutation opcodes before executing Python.

```js
import { assertBinding } from '../src/runtime/adapters.mjs';
test('version similarity never qualifies another image', () => {
  const m = manifestFixture();
  const evidence = { imageRef:m.image.ref, platform:'linux/arm64',
    sourceRevision:m.image.sourceRevision, qualified:true };
  assert.throws(() => assertBinding(m, evidence), {code:'ADAPTER_BINDING_MISMATCH'});
});
```

- [ ] **2. Run red:** `node --test tests/adapters.test.mjs`.
- [ ] **3. Inspect exact upstream source before writing native mappings.** Resolve the approved stable release/platform image metadata and source binding read-only. Inspect Dockerfile, init/stage2, main wrapper, exec shim, s6 gateway/profile reconciler, config/profile resolution, file-write policy, Kanban init/status/dispatch, plugin installation/scanning/provenance and memory activation. Save small license-compliant fixtures plus source URLs/digests under `tests/fixtures/runtime/`; record exact commands and write/network effects in adapter README. The planning snapshot `28e6496a5e3adfea57bebfc9571b981bff378523` is evidence, not a selected release pin.
- [ ] **4. Implement binding and bridge using only those verified operations.** Required bridge operations: `observe-runtime`, `observe-config`, `observe-plugin`, `observe-board`, `assert-quiescent`, `configure-owned`, `initialize-board`, `admit-plugin`, `verify-memory-local`. Whitelist output keys. Read-only observations must not import a module whose top level initializes DB/plugins or contacts a model. Do not use `hermes doctor`, plugin listing or `kanban connect` without auditing their side effects. Run bridge only as verified nonroot IDs using image Python and startup-equivalent environment.

```js
export function assertBinding(manifest, evidence) {
  if (evidence.imageRef !== manifest.image.ref ||
      evidence.platform !== manifest.image.platform ||
      evidence.sourceRevision !== manifest.image.sourceRevision)
    throw new ToolkitError('ADAPTER_BINDING_MISMATCH', 'Image/source/platform binding differs.');
  if (evidence.qualified !== true)
    throw new ToolkitError('RUNTIME_NOT_QUALIFIED', 'Runtime qualification is required.');
}
```

The real binding additionally verifies provenance, adapter ID/version and reviewed fixture digests, not just strings. Qualification bypass exists only in the explicitly authorized disposable test harness, never as a production CLI flag. The harness may instantiate a source-reviewed candidate driver with an in-memory fixture capability; it must reject the real target/home and refuse unsigned arbitrary adapter paths.
- [ ] **5. Run green offline tests.** If disposable Docker scope is authorized during execution, run the initial bootstrap portion of `node scripts/qualify-runtime.mjs --phase bootstrap --repo <fixture>` and verify exactly two mounts, one service, UID/GID, `/opt/data` HOME/state backing, safe roots and maintenance quiescence. Otherwise record that live step as pending; no qualified production binding is emitted. Final integration qualification occurs in Task 12.
- [ ] **6. Commit:** `git add src/runtime/adapters.mjs adapters scripts/qualify-runtime.mjs tests docs/runtime-qualification.md; git commit -m "feat: bind runtime operations to reviewed official image contracts"`.

## Task 5: Two-mount Compose rendering and exact read-only planning

**Files:** Create `src/manifest/compose.mjs`, `src/manifest/select.mjs`, `src/manifest/drift.mjs`, `src/lifecycle/plan.mjs`, `tests/compose.test.mjs`, `tests/plan.test.mjs`; wire `plan` in `bin/hermes-repo.mjs`.

**Interfaces:** `makeCompose({identity,manifest,uid,gid,mode,launch}) -> object`; `selectManifest({requested,stored,bundled,identity}) -> Manifest`; `calculateDrift({desired,observed,command}) -> Plan`; `planCommand(options,deps) -> Promise<Result>`. `launch` is bundled adapter data, not arbitrary manifest commands. New plans default to maintenance, not gateway.

- [ ] **1. Write failing plan tests.** Assert fresh plan performs no writes/pulls, prints resolved immutable selection, reuses an installed manifest despite newer bundled defaults, and blocks foreign/missing receipt residue. Docker discovery failure is unknown/blocked, never “no existing container.”

```js
import { makeCompose } from '../src/manifest/compose.mjs';
test('state alias has one physical backing and official canonical path', () => {
  const manifest = manifestFixture(); const identity = manifest.identity;
  const compose = makeCompose({identity,manifest,uid:1000,gid:1000,
    mode:'maintenance',launch:{command:['fixture-maintenance']}});
  const service = compose.services.hermes;
  assert.deepEqual(service.volumes.map(v => [v.source,v.target]), [
    [identity.repoPath,'/workspace'], [identity.repoPath+'/.hermes','/opt/data']]);
  assert.equal(service.environment.HERMES_HOME,'/opt/data');
  assert.equal(service.environment.HERMES_WRITE_SAFE_ROOT,'/opt/data:/workspace');
  assert.equal(service.user,undefined); assert.equal(service.ports,undefined);
  assert.equal(Object.keys(compose.services).length,1);
});
```

`fixture-maintenance` is test data only and must never appear in a production adapter.
- [ ] **2. Run red:** `node --test tests/compose.test.mjs tests/plan.test.mjs`.
- [ ] **3. Implement Compose JSON and deterministic drift.** Persist only Compose object, not an envelope. Escape `$` for Compose interpolation while preserving literal host paths. Explicit raw bootstrap env-file; no native `.env` injection. One service, scoped network, mandatory toolkit labels and hashed project. No HOME/XDG overrides contradicting qualified official behavior. Add comparisons for canonical state alias and default profile rather than insisting gateway process cwd equals `/workspace`.

```js
export function selectManifest({requested,stored,bundled,identity}) {
  const selected = validateManifest(requested ?? stored ?? bundled);
  if (selected.identity.repoId !== identity.repoId)
    throw new ToolkitError('MANIFEST_REPO_MISMATCH', 'Desired state belongs to another repository.');
  return selected;
}
```

A bundled release template has no repo identity; instantiate it with the freshly resolved identity before this function. Existing manifests cannot be rebound to another repo automatically. Calculate actions from owned keys only; incompatible preserved state becomes a blocker. Make pending/dry plan results distinct from qualified runnable plans.
- [ ] **4. Run green:** `npm test`; optional installed Compose `config --quiet` schema test uses temporary nonsecret fixtures and never starts a container. Snapshot file listings/contents before/after `plan`; they must match.
- [ ] **5. Commit:** `git add src/manifest src/lifecycle/plan.mjs bin tests; git commit -m "feat: plan declarative two-mount Hermes deployments"`.

## Task 6: Default identity, workspace and manual Kanban

**Files:** Create `src/integrations/identity.mjs`, `src/integrations/kanban.mjs`, `tests/kanban.test.mjs`, `tests/repo-identity.test.mjs`; extend verified bridge operations and fixtures only where source supports them.

**Interfaces:** `desiredKanban() -> object`; `reconcileKanban({driver,manifest,observed}) -> Promise<Gate>`; `reconcileIdentity({driver,identity,observed}) -> Promise<Gate>`. Native config writes use expected preimages and narrow owned keys; SOUL identity updates preserve all unrelated text.

- [ ] **1. Write failing tests for both independent booleans, explicit empty allowlist, platform-specific tool enablement, global deny conflicts, alias DB mismatch and an existing active roster.** Default must not become a worker or be named after the Docker container.

```js
import { desiredKanban } from '../src/integrations/kanban.mjs';
test('manual board is not an automatic worker pool', () => {
  assert.deepEqual(desiredKanban(), {
    orchestrator_profile:'default', dispatch_in_gateway:false, auto_decompose:false,
    dispatch_profiles:[], auto_subscribe_on_create:false, notify_in_gateway:false,
    default_workdir:'/workspace'
  });
});
```

- [ ] **2. Run red:** `node --test tests/kanban.test.mjs tests/repo-identity.test.mjs`.
- [ ] **3. Implement the literal baseline above and native initialization with side-effect accounting.** Preserve unrelated config and selected channel toolsets. Resolve effective board and SOUL loader paths through the adapter. Kanban initialization is a named write step under lock/quiescence; receipt records observed canonical `/opt/data/kanban.db`, not an assumed filename. Reject unexplained alternate DB overrides or existing team state rather than choosing a different board. Set terminal local `/workspace` and repo basename identity without altering model/auth. Verify changed key set semantically after each native config operation.

```js
export function desiredKanban() {
  return { orchestrator_profile:'default', dispatch_in_gateway:false,
    auto_decompose:false, dispatch_profiles:[], auto_subscribe_on_create:false,
    notify_in_gateway:false, default_workdir:'/workspace' };
}
```

- [ ] **4. Run green:** targeted tests and `npm test`. Fake driver call log must contain no `create-card`, `create-profile`, `dispatch`, `decompose`, `send-message` or inference operation. Assert fresh-session schema observation is separate from merely writing toolset config.
- [ ] **5. Commit:** `git add src/integrations/identity.mjs src/integrations/kanban.mjs adapters tests; git commit -m "feat: prepare default-profile manual Kanban"`.

## Task 7: Immutable Superpowers admission without force bypass

**Files:** Create `src/integrations/superpowers.mjs`, `tests/plugin-policy.test.mjs`, `tests/plugin-install.test.mjs`; extend bridge admission and provenance mappings, receipt evidence and parser approval handling.

**Interfaces:** `decidePluginAdmission({source,sha,repoId,scan,priorReport,approval}) -> {state,code}`; `reconcileSuperpowers({driver,manifest,approval,receipt}) -> Promise<Gate>`. `scan` contains `{verdict,scannerId,findingsDigest,treeDigest,findings}`. Digests apply to plugin source/findings only, never secrets. `approval` must match source/SHA/repo/current scan binding; an earlier report alone is not approval.

- [ ] **1. Write failing safe/caution/dangerous/disabled/scanner-error tests; force-replacement and changed-tree regressions.** Include a safe old pin upgrading to a caution new pin and a previously approved caution that becomes dangerous.

```js
import { decidePluginAdmission } from '../src/integrations/superpowers.mjs';
test('dangerous blocks even a matching approval', () => {
  const base = {source:'obra/superpowers',sha:'a'.repeat(40)};
  const scan = {verdict:'dangerous',scannerId:'fixture',findingsDigest:'f',treeDigest:'t',findings:[]};
  const approval = {...base,repoId:'r',scannerId:'fixture',findingsDigest:'f',treeDigest:'t'};
  assert.deepEqual(decidePluginAdmission({...base,repoId:'r',scan,approval,priorReport:scan}),
    {state:'blocked',code:'PLUGIN_BLOCKED'});
});
test('caution is not implicitly accepted by an install request', () => {
  const scan = {verdict:'caution',scannerId:'fixture',findingsDigest:'f',treeDigest:'t',findings:[]};
  assert.equal(decidePluginAdmission({source:'obra/superpowers',sha:'a'.repeat(40),
    repoId:'r',scan,approval:null,priorReport:null}).code,'PLUGIN_APPROVAL_REQUIRED');
});
```

- [ ] **2. Run red:** `node --test tests/plugin-policy.test.mjs tests/plugin-install.test.mjs`.
- [ ] **3. Implement a closed decision table, report binding and native pin install.** Safe proceeds; caution requires the exact explicit owner decision; dangerous/unknown/disabled blocks. Do not turn on disabled scanning silently. Render bounded findings without source contents/secrets or terminal escapes. Add verified `--ref <sha> --enable` installation arguments, inspect the result's native provenance and compare actual revision. No floating install or `plugins update` for pins.

```js
if (scan.verdict === 'dangerous') return {state:'blocked',code:'PLUGIN_BLOCKED'};
if (scan.verdict === 'safe') return {state:'pass',code:'PLUGIN_ADMITTED'};
if (scan.verdict !== 'caution') return {state:'blocked',code:'PLUGIN_SCAN_UNAVAILABLE'};
```

Complete the caution branch by comparing every approval binding listed in the interface, including the current repo ID and prior reported candidate. A changed scanner/findings/tree invalidates the decision. Native capability grants are not implied.

For replacement, the inspected upstream core scans a staged candidate before publication and `force=True` can accept caution. Qualify a native staged admission route that gates that exact candidate with `force=False` before any force-replacement continuation. One available source interface to investigate is the native `before_swap` callback in `_install_plugin_core`; its private status requires exact revision binding and tests. Reject owner-modified trees. If final candidate identity/admission cannot be guaranteed, return `PLUGIN_REPLACEMENT_UNSUPPORTED`; do not pre-scan one tree and force-install a different one or patch the scanner.
- [ ] **4. Run green:** targeted tests and `npm test`. Inject a mock native installer that would accept caution with force; prove the wrapper rejects it before replacement. Verify disabled/project-alias plugin discovery cannot load a second instance. An unchanged pin/enablement is a no-op.
- [ ] **5. Commit:** `git add src/integrations/superpowers.mjs src/cli adapters tests; git commit -m "feat: pin Superpowers with explicit scanner admission"`.

## Task 8: Source-backed memory and integration readiness

**Files:** Create `src/integrations/memory.mjs`, `tests/memory.test.mjs`; extend bridge memory operations and `docs/runtime-qualification.md` with actual provider provenance.

**Interfaces:** `reconcileMemory({driver,manifest,observed}) -> Promise<Gate>`; `memoryVerdict(observations) -> Gate`, where observations contain separate `dependencies`, `hrr`, `vectors`, `canary`, `cleanup`, `loaded`, `durability` gates. Native provider method names come from reviewed source, not guessed imports.

- [ ] **1. Write failing tests for plain-Python versus startup-equivalent dependencies, FTS5 absence, cleanup failure, DB alias mismatch, existing different provider, and local-only pass with loaded integration still pending.**

```js
import { memoryVerdict } from '../src/integrations/memory.mjs';
test('local persistence does not prove loaded gateway integration', () => {
  const pass = {state:'pass',code:'VERIFIED',evidence:{}};
  const result = memoryVerdict({dependencies:pass,hrr:pass,vectors:pass,canary:pass,
    cleanup:pass,loaded:{state:'pending',code:'LOADED_UNVERIFIED',evidence:{}},durability:pass});
  assert.equal(result.state,'pending');
});
```

- [ ] **2. Run red:** `node --test tests/memory.test.mjs`.
- [ ] **3. Implement using the full memory skill/setup reference, overriding only its obsolete container-home assumption with this approved `/opt/data` mapping.** Source basis is `NousResearch/hermes-plugin-holographic`; inspect the actual selected provider. Verify packages before proposing install; keep supported additions under persistent state. New config uses `/opt/data/memory_store.db`, Holographic, `auto_extract:false`; preserve existing tuning/provider decisions. A required standalone provider pin/dependencies must be recorded and scanned through the same native admission policy, not exempted because it is a memory integration.

Use unique ASCII canary, returned fact ID, search `min_trust:0.0`, fresh-process reopen and `finally` cleanup of only that ID. Compare canonical DB before initialization; close all probe processes. Never print existing memories or invoke a second agent. Report basic keyword capability explicitly and keep HRR required readiness pending/blocked. Unsupported persistent dependency setup is a blocker, not permission to build an image.

```js
export function memoryVerdict(observations) {
  const required = ['dependencies','hrr','vectors','canary','cleanup','loaded','durability'];
  const gates = required.map(key => observations[key] ??
    {state:'pending',code:`MEMORY_${key.toUpperCase()}_UNVERIFIED`,evidence:{}});
  for (const state of ['blocked','unsupported','pending']) {
    const gate = gates.find(item => item.state === state);
    if (gate) return gate;
  }
  return {state:'pass',code:'MEMORY_VERIFIED',evidence:{}};
}
```

Validate Gate enums before aggregation. Add a test where dependencies are pending and cleanup is blocked: cleanup must win; keep every individual observation in the receipt, not just the aggregate verdict.
- [ ] **4. Run green:** targeted tests and `npm test`; cleanup failure preserves the returned canary ID for a safe next action and blocks readiness. No host Python command appears in the fake process log.
- [ ] **5. Commit:** `git add src/integrations/memory.mjs adapters tests docs/runtime-qualification.md; git commit -m "feat: verify persistent memory without false readiness"`.

## Task 9: Idempotent install and private setup

**Files:** Create `src/lifecycle/install.mjs`, `src/lifecycle/setup.mjs`, `tests/install.test.mjs`, `tests/setup.test.mjs`, `tests/helpers/fake-driver.mjs`; wire commands and protected native launcher generation.

**Interfaces:** `installCommand(options,deps) -> Promise<Result>`; `setupCommand(options,deps) -> Promise<Result>`; `createFakeDriver(initial) -> Driver & {calls,observed,injectCrash}`. Inject `withLock`, storage and driver dependencies; production entry points never accept driver code from user input.

- [ ] **1. Write failing fresh/rerun/crash/private-TTY tests.** Cases: unavailable image may be pulled after preview, unsupported adapter cannot deploy, wizard is not called by install, second install no changes, create succeeded then process died before receipt, and pipe-mode setup refuses before exec.

```js
import { installCommand } from '../src/lifecycle/install.mjs';
import { setupCommand } from '../src/lifecycle/setup.mjs';
test('setup never captures a noninteractive credential flow', async () => {
  const driver = createFakeDriver({});
  const result = await setupCommand({repo:'/tmp/fixture'},
    {driver,isTTY:false,withLock:async () => { throw Error('must not acquire'); }});
  assert.equal(result.code,'SETUP_REQUIRES_TTY'); assert.deepEqual(driver.calls,[]);
});
```

- [ ] **2. Run red:** `node --test tests/install.test.mjs tests/setup.test.mjs`.
- [ ] **3. Implement ordered preparation.** Discover read-only → preview → protect/lock/revalidate → select/verify immutable desired state → recover or publish generated artifacts → scoped pull/create same maintenance service if genuinely absent → verify IDs/mounts/quiescence → configure default/workspace/identity/Kanban → plugin admission → prepare memory dependencies when safe → persist only observed gates → release and return private setup next action.

```js
// Install must never call driver.runSetup or startGateway.
return {exitCode:10,code:'AWAITING_USER_SETUP',runtime:'prepared',alignment:'partial',
  development:'not-assessed',integrations,changes,preserved,
  nextAction:{command:'setup',repo:identity.repoPath}};
```

A plugin caution may be the earlier next action; do not overwrite it with a false all-integrations-prepared state. Reuse valid existing configuration instead of forcing setup; an already ready unchanged installation returns a no-op result with refreshed read-only observations, no canary or timestamp churn.

`setup` holds the lock, verifies maintenance/all writers and trusted exec mapping, then inherits the user's terminal directly. Native wizard stdout/stderr never enter report buffers. Propagate native exit/signal; on completion record only completion evidence and request `activate`. Generated main launcher routes setup through this same coordination path.
- [ ] **4. Run green:** `npm test`; crash injection after Docker create must resume by observing the owned container, not repeating creation. Reject an intervening foreign resource. Assert credentials and unrelated source bytes unchanged.
- [ ] **5. Commit:** `git add src/lifecycle/install.mjs src/lifecycle/setup.mjs src/storage bin tests; git commit -m "feat: install once and hand off private setup"`.

## Task 10: Activation and manifest-driven apply

**Files:** Create `src/lifecycle/activate.mjs`, `src/lifecycle/apply.mjs`, `tests/activate.test.mjs`, `tests/apply.test.mjs`; wire commands and upgrade/recovery receipts.

**Interfaces:** `activateCommand(options,deps) -> Promise<Result>`; `applyCommand(options,deps) -> Promise<Result>`. `driver.replaceService({manifest,identity,previous,recovery})` never overlaps old/new runtime containers and never deletes state. `driver.verify` reports fresh bounded gates, not health-by-uptime.

- [ ] **1. Write failing missing-auth, inherited-dispatch override, config-drift, immutable-upgrade, no-overlap and failed-recovery tests.** Updating bundled defaults alone must not generate an image change; selected `--manifest` does. Refuse apply on absent/corrupt identity. Maintenance stays maintenance.

```js
import { calculateDrift } from '../src/manifest/drift.mjs';
test('unchanged desired pin never follows a moving upstream tag', () => {
  const desired = manifestFixture();
  const observed = observedFixture(desired);
  observed.availableUpstream = {ref:'nousresearch/hermes-agent@sha256:'+'b'.repeat(64)};
  assert.equal(calculateDrift({desired,observed,command:'apply'}).actions
    .some(a => a.kind === 'pull-image' || a.kind === 'replace-service'),false);
});
```

Define `observedFixture(manifest)` in the shared fixture helper with complete passing synthetic observations; mark it test-only.
- [ ] **2. Run red:** `node --test tests/activate.test.mjs tests/apply.test.mjs`.
- [ ] **3. Implement activation gates and explicit drift reconciliation.** Reacquire lock, validate completion/ownership, re-observe all writers and native settings. Verify auth availability without network refresh, workspace/default profile, both Kanban booleans and effective env overrides, plugin admission/loading and memory local prerequisites. Start only selected default services, then bounded readiness and required recreation persistence. No automatic message/model call. CLI-only readiness is separate from absent channel configuration.

Apply compares selected desired manifest to owned config/artifacts and actual runtime, previews exact deltas, then repairs only owned settings. Upgrades need prior immutable identities and consistent private backups with verified restore compatibility; otherwise block before stopping the service. Reject active teams/workers or owner plugin edits. Perform required stopped-container removal and recreation sequentially if Compose would overlap containers. Never `down -v`; do not use `docker restart` to reload injected env.

```js
// Mandatory ordering in a qualified replacement method:
await driver.assertQuiescent({manifest:previous,identity});
await recovery.verify();
await driver.replaceService({manifest,identity,previous,recovery});
const gates = await driver.verify({manifest,identity,readOnly:true});
```

`recovery` is a toolkit-created object with `verify()` checking a recorded private consistent backup and supported source/target data compatibility; it is not executable data from a manifest. Failed readiness is one reported failed attempt, not an automatic restart/rollback loop. Record observed new reality and retained recovery state honestly.
- [ ] **4. Run green:** `npm test`; drive fake container-count tracking through replacement and assert maximum 1, including stopped containers. Preserve all fixture auth/session/board/source bytes across approved pin changes. Run the plugin force-bypass regressions again.
- [ ] **5. Commit:** `git add src/lifecycle src/storage bin tests; git commit -m "feat: activate and reconcile explicit repository manifests"`.

## Task 11: Truly read-only status and concise user documentation

**Files:** Create `src/lifecycle/status.mjs`, `tests/status.test.mjs`, `README.md`, `docs/operations.md`; extend report renderer and CLI.

**Interfaces:** `statusCommand(options,deps) -> Promise<Result>`; `summarizeGates(observed) -> Result`. Status never calls storage mutators or creates receipts; missing state is reported, not initialized. JSON schema matches `Result` exactly with fixed reason codes.

- [ ] **1. Write failing tests for no mutation, WAL-visible observations, unknown health, raw output redaction, disabled plugin with correct SHA, stale receipt and gateway-up/channel-unconfigured distinctions.** Use a file snapshot and driver spy; no credential values in diagnostics.

```js
import { statusCommand } from '../src/lifecycle/status.mjs';
test('status does not repair drift or record a checked timestamp', async () => {
  const fixture = await temporaryRepo();
  try {
    const before = await fixture.snapshot();
    const driver = createFakeDriver({ drift:true });
    const result = await statusCommand({repo:fixture.repoPath},{driver});
    assert.notEqual(result.alignment,'aligned');
    assert.deepEqual(await fixture.snapshot(),before);
    assert.ok(driver.calls.every(c => ['collect','verify-read-only'].includes(c.method)));
  } finally { await fixture.cleanup(); }
});
```

Add `snapshot()` and `repoPath` to the shared fixture helper. It hashes only test fixture contents, not actual user secrets.
- [ ] **2. Run red:** `node --test tests/status.test.mjs`.
- [ ] **3. Implement observed-state aggregation and documentation.** Read-only SQLite observations must be source-qualified and WAL-consistent without creating SHM/journal/schema; if that cannot be proven in the active runtime, report `BOARD_OBSERVATION_UNSUPPORTED`, not success. Never use `immutable=1` against a live WAL DB, `dispatch --dry-run`, a mutating plugin import, token refresh or raw logs. Absence of a safe observer is an explicit pending/unsupported gate.

README includes prerequisites; six commands; host/state mount diagram; private setup; immutable manifest upgrade workflow; caution approval with exact SHA; no default worker execution; no automatic dashboard; HOME versus HERMES_HOME; Superpowers fresh-session/compaction caveat; file-tool policy versus OS confinement. Operations docs cover lock contention, interrupted effects, name collision, retained recovery artifacts and unsupported uninstall. Document source license/provenance for reused code.
- [ ] **4. Run green:** `npm test`; assert default report leads with concise runtime/alignment and one next action, no giant Docker inventory. Validate README commands against CLI `--help` in a test.
- [ ] **5. Commit:** `git add src/lifecycle/status.mjs src/cli bin tests README.md docs/operations.md; git commit -m "feat: report read-only health and document lifecycle commands"`.

## Task 12: Adversarial acceptance and official-image qualification

**Files:** Create `tests/acceptance.test.mjs`, `tests/runtime/lifecycle.test.mjs`, `tests/runtime/kanban.test.mjs`, `tests/runtime/plugin.test.mjs`, `tests/runtime/memory.test.mjs`, `tests/runtime/helpers.mjs`; finalize `scripts/qualify-runtime.mjs`, adapter metadata and `manifests/release.json` only with real evidence; update `docs/runtime-qualification.md`.

**Interfaces:** `runtimeFixture({authorization,manifest}) -> {repoPath,run,inspect,cleanupOwnedContainer,preserveEvidence}`; test harness owns one disposable repo/container and never shares a real home. `QualificationReport` binds exact image/source/platform, adapter source digest, plugin revision, test names/results and untested scopes. Production `selectAdapter` accepts only complete required qualification evidence.

- [ ] **1. Add failing acceptance matrix covering all spec cases.** Reuse actual CLI subprocesses, fake Docker, real lock processes and fault-injected storage, not only unit functions. SIGKILL every journal/external-effect boundary. Preserve pre-existing dirty tracked/untracked files and gitlinks. Verify no extra container even in failure branches.

```js
for (const point of ['before-create','after-create','before-receipt','after-receipt']) {
  test(`resume after ${point} never duplicates the repository service`, async () => {
    const harness = await acceptanceFixture({crashAt:point});
    try {
      await harness.installExpectInterrupted();
      await harness.installResume();
      assert.ok(harness.maxContainerCount <= 1);
      assert.deepEqual(await harness.userState(),harness.initialUserState);
    } finally { await harness.cleanup(); }
  });
}
```

Define `acceptanceFixture` in `tests/helpers/fixture.mjs` as an actual child-process harness backed by fake Docker's durable state file, so a killed Node parent does not erase observed daemon effects.
- [ ] **2. Run red/green:** `node --test tests/acceptance.test.mjs`, fix implementation defects by regression, then `npm test`.
- [ ] **3. Obtain the execution handoff's explicit disposable Docker scope before live tests.** Preview selected official immutable image, expected downloads/resources, fixture paths and no-model/no-message limits. Without that scope, stop with runtime qualification pending. Do not treat the user's approval for implementation planning as runtime permission.
- [ ] **4. Run official-layout bootstrap and lifecycle qualification.** Use `node scripts/qualify-runtime.mjs --phase all --repo <owned-fixture-repo>` with explicit harness authorization. Verify one container/two binds, maintenance quiescence, effective nonroot IDs/HOME/tool HOME, local backend `/workspace`, safe-root file-tool behavior, immutable `/opt/hermes`, default profile, private file modes and no project-alias double loading. Exercise native setup mapping with a nonsecret fixture interface only; never automate a real private wizard or claim actual auth verified.
- [ ] **5. Test persistence and upgrade.** Create synthetic session/board/memory records and benign source changes through supported APIs; recreate and compare exact identities/content. Upgrade only between two qualified official pins with supported recovery and test state preservation. If only one version qualifies, recreation passes but image-upgrade acceptance remains unverified; do not label it passed.
- [ ] **6. Test Kanban behavior distinctly.** Create a ready card in the disposable board and observe through more than the configured dispatch interval while both automatic controls are false; prove no claim/spawn. For manual dispatch, explicitly create a test-only specialist and allowlist in that same fixture container, use a source-qualified local stub worker/provider, invoke native dispatch once, then verify claim/execution/history persistence. Record “native dispatch with stub worker verified; model work untested.” If upstream cannot exercise that path without real inference, retain manual execution as pending rather than spending credentials or faking it. The production baseline remains default-only with empty allowlist.
- [ ] **7. Test native plugin admission and memory.** Install the reviewed full Superpowers pin; preserve an exact caution report and stop if owner approval is required. Use synthetic harmless scanner fixtures for safe/caution/dangerous behavior; never deploy intentionally malicious plugins. Verify actual pin, enablement, skill schema/startup hook and recreation durability. Verify memory startup environment, functional HRR, canary reopen/removal and loaded-state observations without an LLM. Any unobservable hook/loading gate remains pending, not inferred from import success.
- [ ] **8. Publish qualification evidence only for checks actually run.** Bind a successful candidate to `manifests/release.json`; real SHA/digest values come from recorded verified artifacts, not examples. No qualified manifest is emitted with missing required gates. Save residual limitations, cleanup/preserved-evidence paths and exact tested platform. Do not reuse a planning-main source SHA as a stable image proof.
- [ ] **9. Final verification and review.** Run `npm test`, `node bin/hermes-repo.mjs --help`, shell syntax checks, all authorized runtime suites and `git diff --check`. Use the verification-before-completion and requesting-code-review skills. A fresh reviewer checks manifest/actual-state agreement, scanner bypasses, one-container cardinality, privacy, process lifetime and recovery. Fix findings with regression tests. Report implementation, offline tests, runtime qualification and untested model/channel/worker scopes separately.
- [ ] **10. Commit:** `git add tests scripts/qualify-runtime.mjs adapters manifests docs/runtime-qualification.md; git commit -m "test: qualify repository-local Hermes lifecycle and recovery"`. Do not push, deploy or publish without a separate request.

## Coverage map and self-review

| Spec area | Tasks |
| --- | --- |
| Host-local state, official HOME, write-policy distinction | 4, 5, 11, 12 |
| Six commands, immutable desired state, no implicit updates | 1, 5, 9, 10, 11 |
| Ownership, aliases, stopped duplicates and no overlap | 2, 5, 9, 10, 12 |
| Ignore/index checks, secrets, locks and crash recovery | 3, 9, 10, 12 |
| Manual Kanban, default role and authorized manual fixture | 6, 10, 12 |
| Exact Superpowers pin, scanner decisions and force risk | 1, 4, 7, 12 |
| Holographic capability and separate development gates | 8, 10, 11, 12 |
| Status side effects, WAL, output and operational docs | 4, 11, 12 |
| Provenance, qualification and upgrade recovery | 4, 10, 12 |

Self-review completed against the spec and Review Focus. Corrected approval `repoId` input, optional plan output, help parsing, standalone memory-plugin pin representation, and memory blocked-before-pending precedence. Every Review Focus item has an owning test. No production pin or native maintenance command is invented. Runtime-specific implementation is gated on source inspection in Task 4 rather than embedding guessed shell recipes in this plan. Source inspection and live qualification are explicit work items, not completed checks.

Recommended execution: **subagent-driven**, serial single writer with a fresh reviewer per task. The filesystem, scanner and Docker lifecycle boundaries can each corrupt or expose state if a mistake ships. Keep orchestration with the parent; ordinary child workers must not launch their own subagents. Native inline execution remains an option if the user prefers lower context cost, followed by one fresh whole-project review.

Plan review and execution-method selection are the next user decision. No code, Docker pull/build, plugin installation or runtime qualification has been executed by writing this plan.

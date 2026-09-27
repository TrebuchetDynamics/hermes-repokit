> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit A — Host Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build deterministic host administration with honest unsupported-operation gates.

**Architecture:** Pure validation/planning precedes locked effects. Node built-ins call fixed argument-array adapters; observation has no mutator capability.

**Tech Stack:** Node >=22 ESM/node:test; Linux local filesystems; util-linux flock; proposed Compose >=2.30, qualified in A5.

**Spec:** [Approved design](../specs/2026-09-27-repokit-team-runtime-design.md) §§1–2,5,9–12; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout).

## Global Constraints

All master constraints apply. Eight command surfaces including state-preserving `uninstall`; no `memory index`. No pulls/initialization in plan/status. No second Hermes, socket, recursive source chown, `down -v`, prune, global container names, secrets in argv or arbitrary manifest executable paths. Null artifact/plugin selections mean blocked, not defaults.

## Review Focus

1. Ignored but indexed `.hermes` refuses (A2).
2. Symlink/hardlink/ancestor substitution and relocated receipts refuse (A2/A3).
3. Surviving child holds administrative exclusion after parent death (A3).
4. Missing memory qualification does not block independent infrastructure (A4/A5).
5. Unknown writers, stale preimages and partial effects preserve recovery evidence (A6).

---

All snippets are proposed future tests/core code. Each task includes additional named cases to expand its executable seed. No fixture adapter is admitted as production support.

### Task A1: CLI boundary and strict immutable desired state

**Files:** Create `package.json`, `bin/hermes-repokit.js`, `src/cli.js`, `src/contracts.js`, `src/manifest.js`, `contracts/repokit-v1.json`, `test/manifest.test.js`.

**Interfaces:** Consumes master Desired/Identity definitions. Produces `validateManifest(input)->Desired`, `main(argv,deps)->Promise<number>`, `validateRecord(input,keys)->void` in contracts.js. `deps` contains separate readers/mutators/privateIO/stdout/stderr/now; parsing precedes access.

- [ ] Write failing tests, starting with the complete module:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {validateManifest} from '../src/manifest.js';
import {main} from '../src/cli.js';
test('private flags and future commands reject without echo', async () => {
  let out = '';
  const deps = {stderr:{write:s => { out += s; }}};
  assert.equal(await main(['setup','--api-key=CANARY'], deps), 2);
  assert.equal(await main(['memory','index'], deps), 2);
  assert.equal(out.includes('CANARY'), false);
  assert.throws(() => validateManifest({schemaVersion:0}),
    {code:'INVALID_MANIFEST'});
});
```

Add positive three-service/null-selection fixture; reject extra keys at every nesting, mutable tags in artifact IDs, non-full Superpowers SHA, non-finite/deep/oversized records, managed !=1 and fleet <=1. Safe retains a positive configured limit. Fresh desired state has `team.target:default`; managed/fleet require `engineering`. SAFE with engineering remains valid after downgrade. Validate TeamAdmission/command/actor receipt fields without treating them as task lifecycle. A valid example is assembled from literal master fields, not an undefined fixture helper.

- [ ] Red: `node --test test/manifest.test.js` → missing module/export, then failing validation assertions.
- [ ] Implement package with `type:module`, `engines.node:>=22`, `scripts.test:node --test test/*.test.js`, bin entry. Bound input JSON to 64 KiB/depth 8 before recursive validation; recursively freeze a newly constructed normalized object. Core field check:

```js
export function validateRecord(input, keys) {
  if (!input || Object.getPrototypeOf(input) !== Object.prototype ||
      Object.keys(input).some(k => !keys.includes(k)) ||
      keys.some(k => !Object.hasOwn(input, k))) {
    throw Object.assign(new Error('invalid manifest'), {code:'INVALID_MANIFEST'});
  }
}
```

Parser accepts `plan|install|setup|activate|status|apply|mode|uninstall`, repository/context and nonsecret selection arguments; mode requires safe/managed/fleet. Unknown options report static text, never argv. Shim sets `process.exitCode`, library never exits process. Until G2 supplies teardown, uninstall returns an explicit unsupported-operation blocker with zero effects, not success. Package metadata/bin/test scripts use `hermes-repokit`; no Pi dependency, executable lookup or package bootstrap. User test runners are Node/Python directly, not Pi.
- [ ] Green: rerun red command; `node --check bin/hermes-repokit.js`.
- [ ] Commit: `git add package.json bin src/cli.js src/contracts.js src/manifest.js contracts test/manifest.test.js && git commit -m "feat: define RepoKit CLI and desired contract"`.

### Task A2: Canonical identity and private filesystem admission

**Files:** Create `src/identity.js`, `src/private-layout.js`, `test/identity.test.js`.

**Interfaces:** Consumes Identity. Produces `resolveIdentity(repoPath,dockerContext)->Promise<Identity>`, `assertOwnership(identity,receipt,containers)->void`, `inspectPrivateLayout(identity)->Promise<Gate>`, `bootstrapPrivateLayout(identity)->Promise<void>`.

- [ ] Write failing test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {assertOwnership} from '../src/identity.js';
test('relocation is not adoption', () => {
  const id = {repoRoot:'/a',repoId:'a',projectName:'repokit-a',dockerContext:'local'};
  assert.throws(() => assertOwnership(id,{...id,repoRoot:'/b'},[]),
    {code:'OWNERSHIP_CONFLICT'});
});
```

Add disposable `mkdtemp` + `git init` cases using `execFile` argument arrays: ignored-and-unindexed passes; `git add -f .hermes/probe` rejects even when ignored; symlink root aliases resolve the same identity; symlink private children, hardlink counts >1, unsafe ancestor UID/mode, foreign receipts/labels/mounts and duplicate owned containers reject. Test no source ownership changes.
- [ ] Red: `node --test test/identity.test.js` → unresolved export/assertion.
- [ ] Implement realpath of Git toplevel, explicit Docker context, SHA256 identity and full digest project name. Core identity comparison (also compare UID, labels and actual mount destinations/sources in complete function):

```js
export function assertOwnership(identity, receipt, containers) {
  const fields = ['repoRoot','repoId','projectName','dockerContext'];
  if (fields.some(k => receipt[k] !== identity[k]) ||
      containers.filter(c => c.service === 'hermes').length > 1) {
    throw Object.assign(new Error('ownership conflict'), {code:'OWNERSHIP_CONFLICT'});
  }
}
```

Private admission checks every ancestor and entry via lstat/open-no-follow/fstat, ownership, modes and nlink. Check Git ignore and index independently. Bootstrap uses exclusive repo-local mkdir plus no-follow descriptor verification; if a peer wins mkdir, re-inspect, never assume ownership. Create only known private directories 0700/files 0600, preserve unknown supported state; per-service exceptions need later qualified UID evidence. No migration/adoption of ambiguous legacy state. Read-only callers never bootstrap.
- [ ] Green: rerun red command and A1 tests.
- [ ] Commit: `git add src/identity.js src/private-layout.js test/identity.test.js && git commit -m "feat: admit canonical private repository state"`.

### Task A3: Stable bounded lock and crash-safe publication

**Files:** Create `src/locks.js`, `src/transaction.js`, `test/locks.test.js`, `test/fixtures/lock-process.mjs`.

**Interfaces:** Consumes A2 admission. Produces `withAdminLock(identity,fn)->Promise<T>`, `atomicPublish(path,bytes,expectedDigest)->Promise<void>`. Callback receives inherited lock FD. Busy lock rejects `LOCK_BUSY`; no polling/wait queue.

- [ ] Write real cross-process failing test. Fixture mode `hold` takes lock, starts `inheritor` with the same FD mapped to descriptor 3, prints `HELD` only after child's IPC `READY`; inheritor waits on stdin/IPC, not a timer. `probe` tries nonblocking lock and exits 75 on contention. Tests must assert descriptor lifetime, not merely same-process serialization:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {spawn,spawnSync} from 'node:child_process';
import {once} from 'node:events';
import {mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
test('different process cannot enter held critical section', async () => {
  const root = await mkdtemp(join(tmpdir(),'repokit-lock-'));
  const child = spawn(process.execPath,['test/fixtures/lock-process.mjs','hold',root],
    {stdio:['pipe','pipe','inherit','ipc']});
  try {
    const [msg] = await once(child,'message');
    assert.equal(msg.type,'HELD');
    const probe = spawnSync(process.execPath,
      ['test/fixtures/lock-process.mjs','probe',root],{timeout:2000});
    assert.equal(probe.status,75);
  } finally {
    const exited = once(child,'exit'); child.send({type:'RELEASE'}); await exited;
    await rm(root,{recursive:true,force:true});
  }
});
```

Extend fixture to relay inheritor PID/IPC endpoint: SIGKILL administrator after READY; probe still exits 75; explicitly release inheritor, wait its exit, probe succeeds. Test lock-busy deadline, swapped symlink/hardlink/inode and crash at each publish boundary. Cleanup signals are test-only; runtime never kills work to claim quiescence.
- [ ] Red: `node --test test/locks.test.js` → fixture/lock missing.
- [ ] Implement only stable `.hermes/repokit/locks/admin.lock` after A2 bootstrap, never unlink it or add a global host lock. Open `O_CREAT|O_RDWR|O_NOFOLLOW` 0600, fstat ownership/nlink and compare path inode. Use the same open-file description across flock and every mutating child:

```js
import {spawnSync} from 'node:child_process';
export function acquireDescriptor(fd) {
  const r = spawnSync('flock',['--exclusive','--nonblock','3'],
    {stdio:['ignore','ignore','ignore',fd],timeout:1000});
  if (r.status !== 0) throw Object.assign(new Error('lock unavailable'), {code:'LOCK_BUSY'});
}
```

Keep parent descriptor open until callback and owned children finish. Closing helper's duplicate must not release shared flock; prove this on supported hosts. Atomic publication: exclusive same-directory temp, fsync file, compare nonsecret preimage under lock, rename, fsync directory. Interruption preserves receipt evidence. Bootstrap collision/interruption re-inspects directories; no stale PID deletion.
- [ ] Green: rerun red command; repeat SIGKILL fixture 20 times with deterministic IPC barriers.
- [ ] Commit: `git add src/locks.js src/transaction.js test/locks.test.js test/fixtures/lock-process.mjs && git commit -m "feat: hold administration locks through child lifetime"`.

### Task A4: Deterministic topology and operation-specific gates

**Files:** Create `src/compose.js`, `src/plan.js`, `test/compose.test.js`.

**Interfaces:** Consumes Desired/Qualification/Observation. Produces `renderCompose(desired,catalog)->object`, `operationGate({operation,component,evidence})->Gate`, `buildPlan(...)->Plan` as master.

- [ ] Write failing test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {operationGate} from '../src/plan.js';
test('privacy gate is not an infrastructure veto', () => {
  const evidence = {artifactQualified:true,authority:true,privacyCompatible:false};
  assert.equal(operationGate({operation:'start',component:'openviking',evidence}).allowed,true);
  assert.equal(operationGate({operation:'memory-session',component:'hermes',evidence}).allowed,false);
});
```

Add snapshot assertions for one Hermes + two support services, exact official binds/env, no sidecar repo/ports/socket, OpenViking bot disabled, explicit resources and internal network. Test remote-provider egress denied without separately selected policy; strict internal network cannot promise remote access. Null/unqualified artifacts must not render runnable Compose.
- [ ] Red: `node --test test/compose.test.js`.
- [ ] Implement pure enum switch, not policy language. Core dependent gate:

```js
const required = {
  observe:[],
  start:['artifactQualified','authority'],
  'memory-session':['artifactQualified','authority','privacyCompatible','memoryIdentityQualified'],
  execution:['authority','executionQualified','reviewQualified'],
  download:['authority','downloadBounded'], inference:['authority','budgetBounded'],
  replace:['artifactQualified','authority','quiescent','preimageMatches'],
  'release-claims':['authority','executionQualified','reviewQualified','workersReady',
    'backlogApproved','budgetBounded','notificationsApproved','concurrencyQualified'],
  provision:['artifactQualified','authority'], 'configure-private':['authority'],
  'suspend-claims':['authority','nativeSuspensionQualified'],
  'stop-runtime':['authority','ownershipVerified','quiescent'],
  'remove-runtime':['authority','ownershipVerified','quiescent','retentionVerified']
};
export function operationGate({operation,evidence}) {
  const keys = required[operation];
  const reasons = keys ? keys.filter(k => evidence[k] !== true) : ['UNKNOWN_OPERATION'];
  return {allowed:reasons.length === 0,reasons};
}
```

Gate call sites additionally validate evidence freshness/selection scope. Compose is canonical JSON, fixed service names, no `container_name`; Laya cache bind only after D1 qualification. Sidecars have independent actions/blockers.
- [ ] Green: rerun red command and `node --test test/*.test.js`.
- [ ] Commit: `git add src/compose.js src/plan.js test/compose.test.js && git commit -m "feat: render owned topology and independent operation gates"`.

### Task A5: Qualified adapters, read-only observers and private command flow

**Files:** Create `src/observe.js`, `src/lifecycle.js`, `src/adapters/hermes.js`, `test/lifecycle.test.js`, `qualification/hermes-adapter.md`, `tests/runtime/test_host_adapter.py`, `docs/host-lifecycle.md`; modify `src/cli.js`.

**Interfaces:** Consumes A1–A4. Produces master observe/runCommand/getHermesAdapter/Adapter methods. `readers` expose bounded filesystem/Docker inspection only; `privateIO` owns private FDs; mutators are not passed into observers.

- [ ] Write test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {getHermesAdapter} from '../src/adapters/hermes.js';
test('source-only evidence cannot admit production adapter', () => {
  assert.equal(getHermesAdapter({id:'fixture'},{level:'source',supported:true}),null);
});
```

Add observer tests for absent DB/no directory creation, unchanged file bytes/mtime, WAL/schema mismatch unknown, timeout/output cap, freshness, denied credential refresh and canary-free output. Install safe/default-only, setup-without-native-specialists, activate-without-roster-expansion and private staging receive independent capability spies. Assert zero specialist-create calls before explicit managed entry, zero credential cloning and no Pi subprocess/import/config reference.
- [ ] Red: `node --test test/lifecycle.test.js`.
- [ ] Inspect exact Hermes research files `Dockerfile`, `docker/main-wrapper.sh`, `hermes_cli/profiles.py`, `hermes_cli/kanban_db.py`, `hermes_cli/kanban_db_dispatch.py`, `hermes_cli/plugins_cmd_install.py`. Record at selected source/image: official UID/HOME behavior; safe root-default/board initialization without credential clone or specialist creation; later idempotent specialist provisioning from protected setup material without overwriting existing credentials; true read-only WAL semantics; native claim entry points; exact CLI argv and private auth channel. Acceptance is captured argv/config fixtures plus authorized disposable-runtime results, not guessed native commands. Qualify Node/flock/Compose versions, local FS locking, supported architecture and rootless namespace availability; unsupported hosts fail clearly, never install Docker/change groups.
- [ ] Implement fixed adapter registry, no manifest-driven import; reject absent evidence:

```js
export function admitted(q, selection, operation) {
  return q?.supported === true && q.level === 'runtime' &&
    q.selectionId === selection.id && typeof q.evidenceDigest === 'string' &&
    Array.isArray(q.operations) && q.operations.includes(operation);
}
```

Resolve the catalog through `selection.catalogDigest` and compare qualified `artifactId`/`adapterId` with the selected Hermes entry before returning an adapter. Each actual Adapter method must be covered by the qualification's operation scope: observe/inspectQuiescence → observe; initialize → provision; configurePrivate → configure-private; provisionProfiles → both provision and configure-private; start → start; suspendClaims → suspend-claims; reconcile → replace; setMode → suspend-claims or release-claims; stopOwnedRuntime → stop-runtime; removeOwnedRuntime → remove-runtime. Release evidence cannot authorize profile creation or teardown. Observation evidence never admits mutation; inability to suspend all claims does not disqualify read-only observation. Add a fixture where observe-only qualification admits no initialize/start/reconcile call. Retain separate runtime records when inference qualification is added. Use explicit Docker context/project/Compose path and sanitized environment. Unsupported method returns its blocker before any effects. Fresh host install initializes SAFE/default-only native state and admitted non-autonomous components; no specialist homes or model acquisition. Setup handles default auth and protected staged specialist selections under `.hermes/repokit/private-setup/`, separating provider/data/retention/egress/download/compute authority; no native specialist creation, sessions or credential cloning. Activate starts eligible services without roster expansion, dispatch release or incompatible memory sessions. Reinstall reconciles retained profiles/state, never resets to a fresh default-only roster. Native actor/session releases require qualified privacy regardless of SAFE status. Status reports every spec §9 field with unknown where unsupported; no init/migrate/dispatch-dry-run observer.
- [ ] Green offline command; later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_host_adapter.py'` must prove actual mounts/native behavior before admission. Never promote an offline fixture result.
- [ ] Commit: `git add src/observe.js src/lifecycle.js src/adapters src/cli.js test/lifecycle.test.js qualification/hermes-adapter.md tests/runtime/test_host_adapter.py docs/host-lifecycle.md && git commit -m "feat: gate native operations on adapter qualification"`.

### Task A6: Transactional apply and exact Superpowers admission

**Files:** Create `src/admission.js`, `test/apply.test.js`, `test/admission.test.js`; modify `src/transaction.js`, `src/lifecycle.js`, `docs/host-lifecycle.md`.

**Interfaces:** Consumes Selection/Receipt/Quiescence/Adapter. Produces `runTransaction({identity,selection,intent,adapter,io})->Promise<Receipt>`, `admitPlugin({revision,verdict,scannerEnabled,findingsDigest,approval})->Gate`.

- [ ] Write failing tests:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {admitPlugin} from '../src/admission.js';
test('replacement caution approval binds candidate and current findings', () => {
  assert.equal(admitPlugin({revision:'a'.repeat(40),verdict:'caution',
    scannerEnabled:true,findingsDigest:'b'.repeat(64),
    approval:{revision:'a'.repeat(40),findingsDigest:'c'.repeat(64)}}).allowed,false);
  assert.equal(admitPlugin({revision:'a'.repeat(40),verdict:'caution',
    scannerEnabled:true,approval:{revision:'a'.repeat(40)}}).allowed,false);
});
```

Apply tests inject each effect boundary: no-op leaves bytes/mtime/restart count unchanged; unknown/active writer refuses; selection/preimage substitution refuses; partial sidecar failure preserves unaffected services; old Hermes stop precedes new create; interrupted receipt survives. Dangerous/missing/disabled scanner refuses even with force; exact force-replacement uses candidate revision AND current findings approval, never old installed approval.
- [ ] Red: `node --test test/apply.test.js test/admission.test.js`.
- [ ] Implement scanner predicate core:

```js
export function admitPlugin(x) {
  const allowed = /^[a-f0-9]{40}$/.test(x.revision) &&
    /^[a-f0-9]{64}$/.test(x.findingsDigest) && x.scannerEnabled === true &&
    (x.verdict === 'safe' || (x.verdict === 'caution' &&
      x.approval?.revision === x.revision &&
      x.approval?.findingsDigest === x.findingsDigest));
  return {allowed,reasons:allowed ? [] : ['PLUGIN_ADMISSION_BLOCKED']};
}
```

Qualify native scanner/install source (A5 files plus `hermes_cli/plugins_cmd.py`): does force replace rescan exact candidate, verify final tree/SHA, enabled discovery across aliases and fresh-session requirements? Persist nonsecret findings/approval/tree evidence; refuse any native bypass semantics. Transaction records prepared preimages before mutation, suspends verified claims, checks quiescence, records each observed effect, retains recovery identity, publishes committed only after reconciliation. Preserve sessions/auth/history/board/memories/data/cache and unknown supported files. Never infer rollback safety from old image after migration. Apply retains desired mode and roster intent but actual dispatch suspended until explicit B4 release; F2 adds immutable host promotion. Test that unchanged desired pins remain unchanged even when fake upstream latest moves, including both RepoKit plugin tree hashes. Add an explicit desired-change positive case; never fetch a mutable update selector as authority. G qualifies final recovery/retaining uninstall, not a destructive cleanup shortcut.
- [ ] Green: red command then `npm test`; live preservation remains F1/F4, not an A claim.
- [ ] Commit: `git add src/admission.js src/transaction.js src/lifecycle.js test/apply.test.js test/admission.test.js docs/host-lifecycle.md && git commit -m "feat: reconcile selected state with recoverable admission"`.

**Exit gate:** A reviewer checks offline behavior and identifies exactly which adapter methods have real evidence. A is not full-team readiness; unqualified production actions remain blocked while subsequent slices proceed.

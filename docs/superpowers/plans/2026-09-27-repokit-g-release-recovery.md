> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit G — Final Release and Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Qualify failure recovery, retained-state uninstall/reinstall and the complete Pi-independent release journey after real self-hosting evidence.

**Architecture:** Exercise the existing host lifecycle and native adapters, not another recovery engine or task database. Host-controlled teardown removes owned runtime resources only; final release combines A–F evidence with this failure matrix.

**Tech Stack:** Node built-in tests, Python unittest, actual immutable disposable-runtime fixtures under F1 authority; no Pi runtime/test dependency.

**Spec:** [Revised design](../specs/2026-09-27-repokit-team-runtime-design.md) §§5,10–12; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout).

## Global Constraints

All master constraints apply. One Hermes runtime/one Compose project per repository; OpenViking/Laya are internal sidecars. Fresh install is default-only SAFE; managed deliberately provisions the team. Never delete locks by stale PID, adopt foreign containers, reset corrupt receipts, bypass scanner decisions, blindly replay mutations or substitute old images for compatible data recovery. Uninstall preserves source and the entire private state tree. No purge option, volume/image deletion or automatic credential revocation. Pi is development-only; do not uninstall it from the developer's machine.

## Review Focus

1. Dead administrator versus surviving inherited lock holder requires real process evidence (G1).
2. Corrupt receipt or same-named foreign container must not authorize destructive recovery (G1/G2).
3. Failed pull/scanner rejection must not silently change admitted pins or erase the previous runtime (G1).
4. Retained directories alone do not prove state-preserving uninstall or safe rollback (G2).
5. Hiding one Pi executable does not establish dependency-free self-hosting (G3).

---

### Task G1: Integrated failure and recovery matrix

**Files:** Create `test/recovery-acceptance.test.js`, `test/fixtures/recovery-scenario.mjs`, `tests/runtime/test_recovery_acceptance.py`, `qualification/recovery-matrix.md`; modify `docs/host-lifecycle.md` only for the verified recovery contract.

**Interfaces:** Consumes A `runCommand`, `runTransaction`, real lock fixture and F1 authority/F2 writer evidence. Produces test-only `runScenario(name:string)->Promise<ScenarioResult>` and `atBoundary(name:string,failAt:string|null,action:Function)->Promise<unknown>`. ScenarioResult is `{exitCode:number,destructiveEffects:string[],preserved:boolean,maxHermes:number,claimsReleased:boolean}`; values derive from actual fixture observations/effect logs, never a success table. Runtime evidence uses the same scenario names against qualified adapters.

- [ ] Write the failing integrated test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {runScenario} from './fixtures/recovery-scenario.mjs';
test('corrupt receipt never authorizes destruction', async () => {
  const r = await runScenario('corrupt-receipt');
  assert.notEqual(r.exitCode, 0);
  assert.deepEqual(r.destructiveEffects, []);
  assert.equal(r.preserved, true);
  assert.equal(r.claimsReleased, false);
});
```

Add these exact scenario families:

| Scenario | Required observed behavior |
| --- | --- |
| fresh-install | Default-only native inventory, dispatch/decomposition off, at most one Hermes, no inference; setup/activate do not expand profiles. |
| interrupted-install | Failure before/after each publication/native effect; retry reconciles owned partial state without reset, duplicate profile or credential overwrite. |
| stale-lock | Kill administrator while inherited child survives: busy; after last holder exits acquire the same lock inode, without unlinking. |
| corrupt-receipt | Truncated/invalid/unsupported receipt blocks mutation and retains original bytes; status reports unknown/recovery-required. |
| container-collision | Matching service/name with foreign labels, mounts or Docker context refuses adoption/removal. |
| failed-pull | Requested candidate not admitted; retain prior usable runtime if replacement has not started, otherwise report truthful partial state without dispatch release. |
| scanner-rejection | Dangerous/failed/missing scanner and stale caution approval refuse; previous accepted plugin/runtime remains intact. |
| sidecar-unavailable | Laya UNKNOWN/WATCH and OpenViking degraded independently; already-authorized native work continues without fake memory, while privacy-incompatible sessions remain blocked. |

- [ ] Red: `node --test test/recovery-acceptance.test.js` → missing fixture/behavior, not skipped success.
- [ ] Implement the fixture using disposable Git roots, actual private filesystem/receipt bytes, existing lifecycle functions and instrumented fixed adapter methods. Inject errors at selected effect boundaries; record resulting native-like inventory, filesystem/private-permission snapshots and claim counters. Use real IPC lock tests for the stale-lock case, not simulated PID arithmetic. Core boundary injector:

```js
export async function atBoundary(name, failAt, action) {
  if (failAt === `${name}:before`) throw new Error('INJECTED_FAILURE');
  const result = await action();
  if (failAt === `${name}:after`) throw new Error('INJECTED_FAILURE');
  return result;
}
```

Runtime counterparts require an explicit disposable-host receipt and selected images; simulate pull failure through a controlled fixture boundary, never destroy an unrelated registry/runtime. Corrupt-receipt recovery is documented operator action only after independent canonical identity, actual mounts/labels, immutable selection and journal/backup evidence agree. If evidence is insufficient, remain blocked; never generate a fresh success receipt over the corrupt file. Do not add a speculative public repair command. Status remains strictly nonmutating.
- [ ] Green: rerun the Node command and A/F offline tests; later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_recovery_acceptance.py'`. Report offline versus actual runtime results separately.
- [ ] Commit: `git add test/recovery-acceptance.test.js test/fixtures/recovery-scenario.mjs tests/runtime/test_recovery_acceptance.py qualification/recovery-matrix.md docs/host-lifecycle.md && git commit -m "test: qualify integrated lifecycle failure recovery"`.

### Task G2: State-preserving uninstall, reinstall and compatible rollback

**Files:** Create `src/uninstall.js`, `test/uninstall.test.js`, `tests/runtime/test_uninstall_rollback.py`; modify `src/lifecycle.js`, `src/adapters/hermes.js`, `test/fixtures/recovery-scenario.mjs`, `README.md`, `docs/host-lifecycle.md`, `qualification/recovery-matrix.md`.

**Interfaces:** Consumes A ownership/lock/Receipt, F2 drain, G1 scenario harness. Produces `uninstallOwnedRuntime({identity,selection,adapter,io})->Promise<Receipt>` through existing `runCommand`; qualifies master `Adapter.stopOwnedRuntime(identity)` and `removeOwnedRuntime(identity)` for the separate `stop-runtime`/`remove-runtime` operations. `assertRetainingTeardown({ownershipVerified,quiescent,retentionVerified})->void` is the pure pre-effect guard in uninstall.js. Uninstall does not require unrelated sidecar inference health, but does require conclusive ownership and writer state.

- [ ] Write failing tests:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {runScenario} from './fixtures/recovery-scenario.mjs';
import {assertRetainingTeardown} from '../src/uninstall.js';
test('uninstall preserves state and removes only owned runtime resources', async () => {
  const r = await runScenario('uninstall');
  assert.equal(r.exitCode, 0);
  assert.deepEqual(r.destructiveEffects,
    ['remove-owned-containers', 'remove-owned-project-network']);
  assert.equal(r.preserved, true);
  assert.equal(r.claimsReleased, false);
});
test('unknown writers refuse teardown', () => {
  assert.throws(() => assertRetainingTeardown({ownershipVerified:true,
    quiescent:false,retentionVerified:true}), {code:'TEARDOWN_BLOCKED'});
});
```

Add corrupt receipt, foreign/shared network, active/unknown writers, interrupted stop/remove, repeat uninstall, and reinstall with retained engineering roster. Assert source/private bytes and permissions, logical histories and addressable sessions survive; only legitimate bounded lifecycle receipt updates may differ. No volume/image removal, credential revocation, prune or purge flag. Reinstall retains prior profile configuration and explicit pins and keeps claims suspended.
- [ ] Red: `node --test test/uninstall.test.js`.
- [ ] Implement the pre-effect guard, then actual locked reconciliation:

```js
export function assertRetainingTeardown(x) {
  if (['ownershipVerified','quiescent','retentionVerified'].some(k => x[k] !== true)) {
    throw Object.assign(new Error('retaining teardown blocked'), {code:'TEARDOWN_BLOCKED'});
  }
}
```

Under the inherited administrative lock, validate ownership/selection/receipt and retained-state paths, journal intent, suspend all claims, drain affected writers, recheck ownership immediately before each effect, stop/remove only owned containers and remove the exclusively owned empty project network. Refuse a shared/foreign endpoint, never disconnect it. Preserve every source/private payload including profile auth, `.hermes/plugins`, board/WAL history, sessions, built-in/OpenViking memories, Nerve history, Laya cache and desired pins. Keep the immutable host CLI/recovery path available. A repeated uninstall of demonstrably already-removed resources is a no-op, not an ownership reset. Unknown receipt/inventory still refuses. Interrupted teardown journals actual remaining resources for explicit retry.

Use A/F recovery machinery for rollback fixtures rather than add a new engine: upgrade a qualified compatible image/config/data tuple, inject failure, then restore the verified previous tuple or consistent backup and prove logical state/actual recall. Also test an incompatible data migration: old-image-only rollback must refuse and preserve recovery evidence. Recoverable means demonstrated on that selected pair, not a universal promise. Actual memory recall requires its own bounded inference/data authority; offline byte survival cannot replace it.
- [ ] Green: Node command plus G1 matrix; later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_uninstall_rollback.py'`. Inference-bearing preservation cases retain F authority separation.
- [ ] Commit: `git add src/uninstall.js src/lifecycle.js src/adapters/hermes.js test/uninstall.test.js test/fixtures/recovery-scenario.mjs tests/runtime/test_uninstall_rollback.py README.md docs/host-lifecycle.md qualification/recovery-matrix.md && git commit -m "feat: uninstall owned runtime while preserving state"`.

### Task G3: Pi-absent complete workflow and final release verdict

**Files:** Create `tests/selfhost/test_release_journey.py`, `tests/offline/test_final_release.py`, `qualification/final-release-protocol.md`; modify `tests/selfhost/release.py`, `qualification/release-matrix.md`, `qualification/release-evidence.md`, `README.md`.

**Interfaces:** Consumes F1 authority, F3 actual same-card team work, F4 actual immutable self-apply and G1/G2 evidence. Extends the existing `release_ready(evidence:dict)->bool`; no second release engine. Produces `recovery_evidence_present(evidence:dict)->bool` in release.py as a required structural subcheck, followed by validated candidate/artifact/fixture-bound evidence references.

- [ ] Write a failing final-gate test without assuming tests is an importable package:

```python
import importlib.util
import unittest
spec = importlib.util.spec_from_file_location('release','tests/selfhost/release.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class FinalReleaseTest(unittest.TestCase):
    def test_every_final_gate_is_required(self):
        complete = dict(piAbsent=True,dependencyClosureVerified=True,
                        recoveryMatrix=True,compatibleRollback=True,
                        retainingUninstall=True,reinstallPreserved=True)
        self.assertTrue(module.recovery_evidence_present(complete))
        for key in complete:
            incomplete = dict(complete)
            incomplete.pop(key)
            self.assertFalse(module.recovery_evidence_present(incomplete))
```

Also reject stubbed/stale/mismatched journey evidence and Pi binary-only absence without dependency closure. Prior F evidence alone must no longer produce final release readiness. These booleans are a structural unit test, never authority to fabricate real evidence.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_final_release.py'`.
- [ ] Implement the subcheck and incorporate it into F's existing evidence validator:

```python
def recovery_evidence_present(evidence):
    keys = ('piAbsent','dependencyClosureVerified','recoveryMatrix',
            'compatibleRollback','retainingUninstall','reinstallPreserved')
    return all(evidence.get(key) is True for key in keys)
```

The actual journey runs on an explicitly authorized fresh disposable host where Pi was never installed, or is inaccessible to all test/runtime processes. Inspect host/container package and executable closure, mounted resources, subprocess/import/service use and generated configuration; Pi executables, npm/Python packages, credentials, extensions or daemon calls cannot supply any step. Merely changing PATH is insufficient. Planning/development documents may reference Pi; do not mistake those mentions for runtime dependencies or remove the developer's installation.

Clone reviewed `hermes-repokit` → install/default-only SAFE → private setup fixture → activate/no roster expansion → managed/provision then release → actual native team receives bounded improvement → optional research/planning → builder changes/tests → distinct reviewer independently approves exact candidate → host invokes improved immutable CLI apply → all state survives → qualified recovery/compatible rollback → retaining uninstall → reinstall with preserved state and suspended claims. Reuse F3/F4 native evidence and G1/G2 failure fixtures; do not replace the actual team with scripted approvals. Every subprocess uses the user-facing Node/Python/RepoKit/Hermes path without Pi. Record actual candidate/artifact/host/fixture identities and nonsecret evidence; missing authority is blocked/not-run, never pass.
- [ ] Green: the offline command and all default tests; later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/selfhost -p 'test_release_journey.py'`. Independent whole-branch/evidence review and explicit operator release remain required.
- [ ] Commit: `git add tests/selfhost/test_release_journey.py tests/offline/test_final_release.py tests/selfhost/release.py qualification/final-release-protocol.md qualification/release-matrix.md qualification/release-evidence.md README.md && git commit -m "test: gate release on Pi-independent recovery journey"`.

**Exit gate:** Actual A–F qualification plus G runtime/recovery/Pi-absent self-host evidence, independent review and explicit operator release. No runtime operations are authorized merely by this plan.

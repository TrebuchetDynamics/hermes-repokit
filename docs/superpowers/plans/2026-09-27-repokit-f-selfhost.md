> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit F — Preservation and Real Self-Hosting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove on a Pi-free host that a default-only install deliberately becomes a real engineering team, improves RepoKit, receives independent same-card review and survives approved host self-apply.

**Architecture:** Three separate evidence tiers prevent stub success from becoming release proof. Host locks/drain/immutable promotion govern apply; native records and authorized actual memory recall prove preservation.

**Tech Stack:** Node host lifecycle/promotion tests; Python opt-in runtime/inference harness; selected qualified A–E stack. No automatic deployment or paid CI. Pi/subagents are development-only; qualification commands and the installed team's work use native tools without Pi.

**Spec:** [Approved design](../specs/2026-09-27-repokit-team-runtime-design.md) §§10–11; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout).

## Global Constraints

Master constraints apply. No Docker socket/control endpoint inside Hermes. Self-hosting is actual model-backed work, not scripted transition simulation. Budgets, private setup, downloads and deployments are future explicit operator actions, not questions blocking this plan. No old-image-only migration rollback, kill-to-quiesce, erased history or fake completion.

## Review Focus

1. Clean host bootstrap cannot depend on its future agent stack (F1).
2. Empty board queue or held lock does not prove writer drain (F2).
3. Mutable PATH/worktree candidate substitution is host arbitrary-code execution (F2).
4. Provider/extraction cost ceilings must be enforceable, not monitoring-only (F3).
5. Surviving directories/hashes do not prove sessions, history or knowledge (F4).

---

### Task F1: Evidence tiers and cross-component preservation matrix

**Files:** Create `test/preservation.test.js`, `tests/runtime/test_apply_recovery.py`, `tests/selfhost/authority.py`, `tests/offline/test_fixture_authority.py`, `qualification/release-matrix.md`; modify `docs/host-lifecycle.md`, create `README.md`.

**Interfaces:** Consumes A–E qualifications/receipts. Produces `tests/selfhost/authority.py:require_authority(receipt:dict,tier:str)->None`; receipt contains fixture scope, selected digests, expiry, actor, data/egress/download ceilings, compute/provider limits and notification policy, no secrets. Test fixtures do not count as production qualifications.

- [ ] Write failing test:

```python
import importlib.util
import unittest

spec = importlib.util.spec_from_file_location('authority','tests/selfhost/authority.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class AuthorityTest(unittest.TestCase):
    def test_runtime_receipt_is_not_inference_permission(self):
        with self.assertRaises(PermissionError):
            module.require_authority({'tier':'runtime','fixture':'disposable'},'selfhost')
```

Node preservation tests use A transaction adapter with explicit synthetic owned state and inject interruption before/after each effect. Assert no-op zero restart/byte/mtime changes, sidecar-specific failure preserves unaffected services, stop-before-create single Hermes, no deletion/prune/volume removal, receipts distinguish actual effects and unknown recovery. Scope scanner cases include force replacement exact candidate/current caution findings and alias/fresh-session discovery.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_fixture_authority.py'`; `node --test test/preservation.test.js`.
- [ ] Implement tier admission core and extend expiry/digest/scope checks:

```python
def require_authority(receipt, tier):
    if receipt.get('tier') != tier or not receipt.get('fixture'):
        raise PermissionError('FIXTURE_AUTHORITY_REQUIRED')
```

Default tests run with no Docker/provider credentials and no network/model downloads. Runtime harness refuses requested tier without receipt (not skipped-green), provisions only disposable owned repo/context/artifacts, records actual mount/UID/network/profile/board/plugin identities and health/auth separately. Runtime stubs are explicitly labeled; selfhost tier refuses them. Clean host fixture starts from known reviewed RepoKit clone with documented A5 prerequisites; missing Docker/Compose/namespace support fails clearly, no privileged group or daemon installation. Fresh install is SAFE, one Hermes, native default only, native board, admitted non-autonomous sidecars/Nerve and enabled pinned Superpowers; pending private gates remain visible. Setup stages specialist private config and activate creates no specialists. The fixture snapshots native inventories before each step, then observes the five-role roster only during explicit managed preparation and no released claims until readiness passes. Re-running install or returning to safe preserves an existing roster.

README eventual operator path is install → setup → activate → mode managed → status, including eight command descriptions (state-preserving uninstall included), immutable selections, staged roster admission, independent readiness and opt-in tiers. The checkout/project/package name is `hermes-repokit`, not the temporary directory name. User-facing scripts/config/dependency closure must not require Pi. G3 runs this full path on a disposable host with Pi absent, not removed from the development host. This is documentation, not permission to execute now. Runtime test inventories planned preservation fields before and after interruption; no secret contents in report.
- [ ] Green both red commands. Authorized runtime later: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_apply_recovery.py'`; record exact tier/artifacts, not just exit code.
- [ ] Commit: `git add test/preservation.test.js tests/runtime/test_apply_recovery.py tests/selfhost/authority.py tests/offline/test_fixture_authority.py qualification/release-matrix.md README.md docs/host-lifecycle.md && git commit -m "test: separate runtime and real self-host evidence tiers"`.

### Task F2: Writer drain and immutable reviewed host candidate

**Files:** Create `src/promotion.js`, `test/promotion.test.js`, `tests/runtime/test_writer_drain.py`; modify `src/locks.js`, `test/locks.test.js`, `src/transaction.js`, `src/adapters/hermes.js`, `qualification/hermes-adapter.md`.

**Interfaces:** Consumes Selection, A lock FD and B native suspension. Produces `assertQuiescence(observations:object[])->Quiescence`, `promoteCandidate({source,releaseRoot,selection,reviewEvidence})->Promise<{path,digest}>`, `invokeCandidate({path,digest,argv,lockFd})->Promise<number>`. Candidate path is host-controlled versioned directory outside all Hermes mounts; digest covers entire executable source/package tree, not just shim.

- [ ] Write failing test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {assertQuiescence} from '../src/promotion.js';
test('unknown finalizer cannot be treated as drained', () => {
  const q = assertQuiescence([{kind:'workers',active:0,known:true},
    {kind:'session-finalizers',active:0,known:false}]);
  assert.equal(q.quiescent,false);
  assert.equal(q.unknown,true);
});
```

Add substituted PATH, changed candidate file after review/copy, symlink/hardlink destination, stale selection, wrong native candidate fingerprint and surviving mutator lock cases. Extend the real-process A3 fixture: inherited FD self-apply enters once without deadlock, unrelated probe remains busy, and forged/unrelated FD cannot bypass canonical lock. Real writer-drain fixture includes worker, interactive, gateway, setup, session commit/extraction and Nerve producer/consumer; active/unknown any affected writer blocks disruption. Stopping new claims is not cancelling active runs.
- [ ] Red: `node --test test/promotion.test.js`.
- [ ] Implement required-category quiescence core:

```js
export function assertQuiescence(observations) {
  const required = ['workers','interactive','gateway','setup','session-finalizers',
    'memory-extraction','nerve-producers','nerve-consumer'];
  const unknown = required.some(kind => !observations.some(x => x.kind === kind && x.known));
  const active = observations.some(x => x.known && x.active > 0);
  return {quiescent:!unknown && !active,unknown,
    reasons:unknown?['WRITER_STATE_UNKNOWN']:active?['WRITERS_ACTIVE']:[],
    observedAt:new Date().toISOString()};
}
```

Qualify all native claim/writer entry points in A5 files plus `agent/memory_manager.py` session finalization and OpenViking extraction lifecycle. Adapter suspends each admitted path, reconciles observed PIDs/runs/leases and waits with finite operator-selected drain deadline; unknown never becomes zero on timeout. Native completion/history is not rewritten to manufacture drain. Stop Nerve only after producers quiesce; record deliberate coverage gap. Sidecar-specific apply drains only affected shared state, but full self-apply covers all categories.

Host explicitly reviews/selects source revision and ReviewBinding/ExecutionEvidence. Under A lock, copy verified tree to exclusive versioned host releaseRoot/digest directory, no symlinks, verify tree against reviewed digest, fsync and atomically publish selector retaining previous known executable. Recheck immediately before direct invocation of immutable absolute shim with pinned supported Node, sanitized env and inherited lock FD; never invoke editable worktree or PATH alias. Read-only host files are not a claim of security against administrator/same-UID compromise; B2 supported tools cannot access releaseRoot/selectors/manifests or socket. Child inherits exclusion through crash. For self-apply handoff, `invokeCandidate` maps the held descriptor to FD 3 with host-internal `REPOKIT_ADMIN_FD=3`; A's `withAdminLock` validates that descriptor against the canonical admin.lock inode/owner/nlink and reacquires nonblocking on the same open-file description, rather than opening a competing descriptor. This is not a public CLI lock-bypass flag; invalid/missing handoff falls back to ordinary admission or refuses, never skips locking. Interrupted publication preserves old direct recovery path; no deletion of partial evidence until explicit safe recovery.
- [ ] Green red command; authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_writer_drain.py'` proves real writer coverage. If selected APIs cannot prove extraction/session drain, disruptive apply remains blocked.
- [ ] Commit: `git add src/promotion.js src/locks.js test/locks.test.js src/transaction.js src/adapters/hermes.js test/promotion.test.js tests/runtime/test_writer_drain.py qualification/hermes-adapter.md && git commit -m "feat: drain writers and invoke immutable reviewed CLI"`.

### Task F3: Actual bounded team dogfood on one change card

**Files:** Create `tests/selfhost/test_team_dogfood.py`, `tests/selfhost/budget.py`, `tests/offline/test_budget.py`, `qualification/selfhost-protocol.md`.

**Interfaces:** Consumes all A–E runtime qualifications, F1 authority, B ReviewBinding and native history. Produces `budget.exhausted(usage:dict,limits:dict)->bool` and actual nonsecret dogfood evidence with task/run/profile/candidate/acceptance/usage provenance. Harness observes native team work; it must not fabricate builder/reviewer calls or outcomes.

- [ ] Write failing test:

```python
import importlib.util
import unittest

spec = importlib.util.spec_from_file_location('budget','tests/selfhost/budget.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class BudgetTest(unittest.TestCase):
    def test_missing_usage_is_not_zero(self):
        self.assertTrue(module.exhausted({'tokens':None},{'tokens':100}))
        self.assertTrue(module.exhausted({'tokens':100},{'tokens':100}))
```

Actual fixture first asserts no synthetic adapter/provider/results, all selections qualified, memory privacy compatible and scoped runner live. Wrong/missing authority must fail before any inference or download. No source-only or offline evidence can satisfy selfhost tier.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_budget.py'`.
- [ ] Implement fail-closed ceiling check:

```python
def exhausted(usage, limits):
    return any(type(limit) not in (int,float) or limit <= 0
               or type(usage.get(key)) not in (int,float)
               or usage[key] >= limit for key, limit in limits.items())
```

Extend finite/nonnegative validation and reject empty limits. Protocol predeclares enforceable maxima: runs, correction/re-review cycles, wall time, aggregate tokens/external spend including memory embeddings/extraction, download bytes and local compute. Monitoring alone is not enforcement: use selected provider project hard caps/reservations and request ceilings, or qualified fully local bounded compute for relevant services. Hermes exposes no extraction spend cap; if selected remote extraction cannot be bounded through provider authority, fixture is blocked. No new proxy/gateway or pretend cap. Exhaustion suspends new claims, bounds/cancels supported pending requests, preserves active/incomplete evidence and reports incomplete, never forces done. Notifications remain off unless separately authorized.

Operator privately sets actual providers/keys, permitted nonsecret fixture lesson and data/retention/egress; authorize model acquisition/calibration separately. Activate eligible components without specialist creation, then explicitly select managed concurrency 1: journal/provision the engineering roster with claims suspended, verify actual profile/tool/authority evidence, and release only after the gates pass. Submit one real bounded RepoKit improvement (e.g. add stale-component age to `status`) with allowed paths, interfaces, acceptance and research/planner handoff only if useful. Actual default/builder/reviewer models perform work; researcher/planner participate only when useful. No Pi coordinator, Pi subprocess or scripted stand-in performs installed team work. Builder submits candidate and tests then requests same-card review; independent reviewer uses qualified snapshot tests, requests changes if needed and records exact candidate approval. A changes/re-review cycle may be separately induced within preapproved correction ceiling; never script its approval. Capture native IDs/runs/history, not transcripts/secrets; test rejects builder/default/self/stale completion. C3 actual shared recall/negative foreign-key and D4 calibrated typed workload are included under same explicit budget envelope or separately authorized bounded fixture receipts.
- [ ] Green offline command; actual future command `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/selfhost -p 'test_team_dogfood.py'` only after operator authority. Missing authority is a blocker report, not test success. Save actual candidate digest and native independent-review evidence for F4.
- [ ] Commit: `git add tests/selfhost/test_team_dogfood.py tests/selfhost/budget.py tests/offline/test_budget.py qualification/selfhost-protocol.md && git commit -m "test: require actual bounded same-card dogfood"`.

### Task F4: Real self-apply, preservation and self-host evidence

**Files:** Create `tests/selfhost/test_self_apply.py`, `tests/selfhost/release.py`, `tests/offline/test_release.py`, `qualification/release-evidence.md`; modify `README.md`, `qualification/release-matrix.md`.

**Interfaces:** Consumes F3 actual candidate, F2 immutable promotion/drain, A transaction and actual component evidence. Produces `release_ready(evidence:dict)->bool` as the shared evidence validator that G3 extends for final release; F alone does not authorize release. Evidence records separate offline/runtime/selfhost tiers and per-component selected artifact/qualification/freshness, no secret hashes.

- [ ] Write failing test:

```python
import importlib.util
import unittest

spec = importlib.util.spec_from_file_location('release','tests/selfhost/release.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class ReleaseTest(unittest.TestCase):
    def test_stubs_never_satisfy_actual_selfhosting(self):
        self.assertFalse(module.release_ready({'offline':True,'runtime':True,
                                               'selfhost':False,'stubbed':True}))
```

Add release rejection for directory-only memory persistence, missing shared recall, unqualified runner/artifact, synthetic approval, unknown drain, public secret fingerprints and missing recovery result. Real interruption points: before mutation, after drain/stop, during each component replacement, after effects/before receipt publication; actual at-most-one Hermes observation throughout.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_release.py'`.
- [ ] Implement release predicate core:

```python
def release_ready(evidence):
    required = ('offline','runtime','selfhost','independentReview','immutableApply',
                'preservedState','sharedRecall','crossRepoDenied','recoveryQualified')
    return evidence.get('stubbed') is False and all(evidence.get(k) is True for k in required)
```

Require evidence digests/artifacts/times in complete validator, not caller boolean assertions. Host reviews actual improved CLI, selects revision/digest and directly executes promoted candidate after all writer categories drain. Preserve source changes, canonical project/mount/board identity, logical Kanban cards/runs/events, addressable sessions, auth bytes/permissions, local memories, Nerve history and Laya cache. Credential comparisons happen privately and export booleans only (no secret hashes). SQLite uses consistent logical snapshots, not live-file hash equality. OpenViking preservation must include authorized actual durable write/extraction/recall before and after, not surviving directory. Keep cross-repo denial and all profile identity after restart.

Verify dispatch remains suspended and desired prior intent preserved until host explicitly re-enables managed after fresh gates. Interrupt/recover selected candidate self-apply; observe receipts/current actual identities before further effects, no blind replay. Retain previous known host executable/configuration and consistent backup. Data migration rollback only with separately tested compatible recovery, never just switch old image. Partial sidecar failure leaves unrelated services/state intact.

Release matrix names unsupported/blocked operations and tier separately. Optional E2E CI is opt-in only with explicit fixture/provider/budget/download/cleanup authority; it follows clean host → install → private fixture setup → activate → managed → actual task/review → drained immutable host apply → real preservation. No unspecified paid provider, auto-notifications or stub substitution.
- [ ] Green offline command and all default tests. Later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/selfhost -p 'test_self_apply.py'` must produce actual evidence before the self-host qualification verdict. G1–G3 must still pass before final release. Review full branch and release evidence independently.
- [ ] Commit: `git add tests/selfhost/test_self_apply.py tests/selfhost/release.py tests/offline/test_release.py qualification/release-evidence.md qualification/release-matrix.md README.md && git commit -m "test: gate release on preserved real self-apply"`.

**Exit gate:** Full qualified A–E plus F3/F4 actual inference/self-host evidence and independent reviewer approval. Final release additionally requires G recovery, Pi-absence and retaining-uninstall evidence plus explicit operator release. Partial components, offline green or healthy containers are not completion.

> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit C — Native OpenViking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provision isolated native memory infrastructure and admit sessions only when the preserved privacy contract is actually supported.

**Architecture:** One internal OpenViking service per repository; Hermes native external provider only. Infrastructure observation is independent of session/upload authorization and real memory qualification.

**Tech Stack:** Node host configuration, Python native-provider contract tests, selected official OpenViking image, native profile secrets.

**Spec:** [Approved design](../specs/2026-09-27-repokit-team-runtime-design.md) §§2,6,9; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout).

## Global Constraints

Master constraints apply. No second provider, monkey patch, interception/filter pipeline, broad transcript consent or invented opt-out flag. Native enhancement scope approval is not authority to modify external repositories. Research pin is incompatible for ordinary memory-connected sessions; affected sessions remain blocked, not unrelated infrastructure. No automatic source ingestion or v1 `memory index` command.

## Review Focus

1. Environment/linked ovcli overrides can silently change identity/peer (C1/C3).
2. Turn sync, shutdown and recovery disclose before extraction (C2).
3. User key does not hide root key through shared mounts (C3).
4. HTTP health does not prove authorization/recall (C4).
5. Outage/ingestion must not imply lossless delivery or secret filtering (C4).

---

### Task C1: Sidecar provisioning and explicit native identity

**Files:** Create `components/openviking.js`, `test/openviking.test.js`, `qualification/openviking-artifact.md`; modify `src/compose.js`, `src/adapters/hermes.js`.

**Interfaces:** Consumes A Identity/Selection and privateIO. Produces `nativeMemoryConfig(repoId:string)->object`, `openVikingConfig()->object`; private provisioning creates project-bound ordinary key and root-only server config through FDs, never returned by these pure functions.

- [ ] Write failing test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {nativeMemoryConfig,openVikingConfig} from '../components/openviking.js';
test('native identity has no peer or embedded key', () => {
  const cfg = nativeMemoryConfig('repo-one');
  assert.equal(cfg.memory.provider,'openviking');
  assert.equal(cfg.memory.openviking.user,'repo-one');
  assert.equal(Object.hasOwn(cfg.memory.openviking,'agent'),false);
  assert.equal(JSON.stringify(cfg).includes('api_key'),false);
  assert.equal(openVikingConfig().storage.workspace,'/app/.openviking/data');
});
```

Add Compose snapshots for `.hermes/openviking:/app/.openviking`, no public ports/repo bind, bot disabled, independent repo network/data, auth-mode API key; null image selection refuses provisioning. No root/user key in reports.
- [ ] Red: `node --test test/openviking.test.js`.
- [ ] Inspect selected OpenViking `Dockerfile`, `deploy/docker/openviking-entrypoint.sh`, `docs/en/guides/01-configuration.md`, `openviking/server/auth/plugins/api_key.py`, `docs/en/guides/04-authentication.md`. Questions: exact supported bot-disable flag/env; local AGFS/vector defaults; service UID and config ownership; bootstrap account/user creation and private key delivery; official digest/platform. Record accepted image/source relationship and license obligations; source review alone cannot admit production.
- [ ] Implement core native config:

```js
export function nativeMemoryConfig(repoId) {
  return {memory:{provider:'openviking',memory_enabled:true,user_profile_enabled:true,
    openviking:{endpoint:'http://openviking:1933',account:'repokit',user:repoId}}};
}
export function openVikingConfig() {
  return {server:{auth_mode:'api_key'},storage:{workspace:'/app/.openviking/data'}};
}
```

Complete provider/backend config only from qualified native fields; not generic manifest YAML. Intentionally provision shared project identity without cloning personal credentials. Before Managed entry, configure native default only and stage intended specialist settings privately; do not create specialist profiles to satisfy memory setup. Apply the shared identity to each specialist during managed preparation. Set `OPENVIKING_API_KEY` via approved native profile secret configuration. Enforce no agent/peer in environment, linked ovcli or YAML; disallowed effective override blocks memory sessions. Egress remains an explicit selection, never opened to repair health. Provisioning may proceed while C2 session gate is blocked.
- [ ] Green red command and A Compose tests.
- [ ] Commit: `git add components/openviking.js test/openviking.test.js qualification/openviking-artifact.md src/compose.js src/adapters/hermes.js && git commit -m "feat: provision native project memory identity"`.

### Task C2: Native privacy qualification, not a fabricated feature

**Files:** Create `runtime/repokit/memory_contract.py`, `tests/offline/test_memory_privacy.py`, `tests/runtime/test_native_upload_contract.py`, `qualification/native-upload-contract.md`; modify `src/plan.js`.

**Interfaces:** Consumes native artifact/source qualification and scoped data authority. Produces `privacy_gate(evidence:dict)->Gate`; evidence fields: turnUploadDisabled, toolUploadDisabled, switchCommitDisabled, endCommitDisabled, recoveryUploadDisabled, mirroringDisabled, deliberateMemoryPreserved, qualifiedRevision. All must be verified against the selected native build.

- [ ] Write failing test:

```python
import unittest
from repokit.memory_contract import privacy_gate

class PrivacyTest(unittest.TestCase):
    def test_extraction_disable_is_not_upload_disable(self):
        result = privacy_gate({'extractionEnabled':False,
                               'qualifiedRevision':'research-only'})
        self.assertFalse(result['allowed'])
```

Build a native-provider transport recorder fixture for synthetic forbidden markers: ordinary turn + non-recall tool, session switch, session end, pending-session recovery, built-in mirror. Assert zero automatic upload/commit/mirroring and successful separately invoked deliberate remember/recall. Exercise actual native lifecycle APIs at selected revision, not a RepoKit substitute provider.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_memory_privacy.py'`.
- [ ] Inspect Hermes `plugins/memory/openviking/__init__.py` sync_turn/commit/recovery/config resolver, `agent/memory_manager.py` background lifecycle, `agent/agent_init.py` memory initialization, and `tools/memory_tool.py` mirroring; OpenViking `openviking_cli/utils/config/memory_config.py`. Record exact feature absence/presence and native API paths tested. At research SHA the acceptance result is **unsupported**, not a made-up config. Write enhancement acceptance contract for a native opt-out covering all six automatic paths while preserving deliberate memory; any external implementation needs separately authorized repository work and a newly selected qualified build.
- [ ] Implement gate core, wiring only documented native settings if later actually qualified:

```python
def privacy_gate(evidence):
    keys = ('turnUploadDisabled','toolUploadDisabled','switchCommitDisabled',
            'endCommitDisabled','recoveryUploadDisabled','mirroringDisabled',
            'deliberateMemoryPreserved')
    reasons = [key for key in keys if evidence.get(key) is not True]
    if not evidence.get('qualifiedRevision'):
        reasons.append('UNQUALIFIED_REVISION')
    return {'allowed':not reasons,'reasons':reasons}
```

No local monkey patch or hidden tool substitute. Fixture contract can be green while selected research build remains unsupported; report those separately. Existing raw-session retention/remediation requires distinct authority, never automatic deletion. Missing compatibility only withholds affected memory-connected sessions/private processing.
- [ ] Green offline command; future authorized native contract command `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_native_upload_contract.py'`. Production memory gate requires actual native zero-disclosure result, never a recorder-only simulation.
- [ ] Commit: `git add runtime/repokit/memory_contract.py tests/offline/test_memory_privacy.py tests/runtime/test_native_upload_contract.py qualification/native-upload-contract.md src/plan.js && git commit -m "feat: require native privacy compatibility for sessions"`.

### Task C3: Secret boundaries and key-bound shared profiles

**Files:** Create `tests/offline/test_memory_identity.py`, `tests/runtime/test_memory_identity.py`, `qualification/memory-identity.md`; modify `runtime/repokit/memory_contract.py`, `src/adapters/hermes.js`.

**Interfaces:** Consumes B1 profile contracts/B2 runner, B4 prepared engineering roster with dispatch still suspended, and C1/C2 gates. No requirement to release claims before qualification. Produces `validate_identity(resolved:dict,repo_id:str)->bool`, operation qualification proving root secrecy, authorized project key sharing and cross-repo denial.

- [ ] Write failing test:

```python
import unittest
from repokit.memory_contract import validate_identity

class IdentityTest(unittest.TestCase):
    def test_peer_override_rejects(self):
        cfg = dict(endpoint='http://openviking:1933',account='repokit',
                   user='one',agent='foreign')
        self.assertFalse(validate_identity(cfg,'one'))
```

Add endpoint/user/account precedence mismatches and missing key binding. Runtime test checks actual service UIDs and read/exec tools against `/opt/data/openviking/ov.conf` and `/workspace/.hermes/openviking/ov.conf`, root config writes and symlink aliases. Ordinary user credentials may be available only to admitted native memory implementation, not generic read tools.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_memory_identity.py'`.
- [ ] Implement effective identity predicate:

```python
def validate_identity(resolved, repo_id):
    return (resolved.get('endpoint') == 'http://openviking:1933'
            and resolved.get('account') == 'repokit'
            and resolved.get('user') == repo_id
            and resolved.get('agent') in (None,''))
```

Combine with actual key-bound account/user evidence; namespace text alone is insufficient. Inspect Hermes native resolver precedence and OpenViking `openviking/core/namespace.py` plus API-key plugin at exact selected revisions. Qualify minimal per-service ownership/modes and B2 mount exclusion without inventing new secret infrastructure or recursive chown. If official Hermes UID can access root key through a supported tool/runner, withhold that surface/dispatch; do not equate a user key with isolation.

Authorized inference tier writes one permitted durable lesson through native remember, waits for observed extraction/indexing, then recalls through all five independently configured profiles. A second disposable repo with distinct service/network/key must fail foreign-key, URI/header and account-scope spoof attempts. Real recall, not directory existence, establishes sharing. Admission revocation blocks subsequent affected processing without claiming erasure.
- [ ] Green offline command; runtime denial command uses `tests/runtime/test_memory_identity.py`; actual write/recall run additionally requires F3 budget/data ceilings and compatible C2 native build.
- [ ] Commit: `git add runtime/repokit/memory_contract.py src/adapters/hermes.js tests/offline/test_memory_identity.py tests/runtime/test_memory_identity.py qualification/memory-identity.md && git commit -m "feat: qualify key-bound memory and secret isolation"`.

### Task C4: Independent status, ingestion admission and outage truthfulness

**Files:** Create `test/memory-status.test.js`, `tests/offline/test_ingestion.py`; modify `components/openviking.js`, `src/observe.js`, `runtime/repokit/memory_contract.py`, `docs/host-lifecycle.md`.

**Interfaces:** Consumes ComponentObservation and C gates. Produces `memoryObservation({health,ready,auth,privacyCompatible,observedAt})->ComponentObservation`, `admit_resource(path:str,ignored:bool,size:int,linked:bool)->bool`; resource admission is future internal contract only, no new command/automatic ingestion.

- [ ] Write failing test:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {memoryObservation} from '../components/openviking.js';
test('health does not imply auth or compatible sessions', () => {
  const result = memoryObservation({health:true,ready:true,auth:false,
    privacyCompatible:false,observedAt:'2026-09-27T00:00:00Z'});
  assert.equal(result.verdict,'blocked');
  assert.notEqual(result.evidence,'unknown');
});
```

Add ready 503/pending, unavailable external recall, uncertain remember delivery, extraction pending and stale observations. Cross-component fixtures keep authorized Kanban work running during memory outage, retain provider selection, and reject newly privacy-incompatible sessions; generic health must not become a dispatch dependency. Ingestion tests reject ignored/linked/oversized (1 MiB proposed bound), `.env*`, keys, `.git`, `.hermes`, dependencies/build/dumps even beneath docs; admit only explicit README*/AGENTS.md/CONTRIBUTING*/docs/**/adr/**/architecture/**/specs/** with ownership checks and approved contents. File patterns alone do not detect secrets.
- [ ] Red: `node --test test/memory-status.test.js`; `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_ingestion.py'`.
- [ ] Implement core observation:

```js
export function memoryObservation(x) {
  const blocked = x.auth === false || x.privacyCompatible === false;
  const known = [x.health,x.ready,x.auth,x.privacyCompatible].every(v => typeof v === 'boolean');
  return {component:'openviking',verdict:blocked?'blocked':
    known && x.health && x.ready?'healthy':'degraded',
    evidence:known?'known':'unknown',observedAt:x.observedAt,
    desiredAlignment:'unknown',reasons:blocked?['MEMORY_SESSION_BLOCKED']:[]};
}
```

Qualify `/health` and `/ready` through `openviking/server/routers/system.py`; status uses nonmutating probes only, never remember or credential refresh. Keep native built-in memory/user profile enabled under profile-local `memories/`; no replication/provider switch/deletion/backfill promise. Pending recovery is separate from delivery guarantees. Working-method docs require attempting permitted recall before substantive work and deliberate durable lessons afterward, never raw logs/diffs; unavailable recall is reported explicitly and does not stop already-authorized engineering work or produce imaginary context. Queue/retry/delivery is not promised, and privacy-incompatible sessions remain gated; explicitly state policy does not prevent native background upload. Curated upload remains disabled until native resource admission/snapshot behavior and opt-in data scope qualify; no watch or implicit `recall_resources` enablement.
- [ ] Green both commands, plus authorized outage/persistence cases in C3/F4.
- [ ] Commit: `git add components/openviking.js src/observe.js runtime/repokit/memory_contract.py test/memory-status.test.js tests/offline/test_ingestion.py docs/host-lifecycle.md && git commit -m "feat: report truthful native memory readiness"`.

**Exit gate:** Independent infrastructure can be useful before privacy resolution. Full memory support requires compatible native build, real protected-key evidence, authorized shared recall/cross-repo denial and persistence; none is established by writing this plan.

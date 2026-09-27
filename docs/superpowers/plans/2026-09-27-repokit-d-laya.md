> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit D — Reproducible Laya Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Supply a provenance-bound, bounded typed sidecar and strict advisory client without pretending stock release support exists.

**Architecture:** Pin-aware bootstrap injects the selected Router into Laya's supported application factory; server admission bounds actual ongoing work. Nerve owns a fixed rubric and locally correlates responses to current native evidence.

**Tech Stack:** Python 3.11+, standard-library offline tests, selected `laya[serve]`/FastAPI sidecar dependencies locked only during authorized packaging. No torch inside Hermes.

**Spec:** [Approved design](../specs/2026-09-27-repokit-team-runtime-design.md) §8; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout); [source findings](../../research/2026-09-27-repokit-integration-findings.md).

## Global Constraints

Master constraints apply. No qualified official Laya image currently established. R and S both report 0.3.20 but differ; released wheel cannot inherit S features. No model acquisition during planning/default tests. Router selects checkpoints, never profiles. Unavailable/low confidence is UNKNOWN/WATCH, never fallback BLOCK.

## Review Focus

1. Offline cache mode does not prove immutable artifact selection (D1).
2. Questions/answers are mappings, not arrays; noul is scalar (D2/D3).
3. Alias auto-routing and stale local evidence can invalidate a syntactically valid answer (D3).
4. Client timeout does not end server compute (D4).
5. Context truncation/calibration gaps must not become confident assessments (D2/D4).

---

### Task D1: Pin-aware artifact bootstrap with honest unsupported baselines

**Files:** Create `components/__init__.py`, `components/laya/__init__.py`, `components/laya/artifacts.py`, `components/laya/bootstrap.py`, `components/laya/Dockerfile`, `components/laya/requirements.lock`, `tests/offline/test_laya_artifacts.py`, `tests/runtime/test_laya_artifacts.py`, `qualification/laya-artifacts.md`.

**Interfaces:** Consumes A Selection/catalog, download/compute authority. Produces `artifacts.verify_file(path:Path,expected_sha256:str)->bool`, `bootstrap.build_router(selection:dict)->object` using qualified native constructor, never monkey patch. Selection binds source/package/wheel/dependency-lock/base-image/image/model revision and all tokenizer/config/weight hashes, alias/display mapping and cache policy.

- [ ] Write failing test:

```python
import hashlib
import tempfile
import unittest
from pathlib import Path
from components.laya.artifacts import verify_file

class ArtifactsTest(unittest.TestCase):
    def test_cache_content_not_environment_controls_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'config.json'
            path.write_bytes(b'changed')
            self.assertFalse(verify_file(path,hashlib.sha256(b'reviewed').hexdigest()))
```

Add missing/extra artifact, symlink cache, modified tokenizer, same package version/different source, wrong offline cache revision and absent weights. Tests use tiny non-model bytes only. Missing weights must never initiate downloads without acquisition authority.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_laya_artifacts.py'`.
- [ ] Inspect R `laya/agent.py:211–270`, S `laya/agent.py`, `laya/revisions.py`, `laya/router.py`, `laya/serve.py:230–240`, `Dockerfile`, `pyproject.toml`. Record exact constructor revision/digest arguments, factory injection, lazy loading, tokenizer writes, HTTP startup and UID/GID. Acceptance: source-bound constructor test captures all pinned load parameters; selected server never constructs a default unpinned Router. R must be reported unsupported; S stock entrypoint also unsupported unless the reviewed bootstrap wiring passes. Do not fake a nonexistent stock CLI revision flag.
- [ ] Implement streaming verification core:

```python
import hashlib

def verify_file(path, expected_sha256):
    if path.is_symlink() or not path.is_file():
        return False
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest() == expected_sha256
```

Complete descriptor-based ownership/no-follow checks from A safety contract. Bootstrap constructs only qualified selected Router via documented `create_app(router=...)`; test captures actual pin-aware constructor calls at selected S or compatible revision. If no supported constructor can bind all artifacts, leave component unsupported, not a pretend adapter. Immutable acquisition directory is verified before copying into writable runtime cache; revalidate allowed tokenizer transformations against recorded hashes, never mutate original acquisition material.

During later authorized packaging, generate complete hash-locked transitive requirements and base/image digest provenance; commit actual lock, never a fabricated lock or floating tag. Until generated, packaging substep is blocked with report, not a success containing empty lock. Use UID/GID 10001 only after image verification, cache `/home/laya/.cache/huggingface`, explicit HTTP startup, no host port/repo bind. Runtime fixture proves offline selection, permissions, loaded checkpoint and restart persistence. Resource sizes are measured for authorized workload, not universal 8GB/VRAM promises.
- [ ] Green offline command; authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_laya_artifacts.py'` before artifact admission. No downloads implied by offline pass.
- [ ] Commit: completed contract/source work uses `git add components tests/offline/test_laya_artifacts.py tests/runtime/test_laya_artifacts.py qualification/laya-artifacts.md && git commit -m "feat: bind Laya bootstrap to verified artifacts"`. Include Dockerfile/lock only once actual packaging substep completes; report blocked packaging explicitly otherwise.

### Task D2: Fixed versioned mapping rubric and context admission

**Files:** Create `runtime/repokit_nerve/rubric.py`, `tests/offline/test_rubric.py`; extend `contracts/repokit-v1.json` and `runtime/repokit/contracts.py` with master assessment DTO.

**Interfaces:** Consumes AssessmentInputV1. Produces `QUESTIONS:dict`, `make_request(observation:dict,model_alias:str)->dict`. Trusted context supplies admitted model alias, not tool caller. State contains sanitized DTO only; no arbitrary prompts/source/logs. B1 supplies shared Python validators; D1's reviewed artifact contract supplies the alias schema but actual acquisition is not required for synthetic offline rubric tests. E1 owns `__init__.py` and native plugin registration; before E1 these pure modules use Python namespace-package imports, without loading a plugin or model.

- [ ] Write failing test:

```python
import unittest
from repokit_nerve.rubric import QUESTIONS

class RubricTest(unittest.TestCase):
    def test_four_typed_mapping_entries(self):
        self.assertEqual(set(QUESTIONS),{'progress','evidence','escalation','retry'})
        self.assertEqual(QUESTIONS['evidence']['criteria'],
                         ['insufficient evidence','complete evidence'])
        self.assertEqual(QUESTIONS['escalation']['type'],'noul')
```

Add reject incomplete coverage, unknown alias, arbitrary question/model parameters and oversized state; context accounting includes system/rubric/tokenizer overhead. If exact selected tokenizer capacity cannot be checked within qualified contract, return UNKNOWN before HTTP rather than let upstream truncate.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_rubric.py'`.
- [ ] Implement immutable rubric version `repokit.nerve.rubric.v1` and deep-copy requests:

```python
QUESTIONS = {
 'progress': {'type':'choice','instructions':'Recommend supervision only from observed evidence.',
  'criteria':{'CONTINUE':'No observed concern','WATCH':'Uncertain or incomplete evidence',
              'REPLAN':'Observed approach repeatedly fails','BLOCK':'Evidence warrants human attention'}},
 'evidence': {'type':'score','instructions':'Rate observed acceptance evidence, not truth.',
  'criteria':['insufficient evidence','complete evidence']},
 'escalation': {'type':'noul','instructions':'Estimate need for human attention; never authorize actions.',
  'criteria':{'false':'No observed need','true':'Observed need for attention'}},
 'retry': {'type':'choice','instructions':'Recommend a bounded next consideration, not execution.',
  'criteria':{'retry-same':'Transient evidence with unchanged approach',
              'retry-after-delay':'Temporary unavailable dependency',
              'replan':'Approach needs revision','human-input':'Authority or context is missing'}}
}
```

`make_request` uses exact `{state,questions,model}` object, JSON byte cap 16 KiB, DTO allowlist and D1 alias allowlist. Validate counters are null or bounded/nonnegative and units unchanged; no invented zeros. Record context-capacity qualification from `laya/common.py` tokenization/truncation and selected model configs. Native context-length limits must be known for KNOWN eligibility.
- [ ] Green red command plus DTO golden fixture round-trip under Node/Python validators.
- [ ] Commit: `git add runtime/repokit_nerve runtime/repokit/contracts.py contracts/repokit-v1.json tests/offline/test_rubric.py && git commit -m "feat: freeze bounded Nerve typed rubric"`.

### Task D3: Strict actual answer mapping and provenance correlation

**Files:** Create `runtime/repokit_nerve/answers.py`, `runtime/repokit_nerve/laya_client.py`, `tests/offline/test_laya_answers.py`.

**Interfaces:** Consumes D1 selected artifact/alias/display, D2 request and master Binding. Produces `validate_response(response,request,model_display)->dict`, `validate_noul(answer:dict)->float`, `assess(observation,context)->AssessmentResultV1`.

- [ ] Write failing test:

```python
import unittest
from repokit_nerve.answers import validate_noul

class AnswerTest(unittest.TestCase):
    def test_noul_is_probability_scalar_not_probability_map(self):
        self.assertEqual(validate_noul({'noul':0.75,'confidence':0.75,
                                       'answer_confidence':0.75}),0.75)
        for value in [True,float('nan'),-0.1,1.1,{'true':0.75}]:
            with self.assertRaises(ValueError):
                validate_noul({'noul':value,'confidence':0.75,'answer_confidence':0.75})
```

Add complete literal actual-source answer fixture in test file: progress/retry choice label + complete probabilities; evidence score + legend `{'0':'insufficient evidence','1':'complete evidence'}` and probabilities; escalation scalar noul, no required probabilities. Test missing/extra IDs, malformed type/enum, NaN, extra probability labels, sum error >0.001, score outside [0,1], wrong legend/model/routing, stale local observation and deployment digest mismatch. `action` metadata never grants authority. Selected adapter defines admitted extra native fields exactly, not arbitrary permissiveness.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_laya_answers.py'`.
- [ ] Inspect selected `laya/agent.py:565–814`, `laya/router.py:178–239`, `laya/serve.py` to capture exact answer keys/types, display versus alias and usage/routing shape. No assumption top-level display equals request alias; verify known mapping and routed checkpoint.
- [ ] Implement strict answer validation. Core scalar validation:

```python
import math

def validate_noul(answer):
    for key in ('noul','confidence','answer_confidence'):
        value = answer.get(key)
        if type(value) not in (int,float) or not math.isfinite(value) or not 0 <= value <= 1:
            raise ValueError('INVALID_ANSWER')
    return answer['noul']
```

Choice/score validate all finite probability values and sum/legend; score means expected ordinal, not truth probability. Capture Binding before sending, re-read current evidence on completion, reject superseded fingerprint/run/candidate/selection. Provenance comes from trusted deployment receipt, not invented response fields. Return KNOWN only after calibrated thresholds and complete context; reasons use E evidence templates, never model rationale. Everything else returns UNKNOWN and optional explicitly stale last-known; advisory renderer shows WATCH.
- [ ] Green red command; run D2 tests.
- [ ] Commit: `git add runtime/repokit_nerve/answers.py runtime/repokit_nerve/laya_client.py tests/offline/test_laya_answers.py && git commit -m "feat: validate typed answers and local evidence binding"`.

### Task D4: Bound client work and actual server admission

**Files:** Create `components/laya/admission.py`, `tests/offline/test_laya_bounds.py`, `tests/runtime/test_laya_admission.py`, `tests/selfhost/test_laya_calibration.py`; modify `components/laya/bootstrap.py`, `runtime/repokit_nerve/laya_client.py`.

**Interfaces:** Consumes master assessment request/result. Produces `Admission.try_enter()->bool`, `Admission.leave()->None`; `assess` context injects bounded HTTP transport/monotonic clock for tests. Server ceiling initially one active computation, zero waiting requests; exceeding returns bounded 503, not another model process.

- [ ] Write failing test:

```python
import unittest
from components.laya.admission import Admission

class BoundsTest(unittest.TestCase):
    def test_no_queue_after_client_disconnect(self):
        admission = Admission()
        self.assertTrue(admission.try_enter())
        self.assertFalse(admission.try_enter())
        admission.leave()
        self.assertTrue(admission.try_enter())
        admission.leave()
```

Add fake-clock client tests: one in-flight, <=32 coalesced pending tasks, 16 KiB request/response caps, connect 1s/total 10s, redirects refused, at most one delayed transient retry, cooldown 30s, permanent malformed/auth failures not retried. Restart cannot create catch-up storm. Server test disconnects client mid-compute, next request still rejects until original work actually ends.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_laya_bounds.py'`.
- [ ] Implement nonblocking admission:

```python
import threading

class Admission:
    def __init__(self):
        self._slot = threading.BoundedSemaphore(1)
    def try_enter(self):
        return self._slot.acquire(blocking=False)
    def leave(self):
        self._slot.release()
```

Wire at actual model execution boundary in reviewed bootstrap/app integration; release only when compute finishes, not request cancellation. S native admission may be used only after equivalent test; R unbounded lock is unacceptable. API key gates `/v1/systemone`; `/health` never implies inference authorization. Enforce selected CPU/memory/thread budgets and no process restart on timeout. If native factory cannot enforce bounded cancellation/admission, component remains unsupported until qualified packaging, not client-only workaround.

Selfhost calibration uses actual authorized model workload, finite cases spanning four recommendations, low confidence/incomplete/outage, measured context and resource use. Freeze threshold configuration with artifact/rubric and held-out results; no universal accuracy claims. Budget exhaustion reports incomplete, no downloading replacement model/auto-route.
- [ ] Green offline command; later authorized runtime admission and selfhost calibration commands: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_laya_admission.py'` and `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/selfhost -p 'test_laya_calibration.py'`. Separate reports for actual model versus stub compute.
- [ ] Commit: `git add components/laya runtime/repokit_nerve/laya_client.py tests/offline/test_laya_bounds.py tests/runtime/test_laya_admission.py tests/selfhost/test_laya_calibration.py && git commit -m "feat: bound Laya serving and advisory inference"`.

**Exit gate:** Contract tests admit neither image nor model. Production needs locked packaging/provenance, actual bounded serving and scoped download/inference authority; full readiness also needs calibration.

> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit B — Native Fleet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform a default-only SAFE install into the five-role engineering team on explicit Managed admission, with scoped execution and independent same-card review.

**Architecture:** Native profiles and Kanban retain identity/lifecycle. A trusted compiled plugin mediates supported mutation tools; snapshot-bound execution excludes protected state. No Nerve enforcement or separate scheduler.

**Tech Stack:** Python standard library and selected Hermes native plugin/tool APIs; Bubblewrap candidate on qualified Linux user-namespace hosts; Node host mode bridge.

**Spec:** [Approved design](../specs/2026-09-27-repokit-team-runtime-design.md) §§3–5; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout).

## Global Constraints

Master constraints apply. Default is root home, never `profiles/default`. Fresh install/setup/activate create no native specialists; first managed entry provisions them under suspended claims. SAFE downgrade retains all profiles/history. `goal_mode` is excluded until separately qualified. No credential cloning. `file`/`safe` are not read-only. Native worker lifecycle auto-append must be measured. No unrestricted reviewer shell or comprehensive same-UID security claim. Unsupported execution blocks affected dispatch, not observation.

## Review Focus

1. Failed tool resolution must not drop explicit worker pin (B1).
2. Repository tests can be hostile; both `.hermes` aliases and process/network authority need denial (B2).
3. Hook exceptions and race windows must not allow native mutation (B3).
4. Same profile/different run is not independent builder/reviewer identity (B3).
5. Safe downgrade preserves active provenance; fleet refuses overlapping writers (B4).

---

### Task B1: Native fleet provisioning and actual schema admission

**Files:** Create `runtime/repokit/__init__.py`, `runtime/repokit/plugin.yaml`, `runtime/repokit/contracts.py`, `runtime/repokit/native.py`, `runtime/repokit/fleet.py`, `tests/offline/test_fleet.py`, `tests/runtime/test_fleet_schema.py`, `qualification/fleet.md`; modify `src/adapters/hermes.js`.

**Interfaces:** Consumes A Identity/Qualification and shared JSON schema. Produces `Adapter.provisionProfiles(identity,selection,privateIO)->Receipt` for managed preparation only, `repokit.register(ctx)->None` for the separate trusted `repokit-policy` plugin, `fleet.profile_specs()->dict`, `fleet.admit_schema(profile:str,names:set[str],policy_loaded:bool)->dict`, native adapter `observe_board` and `transition` signatures from master. Python contracts validates identical wire keys against shared golden fixtures; no runtime dependency on a second schema framework.

- [ ] Write failing test:

```python
import unittest
from repokit.fleet import admit_schema, profile_specs

class FleetTest(unittest.TestCase):
    def test_default_cannot_implement(self):
        self.assertFalse(admit_schema('default', {'terminal', 'file_write'}, True)['allowed'])
        self.assertFalse(admit_schema('reviewer', set(), False)['allowed'])
        self.assertEqual(set(profile_specs()),
                         {'default','researcher','planner','builder','reviewer'})
        self.assertTrue(admit_schema('builder', {'repokit_apply_patch'}, True)['allowed'])
        self.assertFalse(admit_schema('reviewer', {'repokit_apply_patch'}, True)['allowed'])
        self.assertTrue(admit_schema('default', {'nerve_status'}, True)['allowed'])
        self.assertFalse(admit_schema('default', {'goal_mode'}, True)['allowed'])
```

Add actual resolved-schema fixtures covering write-capable file, browser, plugin/MCP connectors; missing custom read toolset; failed worker resolution; lifecycle auto-append followed by subtractive deny-toolsets. Assert no silently inherited broad tools; read roles cannot gain implementation via secondary connector. Canary personal/default credential is absent from specialist profiles. `profile_specs()` is a side-effect-free five-role blueprint, not an instruction to instantiate all roles at install. Native lifecycle fixtures observe default only through install/setup/activate, then exactly five profiles after managed preparation; partial retries preserve existing profile bytes and independently staged credential selections.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_fleet.py'`.
- [ ] Inspect pinned `hermes_cli/profiles.py` creation/clone/default-home; `toolsets.py`; `model_tools.py` expansion/auto-append; `hermes_cli/kanban_db_dispatch.py` worker selectors; `hermes_cli/plugins.py` registration/veto. Record exact custom toolset registration and profile description/config argv. Acceptance: default-only inventory before managed, five actual profile/session schemas after managed preparation and loaded policy attestations tied to artifact, not desired YAML alone. Qualify specialist creation from protected private setup without personal/default credential copying, plus partial creation discovery/retry. Goal-mode settings/tools/aliases must not provide another claim/completion path; record qualified native exclusion rather than inventing an upstream config flag.
- [ ] Implement descriptions and strict schema admission core:

```python
ADVISORY = {'nerve_assess', 'nerve_status', 'nerve_explain', 'nerve_history'}
ALLOWED = {
    'default': {'repokit_read', 'repokit_transition', 'repokit_memory'},
    'researcher': {'repokit_read', 'repokit_research', 'repokit_transition', 'repokit_memory'},
    'planner': {'repokit_read', 'repokit_plan_handoff', 'repokit_transition', 'repokit_memory'},
    'builder': {'repokit_read', 'repokit_apply_patch', 'repokit_execute',
                'repokit_transition', 'repokit_memory'},
    'reviewer': {'repokit_read', 'repokit_test', 'repokit_transition', 'repokit_memory'},
}
ALLOWED = {role: names | ADVISORY for role, names in ALLOWED.items()}
def admit_schema(profile, names, policy_loaded):
    allowed = policy_loaded and profile in ALLOWED and names <= ALLOWED[profile]
    return {'allowed': bool(allowed), 'reasons': [] if allowed else ['SCHEMA_DENIED']}
```

Package native `plugin.yaml` plus `register(ctx)` as `repokit-policy`, separately from advisory `repokit-nerve`. Qualify manifest capabilities/discovery/enablement and cross-package imports against `hermes_cli/plugins_manifest.py` and `hermes_cli/plugins.py`; fresh runtime fixtures load the actual installed artifacts in every worker, not just via test PYTHONPATH. Selected trusted plugin trees/hashes are included in artifact admission. Advisory names are permitted when E4 is installed but absent advisory tools do not prevent unrelated role readiness.

These are RepoKit-owned exposed tool names, not invented upstream selectors. fleet.py produces `repokit_read(path:str)->dict` (bounded permitted file read), `repokit_research(query:str)->dict` (admitted search connectors only), `repokit_plan_handoff(task_id:str,contract:dict)->dict` (native non-source card context), `repokit_memory(action:str,payload:dict)->dict` (fixed native recall/remember operations through the selected provider), `repokit_apply_patch(task_id:str,base_digest:str,patch:str)->dict` (B2 builder-only bounded patch submission), `repokit_execute(test_plan_id:str)->dict`, `repokit_test(test_plan_id:str)->dict` (B2 approved plans, never caller argv), and `repokit_transition(action:str,task_id:str|None,payload:dict)->dict` (B3 action-specific structured fields, never caller role/run authority). All are role-bound registrations, not generic arbitrary tool proxies. C gates apply before memory-connected sessions and memory operations. In fleet.py register narrow read/research/handoff/native-memory wrappers, patch/execute/test delegate B2, transition delegates B3. Incomplete wrappers are withheld, never dummy success. Pre-tool policy denies non-admitted tools and protected aliases on all enabled surfaces. Omit unavailable tools, but required role tools must be present before release. Native adapter converts exact selected native records; native board reads use noninitializing WAL-aware path qualified in E3. No direct SQL writes.
- [ ] Green same offline command; later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_fleet_schema.py'` captures actual fresh session/worker schemas. Missing runtime evidence leaves roles inspection-only.
- [ ] Commit: `git add runtime/repokit tests/offline/test_fleet.py tests/runtime/test_fleet_schema.py qualification/fleet.md src/adapters/hermes.js && git commit -m "feat: provision and admit effective native fleet"`.

### Task B2: Snapshot-bound scoped runner

**Files:** Create `runtime/repokit/runner.py`, `tests/offline/test_runner.py`, `tests/runtime/test_runner_denials.py`, `qualification/runner.md`.

**Interfaces:** Consumes approved card scope, immutable candidate digest and selected testPlanId. Produces master `runner.execute(spec:dict)->ExecutionEvidence`, `runner.validate_paths(paths:list[str])->None`, `runner.apply_patch(task_id:str,base_digest:str,patch:str,context:dict)->dict` returning `{candidateDigest,changedPaths}`. Trusted context binds current native run, builder role, host-admitted scope, immutable snapshot and B3 publication fence; the model supplies only task/base/patch. Spec fields: role, candidateDigest, snapshotPath, approvedPaths, testPlanId, argv, deadlineSeconds, outputBytes; only trusted host/card admission supplies snapshot/test-plan/argv. Model cannot choose executable or mounts.

- [ ] Write failing test:

```python
import unittest
from repokit.runner import apply_patch, validate_paths

class RunnerTest(unittest.TestCase):
    def test_protected_paths_and_escapes_refuse(self):
        for path in ['.hermes/kanban.db','../secret','/opt/data/config.yaml',
                     'src/../../.hermes/repokit/manifest.json']:
            with self.assertRaises(ValueError):
                validate_paths([path])

    def test_reviewer_cannot_submit_patch(self):
        with self.assertRaises(PermissionError):
            apply_patch('task-one', 'base', 'bounded patch', {'role':'reviewer'})
```

Add actual sandbox fixtures attempting secret reads, source write in reviewer, native CLI/Python/SQLite transitions, plugin/policy/selector mutation, both mount aliases, symlink escape, `/proc` sibling access, background escape, socket access, network and unbounded output/process/time. Assert source digest unchanged for reviewer and protected state unchanged for builder. Shell allowlist alone must fail the qualification test.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_runner.py'`.
- [ ] Implement path core:

```python
from pathlib import PurePosixPath

def validate_paths(paths):
    for value in paths:
        path = PurePosixPath(value)
        if path.is_absolute() or '..' in path.parts or '.hermes' in path.parts:
            raise ValueError('SCOPE_DENIED')
```

Choose Bubblewrap namespace runner: no broad root/home binds; minimal qualified interpreter/dependency image paths read-only, no secrets/runtime/plugin files, new user/PID/IPC/UTS/network namespaces, private proc/dev, cleared environment, no inherited FDs. Read-only immutable snapshot at `/input`, disposable copy at `/work`; reviewer output only scratch, never promoted. Builder submits a bounded unified patch through `repokit_apply_patch`, not through a test-plan identifier or arbitrary executable. `apply_patch` rejects non-builder context before I/O, caps UTF-8 patch bytes at 1 MiB, checks task/run/base digest and parses only relative ordinary-file edits. Apply without shell/Git hooks to a disposable snapshot, inspect every changed/deleted path, and publish the new immutable candidate under the trusted B3 fence only if the base is still current. Return the new digest; tests and handoffs bind that digest. Positive fixtures submit a real source/test edit then execute an approved test plan; negative fixtures cover scope escapes, oversized/malformed/binary patches, stale base and reviewer submission. Trusted mediator alone promotes approved builder output within source scope. Source writes use that mediator, not unrestricted native file write. New symlinks/special files/hardlinks and `.git` changes reject. Read inspection disables Git hooks, external diff/textconv/pagers.

Use cgroup-v2 delegated controls for aggregate memory/process/CPU limits plus wall/output limits and PID-namespace teardown; no privileged container workaround. The supported-host candidate is rootless Linux with nested unprivileged user namespaces and a prequalified delegated cgroup subtree; A5 must record an actual working combination. If Docker's default syscall filter prevents this, qualify a narrowly reviewed namespace syscall profile as part of the explicit artifact selection, never `seccomp=unconfined`, added SYS_ADMIN capability or privileged mode. Runner binaries/dependencies come from the admitted read-only artifact, not repository-controlled paths. Qualify namespace/cgroup availability inside official Hermes on supported host; set finite ceilings from selected fixture plan, validate >0. If rootless nesting/delegation fails, withhold runner/affected dispatch. This is a scoped engineering support gap, not permission to substitute host-operated reviewer evidence. Document writable mounts, process/network authority and candidate evidence hashing precisely.
- [ ] Green offline command; authorized real denial command `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_runner_denials.py'`. A pass requires actual repository test code attempting denial cases, not simulated syscall outcomes.
- [ ] Commit: `git add runtime/repokit/runner.py tests/offline/test_runner.py tests/runtime/test_runner_denials.py qualification/runner.md && git commit -m "feat: qualify snapshot-bound role execution"`.

### Task B3: Fail-closed native review mediation and run fencing

**Files:** Create `runtime/repokit/review.py`, `tests/offline/test_review.py`, `tests/runtime/test_same_card_review.py`, `qualification/review-policy.md`; modify `runtime/repokit/fleet.py`, `runtime/repokit/native.py`.

**Interfaces:** Consumes ReviewBinding, ExecutionEvidence, native current card/history. Produces `review.valid_completion(binding:dict,evidence:dict)->bool`, master `review.transition(action,binding,evidence,native_call)->dict`. `native_call` is trusted compiled `native.transition`, never caller input. Only wrapper accepts model transitions. `fleet.repokit_transition(action,task_id,payload)` validates an action-specific schema; create/context actions may carry bounded nonsecret card scope/interfaces/acceptance, dependency actions carry card IDs, and completion carries an evidence reference. It derives all actor/run/approval fields from trusted context; null task ID is allowed only for create. The `binding` union and coordination context are defined in the master; creating a card does not require a nonexistent review handoff.

- [ ] Write failing test:

```python
import unittest
from repokit.review import valid_completion

class ReviewTest(unittest.TestCase):
    def test_self_and_stale_completion_reject(self):
        b = dict(builderProfile='builder',builderRun=7,reviewerProfile='builder',
                 reviewerRun=8,candidateDigest='a',handoffFingerprint='h',
                 acceptanceFingerprint='e')
        e = dict(candidateDigest='b',result='pass')
        self.assertFalse(valid_completion(b,e))
        e['candidateDigest'] = 'a'
        b.update(builderProfile='builder', reviewerProfile='reviewer')
        self.assertTrue(valid_completion(b,e))
        for wrong in ['default', 'researcher', 'planner', 'reviewer', None]:
            b['builderProfile'] = wrong
            self.assertFalse(valid_completion(b,e))
        b.update(builderProfile='builder', reviewerRun=7)
        self.assertFalse(valid_completion(b,e))
```

Add builder/default completion, wrong reviewer/current run, review not claimed, missing handoff/evidence, reclassification/reassignment, candidate change and superseded handoff. Successful trace uses one card: handoff → reviewer claim → changes → recorded builder → fresh handoff/reviewer run → done. Assert native event provenance, not derived approval state.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_review.py'`.
- [ ] Inspect `model_tools.py` outer hook catch; `hermes_cli/plugins.py`/`plugins_dispatch.py` failure behavior; `hermes_cli/kanban_db.py` request-review/changes/complete/claim; dispatch worker schema again. Acceptance question: can every supported mutation enter a trusted wrapper while direct auto-appended lifecycle tools are removed? Prove schema and execution veto, never rely on precheck alone.
- [ ] Implement core predicate, then native-history/current-run checks:

```python
def valid_completion(binding, evidence):
    return (binding.get('builderProfile') == 'builder'
            and binding.get('reviewerProfile') == 'reviewer'
            and binding.get('builderRun') != binding.get('reviewerRun')
            and bool(binding.get('handoffFingerprint'))
            and bool(binding.get('acceptanceFingerprint'))
            and evidence.get('result') == 'pass'
            and evidence.get('candidateDigest') == binding.get('candidateDigest'))
```

The model supplies only action/task/evidence reference; the trusted wrapper derives profile, run, ReviewBinding and immutable evidence from current native process/history, never trusting caller-supplied identity or fingerprints. Map the fixed actions `create|assign|depend|context|claim|request-review|request-changes|complete` to qualified native APIs, including `kanban_request_review`, reviewer claim, `kanban_request_changes` and `kanban_complete`. Default can create/coordinate admitted cards and dependencies, never complete a change-bearing card or bypass assigned ownership. Researcher/planner may hand off and complete only their own host-classified non-change cards; builder submits review, never change-card completion; reviewer claims a review handoff and requests changes/completes only that exact independently reviewed candidate. Native dispatcher remains the authority for ordinary claim assignment. Add positive tests for default card creation/dependency wiring and researcher/planner completion, alongside denial of role spoofing, non-change reclassification and change-card self-completion. Context edits cannot rewrite protected admission/acceptance or review provenance; approved changes invalidate dependent evidence. Trusted mediator holds per-board supported-transition lock over native history reload, run check and native call; trusted builder patch publication shares this fence. Candidate is immutable/digest bound. Classification comes from host-approved card contract, not worker prose; record admission/handoff fingerprints in native supported event/context records with qualified protection, not parallel approved/done DB. Every supported transition validates inside wrapper, so outer hook exception cannot convert refusal into a native call; mandatory loaded pre-tool policy is additional defense. Lock contention returns retryable denied, never waits indefinitely. Direct native CLI/SQLite are absent from runner.

Inject policy load failure, thrown hooks, concurrent patch/complete/claim and wrapper errors; unauthorized native call count must be zero. If native tools cannot be excluded or trusted native provenance cannot be fenced at selected revision, mark adapter unsupported and withhold release; native enhancement implementation requires separately authorized repository scope. Do not claim universal DB invariant against administrators or hostile same-UID plugins.
- [ ] Green offline command; later `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_same_card_review.py'` must pass actual native mutations/races.
- [ ] Commit: `git add runtime/repokit tests/offline/test_review.py tests/runtime/test_same_card_review.py qualification/review-policy.md && git commit -m "feat: mediate same-card independent review"`.

### Task B4: Explicit modes, native concurrency and sibling contracts

**Files:** Create `runtime/repokit/modes.py`, `test/modes.test.js`, `tests/offline/test_modes.py`, `tests/runtime/test_native_modes.py`; modify `src/lifecycle.js`, `src/adapters/hermes.js`, `docs/host-lifecycle.md`.

**Interfaces:** Consumes A operationGate and B1–B3 qualifications. Produces `mode_config(mode,max_in_progress,*,release_authorized=False)->dict`, `validate_card_contract(card:dict)->bool`; Adapter.setMode uses these native settings only inside the host transition. `release_authorized` is derived from qualified host evidence, never supplied by the model. The host orchestrator separately invokes provisionProfiles with provision/configure-private qualifications; setMode alone cannot provision profiles. Card contract contains scope, interfaces, evidence, acceptance and dependencies on every sibling.

- [ ] Write failing test:

```python
import unittest
from repokit.modes import mode_config

class ModeTest(unittest.TestCase):
    def test_dispatch_is_explicit(self):
        self.assertFalse(mode_config('safe',1)['dispatch_in_gateway'])
        for mode, limit in [('managed',1),('fleet',3)]:
            self.assertFalse(mode_config(mode,limit)['dispatch_in_gateway'])
            cfg = mode_config(mode,limit,release_authorized=True)
            self.assertTrue(cfg['dispatch_in_gateway'])
            self.assertFalse(cfg['auto_decompose'])
```

Add managed !=1/fleet <=1 rejection, incomplete sibling contract, absent budget/backlog/notification/role/worker readiness, safe active-worker/profile preservation and explicit-none notification policy. Exercise host runCommand with fake native adapters: ordered suspend → provision missing profiles → private configuration/effective-schema/review/privacy qualification → persist release intent → native release → observed admission receipt. Fail after each effect; no release before all gates, no overwritten credentials, no duplicate profiles, unresolved release outcomes never become fictional zero runs. Fresh/partial-managed fleet attempts refuse; prior successful managed admission still requires fresh fleet gates.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_modes.py'` and `node --test test/modes.test.js`.
- [ ] Implement core:

```python
def mode_config(mode, max_in_progress, *, release_authorized=False):
    if (mode not in {'safe','managed','fleet'} or type(max_in_progress) is not int
        or max_in_progress < 1 or (mode == 'managed' and max_in_progress != 1)
        or (mode == 'fleet' and max_in_progress <= 1)):
        raise ValueError('MODE_DENIED')
    return dict(orchestrator_profile='default',
                dispatch_in_gateway=mode != 'safe' and release_authorized is True,
                auto_decompose=False,max_in_progress=max_in_progress)
```

Qualify actual native selectors/concurrency fields and all claim entry points in `hermes_cli/kanban_db_dispatch.py` before adapter writes settings. Host lock/intent/evidence gate release; worker tools cannot change mode. Initialize team.target=default. First managed intent selects engineering without changing artifact pins, suspends every claim entry point, reconciles owned native profiles, provisions missing specialists from protected setup, then validates all readiness and authority before release. Record actor, actual per-role effects, selection and successful TeamAdmission without creating a parallel task state machine. `mode_config` with default arguments always keeps dispatch off; a desired mode name or profile description is not release authority. Crash after native release may leave real active runs: reconcile/suspend further claims, preserve those runs, and require observed evidence before marking admission committed. Repeated managed admission is idempotent; mode safe never removes profiles; fleet refuses bypass of initial managed admission. Test real limit 3/global and builder 1, zero overlapping scopes, distinct worktrees, correct same-card ownership. If native per-profile limit semantics differ, use supported native admission hooks proven under native dispatch lock or mark fleet unsupported; never create a RepoKit scheduler. Safe suspends new claims without cancelling or relabeling existing work. Apply never re-enables mode implicitly. Laya failure is advisory UNKNOWN/WATCH; OpenViking outage is degraded memory, not a global workflow kill-switch. Test continued already-authorized native work without fabricated recall, while privacy-incompatible memory sessions remain refused. Do not enable goal_mode as an outage/review fallback. Golden flows exercise direct builder, research→builder and architecture research→planner→builder; review stays same card.
- [ ] Green both offline commands; later authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_native_modes.py'`.
- [ ] Commit: `git add runtime/repokit/modes.py test/modes.test.js tests/offline/test_modes.py tests/runtime/test_native_modes.py src/lifecycle.js src/adapters/hermes.js docs/host-lifecycle.md && git commit -m "feat: gate native modes and bounded fleet release"`.

**Exit gate:** Native schemas, runner denials, race/failure handling and concurrency need real qualification. Profile text, fixture pass and loaded plugin alone do not authorize autonomous execution.

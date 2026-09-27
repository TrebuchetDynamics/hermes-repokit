> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# Hermes RepoKit Team Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Do not restart the historical plan.

**Goal:** Deliver default-only SAFE installation, deliberate Managed engineering-team admission, actual reviewed self-improvement and final recovery/release qualification.

**Architecture:** Deterministic Node host tooling owns lifecycle; trusted Python plugins inside one Hermes own capability mediation and advisory observation. Native Kanban and native OpenViking retain task and memory authority; internal Laya provides bounded typed inference. Operation-specific qualification gates prevent unsupported paths from masquerading as readiness.

**Tech Stack:** Proposed supported target: Linux local filesystems, Node.js >=22 ESM/built-ins, Python 3.11+ standard-library offline tests, util-linux flock, Docker Compose >=2.30. A5 qualifies exact host/runtime combinations; these floors are engineering selections, not measured upstream guarantees. Bubblewrap is the B2 scoped-runner candidate, not a bootstrap dependency for observation.

**Spec:** [Approved team-runtime design](../specs/2026-09-27-repokit-team-runtime-design.md); [immutable research findings](../../research/2026-09-27-repokit-integration-findings.md).

## Global Constraints

- Exactly one Hermes runtime container and one project-scoped Compose project per canonical repository; replacement may temporarily leave zero Hermes containers, never two.
- `HERMES_HOME=/opt/data`, `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`; built-in memory is `memories/`.
- No Docker socket, remote Docker control credentials/endpoints, host credential directory or privileged container access inside Hermes.
- Fresh install creates native default only and initializes `orchestrator_profile: default`, `dispatch_in_gateway:false`, `auto_decompose:false`. Setup/activate create no specialists. Managed provisions the five-role team under suspended claims and only then releases concurrency 1; fleet requires prior successful managed admission. SAFE downgrade retains all profiles/history; `auto_decompose:false` always.
- Pi is development-only, never a dependency of installation, runtime, generated configuration, user tests or self-hosting. Qualify the complete release journey on a disposable Pi-absent host; do not uninstall developer tooling.
- `goal_mode` is excluded from the default reviewed workflow until independently qualified. Exact trusted builder/reviewer profile and distinct run identity checks are mandatory acceptance logic.
- Nerve never completes, blocks, unblocks, requests changes or dispatches work. No second task database/scheduler or memory provider.
- Pins are explicit immutable desired selections, never research defaults or latest. Private authority, downloads, inference, deployments and external-repository edits are not authorized by this plan.
- Planning snippets below and in slices are proposed future code/tests, not implemented features or evidence of runtime support. All commands in task checklists are future execution instructions.

## Review Focus

1. Path aliases/tracked secrets and relocation must refuse unsafe ownership, not silently adopt (A2 tests).
2. A killed administrator with a surviving mutator must retain exclusion; unknown writer state must stop apply (A3/F2 tests).
3. Auto-appended native tools, outer hook failures and stale candidate/run races must not bypass same-card review (B1/B3 tests).
4. Healthy HTTP with incompatible privacy, wrong keys or unpinned models must not imply readiness (C2/C3/D1/D3 tests).
5. Lost multiprocess hooks, client timeouts and partial replacement must remain bounded and truthful, not recreate work or claim completion (D4/E1/E3/F1 tests).

---

## Shared contracts and file layout

This section is the sole cross-slice naming authority. Types below are JSON records unless stated otherwise; JS uses plain objects, Python uses dictionaries with identical camelCase wire keys. Boundary validators reject unknown fields, non-finite numbers, excessive depth/bytes and raw content. Errors expose stable reason codes, not offending values.

```text
bin/hermes-repokit.js       executable shim, no business logic
src/contracts.js           versioned wire validation; schema mirrors Python
src/cli.js                 eight-command parser/main(argv,deps)->Promise<number>
src/identity.js             canonical repo and ownership checks
src/private-layout.js       filesystem admission/bootstrap
src/locks.js                inherited-descriptor administrative lock
src/manifest.js             strict desired validation
src/compose.js              deterministic JSON Compose rendering
src/observe.js              bounded read-only component aggregation
src/plan.js                 pure plans and operation gates
src/lifecycle.js            capability-separated command orchestration
src/transaction.js          crash-safe publication/reconciliation
src/adapters/hermes.js      compiled-in revision-qualified native boundary
src/admission.js            immutable artifacts/Superpowers scanner admission
src/promotion.js            host immutable CLI publication/direct invocation
src/uninstall.js            G2 owned runtime teardown, retaining all private/source state
components/openviking.js    host config/identity, never alternate provider
components/laya/            bootstrap.py, artifacts.py, admission.py, Dockerfile,
                           requirements.lock (generated with approved artifacts)
runtime/repokit/            __init__.py, plugin.yaml (repokit-policy), contracts.py,
                           native.py, fleet.py, runner.py, review.py, modes.py,
                           memory_contract.py
runtime/repokit_nerve/      __init__.py, plugin.yaml (repokit-nerve), hooks.py,
                           ingress.py, consumer.py,
                           observer.py, store.py, tools.py, rubric.py,
                           laya_client.py, answers.py
contracts/repokit-v1.json   shared JSON schema and bounded enum definitions
qualification/             reviewed source/host/runtime evidence, never secrets
test/                      Node *.test.js; fixtures/ for synthetic data/processes
tests/offline/             Python unittest files
tests/runtime/             opt-in actual pinned runtime qualification
tests/selfhost/            opt-in actual inference/self-apply release gate
```

No runtime `.hermes` is created in the development checkout by default tests. Python tests run from repository root with `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_*.py'`; imports use `repokit` and `repokit_nerve`. Test files use standard-library `unittest`; Node uses `node:test`/`node:assert/strict`. Runtime tests must fail, not skip-to-success, when a requested qualification lacks authority or its artifact is absent.

```text
Identity = {repoRoot:string, repoId:string, projectName:string,
            dockerContext:string}; repoId = full sha256(canonical Git root)
Desired = {schemaVersion:1, identity:Identity,
 services:{hermes:ServiceSelection,openviking:ServiceSelection,laya:ServiceSelection},
 method:{superpowersRevision:string|null}, team:{target:"default"|"engineering"},
 intent:{mode:Mode,maxInProgress:int}}
ServiceSelection = {artifactId:string|null, adapterId:string|null}
Mode = "safe"|"managed"|"fleet"
Selection = {id:string,desiredDigest:string,cliDigest:string,catalogDigest:string}
Qualification = {adapterId:string,artifactId:string,selectionId:string,sourceRevision:string,
 evidenceDigest:string,operations:Operation[],
 level:"source"|"runtime"|"inference",supported:bool}
Gate = {allowed:bool,reasons:string[]}
Operation = "observe"|"provision"|"configure-private"|"start"|"memory-session"|
 "execution"|"download"|"inference"|"replace"|"suspend-claims"|"release-claims"|
 "stop-runtime"|"remove-runtime"
Command = "plan"|"install"|"setup"|"activate"|"status"|"apply"|"mode"|"uninstall"
ComponentObservation = {component:string,verdict:"healthy"|"degraded"|"blocked",
 evidence:"known"|"unknown"|"pending",observedAt:string|null,
 desiredAlignment:"match"|"drift"|"unknown",reasons:string[]}
Observation = {observedAt:string,components:ComponentObservation[],ownership:Gate}
Plan = {actions:{component:string,operation:Operation}[],
 blocked:{component:string,operation:Operation,reasons:string[]}[],changed:bool}
TeamAdmission = {selectionId:string,profilesDigest:string,evidenceDigest:string}
Receipt = {schemaVersion:1,transactionId:string,identity:Identity,selectionId:string,
 command:Command,actorRef:string,intent:Desired.intent,
 phase:"prepared"|"applying"|"committed"|"interrupted",
 effects:string[],teamAdmission:TeamAdmission|null,recoveryRequired:bool}
ReviewBinding = {repoId:string,board:string,taskId:string,builderProfile:string,
 builderRun:int,reviewerProfile:string,reviewerRun:int,candidateDigest:string,
 handoffFingerprint:string,acceptanceFingerprint:string}
ExecutionEvidence = {candidateDigest:string,testPlanId:string,runnerIdentity:string,
 result:"pass"|"fail"|"incomplete",outputFingerprint:string}
Quiescence = {quiescent:bool,unknown:bool,reasons:string[],observedAt:string}
```

`team.target` starts as `default`; the explicit managed transaction selects `engineering`, which survives safe downgrade/uninstall/reinstall. It is desired roster shape, not evidence of actual native profile creation or permission to release. Native observation is authoritative for existence; receipts only journal provisioning and selection-bound successful admission. Fleet needs that prior admission plus fresh readiness. Mode changes publish an explicitly authorized new desired digest while keeping component/plugin pins fixed; no incidental upgrade.

`board` is the canonical `/opt/data/kanban.db` identity, never its workspace alias. Desired contains no executable paths, free-form config/commands or secrets. B–E configuration is compiled versioned policy referenced by admitted adapter/artifact IDs; selections/catalog digest bind exact policy/rubric files. Changes to configurable fields require explicit schema review, not an extension bag. Host-owned catalog evidence is outside worker authority. Mutable private authority receipts carry scopes/expiry and references only; private values use FDs and protected native files, never manifest/logs/argv.

Stable cross-language interfaces:

```text
JS validateManifest(input:unknown)->Desired
JS resolveIdentity(repoPath:string,dockerContext:string)->Promise<Identity>
JS inspectPrivateLayout(identity:Identity)->Promise<Gate>
JS withAdminLock(identity:Identity,fn:(fd:number)->Promise<T>)->Promise<T>
JS operationGate({operation:Operation,component:string,evidence:object})->Gate
JS buildPlan({desired:Desired,observed:Observation,qualifications:Qualification[]})->Plan
JS observe(identity:Identity,{readers,now,timeoutMs,maxBytes})->Promise<Observation>
JS runCommand(request:{command,repoPath,dockerContext,selectionId?,mode?},deps)
   ->Promise<{exitCode:number,report:object}>
JS getHermesAdapter(selection:Selection,qualification:Qualification)->Adapter|null
Adapter.observe(identity,limits)->Promise<ComponentObservation[]>
Adapter.initialize(identity,selection)->Promise<Receipt>
Adapter.configurePrivate(identity,privateIO)->Promise<Gate>
Adapter.provisionProfiles(identity,selection,privateIO)->Promise<Receipt>
Adapter.start(identity,component)->Promise<Receipt>
Adapter.suspendClaims(identity)->Promise<Gate>
Adapter.inspectQuiescence(identity)->Promise<Quiescence>
Adapter.reconcile(identity,selection,intent)->Promise<Receipt>
Adapter.setMode(identity,mode,limits)->Promise<Gate>
Adapter.stopOwnedRuntime(identity)->Promise<Receipt>
Adapter.removeOwnedRuntime(identity)->Promise<Receipt>
JS uninstallOwnedRuntime({identity,selection,adapter,io})->Promise<Receipt>
Python native.observe_board(path:str,after:int,limit:int)->dict
  # {events:list[dict],cards:list[dict],checkpoint:int,coverage:str}; read-only
Python native.transition(action:str,binding:dict,evidence:dict)->dict
  # trusted compiled native adapter; supported tool calls only through review.transition
Python review.transition(action:str,binding:dict,evidence:dict,native_call:callable)->dict
Python runner.execute(spec:dict)->dict # ExecutionEvidence; no arbitrary shell string
Python runner.apply_patch(task_id:str,base_digest:str,patch:str,context:dict)->dict
  # builder-only trusted publication -> {candidateDigest,changedPaths}
Python fleet.repokit_transition(action:str,task_id:str|None,payload:dict)->dict
  # null task ID only for create; action-specific fields, trusted actor identity
Python modes.mode_config(mode:str,max_in_progress:int,*,release_authorized:bool=False)->dict
Python memory_contract.privacy_gate(evidence:dict)->dict # Gate
Python repokit.register(ctx)->None # trusted policy plugin entry, B1
Python repokit_nerve.register(ctx)->None # advisory plugin entry, E1
```

Qualification `operations` scopes evidence to named operations, not an entire artifact by implication; observation cannot authorize mutation. Keep source/runtime/inference records separate rather than replacing runtime evidence with an inference label. Transition `binding` is an action-specific trusted context: `{repoId,board,actorProfile,actorRun,taskId}` for coordination/non-change actions, with `taskId:null` only when creating a card; change completion additionally requires the full ReviewBinding. Worker payloads cannot supply or overwrite those fields.

Nerve/Laya DTOs (E owns evidence; D owns typed conversion):

```text
AssessmentInputV1 = {schema:"repokit.nerve.assessment.v1",repoId,board,taskId,
 runId:int|null,observationId,observedAt,evidenceFingerprint,candidateDigest:string|null,
 rubricVersion,checkpointSelectionId,coverage:"complete"|"partial"|"unknown",
 elapsedMs:int|null,heartbeatAgeMs:int|null,iterations:int|null,
 remainingBudget:{value:number,unit:string}|null,reviewCycles:int|null,
 failureFingerprints:string[],reasonCodes:string[]}
Binding = {repoId,board,taskId,runId,observationId,evidenceFingerprint,
           candidateDigest,checkpointSelectionId,rubricVersion}
AssessmentResultV1 = {status:"KNOWN",binding:Binding,recommendation,
 evidenceScore:number,escalationProbability:number,retryClass,
 confidenceByQuestion:object,answerConfidenceByQuestion:object,reasons:string[],
 assessedAt:string,artifactProvenance:object}
 | {status:"UNKNOWN",binding:Binding,reasonCode:string,observedAt:string,
    lastKnownStale?:object}
Python rubric.make_request(observation:dict,model_alias:str)->dict
Python answers.validate_response(response:dict,request:dict,model_display:str)->dict
Python laya_client.assess(observation:dict,context:dict)->dict # AssessmentResultV1
```

String fields without annotations are bounded strings. Recommendations: `CONTINUE|WATCH|REPLAN|BLOCK`; retry classes: `retry-same|retry-after-delay|replan|human-input`. Both confidence maps use exactly progress/evidence/escalation/retry with finite [0,1] numbers. Context binds trusted deployed artifact provenance, alias/display mapping, clock, transport and current evidence. Laya does not echo or attest native task identity; correlate locally, recheck freshness, never invent server attestation.

## Lifecycle and release invariants

`install` → SAFE/default-only + native board + admitted non-autonomous components → private `setup` (specialist material staged, no specialist homes) → `activate` (no roster expansion) → `mode managed` (suspend, provision, verify, release) → optional separately authorized fleet. Managed provisioning is idempotent and preserves credentials/partial effects. `mode_config` represents a target, never permission to write dispatch=true during preparation. Journal release intent before the native effect; a crash after that effect may have created real runs, so recover observed state, suspend further claims and preserve provenance—never report fictitious zero work.

Hermes/OpenViking image digests, Laya package/checkpoint/artifact hashes, Superpowers commit and both RepoKit plugin tree hashes change only through explicit desired selection. Outage is distinct from missing authorization: Laya yields UNKNOWN/WATCH, OpenViking degraded memory; already-authorized native work may continue, but native privacy incompatibility remains a session gate. Nerve never mutates cards; OpenViking never stores authoritative task state.

`uninstall` is the eighth command: after verified ownership, claim suspension and writer drain, remove owned runtime containers and the exclusively owned project network only. Preserve source, all `.hermes` payloads/private permissions, desired/plugin pins and usable host recovery CLI; append truthful lifecycle evidence only. No purge, image/volume deletion, credential revocation or automatic reactivation. Fresh install and reinstall are distinct: reinstall reconciles retained roster/state without reducing it to default, and leaves claims suspended.

## Sequence, dependencies and readiness

| Slice | Tasks / dependency | Independently useful deliverable; remaining gate |
| --- | --- | --- |
| [A lifecycle](2026-09-27-repokit-a-lifecycle.md) | A1–A6 sequential | Offline host foundation; actual adapter qualification before production mutation |
| [B fleet](2026-09-27-repokit-b-fleet.md) | B1 after A5; B2→B3→B4 | Inspection/profile policy; runner/review/native concurrency qualification before dispatch |
| [C OpenViking](2026-09-27-repokit-c-openviking.md) | C1 after A4; C2→C3→C4; C3 needs B2 and B4 prepared roster | Eligible service provisioning; compatible native upload behavior before sessions |
| [D Laya](2026-09-27-repokit-d-laya.md) | D1 after A4; D2 after B1 shared validators; D3 after D2 + D1 contract; D4 after D3 | Typed offline client; packaging/download/runtime/calibration gates remain distinct |
| [E Nerve](2026-09-27-repokit-e-nerve.md) | E1 after A5/B1; E2→E3→E4; B2 for E3 namespace reader; D3 for optional inference | Observation works with unavailable Laya; never lifecycle enforcement |
| [F self-host](2026-09-27-repokit-f-selfhost.md) | F1 after A6; F2→F3→F4; real tiers need A–E | Actual bounded team task + preserved host self-apply on a Pi-free host; G still required |
| [G release/recovery](2026-09-27-repokit-g-release-recovery.md) | G1 after A6/F2; G2 after G1/F4; G3 after A–F/G1–G2 evidence | Recovery matrix, compatible rollback, retained-state uninstall/reinstall and final Pi-absent release |

Retain **subagent-driven execution** after user plan review and execution authorization: fresh task implementer and reviewer, gate each task, then whole-branch review. This planning session launches no implementation workers. Each task's source inspection is allowed only within its later execution authority; an unsupported selected revision produces a documented blocker and working denial path, not a fabricated upstream API. Offline success cannot admit production artifacts. Where a task contains a later authorized qualification/packaging substep, review its independently verified contract portion and record the remaining substep blocked; never check off the whole runtime deliverable or promote fixture-only support. Dependency edges refer to reviewed consumed interfaces for offline work; they do not require unrelated live substeps to complete. In particular, D1 artifact acquisition cannot hold up D2 offline rubric work, and E1–E3 observation needs no loaded Laya model. Runtime execution still requires every dependent operation's actual qualification.

## Coverage and inline self-review

| Spec requirements | Owning tasks |
| --- | --- |
| §§1–2 operation gates, identity, layout, authority, state safety | A1–A6, C1, D1, F1–F2 |
| §3 five roles, native schemas, scoped autonomous tests, bounded trust | B1–B2, C3 |
| §4 DAG sibling context, native same-card review, fencing/fail closed | B3–B4, E3, F3 |
| §5 eight CLI surfaces, staged setup, default-only SAFE → managed team, retained uninstall | A1/A5/A6, B1/B4, F1, G2 |
| §6 native identity, opt-out qualification, curated admission, outage | C1–C4; F3–F4 actual shared recall |
| §7 eight hooks, multiprocess bounds, tools, missing-event reconcile | E1–E4 |
| §8 package/source drift, pinned artifacts, rubric and server/client bounds | D1–D4 |
| §9 independent freshness/degradation, unknowns, redaction | A5, C4, D3–D4, E4, F1 |
| §10 immutable apply, scanner exact replacement policy, recovery | A6, F1–F2/F4 |
| §11 clean Pi-free bootstrap, real same-card dogfood, immutable apply/preservation | F1–F4, G3 |
| §§12–13 seven slices, recovery/uninstall, qualification versus authority | G1–G3 and entire sequence; master readiness table |

Self-review performed against all thirteen spec sections: five review-focus families assigned tests; no duplicated board or provider; shared camelCase DTOs and named Python functions frozen here; safe/preparation versus managed/fleet dispatch corrected; bootstrap uses no second host lock; no live commands executed in planning. Engineering gaps (native opt-out support, runner isolation, review failure/races, exact artifacts) remain explicit acceptance dependencies, not deferred unspecified implementations. Owner gates concern actual scopes/pins/spend, not mechanism selection. Review must reject release claims based on fixture-only adapters.

**Next gate:** review the requested pre-code lifecycle/recovery refinements, preserving the accepted-in-principle plan and already-selected execution method. The plan now contains 29 tasks across A–G; actual runtime/download/inference authority remains separate. No historical implementation task is resumed.

# Hermes RepoKit team runtime

## Current stage

User-approved architectural direction → **new written spec awaiting review** → replacement implementation plan → separately authorized execution. No implementation code, tests or implementation commits have happened. Existing Git history records the historical documents only; do not resume old Task 1.

- [x] Read the redesign and self-hosting requirements.
- [x] Read pinned Hermes/OpenViking/Laya source verification.
- [x] Incorporate self-hosting and privacy/capability reviews, including worker lifecycle-tool auto-append and non-atomic policy limits.
- [x] Write the [new team-runtime spec](docs/superpowers/specs/2026-09-27-repokit-team-runtime-design.md) and concise [source findings](docs/research/2026-09-27-repokit-integration-findings.md).
- [x] Preserve the [historical spec](docs/superpowers/specs/2026-09-26-hermes-repo-installer-design.md) and [historical plan](docs/superpowers/plans/2026-09-26-hermes-repo-installer.md), with superseded headers.
- [x] Address the three independent-review findings: bounded v1 threat model, decision ownership and operation-specific gates.
- [ ] Obtain review approval of the revised written spec; recommendations are not implemented support.
- [ ] Write/review a replacement plan across the six bounded slices, including engineering feasibility/qualification; no mandatory owner questionnaire on mechanisms. Select execution method separately.
- [ ] Implement only after those gates, then qualify authorized offline/runtime/real-inference levels separately.

## Approved direction to preserve

- Product/CLI: **Hermes RepoKit / `hermes-repokit`**. Exactly one Hermes runtime container and one Compose project per canonical repository; internal OpenViking/Laya sidecars allowed, Nerve initially a plugin. No bundled VikingBot or second orchestrator.
- Native profiles own WHO: default/researcher/planner/builder/reviewer with descriptions and effective enforced capabilities, not SOUL-only restrictions. Native Kanban owns WHAT/lifecycle/audit. Nerve observes/advises; Laya supplies typed decisions, never profile routing. Native OpenViking owns project knowledge; Superpowers owns methodology.
- Official `/workspace` + persistent `.hermes` → `/opt/data`; `HERMES_HOME=/opt/data`, `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`. Native built-in path is `memories/`. Preserve sessions, credentials, board/history, plugins, memory, sidecar data/cache and Nerve history. No custom Hermes image just to force HOME.
- Keep canonical identity, private ignored/unindexed state, symlink/ownership protections, held locks, truthful no-op and interruption recovery. No Docker socket or host lifecycle authority inside Hermes.
- Commands: plan/install/private setup/activate/status/apply plus `mode safe|managed|fleet`. Safe initially; managed explicitly authorized with concurrency 1; fleet separately authorized/bounded. `auto_decompose:false`; no ceremonial DAG cards. Same change card moves builder → independent reviewer → done, or back for changes.
- Sidecars have no published ports. Remote-provider egress requires a separate explicit policy; private networking is not proof remote inference works.
- Native `obra/superpowers` at approved full SHA with safe/caution/dangerous admission, no generic force bypass. All component deployment pins remain unselected/unqualified; research SHAs are evidence only.
- Deterministic host bootstrap without a pre-existing agent stack; real reviewed bounded self-hosting card and host-operated quiescent self-apply/persistence are release requirements, not current permission.

## Owner authority, engineering work and scoped gates

**Owner decisions:** actual privacy-contract changes or additional native-enhancement scope; private provider/data/retention/egress/budget/download/compute authority; any proposed reduction in requested autonomy or protected-state guarantees. Exact deployment selections and scanner caution approvals remain explicit. These are not requests for the owner to choose a runner or solve qualification.

**Engineering obligations:** select and demonstrate supported role/execution tools, loaded native completion policy and failure/run/input fencing, protected-path permissions across both mount aliases, pin-aware Laya packaging, candidate artifacts, supported hosts, resource bounds and denial/recovery fixtures. V1 trusts the selected runtime/admitted plugin implementations and host operator, not model output or repository test code. It enforces the ordinary model-tool contract and concrete supported execution guarantees, not comprehensive hostile same-UID/plugin resistance. Runner choice is engineering within the approved topology; no mandated new sandbox architecture.

- **Memory-session gate:** native OpenViking background sync uploads raw turns and non-recall tool data. Curated ingestion, hiding memory tools and disabling server extraction do not stop upload. Block incompatible memory-connected sessions/private processing until a compatible native solution is qualified; preserve the privacy requirement, with owner approval for additional enhancement scope. No second provider, invented toggle or presumed broad consent. Eligible non-inference provisioning/observation can proceed.
- **Affected-dispatch gate:** reviewer tests and builder execution must meet documented source/runtime/credential access limits. No unrestricted reviewer shell/read-only claim, builder self-approval or exposed secret store labeled protected. Worker lifecycle tools auto-append and pre-tool policy has failure/race caveats; qualify supported paths with negative tests. If guarantees cannot be met, withhold the affected execution/dispatch, not unrelated infrastructure work. Nerve never owns transitions.
- **Download/inference gate:** missing provider/data/compute authority blocks the corresponding processing/acquisition; status and health do not infer permission. Nerve can observe without Laya calls.
- **Component qualification gate:** unqualified artifacts block their deployment/use, not independent engineering/observation. Laya release/source pinning and OpenViking root-key exposure remain explicit gaps.

**Release acceptance:** full stack qualification and the real reviewed self-hosting task, host-operated quiescent self-apply and state preservation remain mandatory. Partial component availability, offline passes or non-inference health do not satisfy that gate.

## Future slices and evidence

1. Host lifecycle/identity/manifest/locks and read-only status.
2. Native fleet, modes and independent review policy.
3. Native OpenViking sidecar/auth/consent/degradation.
4. Reproducible typed Laya sidecar.
5. Bounded nonblocking multiprocess Nerve pipeline and read-only reconciliation.
6. Preservation, interruptions and actual self-hosting release qualification.

These are design boundaries, not executable tasks. Offline stubs do not prove live runtime behavior; runtime health does not prove real inference or shared memory. No code/build/install/Docker/model execution, credential access, sibling-repository changes or commits are authorized by this checklist. No automatic whole-source ingestion or v1 `memory index` command.

## Historical Task 1

**Paused/superseded, not completed.** Its default-only/Holographic/six-command schema is obsolete. No old Task 1 source/tests were created, run, staged or committed. Preserve any future discovered partial artifacts and report them rather than restarting the old plan. The old documents remain intact below their new headers for provenance.

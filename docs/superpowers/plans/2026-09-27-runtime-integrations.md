# Ordered runtime integration implementation

User directive: Finish RepoKit Runtime Integrations (2026-09-27).

Preserve the six permanent roles and native authority boundaries. Runtime must
survive removing RepoKit. No automatic hosted supervision, credential collection,
or self-dogfood before disposable acceptance. Work in feat/runtime-integrations.

## Phase 1: Kanban

- [x] Bootstrap: run native init only when absent; native list and diagnostics
  must succeed even for an existing board. Test corrupt/existing boards and
  failure propagation without replacement.
- [x] Extend credential-free native regression to researcher -> planner ->
  executor, structured handoffs, change-request delivery and distinct review
  runs. This is structural evidence, not model-driven acceptance.
- [x] Provide a disposable live acceptance entrypoint using privately configured
  native model/provider state. Default creates the graph; manual native dispatch
  precedes gateway dispatch. Enforce concurrency one and no auto-decomposition.
- [ ] Observe real workers, same-card changes and approval, then recreation.
  Do not qualify review from synthetic lifecycle calls.

## Phase 2: OpenViking (live acceptance follows Kanban acceptance)

- [x] Implement pinned image/config/storage contracts, native init/doctor/validation,
  restart and health readiness; generate internal-only persistent Compose service.
  Pending-config/recreation evidence passed; live configured model services remain open.
- [x] Implement native memory activation/linking across profiles after doctor/health;
  shared account/user, no peer, built-in memory and automatic extraction retained.
  Native linking fixture passed; private live configuration is still required.
- [ ] Prove cross-role recall, restarts/recreation, second-repository isolation.

## Phase 3: Nerve/Laya (integrated work acceptance follows memory acceptance)

- [ ] Reconcile public promoted release with tested commit; rerun upstream suite.
- [x] Produce reproducible sidecar image from pinned upstream dependencies and
  checkpoint; supported service/client URL, cache persistence, no host ports.
- [x] Native install disabled -> configure local -> enable for all six profiles;
  fresh hooks and actual LOCAL_ONLY decisions, coordinated Compose recreation
  and repeated native checks. Default installation embeds the pinned build inputs.
- [x] Qualify six-role local decisions, offline outage/recovery and installer
  removal/recreation in a disposable stack; generated default build and cache
  persistence separately passed.
- [ ] Observe task supervision, correction and offline failure/recovery without
  hosted fallback; preserve canonical Kanban run identity.

## Phase 4: removal-first acceptance

- [x] Remove copied installer executable/source in foundation and local-supervision
  fixtures; native operations survive.
- [ ] Complete removal before genuine integrated main-model work with live memory.
- [ ] Talk only to default; shared memory affects plan; verify review actors,
  supervision, persistence, isolation and standalone launcher.
- [x] Independent read-only configuration/runtime probes and authenticated memory
  health; no plugin loading, dispatch, extraction or inference. Review unqualified.
- [x] Update qualification matrix and operator docs with actual bounded evidence.
- [ ] Complete final validation/review and remaining live gates. Keep status
  FOUNDATION READY / INTEGRATION PARTIAL until integrated acceptance passes.

## Constraints and decisions

Native model, embedding and extraction access is missing from the supplied
configuration. Private native setup is still required; do not borrow host
credentials or substitute canned model responses for acceptance. The authorized
production wiring and independent credential-free/local-inference checks proceed
while live main-model/memory acceptance remains blocked. Do not push or touch the
active root repository's runtime. Work remains on the focused implementation
branch. The user supplied the architectural design and ordered implementation
directive; no additional design approval is necessary.

Baseline: go test ./... passed in isolated worktree before edits.

## Progress ledger

Phase 1 bootstrap RED: both existing-board native-check and corrupt-board failure
tests failed against the old implementation. GREEN after native list/diagnostics.
Docker structural graph passed in 124.04 seconds including planner handoff,
request-change delivery and restart/recreation. All Go tests passed.

Live manual runner added; native provider setup is pending. Read-only review
found copied-launcher routing, startup-sensitive dispatch, expiring worker PIDs
and weak convention provenance issues. Addressed with native plan + mount
preflight, restart before cards, historical observations and independent nonce.
Evidence-validator regression tests cover rejected synthetic/self-approved runs,
missing metadata/dependencies and retained observations after native PID cleanup.
Live main-model dispatch and memory acceptance remain unclaimed. The active
project runtime remains untouched.

Production default installation now includes Hermes/OpenViking/Laya with durable
state and embedded pinned build inputs. Plain setup sequences default/team,
private memory configuration and native local supervision; per-integration flags
resume setup. The foundation plus native-memory-link fixture passed in about
163 seconds and OpenViking pending-mode fixture in about 11 seconds. The native
six-role supervision fixture passed in about 221 seconds with LOCAL_ONLY
inference, offline outage, installer removal and coordinated recreation. That
run used the native helper; the newer public CLI path passed in 535.00 seconds. Read-only probes report memory identity, Nerve configuration and Laya
health separately, keeping review unqualified. See the
[current evidence record](../../qualification/runtime-integrations.md).

Remaining: private main-model/embedding/VLM configuration, real manual/gateway
work, same-card independent review, cross-profile memory recall/persistence,
cross-repository isolation, future specialist qualification and the complete
combined removal-first gate. No v1 completion or self-dogfood claim.

Final reviewed-code receipt: public first-time `setup --supervision`, read-only
component reporting, six-role offline LOCAL_ONLY decisions, local outage,
installer removal and Compose recreation passed in 535.00 seconds. Go tests,
race checks, vet and pinned-container read-only provenance regressions passed.
Live model-driven work and real memory acceptance still require native private
provider configuration.

# Hermes RepoKit implementation progress

The [final bootstrap design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md) and [phase plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md) are authoritative. Earlier twelve-task and A–G plans are superseded. The final user directive authorizes complete implementation and phase commits.

Existing evidence: Go CLI skeleton and qualification evaluator, offline tests, product README. No release/runtime/model/memory qualification yet. No custom Nerve/Laya/provider code will be added. Native OpenViking automatic synchronization/extraction is accepted; durable-only privacy is not a product requirement. Native same-card review first; policy code only after demonstrated enforcement gap.

- [x] Phase 0: Correct authoritative documents.
- [ ] Phase 1: Go CLI and qualification model.
- [ ] Phase 2: Target identity and collision handling.
- [ ] Phase 3: Compose renderer.
- [ ] Phase 4: Atomic installer and locking.
- [ ] Phase 5: Standalone launcher.
- [ ] Phase 6: Setup delegation.
- [ ] Phase 7: Native defaults, engineering profiles and Superpowers.
- [ ] Phase 8: Upstream Nerve and supported Laya.
- [ ] Phase 9: Official OpenViking and native provider.
- [ ] Phase 10: Same-card review qualification.
- [ ] Phase 11: Read-only plan and verify.
- [ ] Phase 12: README and quickstart.
- [ ] Phase 13: Removal-first acceptance.
- [ ] Phase 14: RepoKit dogfood.
- [ ] Phase 15: Release binaries.

## Evidence ledger

- Phase 0: correct spec/plan/TODO before product code. Preserve source research as historical observations, not active requirements.
- Live gates and released component pins remain pending. Do not claim model/memory operation from mocks or cross-compiled binaries.

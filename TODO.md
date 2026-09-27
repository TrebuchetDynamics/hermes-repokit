# Repository-local Hermes lifecycle toolkit

## Goal

Build a standalone six-command toolkit here to install and maintain one Docker Hermes container per repository, with native default-profile Kanban and obra/superpowers. This project builds the toolkit; it does not deploy a production repository or install host Docker.

## Approved requirements

- **Exactly one Hermes runtime container per repository**, including setup, integrations and replacement; never one container per profile.
- Native **default** profile initially, with persistent usable Kanban and no invented specialists.
- Host repo → `/workspace`; host ignored private `<repo>/.hermes` → `/opt/data`.
- `HERMES_HOME=/opt/data`, `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`; accept official Unix/tool HOME behavior. File-tool safe roots are not an OS sandbox.
- No derived image merely to force HOME to `/workspace/.hermes`; qualify official layout first. No image build during planning.
- Explicit `kanban.dispatch_in_gateway: false` and `kanban.auto_decompose: false`; initial empty worker allowlist and no automatic wakes/work.
- Native Superpowers integration, full immutable approved SHA, scanner safe/caution/dangerous handling with no silent force bypass.
- Six commands: `plan`, `install`, private `setup`, `activate`, `status`, declarative `apply`.
- Image/plugin upgrades only through explicitly selected immutable desired state, not because latest moved.
- Deterministic labels plus receipts/mount checks, held locks and interruption-safe recovery; preserve credentials and user state.

## Planning progress

- [x] Inspect destination and reference installer guidance.
- [x] Confirm topology, default-profile Kanban and Superpowers source with user.
- [x] Receive approval of standalone toolkit direction.
- [x] Write initial design specification.
- [x] Receive user review approving planning with explicit layout/manifest/security/ownership amendments.
- [x] Verify official Docker layout and full-SHA plugin support in upstream docs; inspect wrapper and scanner/replacement source at recorded revision.
- [x] Revise [approved design spec](docs/superpowers/specs/2026-09-26-hermes-repo-installer-design.md), superseding conflicting sibling skill policies.
- [x] Write and self-review [implementation plan](docs/superpowers/plans/2026-09-26-hermes-repo-installer.md).
- [ ] User reviews plan and selects subagent-driven or native execution.
- [ ] Establish Git repository/isolation during authorized execution; documents currently uncommitted.
- [ ] Implement the plan with tests; no implementation code exists yet.
- [ ] Qualify actual official runtime in explicitly authorized disposable fixtures; no production image/plugin pin qualified yet.

## Evidence and limits

- Source snapshot inspected: Hermes `28e6496a5e3adfea57bebfc9571b981bff378523`; this is not an image qualification.
- Superpowers native integration and fresh-session requirement are documented upstream; post-compaction bootstrap restoration is not promised.
- Manual board use is available with default alone; manual worker dispatch requires an explicitly provisioned/authorized specialist. Test that only in an isolated fixture.
- Holographic memory remains the reference-derived integration target; verify its actual provider/dependencies separately using `/opt/data`.
- No Docker pull/build/recreation, plugin install, credential setup, model call or production deployment performed during planning.
- Source excerpts downloaded for review are temporary files under `/tmp/hermes-installer-design-review.SG5MHL`; upstream code was read, not executed.
- Sibling `pi-toolset` files were not edited. Its current unrelated `.gitignore` modification is left untouched.

# Hermes RepoKit progress

The maintained scope is one Hermes development container, six native profiles,
shared Kanban, embedded OpenViking and ordinary Compose/native lifecycle.
**Full v1 acceptance remains incomplete.**

- [x] Go bootstrapper and standalone native launcher.
- [x] Repository ownership checks, private-state protection and conservative reruns.
- [x] Six-role identities, native cloning and owner-drift preservation.
- [x] Shared Kanban and source implementation of gated automatic dispatch/review.
- [x] Embedded OpenViking configuration/linking and read-only health inspection.
- [x] Generated development image and optional isolated Docker acceptance daemon.
- [ ] Build and qualify the current single-container image in a Docker-enabled environment.
- [x] Run required Unix-socket tests outside the restricted sandbox.
- [ ] Prove live researcher canary and originating-channel task/result delivery.
- [ ] Prove genuine same-card executor/reviewer correction and approval.
- [ ] Prove cross-profile memory write/recall, restart persistence and repository isolation.
- [ ] Complete integrated removal-first acceptance and bounded self-dogfood.
- [ ] Resolve optional Superpowers candidate scanner admission if selected.
- [ ] Qualify release toolchain/artifacts and supported architectures.

Native profiles, board, credentials and memory remain authoritative after RepoKit
is removed. Installation does not start services. Setup owns private configuration
and activation; verification is observational. Owner-selected optional plugins
remain native Hermes concerns.

Historical credential-free fixture passes are recorded in [runtime observations](docs/qualification/runtime-observations.md).
They do not qualify the current image or live model work. Git delivery validation
passed the full offline and race suites, including Unix-socket operations;
live Docker and model acceptance were not run during delivery.
See [implementation status](docs/implementation-progress.md), [design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md)
and [plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md).

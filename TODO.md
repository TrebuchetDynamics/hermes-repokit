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
- [x] Build and qualify the current single-container image in a Docker-enabled environment (embedded OpenViking pending fixture plus development-runtime, foundation, Kanban and maintenance Docker fixtures pass).
- [x] Run required Unix-socket tests outside the restricted sandbox.
- [x] Prove live gateway-spawned researcher canary and automatic execution of the original queued research task.
- [x] Upgrade the live dogfood deployment's HEAD-era recipe with `install` + Compose recreate; profiles, board, dispatch policy and launcher preserved; `verify` CORE_READY healthy.
- [x] Prove the gateway and dispatcher return after Compose recreate and `docker restart` (native main-wrapper re-registers s6 gateway services; dispatch check passed after restart).
- [x] Run `verify --dispatch-check` with a real provider (live dogfood, after restart). A fresh-deployment run remains open.
- [ ] OpenViking shutdown: the upstream entrypoint's TERM wait race is no longer patched. Not reproducible while memory is unconfigured (pending server only); re-check after `setup --memory`.
- [x] Prove originating-channel task/result delivery: Telegram → default → automatic executor → automatic same-card reviewer approval → result in the same chat (live dogfood, `t_6641c2b0`), surviving `docker restart`.
- [ ] Prove a same-card request-changes correction cycle (approval is proven; a reviewer rejection → executor revision was not exercised).
- [x] Prove a fresh, unrelated-repository bootstrap through the same Telegram loop: s3upload PASS ([record](docs/qualification/fresh-repo-s3upload-2026-09-29.md)). The earlier PMB clone trial was BLOCKED by an upstream Hermes worker import collision ([record](docs/qualification/fresh-repo-pmb-2026-09-29.md)); `verify` now detects it.
- [ ] Fresh-install follow-ups from s3upload: bounded readiness wait for `install` right after Compose start; preferred free/self-hosted tool defaults (owner request; `ddgs` cannot install in the sealed Hermes environment).
- [ ] Track upstream Hermes [#126127](https://github.com/NousResearch/hermes-agent/issues/126127) / [#126277](https://github.com/NousResearch/hermes-agent/pull/126277) (worker `python -m` import shadowing; RepoKit evidence commented). Requalify the pinned image once fixed and retire the `python-imports` probe if workers no longer see the workspace ([record](docs/qualification/python-import-collision.md)).
- [x] Accept a fresh clone's group-writable (775) root when the group is the owner's private group; refusals now name the path and the `chmod` fix.
- [ ] Fresh-install follow-ups from the PMB trial: collision remediation text, `.hermes-repokit.lock` ignore, printing the one-time `gateway start` command, env-only channel observation.
- [ ] Qualify coordinator behavior for vague requests (for example "Improve readme"), not only bounded tasks.
- [ ] Prove cross-profile memory write/recall, restart persistence and repository isolation.
- [ ] Complete integrated removal-first acceptance and bounded self-dogfood.
- [ ] Resolve optional Superpowers candidate scanner admission if selected.
- [ ] Qualify release toolchain/artifacts and supported architectures.

Native profiles, board, credentials and memory remain authoritative after RepoKit
is removed. Installation does not start services. Setup owns private configuration
and activation; verification is observational. Owner-selected optional plugins
remain native Hermes concerns.

Historical credential-free fixture passes are recorded in [runtime observations](docs/qualification/runtime-observations.md).
They do not qualify live model work. Git delivery validation passed the full
offline and race suites, including Unix-socket operations, and a Docker-capable
session passed the credential-free Docker fixtures (embedded OpenViking pending,
development runtime, foundation, Kanban channels and maintenance). Live researcher
execution also passed; memory, independent review and Telegram round-trip
acceptance remain open.
See [implementation status](docs/implementation-progress.md), [design](docs/superpowers/specs/2026-09-27-repokit-bootstrap-design.md)
and [plan](docs/superpowers/plans/2026-09-27-repokit-bootstrap.md).

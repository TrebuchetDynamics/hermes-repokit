# Development runtime implementation plan

Goal: make the same native Hermes team capable of repository development from
CLI and messaging, with optional separate Docker acceptance.
Spec: ../specs/2026-09-28-development-runtime-design.md

- [x] Detect a bounded supported manifest matrix and generate immutable build
      inputs. Test absent, malformed, redirected and unsupported manifests.
- [x] Render the derived Hermes image by default; exact recipe and image identity
      must be checked before native mutations. Preserve existing native state.
- [x] Publish build artifacts under the existing installation lock; test fresh,
      idempotent, exact legacy upgrades and owner drift refusal.
- [x] Add an explicit Docker-test selection/profile with a pinned separate daemon,
      shared scratch paths, no host socket/private mounts and a bounded helper.
- [x] Observe actual runtime tool versions/cwd/mount access and report unsupported
      compiler requirements separately from Docker-test availability.
- [x] Expand executor native coding tools; reconcile only exact older managed
      selections, preserving owner choices and independent review boundaries.
- [ ] Run focused fixtures, full normal/race/vet/format checks and opt-in Docker
      acceptance where permitted; record environmental blockers without skips
      disguised as qualification. Preserve previous source changes, no commit.

Source implementation and offline regression coverage are complete. The final
qualification item remains open: six AF_UNIX cases and real Docker builds are
blocked by this execution sandbox. No live deployment was changed. See
[validation evidence](../../qualification/development-runtime.md).

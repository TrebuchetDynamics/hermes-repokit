# Session audit repair implementation plan

**Goal:** Fix the approved runtime-discovery and orchestration defects without changing live state or credential boundaries.

**Architecture:** Keep one runtime container and native Hermes task lifecycle. Expose existing pinned executables in the login-shell search path. Keep passive verification observational; prove actual terminal behavior in a disposable native-worker fixture. Add explicit runtime/task guidance to managed identities with exact historical migration.

**Constraints:** No provider calls, secret setup, live deployment mutation, commit or push. Preserve custom profiles and generated-file drift. Do not grant workers host Docker access or broaden planner tools.

- [x] Reproduce bare Hermes/Go lookup failure through native `LocalEnvironment` in a disposable container. Test both first and subsequent shell snapshots and compile a tiny project offline.
- [x] Add immutable image command links; verify stable entrypoint targets observationally, separately from worker execution evidence.
- [x] Recognize exact previous recipe/Compose and preserve unrelated state during source-driven upgrade, including interrupted retries and edited-file rejection.
- [x] Add runtime-aware capability preflight, blocked-versus-completed semantics, credential-discovery boundaries and native same-card review to generated identities. Retain exact old SOUL recognition.
- [x] Run affected tests, full normal/race suites, vet, formatting and disposable Docker worker acceptance. Review final diff and record evidence.

Review focus: login shell resets PATH; snapshot reuse; non-Go repositories; old generated recipes and interrupted upgrades; owner-edited SOUL/recipe rejection; absent optional memory must not disable core work. Prompt guidance is advisory: no claim of runtime enforcement or live model behavioral acceptance from static tests.

Validation: original image failed the native terminal lookup fixture; repaired
image passed (17.60 seconds). Full normal/race suites, Docker-tagged vet,
formatting and diff checks passed. Final migration test additions passed focused
development/compose/install/CLI tests. Independent source review found no
actionable issues. No commit, push, private setup or live rollout performed.

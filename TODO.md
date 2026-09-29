# Hermes RepoKit TODO

RepoKit configures a repository-specific Hermes team through public Hermes
interfaces, proves what it can prove, then gets out of the way.

**Released:** [v0.1.0](https://github.com/TrebuchetDynamics/hermes-repokit/releases/tag/v0.1.0)
(one Hermes container per repository, six profiles, automatic dispatch with
same-card review, evidence-based `verify`). The core loop is proven; memory is
optional and not fully qualified.

## Proven in v0.1.0

- Live dogfood upgrade in place: profiles, Kanban, dispatch policy and launcher preserved ([record](docs/qualification/live-dogfood-2026-09-28.md)).
- Telegram → `default` → `executor` → same-card `reviewer` → same chat, then `docker restart` with dispatch still working.
- Fresh unrelated repository, s3upload (Go): install, private setup, dispatch check, real Go fix with tests run by executor and reviewer, Telegram delivery, restart ([record](docs/qualification/fresh-repo-s3upload-2026-09-29.md)).
- Fresh repository PMB blocked by an upstream Hermes worker import bug; `verify` now detects it ([record](docs/qualification/fresh-repo-pmb-2026-09-29.md)).

## Next: fresh-install follow-ups (0.1.x)

Found in the fresh-repository trials. Each needs unit tests plus a real check
against a fresh clone.

- [x] Accept a group-writable (775) root when the group is the owner's private
      group; refusals name the path and the `chmod` fix (#3).
- [x] **Post-start install race.** On a first boot Hermes remaps its user and
      fixes data ownership, so a `docker exec` issued in that window fails with
      `PermissionError: /opt/data/.env`. `install` and `setup` now wait (bounded,
      90 s) for a public `config get` read, then report "runtime still starting"
      instead of "native team inspection unavailable" (#4).
- [x] **Gateway never started on fresh setup.** Explicit `setup` / `setup --team`
      now starts a stopped default gateway through the native `gateway start`
      and waits for a running PID; Hermes keeps it running across restarts.
      `install` reruns never start an owner-stopped gateway and print
      `hermes-<repo> -p default gateway start` instead (#6).
- [x] **`.hermes-repokit.lock` hygiene.** The lock must stay at the root (it
      guards creation of `.hermes` and is shared with the container flock), so
      `install` adds `/.hermes-repokit.lock` to the local, never-committed
      `info/exclude` (worktree-aware), unless already ignored. The tracked
      `.gitignore` is never edited; an unsafe exclude file only warns.
- [ ] **Actionable collision text.** "private .hermes state is tracked by Git"
      names neither the file nor the fix. Report each tracked path and the remedy
      (move it outside `.hermes`, then `git rm --cached`).
      *Test:* repository tracking a file under `.hermes/` → message names it.
- [ ] **Env-only channels are invisible to `verify`.** A Telegram channel
      configured only through `.env` gets no `channel:telegram` row. Use the
      public `hermes -p default tools list --platform <p>`.
      *Test:* env-only Telegram → row reports Kanban/memory from effective tools.
- [ ] **Install output noise.** `install` always prints "Start Hermes with
      ordinary Compose" (even when running) and "Created …" on upgrades.
      Print only what applies.
- [ ] **Monorepo toolchains.** Detection reads root manifests only, so PMB's
      Python/Rust subprojects were missed. Defer unless real projects need it.

## Owner request: `remove`

- [x] `hermes-repokit remove` deletes a RepoKit deployment (Compose project,
      generated image, host command, lock and exclude line, and `.hermes`) after
      the repository name is typed in an interactive terminal; it refuses state
      it cannot prove it generated.
- [ ] Optional: a non-interactive `--yes` for scripted teardown, if ever needed.
- [x] `install` builds and starts the deployment itself; `stop` / `start` stop and
      restart it with state kept (recreation deferred while a card runs).

## Owner request: preferred tool defaults

- [ ] Enable a preferred tool set on `default` at install/setup through native
      commands, favoring free or self-hosted providers, with paid/API-key
      integrations optional. Captured set: see the s3upload trial record.
- [ ] Make the free web backend work: `ddgs` failed to install because the
      sealed Hermes environment has no pip. Provision it where Hermes loads
      lazy packages, or accept a SearXNG URL.
- [ ] Report tools that cannot work headless (Computer Use) instead of silently
      enabling them.
- [ ] Decide: do specialists keep narrow role toolsets or get the full set?

## Memory boundary

- [x] Remove the embedded shared-memory provider. RepoKit neither configures nor
      verifies memory; the operator owns provider setup and behavior.

## Upstream

- [ ] Hermes [#126127](https://github.com/NousResearch/hermes-agent/issues/126127) /
      [#126277](https://github.com/NousResearch/hermes-agent/pull/126277): workers
      import repository modules (`python -m` from the workspace). When a fixed
      release becomes the pinned image, rerun the reproduction in
      [python-import-collision.md](docs/qualification/python-import-collision.md)
      and retire the `python-imports` probe and `HermesImportNames`.
- [ ] Requalify `NativeDefaultSoulSHA256` and `HermesImportNames` on every
      `FoundationImage` change.

## Acceptance tests still open

Live and Docker proof that has not been exercised yet.

- [ ] **Reviewer correction cycle:** reviewer requests changes → executor
      revises on the same card → reviewer approves → result in the same chat.
- [ ] **Vague requests:** "Improve readme" produced a reply but no card; qualify
      how `default` scopes open-ended requests.
- [ ] **Removal-first:** delete the RepoKit binary, then run a real Telegram task
      and a restart; everything must keep working.
- [ ] **More fresh repositories:** a Python repository without colliding root
      modules, and a Node repository, to exercise toolchain adequacy.
- [ ] **Automated upgrade test:** turn the ad hoc old-release → new-release
      upgrade probe into a Docker acceptance test (install with the previous tag,
      upgrade, recreate, restart, state byte-identical).
- [ ] **SELinux:** run on an SELinux-enforcing host (not yet available).
- [ ] **Isolated Docker test daemon:** run `TestDockerIsolatedAcceptanceDaemon`
      (`REPOKIT_DIND_TESTS=1`).

## Release engineering

- [ ] **CI:** there is none. Add GitHub Actions for `go test`, `go test -race`,
      `go vet` (plain and `-tags=docker`), gofmt and `git diff --check`; Docker
      acceptance stays opt-in.
- [ ] **Binaries:** publish linux/amd64 and linux/arm64 builds per release, or
      stop claiming "no host Go" in the README.
- [ ] **arm64:** generated Compose pins `platform: linux/amd64` although the
      recipe supports arm64; qualify arm64 or document amd64-only.
- [ ] Optional Superpowers plugin: scanner admission if ever selected.

## Cleanup (low priority)

- [ ] Retire legacy Compose recognition (Laya, previous names, pre-SELinux,
      pre-docker-tests) once no deployment predates the recipe self-check.
- [ ] Remove the no-op `--engineering` flag.
- [ ] `verify` review evidence reads `kanban list/show`; confirm those never
      migrate the board schema, or gate them on an initialized board.
- [ ] Local housekeeping: decide on the `hermes-s3upload` trial deployment,
      old `feat/*` branches, `/tmp/repokit-review` and the 594 MB
      `.hermes/backups/pre-recipe-upgrade-*.zip`.

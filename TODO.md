# Hermes RepoKit TODO

RepoKit configures a repository-specific Hermes team through public Hermes
interfaces, proves what it can prove, then gets out of the way.

**Released:** [v0.2.17](https://github.com/TrebuchetDynamics/hermes-repokit/releases/tag/v0.2.17)
(repository toolchains with nothing manual, autonomy stated at setup, memory
left to the owner, fresh chats that follow default's identity in v0.2.4;
v0.2.5 recognizes untouched SOULs by a recorded digest; v0.2.6 makes install
one command with live progress and ships text to speech; v0.2.7 adds `repokit update`
and keyless Exa web search by default; v0.2.8 adds `repokit list` and local speech
to text; v0.2.9 teaches the coordinator to talk to the owner plainly; v0.2.10
stops uv scratch locks blocking upgrades; v0.2.11 provisions Godot 4; v0.2.12
gates recurring checks with a monitor script; v0.2.13 adds Godot export
templates and Android builds; v0.2.14 fixes its verify listing; v0.2.15 has
agents do the work they can instead of blocking; v0.2.16 pushes through the
owner's gh login; v0.2.17 adds the single-profile shape, on trial on
hermes-wing, and keeps image builds from filling the disk).
Since v0.3.0, RepoKit is one profile, `default`. The seven-profile same-card
executor → tester → reviewer loop was proven live before that; the one-profile
implementation → verification loop is not yet proven live.

## Released in v0.3.0

- RepoKit is one profile, `default`: it talks with the owner, researches
  through read-only subagents, implements Kanban cards assigned to itself and
  verifies each in a separate fresh run (`kanban_request_review` with
  `reviewer="default"`). The rubric: acceptance shown by command output; no
  deleted, weakened or special-cased tests; scope; new probes. Risky changes
  fan out up to four subagents with distinct jobs; after two change-request
  rounds the card goes back to the owner.
- Memory is on for `default` and holds itemized owner decisions and
  preferences only. Memory and delegation are enabled on every human channel.
- New grants on `default`: `checkpoints.enabled`, `delegation.oneshot_max_children`
  4, `agent.max_turns` 0, `agent.reasoning_effort` high, and the union of the
  old roles' skills. The dispatch allowlist is `dispatch_profiles: ["default"]`.
- Removed: the six worker profiles and their SOULs, `install --team` and
  `.hermes/repokit-team`. `--reset-profile` takes only `default`.
- Migration: `install` retires a seven-profile deployment once, on an idle
  board. Open cards of the six move to `default`; each profile is exported to
  `.hermes/backups/profile-<name>-<UTC time>.tar.gz` (restorable with
  `hermes profile import`) and deleted. A busy board reports `retire-later`;
  rerun install later.
- `verify` counts as review evidence a card `default` implemented and a
  separate `default` run verified and completed. The dispatch-check canary is
  assigned to `default`.

## Released in v0.2.4

- Toolchains from the repository's own manifests, including nested projects:
  Go with staticcheck, Rust, Flutter with Dart, and Flutter's Linux desktop
  toolchain; shellcheck, the browser tool and tirith in every image.
- Autonomy posture stated at `setup`: no approval prompts by default, or
  Hermes's prompts kept on the owner's answer.
- Memory boundary finished: no profile is granted memory and no SOUL relies on
  it; an owner who adds memory keeps it.
- Fresh chats follow default's identity; upgrades never interrupt a running
  card; the safety scan covers grown native state.
- MIT license, security policy, and Docker foundation acceptance on release
  tags and weekly.
- Withdrawn: the board watch cron job (a RepoKit-authored helper on the
  operational path); RepoKit installs, configures and leaves.

## Proven in v0.2.3

- Dogfood and s3upload moved to role-shaped profiles in place: every profile
  reset to the new baseline, all 15 granted official skills installed, each
  specialist's toolsets as designed, and `ast-grep`/`ddgs` working in the
  rebuilt image; `verify` exit 0 on both, s3upload `CORE_READY` healthy.
- The new `sessions` probe flags both Telegram chats, begun before the current
  identity, until `/new`.
- s3upload skills card `t_3359311e`: steward → tester (blocked, then passed) →
  reviewer on the same card.

## Proven in v0.2.2

- A real v0.2.0 deployment of a Go repository, with a full team, upgrades in
  place: the v0.2.0 Compose and recipe are backed up as
  `compose.before-toolchain-cache-*`, the container is rebuilt with the
  toolchain-cache volume, all seven profiles stay current, dispatch is
  unchanged and `verify` reports nothing unhealthy. v0.2.1 refused it. A
  non-Go v0.2.0 deployment upgrades with nothing to change.

## Proven in v0.2.1

- s3upload: a real task (`t_9b06fdfa`, shortener tests) ran executor → tester →
  reviewer on the same card with independent re-verification; `CORE_READY`
  healthy.
- Dogfood and s3upload moved to the toolchain-cache volume in place; a Go build
  in the container writes its caches to the volume, not the repository.
- Fresh Node, Python and Go trial repositories: `plan`, `install`, `verify`,
  `stop`/`start` and `remove` (Node, Go) behaved as documented.

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
- [x] **Actionable collision text.** "private .hermes state is tracked by Git"
      names neither the file nor the fix. Report each tracked path and the remedy
      (move it outside `.hermes`, then `git rm --cached`).
      *Test:* repository tracking a file under `.hermes/` → message names it.
- [x] **Env-only channels are invisible to `verify`.** `verify` now reads
      configured platforms from public `send --list --json` (targets are never
      reported) and judges a platform without a saved tool selection by
      `tools list --platform <p>`.
- [x] **Install output noise.** `install` starts the deployment itself and
      reports an in-place upgrade as an upgrade, not "Created …".
- [x] **Toolchain caches live inside the repository.** The development recipe
      sets `GOMODCACHE=/opt/data/development/go-mod` and
      `GOCACHE=/opt/data/development/go-build` (`internal/development/recipe.go`),
      and `/opt/data` is `<repo>/.hermes`. Go's `./...` skips dot-directories, but
      whole-tree commands do not: `gofmt -l .` exits 2 on third-party cache files
      and invalid fixtures, and `grep -r`/linters wade through the cache. An
      s3upload executor correctly blocked card `t_9b06fdfa` on this, and release
      validation hit it too. Move the caches to a named volume or a path outside
      the bind mount (keeping them across restarts), or at minimum tell the SOULs
      that repository-wide checks exclude `.hermes`.
      *Test:* after a Go build in the container, `gofmt -l .` in the repository
      lists nothing under `.hermes`.
      Fixed: Go caches use the project's `toolchain-cache` volume at
      `/var/cache/repokit`; `install` reports a leftover `.hermes/development`.
- [x] **Monorepo toolchains.** Detection walks nested projects up to three
      folders deep (skipping vendored, generated and hidden trees) and
      provisions Go, Rust and Flutter, plus Flutter's Linux desktop toolchain
      for an app with a `linux/` runner.

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
- [x] Make the free web backend work: the development image ships `ddgs`,
      hash-pinned through `uv --require-hashes`, for keyless web search.
- [ ] Report tools that cannot work headless (Computer Use) instead of silently
      enabling them.
- [x] Decide: do specialists keep narrow role toolsets or get the full set?
      Role-shaped toolsets (v0.2.3); none includes memory, which stays the
      owner's to add. Superseded in v0.3.0: one profile with the full set.

## Memory boundary

- [x] Remove the embedded shared-memory provider. RepoKit neither configures nor
      verifies memory; the operator owns provider setup and behavior.
- [x] Drop memory from RepoKit's contracts: default channels require only Kanban
      (owner tool choices preserved, memory never enabled), `verify` no longer
      reports memory, and the default SOUL no longer guarantees it. The first
      seven-role SOUL generation is frozen so those defaults still upgrade.
- [x] v0.3.0 turns Hermes's built-in memory tool on for `default`, for the
      owner's decisions and preferences. RepoKit still configures no memory
      provider and does not verify recall.

## Upstream

- [ ] Hermes [#126127](https://github.com/NousResearch/hermes-agent/issues/126127) /
      [#126277](https://github.com/NousResearch/hermes-agent/pull/126277): workers
      import repository modules (`python -m` from the workspace). When a fixed
      release becomes the pinned image, rerun the reproduction in
      [python-import-collision.md](docs/qualification/python-import-collision.md)
      and retire the `python-imports` probe and `HermesImportNames`.
- [ ] Requalify `NativeDefaultSoulSHA256` and `HermesImportNames` on every
      `FoundationImage` change.
- [ ] Rootless Podman (owner decision 2026-10-02: stay on Docker for now). A
      spike ran the Docker foundation acceptance through a `podman` Docker
      context on Podman 4.9.3. The image built and the container ran, but the
      Hermes image boots as root and drops to `hermes`, which rootless user
      namespaces cannot satisfy: without `userns_mode: keep-id` the state
      files belong to a sub-ID (the owner cannot read `.hermes`); with it, the
      root-run profile reconcile cannot read `.hermes` and `exec` into
      `/workspace` is refused. Revisit when Hermes ships a rootless-friendly
      image; the RepoKit side is then four small changes: an empty read-only
      volume instead of the tmpfs `.hermes` mask (Podman copies tmpfs up),
      accept that volume in the mount check, strip Podman's `docker.io/` image
      prefix before comparing, and `keep-id` on Podman.

## Acceptance tests still open

Review evidence reads the newest 20 done cards assigned to `default`
(`kanban list --assignee default`) and needs a `default` run requesting review
followed by a separate `default` run completing the card.

Live and Docker proof that has not been exercised yet.

- [ ] **One-profile loop:** on hermes-wing, a real task is implemented by
      `default` and verified and completed by a separate `default` run on the
      same card (qualifies `CORE_READY`), then the same on every live
      deployment.
- [ ] **Migration:** a live seven-profile deployment retires its six profiles
      on install: open cards reassigned, backups written under
      `.hermes/backups/`, dispatch policy rewritten.
- [ ] **Correction cycle:** a verification run requests changes → a new
      implementation run revises on the same card → a new verification run
      completes it → result in the same chat. After two rounds the card goes
      back to the owner.
- [ ] **Vague requests:** "Improve readme" produced a reply but no card; qualify
      how `default` scopes open-ended requests.
- [ ] **Removal-first:** delete the RepoKit binary, then run a real Telegram task
      and a restart; everything must keep working.
- [ ] **More fresh repositories:** a Python repository without colliding root
      modules, and a Node repository, to exercise toolchain adequacy.
- [ ] **SELinux:** run on an SELinux-enforcing host (not yet available).
- [ ] **Isolated Docker test daemon:** run `TestDockerIsolatedAcceptanceDaemon`
      (`REPOKIT_DIND_TESTS=1`).

## Release engineering

- [x] **CI:** GitHub Actions runs gofmt, whitespace checks (pull requests),
      `sh -n install.sh`, `go vet` (plain and `-tags=docker`), `go test` and
      `go test -race`; Docker acceptance stays opt-in.
- [ ] **Binaries:** publish linux/amd64 and linux/arm64 builds per release, or
      stop claiming "no host Go" in the README.
- [ ] **arm64:** generated Compose pins `platform: linux/amd64` although the
      recipe supports arm64; qualify arm64 or document amd64-only.
- [ ] Optional Superpowers plugin: scanner admission if ever selected.

## Cleanup (low priority)

- [x] Remove the no-op `--engineering` flag.
- [ ] `verify` review evidence reads `kanban list/show`; confirm those never
      migrate the board schema, or gate them on an initialized board.
- [ ] Local housekeeping: decide on the stopped `hermes-s3upload` trial
      deployment (deprecated; hermes-wing replaced it as the live tester),
      old `feat/*` branches, `/tmp/repokit-review` and the 594 MB
      `.hermes/backups/pre-recipe-upgrade-*.zip`.

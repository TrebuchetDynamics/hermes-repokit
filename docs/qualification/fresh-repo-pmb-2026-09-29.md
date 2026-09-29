# Fresh unrelated-repository trial: polymarket-mega-bot clone (2026-09-29)

**Result: BLOCKED.** RepoKit installed correctly, then its behavioral dispatch
check exposed that the underlying Hermes worker environment could not safely
execute this repository. Tested PR #1 at `ed0ab75`; RepoKit source was not
changed during the run.

## Sequence

1. Fresh `git clone` of `XelHaku/polymarket-mega-bot` into `~/git/pmb-repokit-trial`
   (the live repository and its existing hand-built Hermes deployment were not read).
2. `plan` correctly refused: the repository tracks
   `.hermes/plans/2026-08-11_051937-env-label-execution-audit.md` (RepoKit owns
   `.hermes` as private state), and the clone root was 775 under umask 0002.
3. Owner prepared the disposable clone: untracked that file (preserved elsewhere)
   and `chmod 755` the root. `plan` then reported no collisions.
4. `install`, Compose start and a second `install` created one
   `hermes-pmb-repokit-trial` container and initialized Kanban.
5. At the owner's request, provider OAuth and the Telegram bot settings were
   copied from the dogfood deployment (no wizard); `hermes-repokit` was stopped so
   only one gateway polled the shared bot. `setup --team` provisioned six
   profiles and configured dispatch. The gateway had never run, so it was started
   once with `hermes-pmb-repokit-trial -p default gateway start`; Telegram connected.
6. `verify`: all configuration/runtime probes healthy; `CORE_READY` unqualified
   (no reviewed work yet); memory inactive.
7. `verify --dispatch-check` failed three times (`t_dd42b4b9`, `t_d6a97fc6`,
   `t_c4dcde67`). The gateway claimed each card automatically, but the researcher
   had no file tools and blocked. Moving `AGENTS.md` aside did not change the
   result (restored byte-identical). Root cause: the
   [Python import collision](python-import-collision.md) caused by the
   repository's root `tools/` package.

The Telegram task, executor/reviewer loop and restart steps were deliberately not
run: they would have exercised a known-broken worker environment. `verify` now
reports this repository's `python-imports` degraded.

## Findings

1. Tracked `.hermes` content is correctly refused, but the message names neither
   the file nor the remedy.
2. A 775 repository root from a normal fresh clone is refused; likely an overly
   strict gate.
3. Toolchain detection reads root manifests only; PMB's Python/Rust subprojects
   were not detected.
4. `install` leaves `.hermes-repokit.lock` untracked and not gitignored at the root.
5. A fresh install ends with the gateway never started and no command given.
6. `verify` observes only channels with saved per-channel tool selections;
   `hermes -p default tools list --platform <p>` could cover env-only channels.
7. Upstream Hermes: workers can import repository modules instead of Hermes'
   (the blocker above), tracked as NousResearch/hermes-agent#126127 with RepoKit's
   evidence commented. Detected by RepoKit's `verify` since this trial.

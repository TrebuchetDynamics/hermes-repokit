# Fresh unrelated-repository acceptance: s3upload (2026-09-29)

**Result: PASS.** PR #1 at `4e30c3f`, frozen during the run. Target: a fresh clone
of `XelHaku/s3upload` (Go, `2837bb2`) in `~/git/repokit-trials/s3upload`. The
owner's existing checkout and its gitignored Azure credential file were not used.

## Supported path, no borrowed state

1. `plan` refused only the 775 clone root; preparation `chmod 755` (recorded).
2. `install`, printed Compose build/start, second `install` (Kanban). The second
   `install` failed once when run under a second after start ("native team
   inspection unavailable") and succeeded on retry: a startup race (finding).
3. Owner ran `setup` privately: own provider login (`gpt-6.1-sol`) and a **new**
   Telegram bot, allowlisted to the owner; OpenViking skipped. No authentication
   was copied from another deployment. RepoKit provisioned six profiles and the
   full dispatch policy; the wizard had removed Kanban from Telegram and RepoKit's
   saved-channel reconciliation restored it.
4. The gateway was never started by setup; started once with
   `hermes-s3upload -p default gateway start` (finding). New bot connected.
5. `verify`: exit 0; `CORE_READY` unqualified (no reviewed work yet),
   `MEMORY_READY` inactive, every configuration/runtime probe healthy including
   Go 1.26.6 and `python-imports`.
6. `verify --dispatch-check`: PASS (researcher `t_5a5e0add`, no manual dispatch).

## Telegram → executor → reviewer → Telegram (UTC)

| Time | Evidence |
| --- | --- |
| 20:36:14 | Owner's task received from Telegram |
| 20:38:08 | `default` inspected the repository and created `t_af6758f3` for executor |
| 20:38:33 | Gateway dispatched executor |
| 20:39:55 | Executor replaced `t.Fatal` inside an HTTP handler goroutine with `t.Error` + `return` in `main_test.go`; recorded baseline and post-change `go test ./...`, `go test -race ./...`, gofmt and `git diff --check`, all exit 0; requested same-card review |
| 20:40:33 | Gateway dispatched reviewer on the same card |
| 20:41:16 | Reviewer independently re-ran gofmt, `go test ./...`, `go test -race ./...`, `git diff --check`; approved |
| 20:41:29 | Completion result sent to the originating chat; owner confirmed receipt |

The change is a genuine Go correctness fix (`FailNow` must not run off the test
goroutine). Only that one hunk changed (all other tracked-file checksums
unchanged), nothing staged, HEAD unchanged. Re-run independently in the
deployment's worker login shell: `go test`, `go test -race -count=1`, gofmt and
`git diff --check` pass.

## Restart

`docker restart hermes-s3upload` at 20:43:21: the gateway returned on its own,
Telegram reconnected at 20:43:38, both cards and the dispatch policy were
unchanged, and the worktree change was preserved. A new `verify --dispatch-check`
passed (`t_7b964ddb`). Final `verify`: `CORE_READY` **healthy** citing
`t_af6758f3`, `MEMORY_READY` inactive, exit 0.

Hermes posted a native "shutting down" notice to the chat on restart.

## Findings (not fixed during the run)

1. A normal fresh clone's 775 root is refused (also seen in the PMB trial).
2. An immediate second `install` after Compose start can hit Hermes still booting;
   RepoKit should wait/retry within a bound instead of a generic error.
3. `.hermes-repokit.lock` is left untracked and not gitignored at the root.
4. Fresh setup ends with the gateway stopped and no command given.
5. Tool-provider defaults (owner request): enable a preferred tool set with
   free/self-hosted providers; the free `ddgs` web backend failed to install in
   the sealed Hermes environment. Separate change after PR #1.

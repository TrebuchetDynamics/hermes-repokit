# RepoKit live dogfood, 2026-09-28

Target: this repository, using the pinned Hermes build
`749220ef0007f8d87bd1531f1c24b0fe93816385` and the generated development image.
This record concerns actual native commands and Docker processes, separately
from offline fixtures. No provider credentials were changed or printed.

## Observed failures and source repairs

- The research card was ready with no worker run because native gateway dispatch
  was disabled. Core setup also incorrectly required OpenViking readiness.
  Core activation now reports memory independently.
- The exact historical local-build Laya Compose was outside the upgrade set.
  The installer now recognizes that generated preimage, backs it up and preserves
  private state while generating the current runtime. Edited stacks still refuse.
- Older managed workers inherited stock Telegram selections. Migration now checks
  complete managed identity and configuration before narrowing those selections
  through native configuration commands. Ambient plugin discovery is excluded
  from the stock comparison; explicitly selected custom plugins remain drift.
- Native `tools enable` saved an ambient plugin during an executor tool upgrade.
  Exact specialist configuration now uses native `config set`. The failed live
  attempt's three tool fields were restored from the pre-repair native backup
  after checking their exact post-operation values, then setup was retried.
- Native gateway runtime status omits `hermes_home`. Both passive readers now
  accept that omission while requiring the correct home in PID and lock records,
  matching process fingerprints, held locks and fresh native health evidence.
- Canary creation supplies and validates native `completion_contract=local-only`.
  The previous empty-field expectation rejected the card that Hermes itself
  created. A safe diagnostic now identifies the failing activation stage using
  fixed function names and line numbers, without printing exception values.
- The first researcher completed its no-write canary, but the checker interpreted
  a shell comment inside a fenced README example as a heading. The v3 canary
  excludes fenced code in both its instructions and the title parser; the earlier
  completed run is preserved.
- Normal native speech-model downloads created Hugging Face cache pointers and
  writable lock/ref metadata. Inspection now recognizes those exact model-cache
  shapes without following links; ownership and ancestor-directory checks remain.

## Live evidence

The native full backup completed before migration. Original Compose, profiles,
credentials, memories, Kanban history, model caches and owner plugins were
preserved. The generated development image built successfully and replaced the
old Hermes runtime through ordinary Compose. The repository now has one runtime
container with embedded OpenViking awaiting private configuration.

The host command resolves through `~/.local/bin` to the repository launcher and
works outside the repository. Its native profile, Kanban and setup interfaces
were exercised. The development image reports the required tools, including
Go 1.26.6. All six managed profiles reconcile and trust `/workspace` for project
skill discovery. The gateway reports connected Telegram and API adapters;
that is not a human message round-trip qualification.

The bundled maintenance plugin passed the native scanner. This does not prove
an agent-initiated restart. OpenViking remains pending and cross-profile recall
is unqualified. Independent executor/reviewer acceptance remains unqualified.

`setup --team` completed successfully. Gateway PID 9452 automatically claimed
researcher canary `t_4e3b0882`; its native run completed in 15 seconds with
`title=NO_MARKDOWN_TITLE` and `changed_files=[]`, and setup archived the card.
RepoKit observed the worker as that gateway's child without manual dispatch.
Read-only verification reports healthy generation, live dispatch, policy, canary,
six-profile identity and development tools while memory remains pending/inactive.
The original research task `t_e40fd4b0` was then automatically claimed and completed
by researcher run 5 in 59 seconds. Its originating Telegram notification/wake
subscription remains present; this record does not assert a human-observed round trip.

Two older, superseded workstream roots were left unassigned during repair to avoid
starting obsolete code changes; their cards, dependencies and history remain.

## Compose coexistence and delivery validation

Existing root Compose files no longer prevent installation. RepoKit retains its
explicit private Compose file and project namespace. The real Docker foundation
fixture started a separate owner stack, then ran plan/install/native setup,
installer removal and Hermes recreation. The owner's container ID, running state
and both Compose files remained unchanged. Ambient `COMPOSE_FILE` and
`COMPOSE_PROJECT_NAME` deliberately selected conflicting values throughout the
fixture; all fixture Compose calls explicitly cleared them. The config-only
fixture also covered an invalid override and conflicting repository `.env`.
Both fixtures passed, with the full foundation run completing in 240 seconds.

The isolated delivery snapshot passed `go test -count=1 ./...`,
`go test -race -count=1 ./...`, Docker-tagged vet, formatting and diff checks.
`CGO_ENABLED=0` built the bootstrap successfully with `CC` pointing to a nonexistent
compiler, and the resulting binary printed its help. Race testing remains source
contributor validation; it is not an installation prerequisite.

These fixtures are credential-free and do not replace the live researcher evidence
above. Concurrent SELinux work in the main checkout was excluded from this delivery.

## Recipe upgrade and restart (2026-09-29)

Before migration: native `hermes backup` (594 MB), a consistent Kanban copy (20
cards) and the generated files were saved; no card was running. `install`
recognized the older generated recipe by its self-certifying fingerprint, saved
`compose.before-recipe-3033add1f051.yaml` and the exact old recipe beside it,
removed the obsolete entrypoint patch file and published the new recipe. The
printed Compose command rebuilt `repokit/hermes-repokit:3fb6500b99fc07711dcd3ef3`
and recreated the one container. A second `install` reconciled all six profiles
without drift and left the operational dispatch policy and gateway untouched.

Afterwards the six SOUL/description/config files and all 20 cards (id, status,
assignee) were unchanged; the dispatch policy was intact; `hermes-repokit` ran
from `/tmp`. The native main-wrapper re-registered the s6 gateway services: the
default gateway was running about 5 seconds after recreation and again after
`docker restart`. `verify` then reported `CORE_READY` healthy (same-card review
evidence `t_b170ab94`) and `MEMORY_READY` inactive (OpenViking not configured).

`verify --dispatch-check` on the restarted gateway passed in 80 seconds: card
`t_ee305797` went created → claimed → spawned → completed → archived with no
manual dispatch; the researcher returned the exact README first line with
`changed_files: []`. Telegram round-trip delivery was not exercised.

## Telegram round trip and restart (2026-09-29)

A bounded documentation task was sent from the owner's phone to the default
bot in a fresh `/new` session. All times UTC; no manual dispatch or CLI step.

| Time | Evidence |
| --- | --- |
| 18:44:58 | Telegram message received by `default` |
| 18:45:23 | `default` created card `t_6641c2b0` for `executor`, preserving the task's bounds |
| 18:45:43 | Gateway dispatched `executor` |
| 18:46:20 | Executor changed only the requested paragraph of `docs/implementation-progress.md`, reported `git diff --check` and `go test ./...` passing, and requested same-card review |
| 18:46:36 | Review-handoff notification sent to the originating chat |
| 18:46:39 | Gateway dispatched `reviewer` on the same card |
| 18:47:16 | Reviewer independently re-ran both checks, made no edits and approved |
| 18:47:22 | Completion woke `default`; result sent to the originating chat |

The owner confirmed both messages arrived in the same chat. The other
uncommitted working-tree paths were unchanged (checksums compared).

After `docker restart`, the default gateway was running within about 5 seconds
and the Telegram adapter re-registered its commands. All 22 cards (id, status,
assignee) matched a consistent pre-restart copy; the dispatch policy was intact;
`verify` reported `CORE_READY` healthy citing `t_6641c2b0`; and a new
`verify --dispatch-check` completed researcher card `t_e8f4e070`.

Not exercised: a reviewer request-changes correction cycle, a vague request
("Improve readme" produced a reply but no card; the conversation was not
inspected), a fresh unrelated repository, and shared memory.

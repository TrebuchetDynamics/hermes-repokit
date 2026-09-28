# Operational gateway dispatch

User-approved contract: bootstrap remains dispatch-off; successful setup must activate the single default gateway with review dispatch, the six-profile allowlist, no decomposition and concurrency one. No manual card dispatch or live state hand-editing.

Implementation sequence:
1. Add pure dispatch policy and read-only live evidence tests (config versus gateway generation, singleton lock ownership, native startup settings).
2. Extend native setup with safe suspension/reconciliation and gated activation. Require current six-role identity/tool contracts, native provider resolution, shared authenticated OpenViking. Preserve drift and active work. Native config commands and graceful gateway restart own changes.
3. Update the coordinator contract and support exact historical SOUL migration. Keep specialist dispatch disabled. Report configured/live dispatch separately through read-only verify.
4. Add an explicit native gateway-only researcher canary during setup, checking real actor/result evidence and archiving only successful canaries. No inference in verify.
5. Run offline tests/race/vet/format checks. Run live canary and Telegram/coding acceptance only where Docker access exists; never report fixture results as live acceptance.

Qualification source: Hermes 749220ef0007f8d87bd1531f1c24b0fe93816385, gateway/kanban_watchers.py and gateway/kanban_watchers_common.py. Local source matches the pinned foundation image revision. Singleton lock: kanban/.dispatcher.lock; per-board native fence: kanban.db.dispatch.lock. Startup logs include max_in_progress and embedded interval. Native gateway restart drains before replacement.

Environment: Docker socket access denied in this session. Existing uncommitted work preserved; starting file hashes recorded outside repository in /tmp/repokit-dispatch-baseline.json. No commits or pushes.

Recovery detail: a canary failure clears the native hot-read dispatch allowlist and saves dispatch-off before attempting a safe graceful restart. This temporarily empty allowlist belongs only to failure recovery, never to operational state. Active/finalizing workers are preserved; the next setup rerun restores the six-role allowlist after gates pass. Historical worker PIDs are checked using native spawn fingerprints so recycled PIDs do not permanently block setup. Interrupted terminal canaries are preserved and a separately keyed successor must prove a new run; a successfully observed current canary is archived natively.

Follow-up recovery repair (2026-09-28): regressions reproduced a fixed-key blocked
canary permanently poisoning later activation and an unchanged verified
activation unnecessarily restarting/repeating paid work. Activation now rechecks
required gates before reusing matching live generation/lock/canary evidence.
Terminal prior canaries can lead to a new attempt only after native claim
fencing, exact task-contract revalidation, closed researcher-run evidence and
empty dependency links. Prior cards remain untouched: native Hermes has no atomic
conditional archive, so a read/archive recovery could race an owner edit or link.
A stable task-ID retry chain reuses unfinished work after gateway replacement;
recovery is bounded to 32 terminal predecessors. Historical successful headings
may differ from today’s README, but the new run must prove the current title.
Modified or active cards are preserved. A failed new attempt still suspends
activation; there is no automatic repeated-inference loop or manual dispatch.

Validation: 15 native activation tests pass, including both reproduced failures
and refusal cases for owner edits, dependencies, foreign actors, active runs and
finalizing workers. Full normal/race suites pass every other package but retain
six sandbox-denied AF_UNIX cases in internal/target. Docker-tagged vet passes.
Live gateway/reviewer dispatch and Telegram delivery remain unqualified; no
runtime state was changed and no commit/push was performed.

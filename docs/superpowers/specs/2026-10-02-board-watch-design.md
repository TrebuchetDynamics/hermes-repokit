# Board watch: keep committed work moving

> **Withdrawn 2026-10-02.** Implemented, proven on the dogfood deployment, then
> removed: it left a RepoKit-authored script running after install, which
> breaks RepoKit's rule that it installs, configures and leaves (no sidecar,
> helper or plugin on the operational path). Kept as a record of the Hermes
> limits it established.

The team acts only when something wakes default, the coordinator. Kanban wakes
default in the owner's chat when a card it subscribed to completes, blocks or
changes review stage. A quiet board raises no event, so live deployments
stalled: Megabot idled nine hours with open goals, Wing's validation card sat
in triage, and sdrhf's owner-input card waited without a reminder. Owner goals
also lived only in chat and `/goal`, which end when RepoKit freshens a chat.

The owner wants two things, in this order: keep committed work moving (recover
stuck cards, keep goals from expiring, remind the owner of blockers that need
them), then start new work toward stated goals when the board is idle. Never
work no goal asked for.

## Constraints from Hermes (verified on the installed release)

- Gateway "autonomy", a heartbeat plugin and a proactive feature router do not
  exist in this Hermes. `/goal`, `/heartbeat`, `/loop`, cron and webhooks do.
- A dispatched worker cannot list the board or unblock cards
  (`kanban_list` and `kanban_unblock` are orchestrator-only), so goal progress
  cannot be judged by a goal card running as a worker.
- Blocking a card twice for the same reason routes it to triage
  (`BLOCK_RECURRENCE_LIMIT = 2`), so blocking cannot park a recurring card.
  `scheduled` parks a card without dispatch and without that limit; nothing
  moves it on a timer.
- A card created outside a messaging session gets no notification
  subscription. Any CLI context can add one with
  `hermes kanban notify-subscribe --delivery-mode notify+wake`; a subscription
  targets the chat, so it survives RepoKit freshening the conversation.
- A cron job with `--monitor-script` runs the script each tick and skips the
  model when its output is byte-identical to the previous tick (observed:
  72 ms, `no_change (agent run suppressed)`). The first tick always runs the
  model once as a baseline.
- A cron-run default has `kanban_list` and `kanban_create`. Each wake costs
  about 98,000 prompt tokens with default's full identity.

## Design

### Goal cards

When the owner states a goal, the coordinator records it on the board with
`hermes kanban create` without an assignee (never dispatched) and parks it
with `hermes kanban schedule`. The title begins `Goal:`. The body holds the outcome,
acceptance criteria, a budget (a number of cards or an end date) and the
owner's constraints. Later constraints and snoozes are comments. The owner
sees, edits and archives goals on the board. A constraint the owner gives only
in chat must be recorded on the affected goal or card.

### Watch job

RepoKit installs one cron job on default named `repokit-board-watch`,
scheduled every 5 minutes, with `--monitor-script repokit-board-watch` and
delivery to the home channel. It sets no workdir: one would load the
repository's AGENTS.md into every wake (observed: about 209,000 prompt tokens
instead of about 98,000). The script lives in
default's `scripts/` and reads only the board and the session database.

The script repeats its previous output (exactly `busy` at first) when any of
these hold, so neither a busy board nor the board getting busy again wakes the
model, and the job's prompt tells a run that sees `busy` (Hermes's first
baseline run) to reply `[SILENT]`:

- a card is running, or ready or todo with an assignee;
- the owner's chat was active in the last 15 minutes;
- there is no open goal card and no stuck card.

Otherwise it prints a digest: open goal ids, triage card ids, blocked card
ids, the time the board went idle, and an idle back-off bucket (30m, 1h,
2h, 4h, 8h, then one per day since the board went idle). The digest carries no
timestamps, so the model runs only when the board's stuck state changes or the
bucket advances: at most about six wakes in an idle day, none while busy.

On every tick the script also subscribes the home chat, with `notify+wake`, to
each card linked to a goal card that lacks a subscription, so completions of
goal work wake the chat coordinator as today.

### Woken coordinator

The job's prompt directs one bounded pass:

1. Return each triage card with `kanban specify`, then immediately restore its
   original title and body with `kanban edit`.
2. For each open goal within budget, create the next card linked to the goal
   card, after checking the board for a card that already covers it. Complete
   the goal card when its acceptance is met with evidence; block it with one
   question when its budget is spent.
3. Once a day, remind the owner of cards blocked on them, skipping snoozed
   ones.
4. Otherwise reply `[SILENT]`.

It never creates work outside a goal card.

### Lifecycle

- RepoKit creates the job and script when it sets up or upgrades the team, and
  recognizes the job by name and the script by content.
- An owner who pauses, edits or removes the job opts out: a marker in default's
  state records RepoKit's creation, and RepoKit never recreates or rewrites a
  job the owner changed.
- `verify` reports the job as active, paused, owner-modified or absent.
  `repokit remove` removes it with the deployment.
- Default is also granted `goals.max_turns: 100` at creation or reset, so a
  chat `/goal` does not expire after 20 turns.

## Testing

- Table tests for the script's decisions: busy conditions, each idle bucket,
  snoozes, digest stability, subscriptions added once.
- Go unit tests for install, upgrade, owner opt-out, verify states and remove.
- Live proof on the dogfood deployment: a goal reaches a completed card,
  idle back-off suppresses repeat wakes, a triage card is recovered, and a
  busy board produces no model run.

## Out of scope

Webhooks (they need an owner-exposed endpoint and secret), work not tied to a
goal card, and any change to Hermes itself.

# Single-profile RepoKit

Date: 2026-10-03. Status: approved by the owner in conversation.

## Intent

RepoKit installs one Hermes profile, `default`, per repository and makes it
excellent: it talks with the owner, researches through read-only subagents,
implements Kanban cards and verifies each in a separate fresh run, and keeps
the owner's decisions in memory. The seven-profile team is retired, its code
deleted, and live deployments migrate on their next install.

Success, in the owner's words: quality, throughput, autonomy and simplicity.
Owner constraint: do it fast; no new tests for now. The existing suite stays
green (tests for deleted code are deleted with it); proof is live on
hermes-wing before the rollout to every deployment.

## Evidence behind it

- Live data, ~45 cards: tester (fresh-context verifier) requested changes on
  ~27% of cards, nearly all real bugs; reviewer after tester caught one
  documentation nit in 36 runs at 11% of worker time and 15% of worker tokens;
  researcher/planner ran rarely and serialized behind the one-card board; a
  tool one profile installed was invisible to another; busy profiles kept
  stale SOULs; owner context was lost whenever a chat was freshened.
- Research: one writer with read-only helpers (Cognition 2025/2026,
  Anthropic, LangChain); multi-agent variants 39-70% worse on sequential
  tasks (arXiv 2512.08296); verification works when fresh-context and
  execution-grounded (Devin Review, Anthropic harness design, Self-Debug);
  identical reviewers add little (correlated errors, Apple 2026); agents game
  tests (ImpossibleBench); memory should be small, itemized, owner-sourced
  (ACE, Letta, poisoning studies).
- Hermes 0.21.5 facts: a card run is a one-shot `chat -q` process with the
  profile's CLI toolsets; same-profile same-card review is supported and the
  review lane force-loads `sdlc-review`; one-shot runs may spawn at most
  `delegation.oneshot_max_children` (default 2) subagents in total, run
  synchronously; subagents die with the gateway process; memory is a
  session-start snapshot and card runs effectively never write it;
  `session_search` hides card-run sessions; checkpoints are off by default.

## Design

### 1. One profile

`default` is the whole team, with one SOUL assembled from: the runtime
contract, the repository identity header, channel and self-maintenance rules,
the shared contract (`common.md`), the single-profile role (`single.md`) and
the owner-conversation rules (`owner.md`). There is no roster, no steward and
no team shape option. Specialization comes from skills.

### 2. Work flow

- **Conversation.** Answers, inspects, decides. Research, comparisons and
  plans fan out to parallel background subagents with a brief each:
  objective, output format and size cap, allowed tools, boundaries. Subagents
  never change the repository; only `default` writes, and only through cards
  for anything beyond a small owner-asked edit.
- **Cards.** Every repository change is a card assigned to `default` with
  `workspace_kind: dir`, `workspace_path: /workspace`; one card runs at a time
  (one writer on one checkout).
- **Implementation run.** Test-first; evidence as command output; decisions
  and rejected approaches recorded on the card; `kanban_request_review` with
  `reviewer="default"`. Never completes its own card.
- **Verification run.** Fresh session; sees the card, the change and the
  repository, never the implementer's reasoning. Fixed rubric, in severity
  order: every acceptance criterion met with command evidence; no test
  deleted, weakened or special-cased; nothing outside scope changed; new
  probes find no regression. Only correctness and requirement gaps block;
  style is advisory. Risky changes (concurrency, auth/security, migrations,
  parsers, large diffs) fan out up to four subagents with distinct jobs
  (property tests, mutation of the new tests, end-to-end path, security).
  After two change-request rounds on a card, report to the owner instead of
  looping.

### 3. Memory

Hermes built-in memory. Entries are itemized lines:
`type (decision|preference|convention|gotcha): text (source, date)`. Only
the owner's own messages create entries, never repository files, issues or
web pages. Edit one entry at a time; a conflicting entry replaces the old
one. No task state, card ids or anything git or the board records. Card runs
read the snapshot and do not write. A reply that changes memory says so in
one line.

### 4. Hermes configuration RepoKit grants

One-time grants at creation, reset or migration; the owner may change any:

| Key | Value | Why |
| --- | --- | --- |
| `checkpoints.enabled` | `true` | `/rollback` recovery for writes |
| `delegation.oneshot_max_children` | `4` | verification-run fan-out |
| `agent.max_turns` | `0` | card runs use the whole card |
| `agent.reasoning_effort` | `high` | default now does the work |
| `kanban.dispatch_profiles` | `["default"]` | the only profile |

Kept: approvals off, Exa keyless search, 10 s dispatch tick, 100 goal turns,
the coordinator toolset with memory and delegation on every human channel.
Skills granted to `default`: the union of the old role grants that still
serve one agent (ast-grep, rest-graphql-debug, subagent-driven-development,
agent-merge-conflict-arbiter, adversarial-ux-test, decision-questionnaire,
dynamic-workflow, domain-intel, code-wiki, duckduckgo-search, grill-me).

### 5. Owner communication

Unchanged rules from `owner.md`: plain language, `[SILENT]` on review
handoffs, short reports on completion, block or decision, exact steps when
the owner must act.

### 6. RepoKit code

Delete: the six worker roles and their SOULs (`researcher`, `planner`,
`executor`, `tester`, `reviewer`, `steward`, `default.md`), per-role
toolset/skill tables, seven-profile runtime/review text, the `--team` flag and
`.hermes/repokit-team`, seven-profile review evidence and per-role verify
probes. `--reset-profile <role>` becomes `--reset-profile default` only
(kept as a flag name for compatibility). `verify` checks one profile and
accepts "a default run requested review, a separate default run completed
it" as review evidence.

### 7. Migration

On `install` (team step) for a deployment that still has any of the six old
profiles: reassign its open (not running) cards to `default`; export the
profile to `.hermes/backups/` with `hermes profile export -o`; delete it with
`hermes profile delete -y`; rewrite the managed dispatch policy to
`dispatch_profiles: ["default"]`. A profile with a running card is left for
the next install and reported. The old seven-profile managed policy is
recognized as RepoKit's (not owner drift) so it can be rewritten.

### 8. Proof and rollout

Ship to hermes-wing (already on the single trial), run real cards, check
`verify`, then roll out to every live deployment.

## Out of scope

More than one card at a time; worktree-per-card parallel writers; external
memory providers; Hermes's `/review` engine (background, not durable).

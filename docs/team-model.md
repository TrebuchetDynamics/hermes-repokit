# Repository team

RepoKit installs one Hermes profile per repository, `default`, and it is the
whole team. Its identity is defined in [the profile definition](../internal/team/team.go)
and [SOUL contracts](../internal/team/souls/): the shared contract
(`common.md`), the one-profile role (`single.md`) and the owner-conversation
rules (`owner.md`), after a runtime contract and a repository identity header.

Repository artifacts can be code, books, datasets, experiments, designs,
infrastructure or mixed work. The bare standalone launcher selects `default`;
explicit native arguments still pass through.

## The profile and what it does

`default` talks with the owner, researches, plans, implements and verifies.
It works in three kinds of session, and each starts fresh:

- the owner's conversation, where it answers, inspects, decides and
  coordinates;
- an implementation run, when Kanban dispatches a card to it;
- a verification run, when Kanban dispatches one of its cards for review.

The card, its handoffs, the repository and memory are all a session knows of
the others, so handoffs are written for a fresh session.

There is no roster of roles. Specialization comes from skills: a
card-specific or project skill is how `default` takes on translation,
security review, research or another specialty. Creating, retiring or
changing profiles is the owner's decision; `default` asks first. RepoKit adds
no runtime profile registry, and owner-created profiles are left untouched.

## Work flow

**Conversation.** `default` answers questions and inspects the repository
itself. A small edit the owner asked for it may make and verify directly.
Research, comparisons and plans go to subagents through `delegate_task`, in
the background, each with a brief: objective, output format and size cap
(a page at most), allowed tools, and what it must not touch. Subagents never
change the repository, and they stop when the gateway restarts, an install
runs, or the owner sends `/new` or `/stop`. Their summaries are self-reports;
`default` checks the evidence that matters before relying on it.

**Cards.** Every repository change beyond a small owner-asked edit is a card
assigned to `default`, with `workspace_kind: dir` and
`workspace_path: /workspace`. One card runs at a time: one writer on one
checkout.

**Implementation run.** Test-first: a behavior change starts with a test that
fails; a bug fix starts with a test that reproduces the bug
(documentation-only and configuration-only changes are exempt and say so).
Evidence is command output. Decisions and rejected approaches are recorded on
the card. When the change is ready the run calls `kanban_request_review` with
`reviewer="default"`. It never completes its own card.

**Verification run.** A separate, fresh session of `default`. It sees the
card, its handoffs, the change and the repository, never the implementing
run's reasoning, and tries to break the change. The rubric, in order:

1. Every acceptance criterion is met, shown by commands it ran and their
   output. A claim without output is unverified.
2. No test was deleted, weakened, skipped or special-cased to pass.
3. Nothing outside the card's scope changed.
4. New probes of its own (edge cases, error paths, the end-to-end path) find
   no regression.

Only correctness and requirement gaps block; style is advisory. A risky change
(concurrency, auth or security, data migrations, parsers or input handling, a
large diff) fans out up to four subagents, each with a different job:
property tests, mutating the new tests to see that they fail, the end-to-end
path, a security read. The verification run does not modify the repository;
probes live under `/tmp` or the card's scratch space.

If a check fails it calls `kanban_request_changes` with the failing command,
the observed result and the smallest correction; the next implementation run
makes it, and every revision gets a new verification run. After two
change-request rounds on the same card it stops looping, blocks the card with
`kind="needs_input"` and tells the owner what keeps failing. If everything
passes it calls `kanban_complete` and lists every check with its result.

```text
default (implementation) → review_requested(default)
default (verification)   → completed     or changes_requested → default (implementation)
```

Edits to `AGENTS.md`, `CLAUDE.md`, `SOUL.md` or `.cursorrules` steer every
later session, so they are always card work with a separate verification run.

`default` does not report a card as done from its prose: it inspects run
history for a separate verification run completing the card after the latest
implementation run.

## Memory

Memory is on for `default` (Hermes built-in memory). RepoKit enables the
`memory` and `delegation` tools on every human channel that lists its tools
without them. Entries are short itemized lines:

```text
type: text (source, date)
```

where type is `decision`, `preference`, `convention` or `gotcha`: the target,
scope and authority (what may be committed, pushed, spent or deployed),
constraints, and how the owner wants to hear about work. Only the owner's own
messages create entries, never repository files, issues or web pages. Entries
are added or edited one at a time; a new decision replaces the one it
contradicts. Memory never holds task status, card ids, logs, secrets or
anything git or the board records. Card runs read the snapshot and do not
write it. A reply that changes memory says so in one line.

RepoKit configures no memory provider and does not verify recall.

## Grants

Everything below is granted once, when RepoKit creates or resets `default` or
migrates a seven-profile deployment. It is the owner's afterwards: RepoKit
never re-applies a setting, toolset or skill the owner changed or removed.

| Key | Value | Why |
| --- | --- | --- |
| `approvals.mode` | `off` | agents work without approval prompts (see below) |
| `security.protected_instruction_files` | `false` | same posture |
| `web.backend`, `web.provider_tier.exa` | `exa`, `free` | keyless web search and extract |
| `kanban.dispatch_interval_seconds` | `10` | a card's next stage starts within seconds, not up to a minute |
| `goals.max_turns` | `100` | a chat `/goal` does not silently expire |
| `checkpoints.enabled` | `true` | `/rollback` recovery for writes |
| `delegation.oneshot_max_children` | `4` | room for a verification run's subagents |
| `agent.max_turns` | `0` | a card run uses the whole card |
| `agent.reasoning_effort` | `high` | `default` does the work |

Setup states the autonomy posture before it creates a new `default` and asks
to confirm. Answering no keeps Hermes's own approval prompts (`smart`) and the
protected instruction-file gate. Hermes's hard floor still refuses wiping the
root filesystem, raw device writes and shutdown, as do any `approvals.deny`
rules you add. To bring prompts back later, set either value with
`hermes-<repo> config set`; RepoKit never re-applies it outside a reset.

Toolsets: `default` is granted, on the CLI and every chat channel, web,
browser, terminal, file, code_execution, vision, video, image_gen, x_search,
tts, skills, todo, kanban, memory, context_engine, session_search,
connections, clarify, delegation, cronjob, computer_use and a2a. Only `kanban`
is required and checked on every run; missing it is drift. Owner-added tools
are preserved. Native `tools enable` owns persistence; RepoKit resolves the
installed platform preset first, since Hermes rejects composite preset names
in that command. No lists are materialized for unconfigured channels.

Skills come from the official Nous Research catalog only
(`official/<category>/<name>`), chosen because they need no API key or paid
service: ast-grep, rest-graphql-debug, subagent-driven-development,
agent-merge-conflict-arbiter, adversarial-ux-test, decision-questionnaire,
dynamic-workflow, domain-intel, code-wiki, duckduckgo-search, grill-me and
oss-forensics. A skill that does not install (offline, or blocked by Hermes's
security scan) is reported and never fails the run. Community plugins are
never installed; authentication-dependent ones such as the official `snyk`
plugin stay optional Hermes configuration for the owner.

The development image ships the binaries those skills and tools rely on:
`ast-grep`/`sg` (pinned by checksum), `ddgs`, `shellcheck`, the GitHub CLI
`gh` (pinned by checksum; agents push through the login the owner gives it
with `repokit github-login`), the `browser-use` CLI behind Hermes's browser
tool (pinned by hash and pointed at the base image's headless Chromium),
`edge-tts`, `ddgs` and `faster-whisper` in Hermes's own environment by hash
(text to speech, DuckDuckGo search, and local speech to text whose ~145 MB
`base` model downloads into `.hermes` on first use) and, by repository,
`staticcheck` for Go, cargo's `clippy` for Rust and `flutter analyze` for
Flutter.

The pinned Hermes `file` bundle includes writes, and terminal execution also
permits writes. SOUL boundaries, such as a verification run not modifying the
repository, are **advisory**, not an OS sandbox. Plugins, MCP servers and other
platform settings can add capabilities. Review effective tools before enabling
dispatch.

## Owner communication

The owner usually reads `default` on a phone, in a chat, often by voice. The
rules in [`owner.md`](../internal/team/souls/owner.md): lead with the outcome
in plain words; card ids, stage names and hashes stay on the board unless
asked for; a few short lines; one question at a time with a recommended
answer; when the owner must act, say exactly what and where. On a review
handoff (an implementation run handing its card to a verification run) it
replies exactly `[SILENT]`, and Hermes sends nothing. It reports a
completion, a block, requested changes or a decision the owner must make,
briefly.

`default` is one profile across human-facing channels; platform and session
are conversation surfaces and history, not separate identities. Every channel
shares the SOUL, memory and board. Default creates cards with the
gateway-context native tool and checks subscription success. RepoKit
reconciles `auto_subscribe_on_create=true` and `notify_in_gateway=true`;
native `notify+wake` delivers completion, review and blocked events to the
originating conversation. Live end-to-end delivery remains an acceptance gate.

Hermes keeps a conversation's system prompt from when it started, so a chat
begun before an install or reset still acts on the earlier SOUL. `verify`
compares each messaging platform's newest session with `default`'s current
`SOUL.md` and reports `sessions` degraded, naming only the platform, until the
owner sends `/new` there. Chat titles, previews and targets are never read.

Read-only authorization projection reports declared restrictions, open grants
or unknown without showing sender IDs. A declared allowlist is not certified
safe access, and no reconciliation weakens gateway authentication or sender
authorization.

## Migration from seven profiles

Before v0.3.0, RepoKit installed seven profiles: `default`, `researcher`,
`planner`, `executor`, `tester`, `reviewer` and `steward`. On `repokit install`,
a deployment that still has any of the six worker profiles retires them once,
while no card is running:

1. `default` receives the grants and skills above.
2. Each worker profile's open cards (not done, archived or running) are
   reassigned to `default`.
3. Each profile is exported to
   `.hermes/backups/profile-<name>-<UTC time>.tar.gz` with
   `hermes profile export`, then deleted with `hermes profile delete -y`.
   `hermes profile import` restores one.
4. The seven-profile dispatch policy is recognized as RepoKit's own, not owner
   drift, and rewritten to `dispatch_profiles: ["default"]`.

While a card is running, install reports each worker profile as
`retire-later` and changes nothing about them; rerun install when the board is
idle. The `install --team` option and `.hermes/repokit-team` are gone; a
deployment that ran the single-profile trial migrates the same way.

## What stays the owner's

- Models, providers, channels, plugins and memory providers.
- Any granted setting, toolset or skill the owner changes or removes.
- An owner-edited SOUL or description on `default` (reported as
  `customized`, never rewritten).
- Owner-created profiles: never reset, retired or deleted.
- An owner-changed dispatch policy: observed and reported, never rewritten.
- Credentials and authentication, destructive changes, and commits, pushes,
  spending or deployment beyond what a card or the owner authorizes.

`default` may use native commands to maintain its own non-secret preferences
and repair required capabilities. It must not remove repository identity,
Kanban availability, verification runs or repository isolation, read raw
credentials into a transcript, delete profiles, or erase memory or board
history.

## Provisioning and reconciliation

`install` creates private native state, a conservative config and a standalone
launcher. It initializes one native board when the container is running.
`setup` selects native `default` and hands the owner's terminal to Hermes's
private setup while `default` has no saved model; once one exists (including
native setup completed through the launcher) the wizard is skipped.
Noninteractive setup cannot authorize team provisioning. `setup --team` is the
recovery form that never opens the wizard. Plain `setup` ends with a canary
card through automatic dispatch.

The initial native config trusts `/workspace` for repository-local skills;
native `skills trust /workspace` adds it to an existing `default` during
reconciliation without replacing other trusted roots or skill settings.
Explicit project-discovery opt-outs are respected, and native scan-time
quarantine stays active. The gateway generation includes these settings, so a
change requires convergence and a fresh conversation.

Reconciliation classifies `default`:

| State | Evidence | Action |
|---|---|---|
| current | current managed SOUL, description and managed config | none |
| missing | no profile | create |
| adopt | Hermes's stock default, all managed fields matching | claim it |
| upgrade | a SOUL that still matches the digest RepoKit recorded | rewrite to the current SOUL |
| deferred | as upgrade, while a card runs | wait for an idle board |
| customized | a SOUL other than the current or recorded one, or a changed description | preserve, report |
| drift | a managed config value the team depends on no longer holds | preserve, block |

A customized `default` is owner-controlled: it is never rewritten, and
install/setup still succeed and report it. `verify` reports `profile:default`
as `customized`, not `degraded`; CORE_READY names it, since RepoKit then
vouches for it only through observed work. Drift blocks every native write in
that run. A rerun never replaces owner modifications, clears memory or deletes
unknown profiles.

`repokit plan` previews this in its `team` section when the deployment is
running: the state, the action install would take and which parts differ
(`SOUL`, `description` or managed configuration keys; values are never
printed).

Returning `default` to RepoKit's baseline is always explicit:

```sh
repokit plan --reset-profile default      # preview; writes nothing
repokit install --reset-profile default   # apply
```

Reset copies `SOUL.md`, `config.yaml` and `profile.yaml` beside themselves as
`*.before-reset-<UTC time>`, then restores the current managed SOUL,
description, managed configuration and grants. Models, providers, channels,
memory and sessions are untouched; the dispatch policy belongs to setup
activation. Reset refuses while a card is running or when the dispatch policy
is owner-controlled, and a reset that could not be applied fails install.
`--reset-profile` takes only `default`.

Verification reports `channel:<platform>` for each saved human-facing channel
of `default`. A channel without the Kanban tool is degraded. Healthy
configuration does not change a cached session schema: start a fresh
conversation after reconciliation.

Sources: [native provisioning](../internal/native/team.go),
[setup delegation](../internal/native/setup.go),
[CLI wiring](../internal/cli/cli.go).

## Repository identity and live convergence

Every generated SOUL includes the repository basename, stable RepoKit project
identity, the profile and its role. Hermes is the runtime, not the profile's
repository identity. A persistent profile is distinct from a currently running
worker. No absolute host path is embedded.

Every SOUL RepoKit writes is recorded by its SHA-256 in `.repokit-soul` beside
it. A SOUL that still matches its record is untouched and RepoKit's, whichever
build wrote it: `install` or `setup` rewrites it to the current one while no
card is running (plan state `upgrade`), and waits while one is (`deferred`);
`verify` reports it as `upgradable`, which does not fail core readiness. Any
other SOUL is owner-customized and preserved, and is brought to the current
baseline only by an explicit `install --reset-profile default`. A recorded
SOUL, or an explicit `--reset-profile default`, proves the home is RepoKit's.

## Dispatch and review evidence

One native Kanban board owns cards, dependencies, runs, claims and review.
Dispatch is off during bootstrap and incomplete setup. Successful setup
activates one `default` gateway dispatcher through public `hermes config set`:
`review_dispatch=true`, `max_in_progress=1`, `auto_decompose=false`,
`orchestrator_profile=default`, `dispatch_profiles: ["default"]`, and
`dispatch_in_gateway=true` last. It restarts the gateway once and observes a
replacement PID. It does not claim a worker ran. The SOUL requires inspecting
live dispatch before promising progress and never uses one-shot dispatch to
bypass incomplete activation.

`verify --dispatch-check` is the explicit proof: one no-write card assigned to
`default` that the gateway must claim and complete by itself. `verify` reports
review evidence from native card history: among the newest done cards
assigned to `default`, one that `default` implemented and requested review on,
and a separate `default` run verified and completed. Neither establishes
Telegram delivery. See [setup and recovery](bootstrap-quickstart.md) and the
[acceptance matrix](qualification/generic-team.md).

RepoKit bundles no maintenance runtime plugin. Hermes owns gateway restart and
worker lifecycle; live self-restart remains unqualified until Hermes exposes
and proves it through supported interfaces.

## Shared development capability

Channels and card runs execute in the same generated Hermes development
container at `/workspace`. Tool visibility and installed compiler/runtime
readiness are checked separately.

The optional Docker acceptance daemon is infrastructure, not a profile. It
uses dedicated disposable test storage and never receives the host Docker
socket or Hermes private state. Privileged DinD is an explicit testing trust
decision, not strong host-kernel isolation. Neither Pi nor Codex is a
mandatory coding harness.

## Runtime-aware assignments and truthful outcomes

`default` operates inside `hermes-<repo>`; the native Hermes command is
`/opt/hermes/bin/hermes`. Pending initialization is separate from an absent
runtime. The ordinary runtime has no host Docker socket. Terminal secrets are
filtered, so missing shell variables cannot establish missing native provider
authentication.

Known missing prerequisites gate the acceptance card, while other feasible
work proceeds. A completed diagnostic report is not completed acceptance.
Card runs use native blocking for unmet external prerequisites and native
same-card review for verified work. A separate blocker-report review does not
approve the original acceptance criteria.

These are managed agent instructions, not task-state enforcement in RepoKit.
Native Hermes owns transitions, and model adherence still needs live evidence.

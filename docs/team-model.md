# Universal repository team

RepoKit scaffolds a small repository organization, not a collection of
technology-specific bots. Its seven permanent identities are defined in
[the roster](../internal/team/team.go) and [SOUL contracts](../internal/team/souls/).

| Profile | Responsibility |
| --- | --- |
| default | Primary user-facing assistant, coordinator, orchestrator and decision owner |
| researcher | Resolve unknowns and gather evidence |
| planner | Define a bounded execution contract |
| executor | Produce the requested artifact or change |
| tester | Prove the change's behavior without modifying the repository |
| reviewer | Decide whether the verified change is accepted |
| steward | Maintain profile identities and capabilities |

Normally the user talks to `default`. It answers lightweight questions and
handles discussion directly. It uses Kanban for bounded work when useful,
without requiring ceremonial research/planning stages. Repository artifacts
can be code, books, datasets, experiments, designs, infrastructure or mixed work.
The bare standalone launcher explicitly selects `default`; explicit native
arguments still pass through, including `-p steward` for deliberate direct use.

The permanent profile answers **who is responsible**. The repository context,
card, acceptance criteria and task skills answer **what expertise is required**
and **what success means**. Default owns project priorities and orchestration;
steward owns team changes and reports back to default.

## Team evolution

Skills first: an existing profile plus a card-specific skill is preferred for
one-off translation, security review, research or another specialization.
Steward creates a persistent specialist only for recurring work, a distinct
identity, model/provider or capability boundary, long-lived skills, independent
responsibility, or an explicit request for a dedicated role.

Steward uses native Hermes profile inventory, creation, description, export,
distribution updates and deletion. RepoKit adds no runtime profile registry.
New specialists use the qualified native config clone, replace inherited SOUL,
clear only newly copied curated memories, and receive their own description.
Native cloning preserves the supported provider/model baseline; RepoKit does
not manually copy authentication stores or use `--clone-all`/`--clone-channels`.
Provider mechanisms outside native config cloning require native verification;
a copied `.env` fixture is not proof of every OAuth/provider path.

Retirement is the default: stop new assignments in coordination with default,
mark routing metadata retired, and preserve history. Deletion requires explicit
user authorization for the specific profile, inspection, appropriate backup,
and confirmation that no active task depends on it. Never delete `default`.
These rules live in [steward's SOUL](../internal/team/souls/steward.md).

## Provisioning and ownership

`install` creates private native state, a conservative config and a standalone
launcher. It initializes one native board when the existing container is running.
`setup` explicitly selects native `default` and inherits the user's terminal for
private setup. Noninteractive setup cannot authorize team provisioning. After
interactive setup returns successfully and a saved model exists, RepoKit
provisions missing specialists through native config cloning. Plain `setup` then
runs operational activation;
`setup --team` provisions missing specialists after a saved default model is
available, without reopening private setup. This also supports users who completed
native setup through the standalone launcher. Memory setup is user-managed and is
not part of RepoKit's setup stages.

The initial native config trusts `/workspace` for repository-local skills. New
profiles inherit that trust; native `skills trust /workspace` adds it to existing
managed profiles during reconciliation without replacing other trusted roots or
skill settings. Explicit project-discovery opt-outs remain respected, and native
scan-time quarantine remains active. The gateway generation includes project
trust/discovery settings so changes require convergence and a fresh conversation.

Native profile creation uses the final role name because Hermes also registers
profile services/routing. Immediately replace a new clone's identity and clear
its copied `memories/MEMORY.md` and `memories/USER.md`; then configure it and
verify native list/show. On interruption, preserve partial native profiles and
report drift. Do not erase them or re-clear their memory on retry.

Default's pristine upstream SOUL is adopted only when all managed config fields
already match RepoKit defaults and existing description is absent or expected.
Default uses the installed Hermes platform preset's resolved categories plus
explicit Kanban and memory on configured interactive channels. Native `tools
enable` owns persistence and built-in/plugin bookkeeping; existing extra tools
are preserved. Hermes rejects composite preset names in that command, so RepoKit
resolves the native preset first instead of passing an ignored alias. No lists
are materialized for unconfigured channels or reduced programmatic surfaces.
The legacy profile-wide Kanban fallback remains; it is not a memory-provider
readiness claim. Native enablement repairs required disabled categories.
New specialist clones have inherited human-channel selections narrowed to their
role subsets; their
task-scoped lifecycle tools remain native dispatcher behavior. Existing worker
platform Kanban opt-ins are reported as drift.

Other conflicting fields are preserved and reported as drift. Existing role SOUL,
description and managed fields must match for a clean rerun; a rerun
never replaces owner modifications, clears learned memories or deletes unknown
profiles. Existing `builder` or domain profiles are left untouched.

Verification reports `kanban:default:<platform>`, `memory:default:<platform>` and
the profile-wide Kanban fallback, plus a declared channel routing/core matrix.
A missing saved-platform opt-in is degraded. Healthy configuration does not
change a cached session schema: start a fresh conversation after reconciliation.
Setup starts or converges the native gateway only after its activation gates pass;
saving Telegram credentials alone does not establish operational readiness.

Sources: [native provisioning](../internal/native/team.py),
[setup delegation](../internal/native/setup.go),
[CLI wiring](../internal/cli/cli.go).

## Work and review

One shared native Kanban board owns cards, dependencies, runs, claims and review.
Bootstrap defaults are dispatch off, automatic decomposition off, orchestrator `default`,
and `max_in_progress: 1`. The coordinator explicitly receives `kanban`.
Specialists receive native worker lifecycle tools at dispatch.

Verification and review stay on one card, in two native review stages:

```text
executor → review_requested(tester)
tester   → review_requested(reviewer)      or changes_requested → executor
reviewer → completed                       or changes_requested → tester (relay)
```

Tester asks "does this demonstrably work?" and reviewer asks "should this be
accepted?". Tester runs the project's checks and probes edge cases but never
edits the repository; a missing regression test is a change request, and the
executor writes it. Every revision passes tester again, because a code change
invalidates earlier test evidence.

Hermes returns requested changes to whichever profile last requested review.
After tester forwards a card, reviewer's `request_changes` therefore lands on
tester. Tester relays it unchanged with `kanban_request_review(reviewer=
"executor")`; executor fixes it and hands it back to tester. A full rejection
cycle is:

```text
reviewer → changes_requested     (card lands on tester)
tester   → review_requested(executor)   relay, no edits
executor → review_requested(tester)     fix
tester   → review_requested(reviewer)   re-verify
reviewer → completed
```

Nobody accepts their own work: implementer, tester and reviewer are distinct.

The [credential-free fixture](../tests/acceptance/fixtures/team_lifecycle.py)
exercises native transitions in separate profile processes. It does not prove
model-driven dispatch, artifact correctness or adversarial actor isolation.

## Capabilities and limitations

| Profile | Selected CLI toolsets | Boundary |
| --- | --- | --- |
| default | Native CLI preset + kanban + memory | Diagnosis and own non-secret maintenance allowed; artifact implementation delegated |
| researcher | file, web, memory | Artifact writes prohibited by SOUL |
| planner | file, memory | Implementation prohibited by SOUL |
| executor | file, terminal, code_execution, skills, memory | Work limited to the card |
| tester | terminal, memory | No file-editing toolset; repository writes prohibited by SOUL |
| reviewer | file, terminal, memory | Artifact writes prohibited by SOUL; terminal permits verification |
| steward | terminal, file, memory | Profile administration only; project writes prohibited by SOUL |

The pinned Hermes `file` bundle includes reads and writes; terminal execution
also permits writes. These specialist boundaries are **advisory**, not an OS
sandbox. Profile plugins, MCP servers, explicit tool arguments and other
platform settings can add capabilities; CLI bundle selection is not proof of
complete capability isolation. Review effective tools before enabling dispatch.

`default` is one profile across human-facing channels; platform and session are
conversation surfaces/history, not new team identities. All primary channels
share repository SOUL, roster and board. Memory is user-managed and is not part
of RepoKit's team identity model.
Routing to another profile is reported separately. Saved core selections do not
prove credentials, connected adapters or a fresh session's loaded tools.

Default can use native commands to maintain its own non-secret preferences and
repair required capabilities. Credentials, authentication, destructive changes,
and review independence are outside that authority.
Steward still owns specialist lifecycle. These are advisory SOUL boundaries.

RepoKit no longer bundles a maintenance runtime plugin. Hermes owns gateway
restart and worker lifecycle. RepoKit may invoke the public native commands
during setup, but live self-restart remains unqualified until Hermes exposes
and proves the required behavior through supported interfaces.

Messaging is a remote development interface. Default may directly inspect files,
search Git/repository state and run diagnostics; substantive changes use Kanban,
executor and distinct same-card tester and reviewer. Once setup passes its operational gates,
the default gateway automatically claims assigned work and review; users need no
SSH or manual dispatch. Default creates cards with the gateway-context native
tool and checks subscription success. RepoKit reconciles
`auto_subscribe_on_create=true` and `notify_in_gateway=true`; native `notify+wake`
delivers completion, review and blocked events to the originating conversation.
Live end-to-end delivery remains an acceptance gate.

Read-only authorization projection reports declared restrictions, open grants or
unknown without showing sender IDs. Effective access still depends on environment,
pairing and native policy, so a declared allowlist is not certified safe access.
An outbound home-channel destination is not inbound profile-routing evidence.
No reconciliation weakens gateway authentication or sender authorization.

## Identity and shared memory

SOUL defines identity and Kanban holds work state. Memory is user-managed: the
operator chooses and configures any memory provider the repository needs, and
RepoKit neither links nor certifies it. Built-in local memory remains a native
Hermes concern.

`verify` reports scaffold readiness and native integration configuration, but it
does not configure or verify a memory provider. Review remains `unqualified`
until actual same-card work is accepted. See
[acceptance matrix](qualification/generic-team.md).

Source: [read-only integration probes](../internal/verify/integrations.go).

## Repository identity and live convergence

Every generated SOUL includes the repository basename, stable RepoKit project
identity, profile role, permanent seven-profile roster and relationship to default.
Hermes is the runtime, not the profile's repository identity. A persistent profile
is distinct from a currently running worker. No absolute host path is embedded.
Exact historical RepoKit SOULs upgrade only when managed configuration and role
description still match; owner edits remain drift and are not overwritten.
Historical SOULs are matched by profile name, never by roster position, so tester
starts with no managed history and no existing profile is remapped to it.
An activated six-profile team whose default still holds the exact six-profile
policy is upgraded by `setup --team`: tester is created, the six SOULs are
rewritten and activation widens the allowlist, restarting the gateway once when
no card is running. Any other owner-changed policy is only observed.

Dispatch is off during bootstrap and incomplete setup. Successful setup activates
one default gateway dispatcher with review dispatch enabled, concurrency one,
automatic decomposition disabled and the explicit seven-profile allowlist.
Memory is user-managed and does not block core activation.
Specialists keep dispatch disabled. SOUL distinguishes persistent profiles
from running workers and requires inspection of live dispatch before promising
progress; it never uses one-shot dispatch to bypass incomplete activation.

Setup configures this policy through public `hermes config set`, restarts the
gateway once, and observes a replacement PID. It does not claim a worker ran.
`verify --dispatch-check` is the explicit researcher proof, and `verify` reports
same-card executor→tester→reviewer evidence from native card history: the final
run is reviewer's completion and a tester hand-off follows the latest
implementation run. Neither
establishes Telegram delivery. See [setup and recovery](bootstrap-quickstart.md).

## Shared development capability

Interactive channels and workers execute in the same generated Hermes development
container at `/workspace`. Tool visibility and installed compiler/runtime readiness
are checked separately. Executor's native coding tools include file, terminal,
code_execution, skills and memory; tester and reviewer can run the same project
checks. Default may perform a tiny authorized direct edit, but substantive
artifact work, verification and review remain executor/tester/reviewer
responsibilities.

The optional Docker acceptance daemon is infrastructure, not a profile. It uses
dedicated disposable test storage and never receives the host Docker socket or
Hermes private state. Privileged DinD is an explicit testing trust decision, not
strong host-kernel isolation. Neither Pi nor Codex is a mandatory coding harness.

## Runtime-aware assignments and truthful outcomes

The managed team operates inside `hermes-<repo>`. The native Hermes command is
`/opt/hermes/bin/hermes`. Memory is user-managed and lives outside RepoKit's
runtime contract. Pending initialization is
separate from an absent runtime. The ordinary runtime has no host Docker socket;
local observation does not require one. Multi-deployment acceptance requires
its own authorized test environment. Terminal secrets are filtered, so missing
shell variables cannot establish missing native provider authentication.

Default supplies inspected Git/board facts to planner or routes inspection to a
profile with the required tools. Planner's file/memory toolsets are unchanged.
Known missing prerequisites should gate the acceptance card, while other feasible
work can proceed. A completed diagnostic report is not completed acceptance.
Workers use native blocking for unmet external prerequisites and native same-card
review for independently reviewed work. A separate blocker-report review does not
approve the original acceptance criteria.

These are managed agent instructions, not new task-state enforcement in RepoKit.
Native Hermes owns transitions. Reconciliation recognizes exact prior managed
SOULs and preserves custom edits; new behavior requires normal setup and fresh
sessions, and model adherence still needs live evidence.

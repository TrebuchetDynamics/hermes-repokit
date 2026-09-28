# Universal repository team

RepoKit scaffolds a small repository organization, not a collection of
technology-specific bots. Its six permanent identities are defined in
[the roster](../internal/team/team.go) and [SOUL contracts](../internal/team/souls/).

| Profile | Responsibility |
| --- | --- |
| default | Primary user-facing assistant, coordinator, orchestrator and decision owner |
| researcher | Resolve unknowns and gather evidence |
| planner | Define a bounded execution contract |
| executor | Produce the requested artifact or change |
| reviewer | Independently verify against the contract |
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
runs private OpenViking configuration/linking and operational activation;
`setup --team` provisions missing specialists after a saved default model is
available, without reopening private setup. This also supports users who completed
native setup through the standalone launcher. `setup --memory` resumes memory setup separately.

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

Review normally stays on one card:
`executor → reviewer → request_changes → executor → reviewer → done`.
The executor does not independently accept its own work; the reviewer checks
actual artifacts and evidence without implementing the requested change.

The [credential-free fixture](../tests/acceptance/fixtures/team_lifecycle.py)
exercises native transitions in separate profile processes. It does not prove
model-driven dispatch, artifact correctness or adversarial actor isolation.

## Capabilities and limitations

| Profile | Selected CLI toolsets | Boundary |
| --- | --- | --- |
| default | Native CLI preset + kanban + memory | Diagnosis and own non-secret maintenance allowed; artifact implementation delegated |
| researcher | file, web, memory | Artifact writes prohibited by SOUL |
| planner | file, memory | Implementation prohibited by SOUL |
| executor | file, terminal, memory | Work limited to the card |
| reviewer | file, terminal, memory | Artifact writes prohibited by SOUL; terminal permits verification |
| steward | terminal, file, memory | Profile administration only; project writes prohibited by SOUL |

The pinned Hermes `file` bundle includes reads and writes; terminal execution
also permits writes. These specialist boundaries are **advisory**, not an OS
sandbox. Profile plugins, MCP servers, explicit tool arguments and other
platform settings can add capabilities; CLI bundle selection is not proof of
complete capability isolation. Review effective tools before enabling dispatch.

`default` is one profile across human-facing channels; platform and session are
conversation surfaces/history, not new team identities. All primary channels
share repository SOUL, roster, board and OpenViking identity.
Routing to another profile is reported separately. Saved core selections do not
prove credentials, connected adapters or a fresh session's loaded tools.

Default can use native commands to maintain its own non-secret preferences and
repair required capabilities. Credentials, authentication, destructive changes,
and review independence are outside that authority.
Steward still owns specialist lifecycle. These are advisory SOUL boundaries.

The bundled `repokit_maintenance` native plugin brokers only graceful restart
requests and successor observations. Setup installs it through Hermes's scanner
and pinned local Git installer; scanner refusal and edited source stop setup.
It neither edits configuration nor invokes RepoKit. The serving default process
must be supervised. This older self-restart broker admits only dispatch-off mode;
with operational dispatch it reports deferred. Ready/running/review work also
defers this broker. Source setup uses its separate native board-lock fence and
graceful restart path for operational reconciliation. Native drain accounting does not include Kanban subprocesses;
the conservative admission snapshot is not atomic with concurrent external CLI
dispatch. Live self-restart remains unqualified until the Docker and originating
channel acceptance tests pass; a request acknowledgement is not success.

Messaging is a remote development interface. Default may directly inspect files,
search Git/repository state and run diagnostics; substantive changes use Kanban,
executor and distinct same-card reviewer. Once setup passes its operational gates,
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

SOUL defines identity. OpenViking holds durable project knowledge. Kanban holds
work state. The permanent team and future specialists should share one
repository OpenViking process inside Hermes and account `repokit`, the same repository user,
and endpoint `http://127.0.0.1:1933`, with no per-profile peer. Built-in local
memory stays enabled alongside native extraction. Check every environment,
YAML and linked-configuration override before activating the connection.

The installer embeds official OpenViking inside Hermes. Private `setup --memory`
checks the effective connection under every profile's native secret scope, requires
a normal repository user key and links the shared native connection across all six
roles. Future specialists cloned from default inherit that link; steward must check
their effective identity and peer overrides before use. Optional plugins remain
owner-managed native Hermes components.

`verify` reports scaffold readiness, native integration configuration and authenticated
memory health. Memory `active` means the shared identity matches, not that recall has
passed. Review remains `unqualified` until actual same-card work is accepted. See
[memory wiring](qualification/openviking-wiring.md), [memory qualification](qualification/generic-team-memory.md)
and [acceptance matrix](qualification/generic-team.md).

Source: [read-only integration probes](../internal/verify/integrations.go).

## Repository identity and live convergence

Every generated SOUL includes the repository basename, stable RepoKit project
identity, profile role, permanent six-profile roster and relationship to default.
Hermes is the runtime, not the profile's repository identity. A persistent profile
is distinct from a currently running worker. No absolute host path is embedded.
Exact historical RepoKit SOULs upgrade only when managed configuration and role
description still match; owner edits remain drift and are not overwritten.

Dispatch is off during bootstrap and incomplete setup. Successful setup activates
one default gateway dispatcher with review dispatch enabled, concurrency one,
automatic decomposition disabled and the explicit six-profile allowlist.
OpenViking authentication is a mandatory activation gate. Specialists keep dispatch disabled. SOUL distinguishes persistent profiles
from running workers and requires inspection of live dispatch before promising
progress; it never uses one-shot dispatch to bypass incomplete activation.

Activation checks replacement-process identity, native singleton-lock ownership,
startup settings and a real gateway-spawned researcher canary. The canary is a
no-write README-title task and is archived only after structured successful
researcher evidence. This does not establish Telegram delivery or independent
review acceptance. See [setup and recovery](bootstrap-quickstart.md).

Successful setup stages finish with one native gateway convergence operation.
A process-bound hash of managed inputs is diagnostic state only: runtime chat,
plugins, Kanban and memory never call RepoKit or read its generation receipt.
`verify` independently compares current files, process identity, native heartbeat,
loop liveness and previously observed adapters. It reports current, stale,
not-running or unknown. Missing/changed process evidence never passes from a
receipt alone. See [qualification and limits](qualification/gateway-convergence.md).

## Shared development capability

Interactive channels and workers execute in the same generated Hermes development
container at `/workspace`. Tool visibility and installed compiler/runtime readiness
are checked separately. Executor's native coding tools include file, terminal,
code_execution, skills and memory; reviewer can run the same project checks.
Default may perform a tiny authorized direct edit, but substantive artifact work
and independent review remain executor/reviewer responsibilities.

The optional Docker acceptance daemon is infrastructure, not a profile. It uses
dedicated disposable test storage and never receives the host Docker socket or
Hermes private state. Privileged DinD is an explicit testing trust decision, not
strong host-kernel isolation. Neither Pi nor Codex is a mandatory coding harness.

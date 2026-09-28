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
runs private OpenViking configuration/linking and native local Nerve activation;
`setup --memory` and `setup --supervision` resume those integration steps.

Native profile creation uses the final role name because Hermes also registers
profile services/routing. Immediately replace a new clone's identity and clear
its copied `memories/MEMORY.md` and `memories/USER.md`; then configure it and
verify native list/show. On interruption, preserve partial native profiles and
report drift. Do not erase them or re-clear their memory on retry.

Default's pristine upstream SOUL is adopted only when all managed config fields
already match RepoKit defaults and existing description is absent or expected.
Any conflicting field is preserved and reported as drift. Existing role SOUL,
description and managed fields must match exactly for a clean rerun; a rerun
never replaces owner modifications, clears learned memories or deletes unknown
profiles. Existing `builder` or domain profiles are left untouched.

Sources: [native provisioning](../internal/native/team.py),
[setup delegation](../internal/native/setup.go),
[CLI wiring](../internal/cli/cli.go).

## Work and review

One shared native Kanban board owns cards, dependencies, runs, claims and review.
Defaults are dispatch off, automatic decomposition off, orchestrator `default`,
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
| default | kanban, memory | No general file or terminal implementation tools selected |
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

## Identity, memory and supervision

SOUL defines identity. OpenViking holds durable project knowledge. Kanban holds
work state. The permanent team and future specialists should share one
repository OpenViking service and account `repokit`, the same repository user,
and endpoint `http://openviking:1933`, with no per-profile peer. Built-in local
memory stays enabled alongside native extraction. Check every environment,
YAML and linked-configuration override before activating the connection.

Nerve and Laya are infrastructure, never extra profiles. Nerve must be installed
disabled, configured for verified local Laya, and only then enabled. Hosted Jev
fallback is forbidden. Do not create a competing task-status database.

The installer scaffolds Hermes, official OpenViking and the pinned local Laya
build. Setup provisions the native roster. Installation enables no provider.
After private native setup,
`setup --memory` checks the effective connection under every profile's native
secret scope, requires a normal repository user key, and links the same private
native connection store across the six roles. Future specialists cloned from
default inherit that link; steward must check their effective identity and peer
overrides before use. Native Nerve is installed disabled, checked against the
pinned revision, configured for local Laya, then enabled across all six roles.
Fresh native hooks and `LOCAL_ONLY` decisions, offline failure, installer removal
and coordinated recreation passed in a real disposable stack. Future specialist
supervision and live memory acceptance still need qualification. See
[current runtime evidence](qualification/runtime-integrations.md), [wiring evidence](qualification/openviking-wiring.md),
[memory qualification](qualification/generic-team-memory.md)
and the [acceptance matrix](qualification/generic-team.md). Full team completion
and RepoKit self-dogfood remain gated on the live integration evidence.

`verify` reports scaffold readiness, six-profile native integration configuration,
authenticated memory health and Laya health separately. It does not load Nerve
or make decisions. Memory `active` means the shared repository user identity
matches; it does not establish recall. Review remains `unqualified` and keeps
the whole-command exit status nonzero. Read the component statuses; configuration
and fixture passes do not make this deployment's full acceptance complete.
Sources: [native supervision](../internal/native/supervision.py) and
[read-only integration probes](../internal/verify/integrations.go).

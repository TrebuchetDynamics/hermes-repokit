# Universal team acceptance — 2026-09-27

The current directive is six permanent profiles: default, researcher, planner,
executor, reviewer and steward. Earlier five-role and engineering-profile
records are historical. **The complete team acceptance gate is not passed.**
This qualification did not deploy RepoKit to itself or perform self-dogfood.
Git delivery is a separate, subsequently authorized operation.

## Executed native scaffold fixture

`REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerFoundation -v`
passed the six-profile fixture in 117.25 seconds against
`nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db`.
Subsequent changes require the final validation record below to supersede that run.

The test uses a disposable Git repository and a copied installer source/binary.
It supplies a synthetic nonsecret provider configuration to exercise native
cloning; it does not claim successful interactive provider login or model chat.
Every required profile resolves through native list/show, descriptions match,
SOULs match distinct role contracts, the `.env` sentinel is cloned through the
native mechanism, and default curated-memory sentinels do not reach specialists.
Fresh role memories survive reruns. Offline tests separately reproduce edited
SOUL/config/description preservation, unknown profiles, partial clone failure,
nonterminal setup, native string config semantics and default routing.

A separate Python process for each relevant role exercises the pinned native
Kanban API against the same board: default creates assigned cards; researcher
completes a parent with metadata; executor sees that metadata, submits review;
reviewer requests changes; executor resubmits; reviewer completes the same card.
Recorded run profiles are executor/reviewer/executor/reviewer. No worker model
is used, and supplied claim names are not an authenticated actor boundary.

After removing the copied installer source/binary, ordinary Compose restart and
forced recreation preserve the board and completed card/parent metadata. The
recreation probe waits for native entrypoint UID/ownership setup before accessing
the board. The test harness is an observer, not a generated runtime dependency.

## Gate matrix

| # | Requirement | Evidence/status |
| --- | --- | --- |
| 1 | All six profile names resolve | Passed in pinned native fixture |
| 2 | Expected descriptions | Passed through native describe |
| 3 | Distinct role SOULs | Passed byte comparisons |
| 4 | No inherited default curated memory | Passed sentinel absence; rerun preserves role memory |
| 5 | Every profile connects to shared OpenViking namespace | Resolver configuration tested; deployed connection pending |
| 6 | One role writes memory another recalls; B cannot see A | Pending real embedding/extraction model configuration and live services |
| 7 | One initialized shared board | Passed native default/profile path checks and persistence |
| 8 | Default creates assigned work | Passed native API fixture; conversational tool use/dispatch pending |
| 9 | Parent metadata reaches dependent role | Passed native worker-context construction in executor process |
| 10 | Same-card review/change/resubmit/done | Passed native state transitions; actual model-driven artifact work pending |
| 11 | Approving reviewer differs from executor | Passed recorded role/run identities; adversarial identity enforcement not claimed |
| 12 | Real local typed Laya decision | Passed [local fixture](generic-team-laya.md) |
| 13 | Nerve consumes Laya without hosted fallback | Passed native plugin invocation in offline fixture; all-six installation pending |
| 14 | Compose restart/recreation preserves board and memory | Board passed; OpenViking recall pending |
| 15 | Removing RepoKit preserves the whole deployment | Native roster/board fixture passed; full memory/supervision team gate pending |

## Outstanding configuration and implementation

The installer does not currently provision the OpenViking service or the
production Laya sidecar, nor install Nerve into every role. Public memory and
local-supervision contracts are emitted by `plan` as **not activated** proposals.
The separate inference fixture is a qualified experiment, not production wiring.

OpenViking needs actual embedding/extraction model endpoints, model identities,
matching vector dimensions and private native credentials where required.
No host credentials were borrowed. See [precise requirements](generic-team-memory.md).
The candidate image is v0.4.21 pinned by digest; it is not claimed fully qualified
from schema checks. Keep the selected provider inactive until live validation.

Nerve/Laya still need a self-contained reproducible sidecar image, production
Compose lifecycle integration, pinned installation/configuration for all six
profiles and dynamic specialists, and the full removal-first integration test.
The local-only enablement order and upstream outage behavior are documented in
[actual Laya evidence](generic-team-laya.md). No automatic hosted fallback exists
in the scaffold because it does not enable Nerve at all.

Capability boundaries for file/terminal-equipped profiles are advisory; see
[the capability matrix](../team-model.md). There is no separate profile registry,
task-status database, custom Nerve implementation or Laya protocol.

## Update triggers

Requalify when the Hermes image, native clone/setup/toolset behavior, roster,
SOULs, profile reconciliation, sidecar image/dependencies/model snapshot, memory
provider schema, or installer wiring changes. Keep credential-free/API fixture
claims separate from real agent work and complete-team acceptance.

## Final validation

The final six-profile Docker run passed in **120.56 seconds** after the explicit
default launcher selection and unknown integration-status reporting changes.
It asserts that `verify` exits nonzero for unestablished OpenViking/Nerve-Laya
acceptance while all native scaffold probes are healthy. Native lifecycle,
restart/recreation and copied-installer removal checks passed in the same run.

`go test ./...`, `go test -race ./...`, `go vet ./...`, `git diff --check`,
formatting and local documentation-link checks passed. A Linux arm64 build of
the CLI passed; this is a development build, not release qualification.
The six-label OpenViking resolver probes passed against the pinned images with
networking disabled. Actual local Laya qualification is recorded separately.

The subsequent Git delivery validation repeated unit/race tests, vet, formatting,
local links, and the pinned offline memory configuration probes successfully.
The full Docker acceptance package passed again; its native fixture completed
in 120.67 seconds. This does not promote the pending integration gates.

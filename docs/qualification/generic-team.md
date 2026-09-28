# Universal team acceptance — 2026-09-27

The current directive is six permanent profiles: default, researcher, planner,
executor, reviewer and steward. Earlier five-role and engineering-profile
records are historical. **The complete team acceptance gate is not passed.**
The current deployment/setup evidence is in the
[runtime integration record](runtime-integrations.md). The older scaffold runs
below are historical. This qualification did not deploy RepoKit to itself or perform self-dogfood.
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

| Requirement | Historical evidence or remaining gate |
| --- | --- |
| Six roles, descriptions and SOULs | Passed native fixture |
| Fresh curated memories and preserved rerun state | Passed sentinels |
| Shared board, handoffs and review transitions | Passed separate-process native API fixture |
| Shared memory connection | Native configuration/linking passed; live recall pending |
| Real executor/reviewer work and channel delivery | Pending model-driven acceptance |
| Current image and integrated removal-first operation | Pending live qualification |

## Outstanding configuration and implementation

OpenViking still needs actual embedding/extraction model endpoints, model
identities, matching vector dimensions and private native credentials where
required. No host credentials were borrowed. See
[precise requirements](generic-team-memory.md). Configuration and authenticated
health do not establish real recall or cross-repository isolation.

## Update triggers

Requalify when the Hermes image, native clone/setup/toolset behavior, roster,
SOULs, profile reconciliation, embedded runtime image/dependencies, memory
provider schema, or installer wiring changes. Keep credential-free/API fixture
claims separate from real agent work and complete-team acceptance.

## Historical scaffold validation

The final six-profile Docker run passed in **120.56 seconds** after the explicit
default launcher selection and unknown integration-status reporting changes.
It asserts that `verify` exits nonzero for unestablished OpenViking
acceptance while all native scaffold probes are healthy. Native lifecycle,
restart/recreation and copied-installer removal checks passed in the same run.

The subsequent Git delivery validation repeated unit/race tests, vet, formatting,
local links, and the pinned offline memory configuration probes successfully.
The full Docker acceptance package passed again; its native fixture completed
in 120.67 seconds. This does not promote the pending integration gates.

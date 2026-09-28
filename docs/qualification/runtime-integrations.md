# Runtime integration qualification — 2026-09-27

Status: **FOUNDATION READY / INTEGRATION PARTIAL**. Production wiring for
default installation and native integration setup is implemented. Full v1 acceptance, live main-model work,
memory recall/isolation and self-dogfood are not complete.

## Implemented deployment and setup

Ordinary `install` emits Hermes, the pinned official OpenViking service and a
local Laya build. The executable embeds the exact build inputs and publishes
them in `.hermes/laya-image`; no RepoKit executable or checkout is needed by
subsequent Compose builds or runtime. Model weights are checksum-verified and
baked into the offline image. Writable caches persist in `.hermes/laya`. Laya
shares Hermes's network namespace and exposes only loopback port 8765.
Explicit qualified local image IDs remain supported.

Plain `setup` follows this sequence:

1. Private native default provider/model setup and native six-role provisioning.
2. Native OpenViking init when configuration is absent, doctor, configuration
   validation, service restart and authenticated health readiness.
3. Native Hermes memory wizard and shared connection linking across all six
   profile scopes, requiring the normal repository user identity.
4. Pinned upstream Nerve admission while disabled, local configuration before
   enablement, fresh native hook loading and actual `LOCAL_ONLY` decisions for
   every role, followed by coordinated Hermes/Laya recreation, health readiness
   and repeated native checks. Supervision setup requires a quiescent board and
   disabled dispatch before this restart.

`setup --memory` and `setup --supervision` resume their respective integration
steps. Credentials remain in native setup/state. Owner edits and interrupted
native state are preserved for inspection.

Sources: [CLI sequencing](../../internal/cli/cli.go),
[installation](../../internal/cli/install.go),
[Compose](../../internal/compose/compose.go),
[memory setup](../../internal/native/memory.go),
[native Nerve setup](../../internal/native/supervision.py), and
[packaging](../../packaging/laya/README.md).

## Executed evidence

| Check | Observed result | What it establishes |
| --- | --- | --- |
| `TestDockerFoundation` with native memory-link fixture | Passed, approximately 163 seconds | Real six-role cloning, native Kanban handoffs/review transitions, shared connection resolution and state preservation; no live memory service or model call |
| `TestDockerOpenVikingPending` | Passed, approximately 11 seconds | Official pending-config behavior, persistent service paths and recreation |
| Generated default Compose build | Passed in a separate disposable repository | Embedded private recipe builds with ordinary Compose; no checkout/runtime dependency |
| `TestLayaImageSelfContained` on that built image | Passed offline as nonroot | Dependency versions and all five pinned model/config/tokenizer hashes |
| Default Hermes/Laya recreation | Passed | Healthy shared-loopback service, no published ports and repository-local cache surviving coordinated recreation |
| `TestDockerSupervisionStack` | Passed, 535.00 seconds | Public `setup --supervision`, six native plugin installs/hooks and real `LOCAL_ONLY` decisions, independent read-only reports, offline operation, fail-closed outage, installer removal and coordinated recreation |

The first approximately 221-second run exercised the native setup helper. The
final 535.00-second run exercised first-time admission through the public CLI,
coordinated recreation and repeated six-profile checks, followed by read-only
component reporting and removal/offline/recreation checks. An intermediate run
that redundantly invoked both helper and CLI exceeded Go's default ten-minute
limit; its disposable containers were removed. Use the explicit 25-minute limit
below for this CPU inference fixture.

Independent code review reproduced and fixed unsafe Git filter execution,
index-flag drift bypass, native environment/proxy overrides and extra settings
that could disable supervision. Regression checks and a real installed plugin
passed through a read-only container mount. `go test ./...`, `go test -race ./...`,
`go vet ./...` and the pinned Docker verification/checksum tests passed.

The generated default-build image used for the content/cache check was
`sha256:597d658da523f9df4d62517081352b099d444ced08f25b7de7811d637ab3baa4`.
That local content ID records this build, not a published registry digest.
The six-role supervision fixture independently uses an explicitly selected
qualified local image; it does not exercise private memory setup or main-model
work. The historical single-worker inference experiment remains documented in
[generic-team Laya qualification](generic-team-laya.md).

Reproduce the credential-free Docker checks only in disposable environments:

```sh
REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerFoundation -count=1 -v
REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerOpenVikingPending -count=1 -v
# Set REPOKIT_LAYA_IMAGE to the full content ID of a qualified local build.
REPOKIT_NERVE_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerSupervisionStack -count=1 -v -timeout=25m
```

Sources: [foundation fixture](../../tests/acceptance/docker_test.go),
[memory-link fixture](../../tests/acceptance/fixtures/memory_link.py),
[pending-service fixture](../../tests/acceptance/openviking_docker_test.go),
[six-role supervision fixture](../../tests/acceptance/supervision_docker_test.go),
and [offline image check](../../internal/supervision/package_test.go).

## Kanban and live manual acceptance

Bootstrap checks initialized and existing boards with native `kanban list --json`
and `kanban diagnostics --json`. Corrupt boards fail and remain preserved. These
commands belong in setup: native list recomputes readiness and connections can
migrate schema; neither belongs in read-only `verify`.

The credential-free fixture demonstrates researcher → planner → executor
structured handoffs and executor → reviewer → request_changes → executor →
reviewer native run actors. The executor receives the reviewer's actual change
request, and board/handoff state survives recreation without installer artifacts.
Native lifecycle API calls in separate profile processes do not establish
model-driven dispatch, independent judgment or gateway dispatch.

The prepared [live manual runner](../../tests/acceptance/kanban_live.py) asks
default to create the graph, then uses native manual dispatch. It requires an
explicitly marked disposable repository and checks launcher routing and matching
container image/mounts before mutation. It restarts Hermes after changing
startup-sensitive dispatcher configuration. A hidden convention marker must
propagate through researcher evidence, planner metadata and the reviewed
artifact. Observed worker PIDs and native run records establish the operational
sequence. Raw output stays in private local logs; failures preserve native state.

This runner has not passed live main-model acceptance. Private native
provider/model configuration is still required; no host credentials were
borrowed. Use the current built CLI and its private `setup` flow when preparing
a new disposable environment. Old session-specific temporary binaries and paths
are not a supported recovery procedure.

## Read-only verification

`verify` reads native files, source/install metadata and bounded Docker state.
It does not discover/load plugins, open the board or invoke models. The memory
probe uses authenticated read-only `GET /health` to confirm the six profiles'
shared repository user identity. `memory: active` does not establish extraction,
recall or persistent cross-profile knowledge. Nerve reports matched native
configuration; Laya separately checks image/namespace/mount identity and
`GET /healthz` without inference.

`review` remains `unqualified`, so the full command still exits nonzero even
when all installed service/configuration checks pass. See
[integration probes](../../internal/verify/integrations.go) and the
[isolated read-only probe](../../internal/verify/integrations_probe.py).

## Remaining gates

Private native main-model and embedding/VLM credentials/configuration block
live model-driven Kanban work and real OpenViking acceptance. Required evidence
still includes manual and controlled gateway dispatch, a genuine same-card
executor/reviewer correction cycle, cross-profile memory write/recall,
recreation persistence of recalled knowledge, cross-repository denial, future
specialist integration and the complete removal-first stack gate. Superpowers
scanner admission remains a separate unresolved decision.

No configuration, health result or local supervision fixture promotes these
gates to passed. Release qualification and full RepoKit self-dogfood remain open.

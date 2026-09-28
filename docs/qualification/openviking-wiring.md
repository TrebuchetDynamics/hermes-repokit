# OpenViking wiring qualification

Observed 2026-09-27 on Linux amd64. This qualifies generated deployment and native
configuration contracts. Live memory acceptance is still pending private model
and authentication configuration.

## Selected artifacts

- Hermes v0.21.5: `nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db`.
- Official OpenViking v0.4.21: `ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05`.
- Full upstream revisions and schema/resolver provenance are in
  [shared memory qualification](generic-team-memory.md).

The image was already cached. The new sidecar runs as the host UID/GID, with
HOME `/app/.openviking`, explicit config `/app/.openviking/ov.conf`, bot disabled,
no published host port and one private persistent bind mount. The official
entrypoint supplies pending-config behavior and the healthcheck. RepoKit adds
no runtime server or memory protocol.

## Executed deployment fixture

`REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerOpenVikingPending -v -count=1`
passed in 10.69 seconds after the review fixes. It:

1. Built the actual CLI from a disposable source copy and installed in a new Git repository.
2. Parsed the generated Compose and started only the pinned OpenViking service.
3. Observed the real native `/health` response return 503 without config.
4. Checked the service UID matches the host owner.
5. Called the pinned wizard's path helpers and confirmed its config and durable
   workspace resolve inside the persistent mount.
6. Created a nonsecret persistence sentinel, reran install, and observed
   pending configuration with matching container identity through metadata-only verification.
7. Removed the copied RepoKit source/binary, recreated the sidecar using ordinary
   Compose, and verified the sentinel survived and no `ov.conf` was invented.
8. Removed the disposable Compose project. The repository's active dogfood
   deployment was not upgraded or reconfigured.

A sentinel establishes bind-mount persistence, not persistence of extracted
memory. The original foundation Docker fixture also passed in 119.33 seconds
with the new generated Compose while starting only Hermes. The extended fixture
then passed in 144.37 seconds: it exercised the actual native config setters and
profile-scoped OpenViking resolver against one private shared connection file
for all six real cloned profiles. It uses an explicit nonsecret dummy key and
never contacts a memory service. The native serializer and resolver intentionally
omit account/user assertion headers for a normal user key; production separately
checks the server-returned identity. This proves native configuration behavior,
not authenticated memory.

## Setup and upgrade contracts

Normal installation renders the service without selecting models, credentials
or activating the Hermes provider. An exact prior Hermes-only Compose can be
upgraded under the existing writer lock, with a private byte-exact backup and
native state preserved. Edited Compose, unknown sidecar data and conflicting
backups refuse. An empty pre-existing sidecar directory is accepted only with
an exact pre-existing backup identifying an interrupted preparation.

`setup --memory` inherits the operator's terminal for native server init/doctor
and native Hermes memory setup. The native wizard writes `ov.conf` atomically
with mode 0600 and places its default workspace alongside that file. Existing
server config skips initialization. Decline the wizard's optional “Start the
server now?” prompt: the pinned entrypoint watches for the first config and
owns the server process. The setup handoff calls doctor even when a
matching running server's Docker health is starting/unhealthy; image, project,
mount and running-state mismatches refuse before private setup.

After doctor succeeds, a one-shot container check requires explicit
`storage.workspace: /app/.openviking/data`, API-key binding on `0.0.0.0:1933`,
and native session extraction enabled (including its native default). Provider
schema and credential validation remain the native doctor's responsibility.
Setup restarts the sidecar so existing config changes take effect, then waits
up to 120 seconds using the pinned entrypoint's `/health` check. These gates
must succeed before the Hermes connection wizard or profile activation.
The checks neither persist a runtime adapter nor print private config values.

Hermes's own wizard creates the private mirrored connection. A one-shot script
uses native configuration commands under the container-held repository lock to
link it across all six profiles. Before any specialist write it checks each
profile's effective native secret scope, no peer, the expected endpoint, and
server-derived normal-user/account/repository identity. Every effective key
must match the native shared link. Other providers, dormant conflicting
connection paths/endpoints, and YAML, linked or effective secret-scope peers
refuse before any profile setter runs. Built-in local memories remain enabled;
their storage remains profile-local. Native session sync/extraction is unchanged.
The script is not persisted as a runtime dependency.

## Verification boundary

`verify` inspects configuration-file metadata and bounded Docker metadata.
It reports container identity separately from runtime health and always keeps
live recall/isolation acceptance unknown. It never reads the private `ov.conf`
contents or calls a service API. In this pin `/ready` actually performs an
embedding request; neither RepoKit verification nor generated healthchecks use
it. The official healthcheck uses `/health`.

Offline regressions cover fresh installation, state-preserving upgrade/rerun,
unknown-file refusal, interrupted preparation, setup error propagation,
shared-link preservation, profile conflicts, unsafe connection files and
metadata-only verification. Two independent-review findings were reproduced
with failing tests and fixed: unhealthy-service diagnosis and unexplained empty
directory adoption.

The 2026-09-27 runtime-integration follow-up added regressions for the ordered
doctor/config/restart/health activation gates, persistent workspace and native
extraction requirements, dormant connection drift, and effective shared-key,
account/user and peer conflicts. `go test ./internal/native -count=1` and its
race-enabled equivalent passed. The isolated pinned OpenViking pending fixture
was rerun successfully in 10.98 seconds with explicit Docker context `default`.
The extended foundation fixture also passed in 162.95 seconds against the
pinned Hermes image: all six profiles resolved the same native shared link,
and conflicting specialist endpoint/key/account/user/peer secrets or a linked
peer were refused before any configuration write. The fixture used only its
named nonsecret key and never contacted a memory service. The full offline
`go test ./...` suite passed after the default-stack verification changes.
These remain configuration, lifecycle and persistence proofs; the private
model-dependent gates below are still open.

## Still required

The operator must select reachable embedding and extraction/VLM models, enter
credentials through native setup, provision the native account and normal
repository user key, then complete the shared connection wizard. No host
credentials were borrowed and no model endpoint was guessed.

The [live matrix](generic-team-memory.md#live-acceptance-still-required) still
requires actual cross-profile write/recall, a second isolated repository,
restart/recreation/removal recall and outage/recovery behavior. Neither health,
file persistence nor a resolver test can close those gates. Production
Nerve/Laya and model-driven Kanban acceptance follow this work; they remain open.

## Final implementation checks

The full offline Go suite, race suite, vet, formatting, diff whitespace and local
documentation-link checks passed. Independent review's two P2 findings were
fixed with RED→GREEN regressions. Validation ran in the isolated
`feat/openviking-production` worktree without changing the active main deployment.

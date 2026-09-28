# OpenViking inside the repository Hermes container

> Delivery follow-up: full offline and race suites now pass, including the
> Unix-socket cases blocked in the earlier record below. Live Docker acceptance
> remains unqualified. See [delivery validation](../implementation-progress.md#git-delivery-validation-2026-09-28).

The owner selected this topology on 2026-09-28. It supersedes the independent
OpenViking sidecar in earlier plans and qualification records. Existing sidecar
evidence remains historical; it does not qualify the new image or lifecycle.

Normal `install` generates `hermes-<repo>` with Hermes and OpenViking inside the
same container. Normal generated deployments contain only this Hermes service.
The existing Hermes s6 supervisor runs OpenViking as the unprivileged Hermes
user. No RepoKit bootstrap executable, host Docker socket, host service, or
separate OpenViking container is required at runtime.

The derived image copies the official v0.4.21 digest's Python installation and
virtual environment into isolated paths. It preserves Hermes's interpreter and
dependencies, checks the relocated Python prefixes, imports OpenViking, and runs
the native server help during image build. The recipe fingerprint covers the
image pin, service scripts and helper. No moving PyPI resolution occurs. A build-time, exact-match repair makes the
pinned upstream shell entrypoint wait for its server child on shutdown, so s6
cannot acknowledge a stopped service while the child still owns the database.
A real subprocess regression test checks this drain ordering; unknown entrypoint
revisions fail the build rather than receiving a speculative patch.

The native service binds `127.0.0.1:1933`. All six Hermes profiles use this URL
and the existing repository account/user contract. `.hermes/openviking` remains
the private persistent bind at `/app/.openviking`; configuration, database,
provider credentials and extraction remain native OpenViking responsibilities.
Missing private configuration remains pending, not healthy. API-key identity
validation is still required despite loopback binding.

`setup --memory` invokes native init/doctor inside Hermes and restarts only the
OpenViking s6 service, with bounded stop/start waits. It does not restart the
gateway or its workers. `verify` checks the derived Hermes image and memory mount,
then a bounded read-only loopback `/health` request. It neither reads private
server configuration nor invokes `/ready`, embeddings, extraction or inference.

## Existing state

The installer refuses to proceed while an old project OpenViking sidecar runs.
Stop it using its original generated Compose file before upgrading. Installation
does not remove containers or data. Exact recognized older Compose configurations
have the existing conservative backup/publication path. Unknown or previously
edited development recipes remain a refusal; this change does not invent an
unsafe recipe migration. No owner state is overwritten to make verification pass.

Native server settings and shared connection files are preserved. The native
Remote wizard mode supplies API-key authentication; the installed wrapper forces
the actual listener to loopback. An old `http://openviking:1933` connection
requires owner private native setup before activation. RepoKit does not rewrite secret-bearing connection stores.

## Acceptance

Unit checks cover topology, setup ordering, mount identity, pending/healthy/failed
component reporting and preserved private configuration. The Docker fixture
`TestDockerOpenVikingPending` now builds the derived Hermes image, checks that
Compose contains no independent OpenViking service, observes the real native 503
pending endpoint, and verifies private storage survives Hermes recreation after
removing the temporary bootstrap executable.

Image build and that Docker fixture must pass in a Docker-capable session before
this topology is live-qualified. Cross-profile recall, extraction, authenticated
identity and cross-repository isolation still require real private provider setup.
Do not reuse old sidecar PASS results for these gates.

Recorded source validation (2026-09-28): Go build, vet with Docker tags,
formatting, diff checks, targeted topology/setup/health tests and the real
shutdown regression passed. Full `go test ./...` and `go test -race ./...`
passed every package except `internal/target`: the six cases of
`TestNativeGatewaySocketsRemainInspectable` failed with sandbox
`setsockopt: operation not permitted`. No tests were weakened or skipped to
hide this. Enabled `TestDockerOpenVikingPending` stopped during disposable
install with `container names could not be inspected`, before creating
containers or building the new image. The owner's stopped installation and
private data were not modified. Embedded Docker acceptance remains blocked.

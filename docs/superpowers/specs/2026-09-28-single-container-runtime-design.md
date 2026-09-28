# RepoKit v1 core runtime: one container per repository

Status: implemented source architecture, retroactive record (2026-09-28).
Supersedes the independent OpenViking sidecar, the Laya service and the
RepoKit-owned Nerve supervision described in earlier plans and qualification
records. Those records remain historical; they do not describe v1.

## Product invariant

Each repository gets exactly one Hermes runtime container:

```text
hermes-<repo>
├── Hermes (six native profiles)
│   ├── default
│   ├── researcher
│   ├── planner
│   ├── executor
│   ├── reviewer
│   └── steward
├── Kanban (sole task/review lifecycle authority)
├── OpenViking (internal s6-supervised process, 127.0.0.1:1933)
└── project development toolchain
```

OpenViking is not an independent Compose service or container. Nerve and Laya
are not RepoKit core components at all; an owner may install Nerve as an
ordinary Hermes plugin, and RepoKit neither configures, qualifies nor verifies
it.

## Generated deployment

Ordinary `install` renders a Compose file containing only the `hermes` service.
The service mounts the repository at `/workspace` and `.hermes` at `/opt/data`.
There are no `openviking:` or `laya:` services, no published memory/model ports
and no inter-service dependencies. Ordinary Compose owns lifecycle; native
setup owns private configuration.

The optional isolated Docker-acceptance profile remains a separate, explicit,
privileged opt-in. It is not part of the runtime invariant.

## Image

The generated development image is built from the exact pinned Hermes digest.
A multi-stage build copies the pinned OpenViking Python runtime and virtual
environment into isolated paths, keeps Hermes's own interpreter untouched, and
asserts the relocated prefixes, an import of OpenViking and the native server
help at build time. The recipe fingerprint covers the image pins, service
scripts and helper; no moving package resolution occurs.

The image contains the Hermes base, the embedded OpenViking runtime and the
project development toolchain. It contains no model runtime for Laya.

## OpenViking

- Runs as the unprivileged Hermes user under the existing s6 supervisor as the
  static service `repokit-openviking`.
- Persistent state root is `/opt/data/openviking`, backed by the repository's
  durable `<repo>/.hermes/openviking` directory through the `/opt/data` bind.
  Configuration is `ov.conf` / `ovcli.conf` under that root; workspace is
  `/opt/data/openviking/data`; mutable state never lives in the image layer.
- Binds `127.0.0.1:1933` only. All six profiles use that loopback URL. Port 1933
  is never published.
- Handles the pre-configuration state: it serves the pinned 503 pending response
  and never crash-loops the container. Private setup remains native
  (`openviking-server init`/`doctor`) and runs inside `hermes-<repo>`.
- A container-level restart policy does not depend on OpenViking readiness, so a
  pending or failed memory process cannot take Hermes down.

## Kanban

Kanban is the sole task and review lifecycle authority. RepoKit no longer owns
any supervision backend, reflex decision service or hosted fallback. Automatic
dispatch and independent review remain native Hermes Kanban behavior, gated by
core readiness and the real researcher canary. Private memory setup is independent.

## Persistence model

```text
/workspace   the project (repository bind)
/opt/data    all durable RepoKit / Hermes state (`.hermes` bind)
image layer  immutable application and runtime
```

Profiles, SOULs, Kanban history, credentials, sessions, repository source and
OpenViking data are preserved across recreation and RepoKit removal.

## Verification

Verification remains read-only and reports container packaging identity,
per-component readiness and live health separately. It never reduces everything
to Docker container status, and it never treats packaging identity as memory
acceptance. It checks the embedded mount, then a bounded loopback `/health`
request; it does not read private configuration or call `/ready`.

## Removed from core

Laya image/build/selection flags, the Laya Compose service, model download and
checksum packaging, local-only supervision setup (`setup --supervision`), Nerve
installation/qualification/verification, dependency and outage gates, and the
OpenViking Compose service generation for new deployments.

Historical migration-recognition code may remain where it is required to safely
upgrade installations created by older RepoKit versions; it is not a supported
new-deployment path.

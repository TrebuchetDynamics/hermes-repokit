# Opt-in Docker acceptance daemon

`internal/dockertest` renders the `docker-test` service behind the `docker-tests`
profile. The normal Hermes service does not set `DOCKER_HOST`. Only an authorized
invocation of `repokit-docker-test` selects this daemon for its child process.

The daemon shares two project-scoped named volumes with Hermes:

| Volume | Path in both services | Purpose |
| --- | --- | --- |
| `docker-test-run` | `/docker-test/run` | Private Unix socket; Hermes mounts read-only |
| `docker-test-work` | `/docker-tests` | Temporary source snapshots and fixture bind mounts |

A third volume, `docker-test-data`, stores `/var/lib/docker` only in the daemon.
Images and build cache persist across test runs. The daemon receives no checkout,
`.hermes`, operator home, or host Docker socket mount. It listens only on
`unix:///docker-test/run/docker.sock`; no TCP API or published ports are configured.
It uses a separate `docker-test` network, with outbound access for image pulls and
build dependencies, and does not join the Hermes network.

Docker-in-Docker uses `privileged: true`. This provides a separate daemon and
storage lifecycle, **not strong isolation from the host kernel**. The user must
opt into this service before authorized Docker-specific work can run unattended.
These files do not start the service or change a running deployment.

## Image provenance

The source pin is Docker Official Image `docker:29.8.0-dind` at index digest
`sha256:5efed980cba3fc126cf54e21a5a6ff8849d05b6e0623d6e7612f48e9cd6cd17e`.
The [official Docker Hub metadata](https://hub-stage.docker.com/layers/library/docker/29-dind/images/sha256-c9da39e30475d7bf353436738239d02fb1c2a52a1c968322beccb6ec239707d8)
identifies that index, version 29.8.0, and the corresponding arm64 platform
manifest. The multi-platform index is selected; no mutable-tag resolution occurs
in generated Compose. Direct registry API access was unavailable during source
implementation. Pulling/starting this pinned image has not been qualified here.

The [official daemon entrypoint](https://github.com/docker-library/docker/blob/master/dockerd-entrypoint.sh)
adds TCP listeners when invoked with no command or just flags. This service passes
an explicit `dockerd` command and an explicit Unix `--host`, retaining the native
DinD initialization without those implicit TCP defaults. Docker documents Unix
[daemon sockets](https://docs.docker.com/reference/cli/dockerd/#daemon-socket-option)
and [daemon-side bind mount resolution](https://docs.docker.com/engine/storage/bind-mounts/).

## Helper behavior

The development image embeds the standalone Python helper at
`/usr/local/bin/repokit-docker-test`; it needs Python 3, Git and `/usr/bin/docker`.
It supports:

```sh
repokit-docker-test
repokit-docker-test -- go test -tags docker ./tests/acceptance -run TestDockerFoundation
repokit-docker-test --timeout 3600 -- go test -tags docker ./...
```

The default command runs the foundation, channel and maintenance Docker suite
(and other `REPOKIT_DOCKER_TESTS` fixtures, including development-runtime
checks) in a fresh Git repository below `/docker-tests`.

Snapshot selection includes dirty tracked files and non-ignored
untracked source. It excludes `.hermes`, `.git`, installer staging/lock paths,
backup directories/files, known credential names (`.env`, `.env.*`, `auth.json`,
`credentials.json`), generated Python bytecode, all ignored paths (even if
force-tracked), symlinks and special files. `.env.example` and `.env.sample` are
allowed source templates; other `.env.*` files are excluded at every directory
depth. Deleted source files stay deleted. It copies no Git credentials,
hooks, operator Docker contexts or private native state.

The child receives a new HOME, Docker config, TMPDIR and cache directories within
its own fixture tree. Credential variables, SSH agent state, inherited Git
configuration, proxy settings and Docker context/TLS controls are not inherited.
Only display settings survive. Fixture bind-mount sources therefore have the
same absolute path in Hermes and the daemon. The CLI wrapper forces the private
config and Unix socket, permits the `default` context used by acceptance tests,
and refuses endpoint/configuration overrides. The daemon must advertise the
`io.repokit.docker-test=1` label before a fixture can start.

Direct `docker run`/`create`, network creation and volume creation receive an
exact per-fixture label. Cleanup removes only resources matching that label, with
bounded lists, then the helper's own scratch directory. Compose-based fixtures
retain their explicit `down` cleanup lifecycle. No image deletion, global prune,
unfiltered resource deletion or removal of another fixture tree occurs. A timeout
terminates the fixture process group before cleanup, escalating to SIGKILL after
a bounded grace period even if the original parent has already exited.

Offline tests prove source filtering, endpoint/configuration handling, child
execution and bounded cleanup. They use a fake daemon response and do not prove
live DinD startup, pulls, Compose health, bind mounts, or Docker acceptance.

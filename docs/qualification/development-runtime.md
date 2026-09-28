# Development runtime qualification

> Delivery follow-up: full offline and race suites now pass, including the
> Unix-socket cases blocked in the earlier record below. Live Docker acceptance
> remains unqualified. See [delivery validation](../implementation-progress.md#git-delivery-validation-2026-09-28).

RepoKit now generates a derived Hermes development image and an optional
separate Docker acceptance service. This is a source implementation, not a
completed live channel or release acceptance claim.

## Generated normal runtime

The exact Hermes digest supplies Git, Bash, Python, Node/npm, curl, ripgrep,
CA certificates, Make and C/C++ build tools. Read-only checks of that digest
confirmed the versions recorded in [packaging provenance](../../packaging/development/README.md).
The recipe adds checksum-pinned jq and Docker Compose/Buildx; a root go.mod adds
checksum-pinned Go 1.26.6. Build-time commands check versions and compile/run
small C and Go/race fixtures. No package manager or compiler is installed during
container startup. Images are named by a fingerprint of the generated inputs.

The initial supported root-manifest matrix is Go, Node/npm, Python and Make.
Detection never executes manifests, hooks or package scripts. Rust/JVM and
unsupported version/package-manager constraints are reported as unqualified.
Nested projects and dependency setup are not inferred. Dependency installation
and project-specific services remain task-specific native development work.

Both CLI and gateway workers use the same /workspace mount. Native profiles
select terminal.backend=local and terminal.cwd=/workspace. Executor has native
file, terminal, code_execution, skills and memory toolsets. Broad orchestrator
Kanban stays exclusive to default. Reviewer can run checks, with its existing
non-writing policy limitation. Default may make a tiny authorized edit directly;
substantive changes still use executor and independent reviewer.

## Optional Docker acceptance

`hermes-repokit install --docker-tests` publishes the separate daemon service.
It does not start it. Run the exact generated Compose profile command printed by
install. Activation is explicit because the daemon is privileged. It has only
project-scoped test data, scratch and Unix-socket volumes: no host daemon socket,
repository checkout or Hermes private-state mount, and no public TCP endpoint.
A dedicated network allows outgoing image/dependency retrieval; it is separate
from the Hermes/OpenViking application network. Privileged DinD shares the host
kernel and is not VM-grade containment for hostile workloads.

After activation, an authorized task can run `repokit-docker-test` inside Hermes.
This native runtime helper survives RepoKit removal. It copies current source
without ignored/private state into shared scratch, creates an independent Git
fixture and uses a fixed isolated Unix daemon. Its default command exercises the
ordinary opt-in Docker fixtures. It does not copy provider secrets
or modify the source checkout. See [helper usage](../../packaging/docker-test/README.md).

Verify checks the generated image recipe/content identity, runs read-only
version commands, checks six terminal workdirs/backends, and reports
`development_environment` independently from `docker_acceptance`. A disabled
test daemon does not degrade normal repository coding. Healthy version checks
are not evidence that models edited code, reviews ran or memory was recalled.

## Evidence boundary

Source fixtures cover detection, build input checksums, opt-in topology,
installer reruns/migrations, owner drift preservation, exact runtime identity,
missing compiler reports, native tool migration and isolated fixture cleanup.
The Docker-tagged `TestDockerDevelopmentRuntime` builds the actual image and
runs an unprivileged compiler/race fixture without network or daemon sockets.
It must pass in a capable environment before live promotion.

`TestDockerIsolatedAcceptanceDaemon` separately builds the generated development
image, starts only Hermes and the opt-in privileged daemon in a disposable
project, removes the copied installer, and invokes the baked helper. It checks
nested daemon-side bind mounts, real Go compilation, private-file exclusions,
source preservation and fixture cleanup. It requires both explicit gates:

```sh
REPOKIT_DOCKER_TESTS=1 REPOKIT_DIND_TESTS=1 \
  go test -tags docker ./tests/acceptance -run TestDockerIsolatedAcceptanceDaemon -v
```

That privileged fixture is intentionally not enabled by the helper's ordinary
Docker-suite gate. Running it inside another test daemon is a separate opt-in.

Current sandbox denies AF_UNIX setsockopt and Docker access from subprocesses.
No live installation, generated .hermes, credentials, gateway or container was
modified to conceal that limitation. Real image builds, nested-daemon operation,
Telegram/CLI coding parity, model-driven review and full dogfood remain pending.

## Source validation, 2026-09-28

| Check | Observed result |
| --- | --- |
| `go test -count=1 ./...` | All packages pass except six existing `TestNativeGatewaySocketsRemainInspectable` subcases: AF_UNIX `setsockopt: operation not permitted` |
| `go test -race -count=1 ./...` | Same six environment failures; other packages pass |
| Focused race checks after topology-regression additions | Verify, offline acceptance and Docker-helper packages pass |
| `go vet ./...` and `go vet -tags docker ./...` | Pass |
| Formatting and `git diff --check` | Pass |
| Docker suite with ordinary and DinD gates enabled | Six fixtures attempted; all blocked at daemon access/container enumeration before runtime qualification |
| Existing source baseline | HEAD unchanged; all 197 baseline paths remain; skill files unchanged |

The offline launcher-removal fixture was updated to acknowledge setup's native
dispatch-preparation call at its fake Docker boundary. The real Docker foundation
fixture now expects the explicit incomplete-integration error after successful
team scaffolding, retains its profile/state assertions, and does not certify
automatic dispatch without private memory. Neither adaptation
removes a production check. The revised Docker fixtures still need a live run.

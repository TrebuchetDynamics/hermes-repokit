# Hermes development image candidate

The generated recipe extends RepoKit's immutable `FoundationImage`. It does not
resolve apt packages or install project dependencies. `go.mod` adds Go 1.26.6;
all projects get jq 1.8.2 and Docker Compose 5.5.1 / Buildx 0.37.1 CLI plugins.
All downloads have architecture-specific SHA256 pins.
The recipe contains no Pi or mandatory Codex installation.

The exact base image (content ID
`sha256:e1e87aec2e3f5b9863c127a0f21f3b3c237c8717c57c323c7184a49ca12a0df6`)
was inspected read-only on 2026-09-28: Git 2.47.3, Bash 5.2.37, curl 8.14.1,
ripgrep 14.1.1, Python 3.13.5, Node 26.5.1, npm 11.17.0, Make 4.4.1,
GCC/G++ 14.2.0 and the CA bundle were present. Go and jq were absent. The base
contains Docker CLI 26.1.5, but read-only `docker compose version` and
`docker buildx version` both failed because these plugins were absent. The
derived recipe installs and checks their versions without installing a daemon.
The derived build asserts these critical interpreter versions, runs a C compiler
smoke test and, for Go, a real race-enabled Go test. A base update requires
requalification. No derived image or Telegram coding acceptance is claimed by
the source tests; build/runtime qualification remains pending.

Sources:

- [Official Go archive checksums](https://go.dev/dl/#go1.26.6)
- [Official jq 1.8.2 checksums](https://github.com/jqlang/jq/blob/master/sig/v1.8.2/sha256sum.txt)
- [Pinned Hermes Dockerfile](https://github.com/NousResearch/hermes-agent/blob/749220ef0007f8d87bd1531f1c24b0fe93816385/Dockerfile)
- [Official Compose 5.5.1 binary checksums](https://github.com/docker/compose/releases/expanded_assets/v5.5.1)
- [Official Buildx 0.37.1 binary checksums](https://github.com/docker/buildx/releases/expanded_assets/v0.37.1)
- [Official Docker image recipe containing the same plugin checksums](https://github.com/docker-library/docker/blob/master/29/cli/Dockerfile)
- [Docker reproducible image guidance](https://docs.docker.com/build/building/best-practices/)

`internal/development` reads bounded regular root manifests without executing
them. It selects Go from `go.mod`, detects Node/npm, Python and Make, and reports
Rust/JVM requirements as unsupported. Go versions newer than the pin and Node,
npm or Python constraints outside a deliberately small numeric comparison
grammar remain unqualified. Nested workspaces, dependency compatibility,
alternative package managers and project bootstrap commands are not inferred.

The image includes `repokit-docker-test` for an explicitly enabled isolated
Docker acceptance service. Its presence grants no host Docker socket. The
normal Hermes container remains unprivileged with only the repository and
private Hermes state mounted; the test helper never relies on the RepoKit
bootstrap executable.

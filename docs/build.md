# Development binaries and release qualification

On Linux with Go 1.26+, install the bootstrap CLI from the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.2.3/install.sh | REPOKIT_REF=v0.2.3 sh
```

To build unreleased `main`, use `curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/main/install.sh | sh`.
The script downloads the selected source, builds it, and installs
`~/.local/bin/hermes-repokit` plus the short
`repokit` alias. Running `./install.sh` from a source checkout builds that
checkout instead. It replaces a command it previously installed (recognized by an
embedded usage marker), preserves any other existing command, and reports a
blocked name with its fix. Generated `hermes-<repo>` launchers live at the same
path, so a repository whose name normalizes to `repokit` already owns the
`hermes-repokit` name; the installer preserves that launcher and reports it
instead of overwriting it. It also reports when `~/.local/bin` is absent from
PATH or an earlier bootstrap command shadows it. The script does not install a
release binary or start a deployment.

The current source builds one static Go executable for Linux amd64 and arm64:

```sh
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags='-s -w -buildid=' -o dist/hermes-repokit-linux-amd64 ./cmd/hermes-repokit
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false -ldflags='-s -w -buildid=' -o dist/hermes-repokit-linux-arm64 ./cmd/hermes-repokit
(cd dist && sha256sum hermes-repokit-linux-* > SHA256SUMS)
```

2026-09-27 development build evidence: local `go1.26.1 linux/amd64`, no external
Go modules, CGO disabled. Both outputs were identified as statically linked ELF
executables. The amd64 executable ran `plan --engineering` against the checkout.
Arm64 was cross-compiled, not executed on an arm64 host.

The development artifacts at `/tmp/repokit-build-20260927/` were built from
implementation commit `189639c` (historical artifacts; later implementation commits are not included):

| File | SHA256 |
| --- | --- |
| hermes-repokit-linux-amd64 | d29294a291ad49637c1d731544a5e09af7f7560ee43921f6e6ea5aa3111b483b |
| hermes-repokit-linux-arm64 | 3b04c0d6fbda36022d918ce7d9d99af478e44542d2eb60ea19f25b823bee23f0 |

These historical binaries predate usable foundation installation and are **not
v1 release artifacts**. Rebuild current source for Hermes-only installation.
Patched-toolchain qualification and full live integration acceptance remain
unfinished. A release needs a patched toolchain, repeat-build comparison,
execution on both architectures, admission evidence, removal-first acceptance
and subsequent native-team dogfood. No release was published.

Final foundation validation ran successfully:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `gofmt -l cmd internal tests` (empty output)
- `git diff --check`
- `REPOKIT_DOCKER_TESTS=1 go test -tags=docker ./tests/acceptance -run TestDockerFoundation -count=1 -v`

Ordinary tests do not use Docker/network. The gated Docker fixture was actually
executed and cleaned up its project. The separate qualification containers were
also removed; existing unrelated containers were left intact.

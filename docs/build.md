# Development binaries and release qualification

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
implementation commit `189639c` (subsequent commits only document evidence):

| File | SHA256 |
| --- | --- |
| hermes-repokit-linux-amd64 | d29294a291ad49637c1d731544a5e09af7f7560ee43921f6e6ea5aa3111b483b |
| hermes-repokit-linux-arm64 | 3b04c0d6fbda36022d918ce7d9d99af478e44542d2eb60ea19f25b823bee23f0 |

These are **not v1 release artifacts**: install still refuses an unqualified
preset, the local compiler is behind current patches, and full live acceptance
is unfinished. A release needs a patched toolchain, repeat-build comparison,
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

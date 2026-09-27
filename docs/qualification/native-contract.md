# Native operation qualification contract

This page records the original Task 1 contract and verification snapshot. For current command behavior and later native observations, see [implementation progress](../implementation-progress.md), [Phase 7 native qualification](phase7-native.md) and [runtime evidence](runtime-observations.md). Task 1 statements below are historical; they do not supersede those later records.

Task 1 establishes the Go command boundary and an evidence model. It does not qualify or run Hermes, Docker, plugins, sidecars or inference. No release image or native runtime version has been selected for deployment.

## Evidence semantics

`internal/qualification` evaluates a single operation for an exact selected Hermes source revision. Operation identifiers are evidence labels, not executable native commands. A selected revision must be a full 40-character lowercase hexadecimal commit SHA and must match the evidence revision exactly.

| Result | Required evidence |
| --- | --- |
| Unknown (zero value) | Default for missing, malformed, mismatched, unknown-enum or incomplete evidence |
| Unsupported | Matching operation/revision, explicit unsupported verdict and a valid source reference |
| Supported | Matching operation/revision, explicit supported verdict, valid source and runtime references |

Source-only positive evidence cannot establish supported operation. Tests use synthetic revisions/references and prove only these evaluation rules. Unsupported and unknown are distinct: a documented incompatibility is not a successful qualification, and missing evidence is not a proven incompatibility.

The evaluator checks completeness and exact scope, not authenticity. Evidence references are relative record paths: slash-separated nonempty segments starting with an ASCII letter/digit and containing only ASCII letters/digits, dots, hyphens or underscores; dot/parent segments, absolute paths, URL syntax and whitespace are rejected. They must identify reviewed, sanitized source/runtime records; a nonempty string is not proof those records exist. Later operation-owning tasks must verify provenance, selected immutable image/artifact correspondence, fixture coverage and freshness before admitting support. Do not expose this model as a user-editable admission manifest or runtime authority. No qualification receipts are loaded by the Task 1 CLI.

Evidence contains only typed operation/verdict, source revision and sanitized evidence references. It has no credentials, arbitrary environment, native argv, captured output or secret hashes. References must not include credential-bearing URLs or private output. Native credential handling belongs to private native setup I/O, never serializable installer observations.

## Pending native contracts

Every row remains **unknown for deployment**. Existing source research may inform later qualification; it cannot substitute for selected-version runtime evidence.

| Operation | Evidence required before enabling |
| --- | --- |
| Native chat | Bare Hermes behavior, qualified zero-argument mapping if needed, terminal and nonterminal behavior |
| Native setup | Exact native spelling and inherited private I/O in the existing Hermes container, exit/signal behavior |
| Native profiles | Exact native syntax, default-only SAFE configuration, credential isolation, dispatch-off engineering preset |
| Native plugins | Exact installation/discovery contract, immutable artifact provenance, scanner admission and actual installed entry point |
| Native Kanban | Exact native operations and same-card provenance, independently qualified role/review enforcement |
| Official exec shim | Selected image's exec UID/GID/Unix HOME behavior, safe startup, HERMES_HOME and workspace contract |
| Read-only probes | Exact non-mutating operations, deadlines/output bounds, no initialization/auth refresh/inference |

The forwarding examples in the design are not declarations that the selected Hermes version supports those spellings. Task 1 adds no Hermes command aliases, default UID, shell wrapper or runtime adapter.

The [integration findings](../research/2026-09-27-repokit-integration-findings.md) inspect Hermes revision `28e6496a5e3adfea57bebfc9571b981bff378523` as source research, not a deployment pin. They record source-level review and memory behavior. The final directive accepts native OpenViking background sync/extraction; the earlier durable-only privacy exclusion is superseded. Review actor enforcement still requires qualification. The [precursor review](../research/2026-09-27-repokit-precursor-review.md) is additional design evidence. Neither record is promoted into a supported operation by Task 1.

## Task 1 command behavior

The host binary exposes exactly `plan`, `install`, `setup` and `verify`. Global or per-command help succeeds. Missing/unknown commands, unknown flags and unexpected positional arguments are usage errors (exit 2). Recognized actions return an explicit not-implemented diagnostic (exit 1); none pretends to install, inspect or verify a target. Arbitrary rejected argument values are not echoed.

The later generated `hermes-<repo>` launcher has a different purpose: no arguments opens native chat, and all explicit arguments pass unchanged. It is not generated in Task 1. Native setup delegation belongs to Task 6; actual installer wiring belongs to Task 11.

## Offline development qualification

Module: `hermes-repokit`, standard library only, language floor `go 1.26.0`. Local development uses the already-installed `go1.26.1 linux/amd64`. No toolchain or module download is required, and no `go.sum` is needed without external modules.

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go vet ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go build -o /tmp/hermes-repokit-task1 ./cmd/hermes-repokit
```

The Go 1.26 series remains supported, but the installed 1.26.1 is behind subsequent security and bug-fix releases. This is a local development check, **not release-toolchain qualification**. Release work must select a then-current patched toolchain and record provenance; no compiler upgrade/download occurred here. See the [official Go release history](https://go.dev/doc/devel/release).

Linux amd64/arm64 release builds, native architecture execution, independent Compose and the removal-first acceptance proof remain Task 12. An offline unit-test pass provides no runtime, privacy, native review or installer-removal evidence.

Task 1 verification also ran the built binary with an empty `PATH` from an unrelated temporary directory: help, all four stubs and invalid inputs returned their expected process exit codes without creating target files. This checks the skeleton only, not later bootstrap/runtime independence.

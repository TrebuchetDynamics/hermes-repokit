# Hermes RepoKit bootstrap design

RepoKit is a short-lived Go bootstrapper for a repository-specific native Hermes
installation. Its maintained commands are `plan`, `install`, `setup` and `verify`.
After bootstrap, ordinary Compose and the generated launcher operate independently
of RepoKit. Native files remain authoritative; diagnostic receipts are optional
observations, never runtime control state.

## Runtime and ownership

Normal installation creates one Hermes development container. The repository is
mounted at `/workspace`, private `.hermes` state at `/opt/data`, and the official
OpenViking runtime's persistent state at `/opt/data/openviking`. OpenViking runs
inside Hermes under the native s6 supervisor and binds loopback, with no host port.
The generated development image carries qualified tools and repository-required
Go. An optional, explicitly selected Docker acceptance daemon uses dedicated test
storage and never mounts the host Docker socket.

The installer preserves owner configuration and unknown state. It refuses identity
collisions, ambiguous ownership, unsafe symlinks, tracked private state and edited
generated artifacts. Recognized upgrades require evidence and backups; no automatic
state deletion or guessed migration. Services start through ordinary Compose.

## Native team and work

The permanent roster is default/researcher/planner/executor/reviewer/steward.
Default coordinates work and human-facing channels; steward manages team evolution.
Native profile creation/cloning supplies provider configuration, while new profiles
receive distinct SOULs and fresh curated memory. Reruns preserve learned memory and
owner edits. Skills are preferred to unnecessary persistent specialists.

One native Kanban board owns cards, dependencies, runs, claims and review.
Fresh/incomplete deployments keep dispatch off. Successful setup activates one
default gateway dispatcher, automatic review, concurrency one, no automatic
decomposition and a six-profile allowlist. Native provider resolution, role/tool
contracts and routing are mandatory gates; shared memory readiness is independent. Setup
fences claims, preserves active/finalizing workers and validates process identity,
singleton ownership, startup policy and a real no-write researcher canary.

Substantive artifact work belongs to executor and independent reviewer, normally
on one card through changes and renewed review. File and terminal toolsets are not
an OS sandbox; role boundaries remain advisory. Real actor independence, artifacts
and originating-channel delivery require live acceptance.

## Setup and memory

Private native default setup runs in the owner's terminal. Plain setup continues
with native team provisioning and OpenViking setup/linking. `setup --team` resumes
reconciliation without another login; `setup --memory` resumes native memory setup.
No credentials are collected into RepoKit logs, receipts or generated public files.

OpenViking uses account `repokit`, the repository user identity and one shared
connection, with no per-profile peer. Built-in local memory remains enabled.
Native synchronization and extraction are intended behavior; owner-selected model
providers determine external processing. Configuration and authenticated health do
not establish recall, persistence or cross-repository isolation.

Optional plugins remain owner-managed through native Hermes admission and setup.
Scanner refusal must be respected; no force-trust path is introduced. The bundled
maintenance adapter is limited to native graceful restart requests and successor
observations, with live behavior separately qualified.

## Launcher and verification

The standalone POSIX launcher uses an absolute Compose path and captured context,
opens `default` with no arguments, forwards explicit native arguments unchanged,
and preserves streams, signals and exit status. It never starts services or calls
RepoKit. Installation includes safe automatic publication of a symlink at
`~/.local/bin/hermes-<repo>`, reusing matching links and preserving conflicts.
Missing PATH entries and unusable host directories are reported; shell aliases,
startup-file edits and automatic PATH changes are excluded.

Verification reads bounded artifact/runtime observations and authenticated memory
health. It does not open/migrate Kanban, load plugins, run models, dispatch work or
write memory. Configuration, observed process state and behavioral acceptance are
reported separately. Generation receipts cannot prove live channel schemas or work.

## Acceptance and release

Offline tests validate control flow and failure handling. Required Docker and
Unix-socket tests remain explicit gates when sandbox permissions prevent execution.
A release requires actual memory write/recall/isolation, same-card independent
review, channel delivery and integrated removal-first operation in a disposable
repository. Remove only the test-owned installer/source copy, restart with ordinary
Compose and prove native history, sessions and recalled memory persist.

The current source does not claim those live gates passed. See [implementation
status](../../implementation-progress.md), [team model](../../team-model.md),
[setup guide](../../bootstrap-quickstart.md) and the [plan](../plans/2026-09-27-repokit-bootstrap.md).

Source boundaries: [CLI](../../../internal/cli/cli.go),
[Compose](../../../internal/compose/compose.go), [native team](../../../internal/native/team.py),
[gateway activation](../../../internal/native/gateway.go), [verification](../../../internal/verify/verify.go).

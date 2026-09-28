# Maintained RepoKit implementation plan

Scope: the Go bootstrapper generates one Hermes development container, six native
profiles, shared Kanban and embedded OpenViking. Ordinary Compose and native Hermes
own the installed lifecycle. See the [design](../specs/2026-09-27-repokit-bootstrap-design.md).

1. Preserve ownership inspection, private-state protection, no-clobber publication
   and the standalone launcher. Keep the four commands: plan, install, setup, verify.
2. Generate the pinned development image with the official OpenViking runtime
   embedded under native s6; persist its configuration/data on the existing mount.
3. Provision default/researcher/planner/executor/reviewer/steward through native
   APIs, preserve owner edits and learned memory, and keep Kanban authoritative.
4. Resume private native setup and shared-memory linking without copying credentials
   or overwriting an established connection.
5. Gate the single default gateway dispatcher on native provider/profile/tool
   readiness, shared-memory authentication, process/startup evidence and a real
   no-write researcher canary. Preserve active workers and queued work.
6. Keep verification read-only. Separate configuration, health, live dispatch,
   memory behavior, same-card review and channel delivery evidence.
7. Qualify the generated development toolchain and optional isolated Docker test
   daemon. Normal installations have only the Hermes service.
8. Complete real memory, independent review, integrated removal-first acceptance
   and bounded self-dogfood before release.

Offline verification: `go test ./...`, `go test -race ./...`, `go vet ./...`,
formatting and documentation links. Docker acceptance is explicitly opt-in and
uses disposable repositories. Required tests blocked by sandbox permissions remain
blocked, never passed or weakened. Live acceptance is still pending.

Update this plan when command ownership, generated topology, profile contracts,
activation gates or evidence requirements change. Current status is in
[implementation progress](../../implementation-progress.md) and [TODO](../../../TODO.md).

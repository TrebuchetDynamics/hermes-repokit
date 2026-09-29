# Runtime integration evidence and limits

Normal generated deployments now contain one Hermes development container. Native
setup provisions six profiles and performs operational activation. Memory providers
and optional plugins are native Hermes features ([decision](../decisions/2026-09-29-hermes-owns-hermes-features.md)).

This page preserves the scope of earlier evidence. It does not certify the current
single-container image or promote source tests to live acceptance.

| Historical check | Recorded result | Scope |
| --- | --- | --- |
| `TestDockerFoundation` with native memory-link fixture | Passed, approximately 163 seconds | Native profiles, board transitions, connection resolution and persistence; synthetic model configuration |
| `TestDockerOpenVikingPending` (removed) | Passed, approximately 11 seconds | Previous official pending-config service behavior and persistent paths; OpenViking is no longer part of RepoKit |
| Native lifecycle fixture | Passed in separate profile processes | Structured handoffs and same-card request-review/changes/resubmit/done; no model judgment |

Source fixture: [foundation](../../tests/acceptance/docker_test.go). The memory-linking
and pending-service fixtures were removed with OpenViking.

Bootstrap can invoke native Kanban initialization/diagnostics, whose implementations
may change state. Read-only verification does not invoke them or open/migrate the
board. Configuration/health observations remain distinct from operational work.

The prepared [manual live runner](../../tests/acceptance/kanban_live.py) remains a
separate, explicitly disposable acceptance fixture. It has not passed live model
acceptance. Private native provider/model configuration is required; unrelated host
credentials are not borrowed. Its manual-dispatch observations cannot certify the
operational gateway or originating-channel notification path.

`verify` reads artifacts, bounded Docker state and native configuration and reports
`CORE_TEAM` and `DISPATCH`. It does not load plugins, run models or probe memory.
`DISPATCH` stays unqualified until same-card review is observed. See [integration probes](../../internal/verify/integrations.go).

Private main-model configuration, actual independent review correction and full
removal-first acceptance remain gates. The delivery pass cleared the full offline and race suites,
including required Unix-socket operations; live acceptance remains unperformed.
See [implementation status](../implementation-progress.md).

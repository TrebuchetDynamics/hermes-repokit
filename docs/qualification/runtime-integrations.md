# Runtime integration evidence and limits

Normal generated deployments now contain one Hermes development container with
OpenViking embedded under native s6. Native setup provisions six profiles, configures
shared memory and performs operational activation. Optional plugins are owner-managed.

This page preserves the scope of earlier evidence. It does not certify the current
single-container image or promote source tests to live acceptance.

| Historical check | Recorded result | Scope |
| --- | --- | --- |
| `TestDockerFoundation` with native memory-link fixture | Passed, approximately 163 seconds | Native profiles, board transitions, connection resolution and persistence; synthetic model configuration |
| `TestDockerOpenVikingPending` | Passed, approximately 11 seconds | Previous official pending-config service behavior and persistent paths; does not qualify the embedded topology |
| Native lifecycle fixture | Passed in separate profile processes | Structured handoffs and same-card request-review/changes/resubmit/done; no model judgment |

Source fixtures: [foundation](../../tests/acceptance/docker_test.go),
[memory linking](../../tests/acceptance/fixtures/memory_link.py),
[pending service](../../tests/acceptance/openviking_docker_test.go).

Bootstrap can invoke native Kanban initialization/diagnostics, whose implementations
may change state. Read-only verification does not invoke them or open/migrate the
board. Configuration/health observations remain distinct from operational work.

The prepared [manual live runner](../../tests/acceptance/kanban_live.py) remains a
separate, explicitly disposable acceptance fixture. It has not passed live model
acceptance. Private native provider/model configuration is required; unrelated host
credentials are not borrowed. Its manual-dispatch observations cannot certify the
operational gateway or originating-channel notification path.

`verify` reads artifacts, bounded Docker state and native configuration. The memory
probe uses authenticated read-only health to check the six profiles' repository
user identity. It does not load plugins, run models or prove extraction/recall.
Same-card review remains unqualified; the overall command can exit nonzero despite
healthy configuration. See [integration probes](../../internal/verify/integrations.go).

Private main-model and embedding/extraction configuration, Docker-enabled execution,
actual independent review, memory persistence/isolation and full removal-first
acceptance remain gates. The delivery pass cleared the full offline and race suites,
including required Unix-socket operations; live acceptance remains unperformed.
See [implementation status](../implementation-progress.md)
and [embedded-memory qualification](embedded-openviking.md).

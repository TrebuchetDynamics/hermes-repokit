# Memory self-check boundary

`hermes-repokit verify` remains read-only. `verify --memory-check` explicitly
requests a bounded write through OpenViking's public `ov` CLI inside the
repository's one Hermes container. RepoKit implements the check in Go and exits;
it installs no diagnostic module into Hermes or OpenViking.

This opt-in check mutates one reserved diagnostic file and attempts exact cleanup.
Creating it may trigger OpenViking indexing, embedding work, and associated cost.
Do not run it as part of routine or unattended read-only verification.

The check first verifies the generated deployment and pinned running container,
private repository CLI configuration, six Hermes profiles' native provider
settings, OpenViking health under a non-admin repository user key, and the
reserved diagnostic namespace. A failed preflight writes nothing. It then creates a
cryptographically unique file under `viking://~/memories/repokit-selfcheck/` with
`--mode create`, waits for processing, reads it, searches for its unique name,
deletes only that URI, and checks that read and search no longer return it.
An uncertain write is never retried. RepoKit attempts exact cleanup and reports
the canary URI if cleanup cannot be proved. It does not delete owner data,
restart services, or initialize Hermes's memory provider.

The JSON probes distinguish the evidence:

- `memory-profile-config`: all six profiles select the same native CLI config;
- `memory-identity` and `memory-server`: the public service presents the
  expected repository-scoped user identity and memory namespace;
- `memory-write`, `memory-read`, `memory-search`, `memory-cleanup`: direct
  OpenViking exact-file operations;
- `memory-recall-same-profile` and `memory-recall-cross-profile`:
  **unqualified**, because a config check or direct OpenViking read does not
  prove that a Hermes agent used its memory tool;
- `memory-extraction`: **unqualified**, because `viking_remember` may add, merge,
  or skip memories asynchronously.

The overall `memory-check` stays unqualified and exits 1 until agent-level
cross-profile recall has its own safe native acceptance path. `CORE_READY` is
independent of OpenViking. `MEMORY_READY` and `FULL_READY` require behavioral
cross-profile proof, and deeper release acceptance still includes extraction,
restart persistence, cross-channel recall, and cross-repository isolation.

This diagnostic is intentionally a narrower contract than the earlier
[Hermes extraction diagnostic proposal](../superpowers/specs/2026-09-28-memory-diagnostic-lifecycle-design.md).
That proposal remains useful for a future extraction qualification, but its
requirements do not block an isolated direct OpenViking file check.

RepoKit's target boundary is a Go configurer/bootstrapper. Durable runtime
capabilities belong to Hermes and OpenViking. This diagnostic uses only their
public CLIs; if they cannot prove a behavior, it reports **NOT VERIFIED**.
The one-shot `setup --memory` linker now uses Go, Hermes `config get/set` and
`memory status`, and the public OpenViking CLI. Its private native wizard is
still run by Hermes/OpenViking, but RepoKit ships no memory Python helper.

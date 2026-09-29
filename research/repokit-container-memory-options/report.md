# RepoKit memory provider comparison (2026-09-29)

## Method and limits

This is an OSS/source study for RepoKit's pinned Hermes revision `749220ef0007f8d87bd1531f1c24b0fe93816385`, not a live memory qualification. I inspected the bundled provider code and documentation in the running `hermes-repokit` image, RepoKit's setup and verification code, and official upstream documentation/source. No provider was activated and no private memory data was changed. Current `verify` reports OpenViking pending setup and cannot qualify recall; it also reports runtime metadata unavailable despite a running container, which needs a separate diagnosis.

## Finding

Keep embedded OpenViking as RepoKit's shared **project** memory provider and Hermes's built-in `MEMORY.md`/`USER.md` as private **profile** memory. The OpenViking server boundary lets the six profiles use the same repository account/user while the service stays inside the one RepoKit container. This is the supported direction for the product architecture, not evidence of live recall. RepoKit's remaining problem is setup and qualification.

Hermes documents Holographic as profile-local: its default SQLite path is under each profile's distinct `$HERMES_HOME`. Forcing all profiles onto one database would be a RepoKit-specific shared mode, outside that documented contract. Its Python connection registry and lock are process-local. Do not choose it as RepoKit's default team store solely because a common path is configurable; it remains useful as an individual-profile provider.

Mem0 OSS with a local Qdrant path is unsuitable for RepoKit's concurrent profile processes: QdrantLocal uses an exclusive file lock and explicitly requires Qdrant server for concurrent access. A Mem0 server, Qdrant server, or pgvector database could run as another supervised process in the same container, but adds dependencies, model configuration, and operational states. `mem0ai` is not installed in the current image. The pinned Hermes Mem0 backend also attempts collection deletion when embedding dimensions change, so migration needs stronger safeguards before it could be managed automatically.

OpenViking already fits the one-container topology as a supervised process and has model-backed extraction. RepoKit's path requires private server and repository-user setup. The new Go `verify --memory-check` performs a narrower direct OpenViking exact-file write/read/search/delete check through the public `ov` CLI; it does not prove Hermes-mediated recall or extraction. Ordinary memory may work; current evidence does not establish that it does. The [previous diagnostic design](../../docs/superpowers/specs/2026-09-28-memory-diagnostic-lifecycle-design.md) and [planning assessment](../../docs/superpowers/plans/2026-09-28-memory-diagnostic-lifecycle.md) found that a true same-identity Hermes extraction partition crosses storage, queues, caches and asynchronous jobs in the pinned OpenViking source. Exact-file deletion is not a complete cleanup receipt for asynchronous extraction.

## Evidence matrix

| Candidate | One runtime container | Shared across profile processes | Setup | Reversible proof | Main limitation |
| --- | --- | --- | --- | --- | --- |
| Hermes built-in files | Yes | No; profile homes differ | Already active | Per-profile only | No shared team memory |
| Hermes Holographic, normal profile path | Yes | No; each profile has a separate `$HERMES_HOME` | No extra service/secret | Per-profile fact ID | Does not provide supported team sharing |
| Hermes Holographic, forced common SQLite path | Yes | Unqualified across separate processes | No extra service/secret | Fact ID | RepoKit-specific topology outside documented profile isolation |
| Mem0 OSS + Qdrant local path | Yes | **No**; exclusive directory lock | LLM/embedder/key and package | IDs available | Fails concurrent profile model |
| Mem0 + internal server/vector DB | Yes, multiple internal processes | Yes in principle | Highest dependency/operations cost | IDs available | Integration and migration complexity |
| Embedded OpenViking | Yes | Yes in principle through HTTP service | Private model, account, user, server setup | Not currently qualified via Hermes | Extraction lifecycle lacks isolated cleanup receipt |

## Design implications

1. Make OpenViking optional relative to `CORE_READY`; report authenticated health, real provider wiring and behavioral qualification separately. `FULL_READY` needs `MEMORY_READY`, not container health.
2. Reconcile one server/account/repository-user binding across six real profiles through native configuration; retain API-key authentication unless a separately reviewed security design changes it. Keep the root/admin key out of profile configuration.
3. Preserve existing OpenViking data and active owner provider settings. No automatic deletion, import, or switch to Holographic/Mem0.
4. Keep the Go/OpenViking exact-file diagnostic distinct from Hermes agent acceptance. Real fresh same/cross-profile recall, extraction and persistence need owner-controlled test data or a qualified native isolation lifecycle. Do not count direct API visibility as an agent recall pass; report the unproved layer as `NOT VERIFIED`.
5. Keep restart persistence and cross-repository isolation as separate, explicitly scheduled acceptance phases; neither is proved by a local health endpoint.

## Primary sources

- [Hermes Holographic provider README](https://github.com/NousResearch/hermes-agent/blob/main/plugins/memory/holographic/README.md) and [implementation](https://github.com/NousResearch/hermes-agent/blob/main/plugins/memory/holographic/__init__.py).
- [Hermes memory provider profile-isolation documentation](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/memory-providers.md) and [OpenViking provider configuration](https://github.com/NousResearch/hermes-agent/blob/main/plugins/memory/openviking/README.md).
- [Hermes Holographic store implementation](https://github.com/NousResearch/hermes-agent/blob/main/plugins/memory/holographic/store.py) and [reported historical write-lock defect](https://github.com/NousResearch/hermes-agent/issues/55503). Current pin has autocommit and a process-wide connection registry; cross-process behavior remains unproven.
- [Hermes Mem0 provider README](https://github.com/NousResearch/hermes-agent/blob/main/plugins/memory/mem0/README.md) and [backend implementation](https://github.com/NousResearch/hermes-agent/blob/main/plugins/memory/mem0/_backend.py).
- [QdrantLocal source](https://github.com/qdrant/qdrant-client/blob/master/qdrant_client/local/qdrant_local.py) and [local-mode documentation](https://github.com/qdrant/qdrant-client/blob/master/README.md).
- [OpenViking repository and license](https://github.com/volcengine/OpenViking) and [RepoKit's current self-check limitation](../../docs/qualification/memory-self-check.md).

## Gaps

- No live OpenViking write/recall/cleanup or restart-persistence evidence for the current deployment.
- The approved same-identity diagnostic design remains blocked by unproven native scope containment and complete async cleanup; no safe shortcut has been identified.
- No benchmark comparing retrieval quality for actual RepoKit team tasks.
- Current `verify` runtime identity mismatch is unexplained; it must not be conflated with provider capability.
- Upstream code is mutable; implementation should pin and test against the exact Hermes image used by RepoKit.

# Memory Diagnostic Lifecycle Implementation Plan — Prerequisite Blocked

> **For agentic workers:** REQUIRED SUB-SKILL for an approved executable plan:
> use superpowers:subagent-driven-development or superpowers:executing-plans.
> This document is a planning assessment, not an executable implementation plan.
> Do not implement the proposed feasibility step without its separate approval.

**Goal:** Qualify a same-service, same-identity Hermes diagnostic write, extraction,
fresh-profile recall and complete cleanup without touching ordinary memory.

**Architecture:** Version-bound Hermes and OpenViking extensions would carry a
server-authorized diagnostic partition through native storage and async work.
The existing unsupported CLI gate remains until that complete lifecycle is proven.
The source study below does not yet support the approved narrow-patch assumption.

**Tech Stack:** Go CLI/build generation, native Python Hermes/OpenViking,
OpenViking durable queues/vector adapters and Rust RAGFS storage.

**Spec:** [Approved diagnostic lifecycle design](../specs/2026-09-28-memory-diagnostic-lifecycle-design.md).

## Global constraints

- Keep ordinary `verify` passive.
- Preserve real profile endpoint, account, user and peer bindings.
- No direct-write substitute, disposable substitute server, new authenticated
  tenant, global service reinitialization or broad account/user deletion.
- No change to image pins, live services, generated Compose or credentials.
- No provider calls, model spending, deployment, publishing or Git delivery.
- Keep the unsupported gate until isolation and complete cleanup are qualified.
- Stop if the required partition cannot be justified as a bounded extension;
  do not replace it with a superficially passing but different test.

## Disposition

The operator approved the written design and requested an implementation plan.
Read-only planning resolved the source checkout mismatch, then found no existing
scope lifecycle encompassing all required state. A URI prefix, session deletion,
or task completion would produce a false cleanup claim.

Same-service isolation may be possible without replacing native memory
algorithms. That is not evidence that it is a narrow patch: the necessary change
crosses request authority, two persistent data models, worker reconstruction,
shared batching, caches, content logs and conditional Rust background work.
The design explicitly requires stopping here rather than inventing implementation
interfaces or silently substituting a different service/identity.

**No ready-to-execute full implementation plan is claimed.** The next decision
is whether to authorize the bounded feasibility step below, or explicitly expand
the design into a broader native storage/lifecycle project.

## Provenance established during planning

The selected OpenViking Git object is available locally at
`3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1`; a temporary `git archive` of that exact
object was inspected without checking out or modifying the newer source tree.
Six installed Python files in the existing running container matched its bytes:

| Selected-source path | SHA-256 |
| --- | --- |
| `openviking/server/identity.py` | `b3a18908173a70dfc01e2bb589678d0fed53405dfb6aeef4ed236526f6bd357d` |
| `openviking/storage/viking_fs/_access.py` | `fda0a1d1f4773a27fa6e195235f3890f35ab0001c5562374954917d59fe50b8f` |
| `openviking/storage/viking_vector_index_backend.py` | `e81a9f04613db4ec74578c27261177977c70326cda536e47fe3f8503d57be3e2` |
| `openviking/storage/queuefs/session_commit_msg.py` | `d1e2835b082557643e9faa26fb30ee556f2e249e92ee439b593409e17ea8a51a` |
| `openviking/service/session_service.py` | `cc5759ad2b3d28cd8369094fa2eeae866351730096051435563475ebde438b2e` |
| `openviking/session/session.py` | `6b0c2e8b0e2d42ad7d28830f47fa2ec684898070e6f90802edf7cad7550f098c` |

This is representative installed-source correspondence, not a complete patch
manifest or proof of native binary correspondence. Every additional patched
module still needs its input/output hashes. The cached newer revision is not the
implementation basis. No server readiness/model API or private configuration was
queried to produce these results.

## Ownership map and implementation impact

All upstream references below are relative to the selected OpenViking revision.
They identify source behavior, not executed regression results.

| Boundary | Existing source | Required change / risk |
| --- | --- | --- |
| Authority | `server/identity.py:92-117` | RequestContext has identity and peer view but no diagnostic scope/generation. Preserve auth identity while adding a separate validated authority. |
| Physical storage | `storage/viking_fs/_access.py:635-695,954-970` | Paths and legacy fallbacks resolve under the ordinary account. Every forward/reverse/fallback path needs containment; a new directory name alone is insufficient. |
| Vector routing | `storage/viking_vector_index_backend.py:946-1024`; `storage/vector_ids.py:53-61` | Native backends share a collection, routed by account. Account/URI-derived record IDs can collide before filtering. Need real partition routing, not only search predicates. |
| Durable commit | `storage/queuefs/session_commit_msg.py:11-37`; `session_commit_processor.py:33-123` | Scope is not serialized; a worker reconstructs ordinary identity. Retry, recovery and cancellation must preserve scope and generation. |
| Memory merging | `session/memory/streaming_memory_updater.py:88-102,127-169,2212-2255` | Global updater keyed by account/user can batch diagnostic and ordinary requests together. Closing flushes buffered work. Add partitioned ownership, generation fences, draining and selective release. |
| Derived jobs | `storage/queuefs/semantic_processor.py:245-324`; `semantic_msg.py`; `embedding_msg_converter.py` | Parent refresh explicitly detaches task context. Cancelling the commit task does not drain all descendants; scopes must survive every queue conversion and publication. |
| Task state | `service/task_work_index.py`; `task_store.py`; `task_tracker.py` | Ownership is account/user/task, not diagnostic scope. Task records and in-memory results outlive session directories. |
| Conditional training | `session/compressor_v3.py`; `session/train/components/policy_trainer.py` | Enabled skill/agent evolution introduces more shared batches and policy writes. Do not silently change effective profile policy to make a check pass. |
| Recall | `service/search_service.py`; `storage/viking_fs/_semantic.py`; `retrieve/context_assembler/` | Search can read session context and write recall bookkeeping. All fallback reads and derived writes need the same partition. |
| RAGFS cache | `crates/ragfs/src/cache/wrapper.rs:999-1017` | Recursive deletion invalidates root/generation state, not an enumeration/erasure of all descendant cached content. Cache failure handling is not a deletion receipt. |
| Replication/snapshots | `crates/ragfs/src/core/multibackend_wrapper.rs`; `storage/viking_fs/_snapshot.py` | Conditional backup jobs and retained snapshots extend outside Python task ownership. Require exclusion or native scoped drain/erase support. |
| Content sinks | `server/body_dump_middleware.py`; `eval/recorder/wrapper.py`; `observability/usage_audit/runtime.py` | Facts or capabilities may survive in logs/recordings. Suppress or own diagnostic content without altering ordinary observability. |

Existing `service/deletion.py` is relevant as a durable cleanup pattern, but it
revokes/deletes real accounts or users. It cannot serve as the canary cleanup API.
A second OpenVikingService instance in the same process is also not a safe shortcut:
service initialization replaces shared filesystem/queue/task globals.

## Recommended next step: bounded offline feasibility probe

**Question:** Can native session extraction and recall retain one authenticated
account/user while operating in a separately owned storage/index partition, with
all async descendants fenced and erased, using a limited, explicit backend envelope?

This is a proposed throwaway probe, not product implementation. If approved:

- [ ] Use a temporary copy of the exact selected source; do not change the
  repository product code, image recipe, running container or dependency pins.
- [ ] Use deterministic local model fixtures only. Start no external services;
  block outbound network in the harness. Do not import personal credentials or
  copy live memory into fixtures.
- [ ] First test the embedded local-storage/local-vector case. Qualify a
  cache-bypass/exclusion path and absence of backup/snapshot/recording sinks in
  the fixture; do not assume the live deployment has those settings or change
  it. If this envelope cannot be established, stop the probe before mutation.
- [ ] Use one ordinary request context and two proposed diagnostic contexts with
  the SAME authenticated account/user and the SAME logical memory URI. The first
  containment test must demonstrate the current missing partition behavior,
  then test the proposed native routing seams against that collision.
- [ ] Drive native session commit and native retrieval, with deterministic
  extraction/embedding fixtures. Patch only the candidate ownership seams in the
  temporary source. Do not replace extraction/recall with a dictionary fake.
- [ ] Pause a real worker before publication, initiate close, then release it.
  Verify no scoped artifact can reappear; repeat for semantic parent refresh and
  streaming updater flush, not only the session-commit worker.
- [ ] Inspect native files, vector records, queue payloads, task records and
  in-memory registries for both scopes. Ordinary sentinels must be unchanged.
  Zero search results alone are not the acceptance signal.
- [ ] Record exact files changed, tests run, before/after storage evidence and
  unexercised backend configurations. Label the code throwaway; do not ship it.

Existing test starting points to inspect before selecting provider-free fixtures:

- `tests/storage/test_session_commit_processor_identity.py`
- `tests/unit/session/test_session_commit_resume.py`
- `tests/session/memory/test_streaming_memory_updater.py`
- `tests/storage/test_semantic_queue_memory_dedupe.py`
- `tests/storage/test_embedding_msg_converter_tenant.py`
- `tests/storage/test_viking_vector_scope_filter.py`
- `tests/retrieve/test_context_assembler_pipeline.py`
- `tests/storage/test_snapshot_deleted_file_binding.py`

**Success signal:** the probe identifies concrete, tested native routing and
ownership seams, with no identity substitution, no ordinary-memory changes and
no remaining late-publication/cache-retention escape inside its declared envelope.
Then explicitly revise supported configurations in the spec and write the full
TDD implementation plan with actual interfaces and code-level task boundaries.

**Stop signal:** central native operations bypass those seams, safe cleanup needs
unbounded global shutdown/deletion, or containment requires replacing the memory
engine. Report the failure and request a scope decision rather than building more
CLI guards or calling partial cleanup success.

## Review focus for the eventual implementation plan

1. Same account and same logical URI in simultaneous scopes must not overwrite
   ordinary vector IDs or share streaming batches.
2. Context reconstruction after queue retry/restart must not silently downgrade
   scoped work to ordinary work.
3. Cancelled parent work must not leave detached semantic/embedding descendants
   able to write after cleanup completes.
4. Recursive file removal must not be mistaken for erasure of caches, backups,
   snapshots, task payloads or content-bearing logs.
5. Unsupported backend/policy configurations must block before creating a scope;
   qualification for a restricted fixture is not a claim about the live setup.

These are the owning probe scenarios above. They must become explicit regression
cases in the production tasks if feasibility is established.

## Spec coverage and why production tasks are not scheduled

| Approved spec section | Planning disposition |
| --- | --- |
| Version-bound patch delivery | Recorded revisions and six input hashes; complete manifest awaits the proven patch surface. |
| Hermes diagnostic initialization | Existing provider entry points identified in qualification; implementation waits for a real server scope interface. |
| OpenViking partition | Central FS/vector seams exist, but durable worker and batching containment remains unproven. |
| Lifecycle and cleanup | Native account/user deletion is not suitable; full scope ownership and late-publication fencing remain unproven. |
| CLI execution and evidence | Keep current unsupported report; no executable adapter can be honestly enabled yet. |
| Test and enablement gates | Probe scenarios and existing test roots named; full implementation and live acceptance are not authorized here. |

Self-review: no new runtime API is presented as existing; same-service identity
and cleanup requirements are not weakened; no disposable-server substitution;
source correspondence is limited to the six files actually checked. No code tests
were executed during planning. Local documentation links and whitespace were
checked; product tests from the earlier CLI gate are not evidence for this lifecycle.

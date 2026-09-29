> Historical proposal for Hermes-mediated extraction and recall. The routine
> [memory self-check](../../qualification/memory-self-check.md) now uses a
> narrower public OpenViking exact-file check and makes no agent recall claim.

# Isolated Hermes memory diagnostic lifecycle

Status: written design approved by the operator. Carrying version-pinned
dependency patches is approved; implementation and deployment are not. Planning
has reached the stop condition described below: same-identity isolation requires
cross-cutting ownership changes whose bounded feasibility is not yet established.
See the [planning assessment](../plans/2026-09-28-memory-diagnostic-lifecycle.md).
No dependency patches, executable canary or live acceptance are implemented by
this document.

## Purpose and accepted boundaries

Make `verify --memory-check` prove a bounded Hermes-mediated write, extraction,
fresh-context recall in the same and a second real profile, and verified cleanup.
A direct OpenViking write, disposable substitute server, or fabricated profile
configuration must not stand in for this proof. Keep ordinary `verify` passive.

The existing unsupported gate stays in place until the complete lifecycle has
passed qualification. Existing profiles, credentials, ordinary memory, Kanban,
repository files and service availability must be preserved. There is no hidden
restart, broad deletion, model spending, image publication, commit or deployment.
Implementing code and authorizing a live model-backed run are separate decisions.

## Evidence and provenance

Current RepoKit evidence and command behavior:

- [Memory self-check qualification](../../qualification/memory-self-check.md):
  inspected Hermes write, recall, forget and initialization limitations.
- [Shared memory qualification](../../qualification/generic-team-memory.md):
  immutable images and recorded source revisions.
- [Development recipe](../../../internal/development/recipe.go): `recipeInputs`
  and `Fingerprint` cover packaged assets and generated build inputs.
- [Existing entrypoint patch](../../../packaging/development/patch-openviking-entrypoint.py):
  precedent for refusing unexpected upstream source rather than speculative edits.

Hermes source: `749220ef0007f8d87bd1531f1c24b0fe93816385`.
Recorded OpenViking image source: `3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1`.
The cached OpenViking checkout inspected during follow-up is instead
`a09a9d20a8e07d08973aee177802d00e08df29e6`. Its administrative deletion APIs must
not be assumed present in the selected image. Before implementation, establish
matching source and installed file hashes for every patched module. Failure to
establish that correspondence blocks patching; it does not justify upgrading pins.

All new interfaces below are proposed contracts, not claims about existing APIs.

## Selected approach and alternatives

Carry narrow, version-bound patches in the derived image: a Hermes diagnostic
lifecycle and an OpenViking diagnostic scope. Reuse native profile resolution,
authentication, memory extraction and recall algorithms; add isolation around
their state and storage rather than implementing a second memory engine.

Rejected alternatives:

- Direct storage writes and exact-file deletion bypass Hermes and cannot undo
  extraction merges or identify all derived data.
- A disposable server or tenant tests a different binding and cannot establish
  installed real-profile wiring. It may be useful for offline integration tests.
- Waiting for an upstream release avoids patch maintenance but leaves the real
  diagnostic unavailable. The fail-closed gate remains the fallback either way.

## 1. Version-bound patch delivery

Package patches, expected input/output hashes, source revisions and diagnostic
protocol version alongside the development image assets. Include all bytes in
`recipeInputs`, `.dockerignore` allowances and the recipe fingerprint. Apply only
at image build time, never by editing the running container or installed profile.

The patcher validates every input before modifying any file, rejects unknown,
partial or mixed versions, and verifies output hashes. Both Hermes and OpenViking
must expose the same qualified protocol version. A tool-name match, healthy
server, changed image tag or version string alone is not capability evidence.

Retain ordinary upstream behavior when diagnostic mode is absent. Do not change
the base image pins as part of this work. Image reconciliation and deployment
remain explicitly operator-controlled; generated Compose is not edited manually.

## 2. Hermes diagnostic lifecycle

Add an explicit diagnostic initialization path to the native OpenViking provider.
It resolves each selected real profile's configuration and credentials through
the existing resolver, without persisting overrides or replacing its endpoint,
account, user or peer identity.

In this mode, initialization must not acquire ordinary run-state ownership,
recover pending sessions, autostart services, register an ordinary exit-time
commit, sync ordinary chat turns, or write built-in memory. Diagnostic bookkeeping
uses its own scope-owned location. Normal initialization remains unchanged.

The writer invokes Hermes's provider tool dispatch for remember/extraction.
Reader processes use the native memory manager/provider recall path, including
normal recall formatting, not direct server search as a substitute. Diagnostic
context is explicit and cannot become a sticky global environment setting.

Use separate fresh processes for writing, same-profile recall and second-profile
recall. Readers receive the lookup question and scope access, but not the expected
secret value or writer history. The parent compares returned evidence with the
expected value. Successful provider retrieval is not agent-answer acceptance.
No main-agent LLM call is required for the provider-path check.

## 3. OpenViking diagnostic scope

Introduce a server-owned diagnostic scope, identified by a random ID and bound
to the authenticated repository account/user and effective peer policy. Require
the real user credentials on every request; a scope handle alone is insufficient.
Return a short-lived opaque scope capability to the owning client. Never log it
or include it in ordinary report output.

The scope is a storage and index partition, not merely a URI prefix or content
tag. Its sessions, archives, extracted memories, summaries, relations, vector
records, caches and jobs are all scope-owned. Diagnostic extraction invokes the
normal extraction algorithm but can neither read nor merge ordinary memories.
Every diagnostic recall operates only inside that same partition.

Ordinary search, browse, direct URI reads, summary generation and extraction must
exclude diagnostic state at the server boundary. An explicit authenticated
scope is required for any access, including traversal and fallback paths. Another
repository identity must be denied even if it knows the scope ID. Paths must be
server-generated; never accept a client deletion path or ordinary-memory URI.

Both profiles retain their actual bindings. Validate matching authenticated
repository identity before writing and before each recall. Shared user keys do
not prove cryptographically distinct profile actors; report cross-profile
configuration/context recall only, not adversarial actor isolation. Conflicting
peer visibility blocks this check rather than rewriting profile configuration.

## 4. Lifecycle and cleanup

Proposed states:

`allocated -> writing -> extracting -> readable -> closing -> deleted`

Any failure after allocation enters `closing`; incomplete cleanup remains
`cleanup-pending`, never success. Allocation uses an idempotent client run ID so
an uncertain response can be reconciled without creating another scope. All
requests and ownership receipts are associated with that run ID.

Closing first atomically fences new work and makes the scope unavailable to
readers. Drain or cancel existing jobs and prevent late results from being
published. Workers must recheck the scope generation/state at publication, not
just when queued. Delete all scope-owned storage, indexes, summaries, archives,
cache entries, provider bookkeeping and queued work through native storage APIs.

The server returns a deletion receipt only after enumerated owned artifacts are
absent and no worker can republish them. A zero-result semantic search is not a
cleanup proof. Retain only a minimal non-content tombstone for replay prevention;
its expiry must exceed every possible job/retry lifetime. No marker, fact,
credential or scope capability may remain in the tombstone.

The client reserves a separate cleanup deadline, attempts cleanup on errors and
interrupts, and records a private recovery handle if cleanup cannot be verified.
A server-owned lease sweeper retries closing abandoned diagnostic scopes after
client crashes, using the same ownership checks. It must never recover ordinary
sessions or delete user/account trees. Failure to prove cleanup is reported even
if every recall passed. Do not claim success merely because a lease will expire.

Cleanup means removal from managed runtime memory/state. It cannot promise erasure
from third-party model-provider logs or retention systems. Disclose that boundary
before an authorized live run; use only generated, non-sensitive test content.

## 5. CLI execution and evidence

Before allocation, pass read-only checks for exact runtime identity, patch and
protocol qualification, two profile bindings, lifecycle support and configured
provider readiness. Passive verification must not initialize a memory provider.
Only then may an explicitly authorized memory-check create its scope.

Use one fact with a unique lookup key and independent random expected value.
Limit to one write/extraction operation and two fresh-profile recall phases.
Set bounded request sizes, operation deadlines and extraction polling, with a
separate cleanup budget. Uncertain writes are reconciled, never blindly replayed.

Before enabling live use, the implementation must expose the embedding/extraction
model cost boundary and require explicit spending authorization. Operation limits
alone are not a monetary cap: if a requested monetary/token budget cannot be
enforced across native model retries, refuse rather than claim it is enforced.
There is no authorization to make these calls during implementation or CI.

Report preflight, write, extraction, same-profile recall, cross-profile recall and
cleanup independently. Preserve `unsupported`/`unqualified` for unavailable or
unperformed layers; distinguish failed execution from unsupported capability.
Exit zero only if every required layer passes, including cleanup. Partial evidence
must survive failures without exposing credentials, native memory or model output.

Restart persistence, cross-channel recall, ordinary unscoped retrieval and
cross-repository operational acceptance remain separately named checks. Scoped
recall must not be advertised as proof of unscoped automatic memory behavior.

## 6. Required tests and enablement gates

All default tests use deterministic local fixtures and no paid providers.

1. **Patch provenance:** exact source applies; altered, partially patched or mixed
   source refuses; all patch bytes affect recipe identity; output hashes verified.
2. **Hermes isolation:** spy on real diagnostic initialization boundaries; no
   pending-session recovery, normal state writes, autostart or exit-time commit.
   Existing normal provider lifecycle tests still pass.
3. **Storage containment:** exercise the native patched server with deterministic
   model fixtures. Ordinary memory and all relevant indexes remain unchanged;
   diagnostic writes/merges/reads stay within the scope. Test URI traversal,
   ordinary endpoints, fallback reads, cross-user access and forged capabilities.
4. **Real provider flow:** two native profile fixtures resolve their real bindings;
   writer and fresh readers use the patched Hermes APIs; wrong credentials, peers,
   endpoints and expected values fail. No direct-write shortcut can pass.
5. **Cleanup races:** failures at allocation, write, extraction and recall;
   uncertain responses; timeout, disconnect, crash and late worker completion;
   idempotent deletion, expired leases and replayed requests. Prove absence through
   storage/index/job inspection, not search results alone.
6. **Reporting:** cleanup failure overrides recall success; all skipped layers
   stay unqualified; private data and capabilities never enter CLI output.
7. **Compatibility:** normal installation, recipe upgrade safety, read-only verify
   and removal-first runtime independence continue to work without RepoKit present.

Only enable the CLI execution path after these gates and independent review.
A separately approved disposable Docker qualification tests the exact built image;
real profile/model acceptance and deployment require their own explicit approval.

## Implementation boundaries and next checkpoint

Expected production areas: packaged dependency patches and build manifests,
`internal/development` recipe assets/fingerprints, the memory-check CLI adapter,
and version-bound verification evidence. Add upstream-shaped native tests and
RepoKit CLI/integration regressions. Preserve unrelated uncommitted work and the
current unsupported gate until the replacement is qualified.

Before a task-level implementation plan, review this written design. The first
planning prerequisite is matching-source provenance and a complete storage/job
ownership map for the selected OpenViking revision. If the native engine cannot
provide a contained partition without a broad rewrite, stop and report that
finding rather than silently substituting a disposable server or unsafe cleanup.

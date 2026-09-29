# Memory self-check safety

## Current command contract

`hermes-repokit verify --memory-check` is a fail-closed qualification gate, **not
an implemented memory canary**. It returns a JSON probe array and exit code 1:

- `memory-check`: `unsupported`, with the missing lifecycle guarantees;
- `memory-write`, `memory-extraction`, `memory-recall-same-profile`,
  `memory-recall-cross-profile`, `memory-cleanup`: `unqualified`, not exercised.

It stops before target/runtime discovery, subprocess execution, provider imports
or initialization. It does not invoke a model, write a canary, recover pending
sessions, change tasks, restart services or attempt deletion. The report describes
RepoKit's qualification boundary, not a live capability discovery result.

Plain `verify`, including `verify --memory-check=false`, retains its existing
read-only behavior. Other commands reject `--memory-check` with usage exit code
2. Unsupported checks never count as a successful health or recall result.

Implementation: [`internal/cli/cli.go`](../../internal/cli/cli.go),
[`internal/verify/memory_check.go`](../../internal/verify/memory_check.go).
Regression coverage: [`internal/cli/memory_check_test.go`](../../internal/cli/memory_check_test.go).

## Why execution remains blocked

Read-only source qualification inspected Hermes revision
`749220ef0007f8d87bd1531f1c24b0fe93816385`. The running container's OpenViking plugin,
setup module and memory manager matched the inspected files by SHA-256. This
establishes the inspected code identity, not live recall or deletion behavior.

In `plugins/memory/openviking/__init__.py` at that revision:

| Entry point | Safety limitation |
| --- | --- |
| `initialize` (line 1388), `_recover_pending_sessions` (2263) | Initialization can commit unrelated abandoned sessions; it is not a passive probe. |
| `on_memory_write` (2431) | Mirrors only additions asynchronously to generated memory URIs; no returned ownership receipt or mirrored removal. |
| `_tool_remember` (2594) | Uses a dedicated source session, but extraction can add, merge or skip memories without a complete output manifest or caller-selected isolated destination. |
| `_validate_forget_memory_uri` (497), `_tool_forget` (2634) | Accepts an exact user memory file, not the source session or generated summary files. Complete cleanup through this tool is unqualified. |
| `_post_prefetch_search` (1565) | Automatic recall supplies no canary namespace filter. A UUID in content alone does not isolate it from ordinary retrieval. |

These are limitations for a reversible diagnostic, not a claim that ordinary
memory operation fails. Changing only the CLI, using direct OpenViking writes,
or replacing the production identity with a disposable one does not qualify
real-profile Hermes write/recall wiring.

## Prerequisites for enabling mutation

Before replacing this gate with an executable adapter, qualify all of:

1. An isolated canary scope, excluded from ordinary extraction merges and recall,
   accessible only to the selected diagnostic contexts of the real profiles.
2. Initialization that cannot recover or commit unrelated pending sessions, with
   diagnostic bookkeeping separate from ordinary profile state.
3. Ownership receipts covering source sessions, extracted/merged content,
   summaries, indexes and asynchronous jobs. Cleanup must finish without altering
   ordinary memory or allowing delayed jobs to recreate the canary.
4. One unique fact written through Hermes, then recalled in fresh contexts of the
   same and second profiles. The expected value must not be supplied in the recall
   query or shared conversation context. Provider retrieval and agent-answer
   acceptance must be reported separately.
5. Explicit authorization for model/embedding spending, bounded operations and
   deadlines, a separate cleanup budget, and no blind retry of uncertain writes.
   Cleanup failure or uncertainty prevents an overall pass and must be reported
   with a safe recovery procedure.

Restart persistence, cross-channel recall and cross-repository isolation remain
separate acceptance checks. No restart is implied by the memory-check flag.

Revisit this document when the pinned Hermes/OpenViking APIs change. A version
bump, healthy `/health`, or tool-name match alone is insufficient to remove the
gate. Until these prerequisites have evidence-backed support, end-to-end memory
acceptance remains blocked.

> **SUPERSEDED / DO NOT EXECUTE.** This old A–G/master plan belongs to the obsolete 29-task runtime-manager direction. Use the [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md) and [draft bootstrap plan](2026-09-27-repokit-bootstrap.md). Historical body retained below for provenance, not implementation authority.

# RepoKit E — Advisory Nerve Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Observe native Kanban with bounded multiprocess ingestion, read-only reconciliation and optional typed assessments.

**Architecture:** Native synchronous hooks perform only bounded projection/nonblocking local sends. One fenced consumer owns derived state and Laya work; missing observations degrade coverage, never native lifecycle.

**Tech Stack:** Python standard-library Unix datagram sockets, flock, SQLite derived store; native plugin `register(ctx)`; D typed client.

**Spec:** [Approved design](../specs/2026-09-27-repokit-team-runtime-design.md) §7/§9; [shared contracts/file layout](2026-09-27-repokit-team-runtime.md#shared-contracts-and-file-layout).

## Global Constraints

Master constraints apply. Nerve never calls complete/request changes/block/unblock/dispatch or enforces B review policy. No synchronous OpenViking dependency. Hooks do not open SQLite, fsync, sleep/retry, spawn, perform remote network I/O or invoke a model. Missing Laya must not prevent observation.

## Review Focus

1. Claim/spawn hooks run under native dispatch lock (E1).
2. Many producers and competing consumer starts must not create unbounded queues/consumers (E1/E2).
3. Request-review/request-changes lack hook events (E3).
4. SQLite live-WAL read must not initialize/create/migrate anything (E3).
5. A BLOCK recommendation or unknown budget is not lifecycle authority/zero budget (E4).

---

### Task E1: Project verified hooks into bounded nonblocking ingress

**Files:** Create `runtime/repokit_nerve/__init__.py`, `runtime/repokit_nerve/plugin.yaml`, `runtime/repokit_nerve/hooks.py`, `runtime/repokit_nerve/ingress.py`, `tests/offline/test_nerve_hooks.py`, `tests/runtime/test_nerve_nonblocking.py`.

**Interfaces:** Consumes native `hermes.observer.v1` additive kwargs. Produces `project_event(name:str,kwargs:dict,identity:dict,sequence:int)->dict|None`, `try_send(sock:socket.socket,payload:bytes)->bool`, master `register(ctx)->None`. Envelope fields: schema=`repokit.nerve.hook.v1`, eventId, eventName, producerInstance, sequence, repoId, board, taskId/runId/profileName optional, observedAt, payload (allowlisted primitives only), droppedSinceLastSend.

- [ ] Write failing test:

```python
import unittest
from repokit_nerve.ingress import try_send

class FullSocket:
    def __init__(self): self.calls = 0
    def send(self, payload):
        self.calls += 1
        raise BlockingIOError()

class HooksTest(unittest.TestCase):
    def test_full_ingress_drops_without_retry(self):
        sock = FullSocket()
        self.assertFalse(try_send(sock,b'{}'))
        self.assertEqual(sock.calls,1)
```

Add projection tests for canary raw reason/summary/config fields excluded, additive kwargs tolerated, dataclass DispatchResult explicitly reduced, taskless tick valid, executing profile not assumed assignee. Actual multiprocessing test starts >=8 producers, stalls/omits receiver, sends 10,000 observations, validates nonblocking socket mode/one syscall/zero fallback and bounds; record latency distribution on supported host, not hard-real-time claim.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_nerve_hooks.py'`.
- [ ] Inspect selected Hermes `hermes_cli/kanban_db.py` emitter, `hermes_cli/kanban_db_dispatch.py` hook sites/result dataclass, `hermes_cli/plugins_dispatch.py` exclusions. Register exact eight hooks: `kanban_task_claimed`, `on_kanban_worker_spawned`, `kanban_task_completed`, `kanban_task_blocked`, `on_kanban_worker_exited`, `on_kanban_worker_stale_claim`, `on_kanban_task_updated`, `on_kanban_dispatch_tick`. Package `plugin.yaml` and `register(ctx)` with native name `repokit-nerve`, separate from `repokit-policy`. Qualify manifest/discovery/enablement and package imports using B1's native installation contract. Qualify loading in dispatcher/worker/mutator, not just interactive process. E1 owns the package entry/manifest; D's independent modules can exist in the namespace before registration, so unavailable Laya packaging cannot block observation.
- [ ] Implement core:

```python
def try_send(sock, payload):
    if len(payload) > 4096:
        return False
    try:
        return sock.send(payload) == len(payload)
    except (BlockingIOError, OSError):
        return False
```

Socket is pre-opened AF_UNIX/SOCK_DGRAM, connected and set nonblocking during plugin setup outside callbacks. Bounded projection visits known keys only, caps string sizes and payload <=4 KiB before send; no generic serialization of arbitrary objects. Producer instance/sequence IDs dedupe; local dropped counter is included in next successful envelope, no disk fallback. Absent consumer registration remains observationally degraded, never blocks native call. Consumer path `.hermes/nerve/events/ingress.sock` uses private verified directory and peer ownership; ingress qualification checks kernel queue ceiling <=256 (including supported host `net.unix.max_dgram_qlen`), no silent sysctl changes. Reject unsupported queue semantics; no unbounded producer buffers. Local IPC is the sole callback I/O.
- [ ] Green offline command; authorized runtime command `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_nerve_nonblocking.py'` proves native lock stall absent/full consumer behavior.
- [ ] Commit: `git add runtime/repokit_nerve tests/offline/test_nerve_hooks.py tests/runtime/test_nerve_nonblocking.py && git commit -m "feat: hand off native hooks without blocking"`.

### Task E2: Single consumer, bounded spool and restart fencing

**Files:** Create `runtime/repokit_nerve/consumer.py`, `runtime/repokit_nerve/store.py`, `tests/offline/test_nerve_store.py`, `tests/runtime/test_nerve_consumer.py`.

**Interfaces:** Consumes HookEnvelopeV1; Produces `store.prune_records(records:list[dict],now:float)->list[dict]`, `consumer.run(home:str,generation:str)->None`. Derived schema stores bounded observations/assessment metadata, not card/transcript copies. Consumer owns spool, SQLite writes and async D calls only after independent inference gate.

- [ ] Write failing test:

```python
import unittest
from repokit_nerve.store import prune_records

class StoreTest(unittest.TestCase):
    def test_age_and_count_both_bound_history(self):
        records = [{'observedAt':i,'eventId':str(i)} for i in range(10001)]
        self.assertEqual(len(prune_records(records,10001)),10000)
        self.assertEqual(prune_records(records,1000000),[])
```

Add real process tests: two starters synchronize at barrier, only one acquires consumer flock; SIGKILL/restart retains bounded checkpoint and rejects late old-generation result. Duplicate, reordered, missing producer sequences and 10,000 full-ingress events stay bounded; no replayed inference storm. Full disk yields degraded coverage, not blocked dispatch.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_nerve_store.py'`.
- [ ] Implement retention core:

```python
def prune_records(records, now):
    eligible = [r for r in records if 0 <= now - r['observedAt'] <= 7 * 86400]
    return sorted(eligible,key=lambda r:r['observedAt'])[-10000:]
```

Keep in-memory ingress/coalescing <=256 events; spool rotation <=16 MiB including active file; assessments <=10,000 or seven days, whichever first. Bound SQLite page count and WAL/checkpoint growth with a configured quota, not record count alone; on full refuse telemetry writes. Deduplicate event IDs/producer sequence and coalesce task/run observations; retain coverage-gap flags when sequence missing or losses reported. Background supervisor startup is one explicit runtime entry under admitted plugin packaging, not one thread per producer. Use persistent consumer lock inode with nonblocking flock and generation receipt; restart cannot steal live lock. Single consumer owns socket unlink/rebind after verified ownership and lock, no arbitrary deletion. Fresh generation fences outstanding results; enqueue only current per-task observation and at most 32 D assessments, one in flight. Startup reconciles once within bounded native page budget before assessing; no history-wide catch-up inference.
- [ ] Green offline command; later `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_nerve_consumer.py'` qualifies supervision/restart inside the single Hermes.
- [ ] Commit: `git add runtime/repokit_nerve/consumer.py runtime/repokit_nerve/store.py tests/offline/test_nerve_store.py tests/runtime/test_nerve_consumer.py && git commit -m "feat: bound and fence one advisory consumer"`.

### Task E3: Noninitializing native WAL reconciliation

**Files:** Create `runtime/repokit_nerve/observer.py`, `runtime/repokit/board_reader.py`, `tests/offline/test_board_observer.py`, `tests/runtime/test_board_observer.py`, `qualification/board-observer.md`; modify `runtime/repokit/native.py`.

**Interfaces:** Consumes master `native.observe_board(path,after,limit)->dict`. Produces `observer.reconcile(snapshot:dict,previous_checkpoint:int)->dict`, `board_reader.read_page(path:str,after:int,limit:int)->dict`. Snapshot contains version-qualified canonical event/card projection only, checkpoint and coverage; unknown budget/heartbeat/live status stays null/unknown.

- [ ] Write failing test:

```python
import tempfile
import unittest
from pathlib import Path
from repokit.board_reader import read_page

class ObserverTest(unittest.TestCase):
    def test_absent_board_is_not_initialized(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'kanban.db'
            result = read_page(str(path),0,100)
            self.assertEqual(result['coverage'],'unknown')
            self.assertEqual(list(Path(directory).iterdir()),[])
```

Add live-WAL fixture containing committed review/changes events absent from hooks; bounded page/checkpoint catches them, no transition replay. Test missing WAL/SHM, schema drift, retention gap, concurrent append, DB locked/absent, alias duplication; assert all native files/directories byte/mtime inventories unchanged and events not copied wholesale into Nerve state.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_board_observer.py'`.
- [ ] Inspect selected `hermes_cli/kanban_db.py` schema, event IDs/retention, review/changes events and native connection initializer; `kanban_db_dispatch.py` heartbeat/reclamation semantics. Pin exact SELECT columns/version fingerprint, accept additive irrelevant fields but unknown schema returns unknown. Do not import native DB convenience connector if it creates/migrates.
- [ ] Implement the bounded reader and reconciliation. Core absent-board path:

```python
from pathlib import Path

def missing_snapshot(path):
    if not Path(path).is_file():
        return {'events':[],'cards':[],'checkpoint':0,'coverage':'unknown'}
    return None
```

Read URI `mode=ro`, query_only, short bounded read transaction and zero busy timeout; never `immutable=1` on live WAL. `mode=ro` alone may create SHM: use a qualified read-only VFS or, for v1, B2-style read-only namespace helper with containing board/WAL/SHM directory mounted read-only and no native init access. Implement helper in board_reader.py; native.py runs it with fixed path/page args, bounded bytes/time, no hooks spawning processes. Existing compatible SHM permits reads; if SQLite needs initialization/write, return unknown rather than repair. Source-only read-only claims are insufficient; actual WAL/SHM no-write fixture must pass. Enforce max 100 events/page and bounded pages/tick with checkpoint; unknown gap does not synthesize review count. Reconciliation may restore evidence, never lifecycle transitions.
- [ ] Green offline command; authorized `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/runtime -p 'test_board_observer.py'` qualifies actual selected DB. Unsupported host/VFS leaves native observation unknown; status does not initialize it.
- [ ] Commit: `git add runtime/repokit_nerve/observer.py runtime/repokit/board_reader.py runtime/repokit/native.py tests/offline/test_board_observer.py tests/runtime/test_board_observer.py qualification/board-observer.md && git commit -m "feat: reconcile missing native events without DB writes"`.

### Task E4: Advisory tools, freshness and degradation

**Files:** Create `runtime/repokit_nerve/tools.py`, `tests/offline/test_nerve_tools.py`; modify `runtime/repokit_nerve/__init__.py`, `runtime/repokit_nerve/consumer.py`, `runtime/repokit/fleet.py`, `tests/offline/test_fleet.py`, `src/observe.js`, `docs/host-lifecycle.md`.

**Interfaces:** Consumes D AssessmentInputV1/AssessmentResultV1 and E store/coverage. Produces `nerve_assess(task_id:str)->dict`, `nerve_status(task_id:str)->dict`, `nerve_explain(task_id:str)->dict`, `nerve_history(task_id:str)->dict`, `display_recommendation(result:dict)->str`. Tools receive configured store/ingress through plugin registration, not caller-supplied paths. Assess requests bounded async work and immediately returns pending/known/unknown.

- [ ] Write failing test:

```python
import unittest
from repokit_nerve.tools import display_recommendation

class ToolsTest(unittest.TestCase):
    def test_unknown_never_becomes_block(self):
        self.assertEqual(display_recommendation({'status':'UNKNOWN',
                                                'reasonCode':'LAYA_TIMEOUT'}),'WATCH')
        self.assertEqual(display_recommendation({'status':'KNOWN',
                                                'recommendation':'BLOCK'}),'BLOCK')
```

Add mutation spies that fail if any native lifecycle/dispatch or memory call occurs for either result; absent/full Laya leaves native outcomes unchanged. Assess returns before blocked model fixture completes; other tools only bounded derived reads. Canary raw source/logs never enter DTO/spool/reasons; unknown budget/cycles remain null; stale last-known explicit.
- [ ] Red: `PYTHONPATH=runtime:. python3 -m unittest discover -s tests/offline -p 'test_nerve_tools.py'`.
- [ ] Implement display core:

```python
def display_recommendation(result):
    if result.get('status') != 'KNOWN':
        return 'WATCH'
    return result['recommendation']
```

Register four native tools and hooks in `register(ctx)` using B1-qualified plugin API, not separate gateway. Re-run role schema admission with all four advisory names present: B1's ADVISORY set permits them without adding lifecycle/write capability, while unavailable Nerve does not remove unrelated role readiness. Exercise actual loaded schemas as well as offline fixtures; never broaden role admission to arbitrary plugin tools. Bounded evidence-derived reason templates cover stale heartbeat, missing acceptance, repeated fingerprint, review cycle and gap; no Laya free-text rationale. Publish consumer/backlog/loaded-process coverage/assessment freshness separately from Hermes/board status. Expose loaded versus configured checkpoint/backend, desired alignment and evidence time. Advisory findings never fulfill B review gate. Document owner/orchestrator intervention through native actors only; no automatic block, retry, reroute or dispatch even for KNOWN BLOCK.
- [ ] Green red command then all offline Python tests; authorized E1/E2 failures show native work continues.
- [ ] Commit: `git add runtime/repokit_nerve runtime/repokit/fleet.py tests/offline/test_fleet.py src/observe.js tests/offline/test_nerve_tools.py docs/host-lifecycle.md && git commit -m "feat: expose bounded advisory Nerve evidence"`.

**Exit gate:** Actual multiprocess/no-stall/single-consumer/WAL evidence is required for runtime support. D availability is optional for observation, not a reason to invent assessments or lifecycle authority.

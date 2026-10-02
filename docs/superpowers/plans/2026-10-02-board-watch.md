# Board Watch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep a deployment's committed work moving while its board is quiet: goal cards record the owner's goals, and a RepoKit-managed Hermes cron job wakes default only when something is stuck or a goal can advance.

**Architecture:** A small Python monitor script, run by Hermes cron every 5 minutes, prints `busy` (Hermes then skips the model) or a stable digest that changes only when stuck state or an idle back-off bucket changes. A Go package `internal/boardwatch` owns the script, the job's managed fields and the create/upgrade/owner-opt-out decision; `internal/native` applies that decision through public Hermes CLI commands inside the container; `internal/verify` reports it. Default's SOUL gains the goal-card rule and default is granted `goals.max_turns: 100`.

**Tech Stack:** Go 1.26 (stdlib, `embed`), Python 3 stdlib in the container, Hermes CLI (`cron`, `kanban`), existing RepoKit runner/compose plumbing.

**Spec:** `docs/superpowers/specs/2026-10-02-board-watch-design.md`

## Global Constraints

- Job name `repokit-board-watch`; script `repokit-board-watch.py` in default's `scripts/`; schedule `every 5m`; workdir `/workspace`.
- The script prints exactly `busy` unless a card is stuck or a goal is open and the board has been idle at least 30 minutes; chat activity in the last 15 minutes is busy.
- Idle buckets: 30m, 1h, 2h, 4h, 8h, then whole days (`1d`, `2d`, …). The digest carries no timestamps.
- Goal cards: title begins `Goal:`, status `blocked` (created with `initial_status: "blocked"`), assignee default; work toward a goal is a parent of the goal card (`kanban_link parent=<work> child=<goal>`).
- RepoKit never recreates or rewrites a job or script the owner paused, edited or removed.
- Files RepoKit writes under `.hermes` are mode 0600 (the safety scan refuses group-writable files).
- Never touch the peer's uncommitted files (`README.md`, `docs/qualification/*`, `internal/cli/cli.go`, `internal/target/target_test.go`, `internal/cli/memory_check*.go`); stage explicit paths only.
- Gate: `go vet ./... && test -z "$(gofmt -l $(git ls-files '*.go'))" && go test ./...`

## Review Focus

- A board whose only open item is a long-blocked owner-input card: expect one wake per day (the `Nd` bucket), never per tick.
- The owner deletes the job: expect RepoKit to leave it deleted on every later install (marker present, job absent ⇒ opted out).
- The owner already has a cron job named `repokit-board-watch` that RepoKit did not create: expect it untouched.
- `kanban list` fails inside the script (DB busy, CLI error): expect `busy` and exit 0, never a traceback delivered to the chat.
- A goal card without any chat subscription (created from the CLI): expect no subscription attempts and no error.

---

### Task 1: Monitor script and its decision tests

**Files:**
- Create: `internal/boardwatch/repokit-board-watch.py`
- Create: `internal/boardwatch/script_test.go`

**Interfaces:**
- Produces: script behavior `REPOKIT_WATCH_SNAPSHOT=<file> python3 repokit-board-watch.py` prints the decision for `{"tasks": [...], "last_chat": <epoch>, "now": <epoch>}` without calling Hermes.

- [ ] **Step 1: Write the failing test** (`internal/boardwatch/script_test.go`)

```go
package boardwatch

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func decide(t *testing.T, tasks []map[string]any, lastChat, now float64) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	dir := t.TempDir()
	snap, _ := json.Marshal(map[string]any{"tasks": tasks, "last_chat": lastChat, "now": now})
	if err := os.WriteFile(filepath.Join(dir, "s.json"), snap, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "w.py"), Script, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, filepath.Join(dir, "w.py"))
	cmd.Env = append(os.Environ(), "REPOKIT_WATCH_SNAPSHOT="+filepath.Join(dir, "s.json"))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func card(id, title, status, assignee string, at float64) map[string]any {
	return map[string]any{"id": id, "title": title, "status": status, "assignee": assignee, "created_at": at}
}

// The script wakes default only for stuck work or an open goal on an idle
// board, with timestamp-free output that changes only per back-off bucket.
func TestScriptDecisions(t *testing.T) {
	const now = 1_000_000.0
	goal := card("t_goal", "Goal: ship it", "blocked", "default", now-40*60)
	for _, tc := range []struct {
		name     string
		tasks    []map[string]any
		lastChat float64
		want     string
	}{
		{"empty board", nil, 0, "busy"},
		{"running card", []map[string]any{goal, card("t_a", "work", "running", "executor", now-3600)}, 0, "busy"},
		{"ready assigned card", []map[string]any{goal, card("t_a", "work", "ready", "executor", now-3600)}, 0, "busy"},
		{"chat active", []map[string]any{goal}, now - 60, "busy"},
		{"idle under 30m", []map[string]any{card("t_goal", "Goal: ship it", "blocked", "default", now-10*60)}, 0, "busy"},
		{"open goal idle 40m", []map[string]any{goal}, 0, `idle {"blocked": [], "goals": ["t_goal"], "idle": "30m", "triage": []}`},
		{"triage card idle 3h", []map[string]any{card("t_t", "work", "triage", "executor", now-3*3600)}, 0, `idle {"blocked": [], "goals": [], "idle": "2h", "triage": ["t_t"]}`},
		{"owner-blocked 50h", []map[string]any{card("t_b", "Owner inputs", "blocked", "default", now-50*3600)}, 0, `idle {"blocked": ["t_b"], "goals": [], "idle": "2d", "triage": []}`},
		{"unassigned ready is not work", []map[string]any{goal, card("t_u", "note", "ready", "", now-3600)}, 0, `idle {"blocked": [], "goals": ["t_goal"], "idle": "30m", "triage": []}`},
		{"done cards only", []map[string]any{card("t_d", "work", "done", "executor", now-9000)}, 0, "busy"},
	} {
		if got := decide(t, tc.tasks, tc.lastChat, now); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

// Output is stable within a bucket, so Hermes suppresses repeat wakes.
func TestScriptOutputStableWithinBucket(t *testing.T) {
	goal := card("t_goal", "Goal: ship it", "blocked", "default", 0)
	a := decide(t, []map[string]any{goal}, 0, 40*60)
	b := decide(t, []map[string]any{goal}, 0, 55*60)
	c := decide(t, []map[string]any{goal}, 0, 61*60)
	if a != b || a == c {
		t.Fatalf("bucket stability: %q %q %q", a, b, c)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/boardwatch/ -run TestScript -v`
Expected: FAIL to compile (`undefined: Script`).

- [ ] **Step 3: Write the script** (`internal/boardwatch/repokit-board-watch.py`) and the embed (`internal/boardwatch/boardwatch.go`)

```python
#!/usr/bin/env python3
"""RepoKit board watch: monitor script of the Hermes cron job repokit-board-watch.

Hermes runs it every tick and wakes default only when the output changes. It
prints exactly "busy" while nothing needs default, so a busy or settled board
costs no model call; otherwise a stable digest of open goals and stuck cards
plus an idle back-off bucket. It also subscribes each goal card's chat to the
work cards linked to that goal, so their completions wake the chat coordinator.
"""
import json
import os
import subprocess
import sys
import time

HERMES = os.environ.get("HERMES_BIN") or "/opt/hermes/.venv/bin/hermes"
GOAL_PREFIX = "Goal:"
CHAT_QUIET_SECONDS = 15 * 60
BUCKETS = [(30 * 60, "30m"), (3600, "1h"), (2 * 3600, "2h"), (4 * 3600, "4h"), (8 * 3600, "8h")]
NOT_CHATS = {"cli", "cron", "kanban", "oneshot", "acp", "api_server", "tui", "webhook"}


def bucket(idle):
    """None under 30 minutes, then 30m..8h, then whole days."""
    if idle >= 86400:
        return "%dd" % (idle // 86400)
    name = None
    for seconds, label in BUCKETS:
        if idle >= seconds:
            name = label
    return name


def is_goal(task):
    return (task.get("title") or "").startswith(GOAL_PREFIX)


def decide(tasks, last_chat, now):
    work = [t for t in tasks if not is_goal(t)]
    goals = sorted(t["id"] for t in tasks if is_goal(t) and t["status"] == "blocked")
    triage = sorted(t["id"] for t in work if t["status"] == "triage")
    blocked = sorted(t["id"] for t in work if t["status"] == "blocked")
    active = any(
        t["status"] == "running" or (t["status"] in ("ready", "todo", "review") and t.get("assignee"))
        for t in work
    )
    if active or now - last_chat < CHAT_QUIET_SECONDS or not (goals or triage or blocked):
        return "busy"
    last = max([last_chat] + [t.get(k) or 0 for t in tasks for k in ("created_at", "started_at", "completed_at")])
    idle = bucket(now - last)
    if idle is None:
        return "busy"
    return "idle " + json.dumps({"idle": idle, "goals": goals, "triage": triage, "blocked": blocked}, sort_keys=True)


def hermes(*args):
    out = subprocess.run([HERMES, *args], capture_output=True, text=True, timeout=60)
    if out.returncode != 0:
        raise RuntimeError("hermes %s: %s" % (args[:2], out.stderr.strip()[:200]))
    return out.stdout


def last_chat_activity():
    try:
        sys.path.insert(0, "/opt/hermes")
        from hermes_state import SessionDB
        rows = SessionDB().list_sessions_rich(limit=50)
    except Exception:
        return 0
    return max([float(r.get("last_active") or 0) for r in rows if r.get("source") not in NOT_CHATS] + [0])


def task_id(entry):
    return entry if isinstance(entry, str) else (entry or {}).get("id")


def subscribe_goal_work(goals):
    for goal in goals:
        targets = json.loads(hermes("kanban", "notify-list", goal, "--json") or "[]")
        if not targets:
            continue
        parents = json.loads(hermes("kanban", "show", goal, "--json")).get("parents") or []
        for work in filter(None, map(task_id, parents)):
            have = {(s.get("platform"), str(s.get("chat_id")))
                    for s in json.loads(hermes("kanban", "notify-list", work, "--json") or "[]")}
            for t in targets:
                if (t.get("platform"), str(t.get("chat_id"))) in have:
                    continue
                args = ["kanban", "notify-subscribe", "--platform", t["platform"],
                        "--chat-id", str(t["chat_id"]), "--delivery-mode", "notify+wake"]
                for flag, key in (("--thread-id", "thread_id"), ("--user-id", "user_id"), ("--chat-type", "chat_type")):
                    if t.get(key):
                        args += [flag, str(t[key])]
                hermes(*args, work)


def main():
    snapshot = os.environ.get("REPOKIT_WATCH_SNAPSHOT")
    if snapshot:
        with open(snapshot) as f:
            s = json.load(f)
        print(decide(s["tasks"] or [], s["last_chat"], s["now"]))
        return
    try:
        tasks = json.loads(hermes("kanban", "list", "--json"))
    except Exception as exc:
        print("repokit-board-watch: %s" % exc, file=sys.stderr)
        print("busy")
        return
    out = decide(tasks, last_chat_activity(), time.time())
    try:
        subscribe_goal_work(sorted(t["id"] for t in tasks if is_goal(t) and t["status"] == "blocked"))
    except Exception as exc:
        print("repokit-board-watch: %s" % exc, file=sys.stderr)
    print(out)


if __name__ == "__main__":
    main()
```

```go
// Package boardwatch owns RepoKit's board watch: the Hermes cron job that
// wakes default when a quiet board has an open goal or a stuck card.
package boardwatch

import _ "embed"

// Script is the cron job's monitor script.
//
//go:embed repokit-board-watch.py
var Script []byte
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/boardwatch/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/boardwatch/repokit-board-watch.py internal/boardwatch/boardwatch.go internal/boardwatch/script_test.go
git commit -m "feat: add the board watch monitor script"
```

### Task 2: Managed job spec, host reads and the lifecycle decision

**Files:**
- Modify: `internal/boardwatch/boardwatch.go`
- Create: `internal/boardwatch/state_test.go`

**Interfaces:**
- Produces:
  - consts `JobName = "repokit-board-watch"`, `ScriptName = "repokit-board-watch.py"`, `MarkerName = "repokit-board-watch.json"`, `Schedule = "every 5m"`, `Workdir = "/workspace"`, `Prompt` (string).
  - `type Spec struct { Prompt, Schedule, MonitorScript, Workdir, Deliver, ScriptSHA string }` (JSON tags snake_case).
  - `type Job struct { ID, Name, Paused bool; Spec }` — `Job{ID string; Name string; Paused bool; Spec Spec}`.
  - `type Marker struct { JobID string; Spec Spec }`.
  - `func Desired(deliver string) Spec`.
  - `type Decision string` with `Create, Upgrade, Current, Paused, OwnerModified, OptedOut, OwnerJob`.
  - `func Decide(job *Job, marker *Marker, diskSHA string, want Spec) Decision`.
  - `func Read(state *os.Root) (*Job, *Marker, string, error)` — state is the `.hermes` root; returns the job named `JobName` (nil if absent), the marker (nil if absent) and the on-disk script's SHA-256 (empty if absent).

- [ ] **Step 1: Write the failing test** (`internal/boardwatch/state_test.go`)

```go
package boardwatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecide(t *testing.T) {
	want := Desired("telegram")
	ours := &Marker{JobID: "j1", Spec: want}
	job := &Job{ID: "j1", Name: JobName, Spec: want}
	older := want
	older.Prompt = "old prompt"
	olderMarker := &Marker{JobID: "j1", Spec: older}
	olderJob := &Job{ID: "j1", Name: JobName, Spec: older}
	edited := *job
	edited.Spec.Schedule = "every 1h"
	paused := *job
	paused.Paused = true
	for _, tc := range []struct {
		name   string
		job    *Job
		marker *Marker
		disk   string
		want   Decision
	}{
		{"fresh", nil, nil, "", Create},
		{"owner's own job", job, nil, want.ScriptSHA, OwnerJob},
		{"current", job, ours, want.ScriptSHA, Current},
		{"owner removed it", nil, ours, want.ScriptSHA, OptedOut},
		{"owner paused it", &paused, ours, want.ScriptSHA, Paused},
		{"owner edited job", &edited, ours, want.ScriptSHA, OwnerModified},
		{"owner edited script", job, ours, "deadbeef", OwnerModified},
		{"older release", olderJob, olderMarker, older.ScriptSHA, Upgrade},
		{"delivery target changed", job, ours, want.ScriptSHA, Current},
	} {
		w := want
		if tc.name == "delivery target changed" {
			w = Desired("discord")
			if got := Decide(tc.job, tc.marker, tc.disk, w); got != Upgrade {
				t.Errorf("%s: got %s want %s", tc.name, got, Upgrade)
			}
			continue
		}
		if got := Decide(tc.job, tc.marker, tc.disk, w); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func TestReadHostState(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cron"), 0700)
	os.MkdirAll(filepath.Join(dir, "scripts"), 0700)
	jobs := `{"jobs":[{"id":"x1","name":"other"},{"id":"j1","name":"repokit-board-watch","prompt":"p","schedule_display":"every 5m","monitor_script":"repokit-board-watch.py","workdir":"/workspace","deliver":"telegram","enabled":false,"state":"paused"}]}`
	os.WriteFile(filepath.Join(dir, "cron", "jobs.json"), []byte(jobs), 0600)
	os.WriteFile(filepath.Join(dir, "scripts", ScriptName), Script, 0600)
	os.WriteFile(filepath.Join(dir, MarkerName), []byte(`{"job_id":"j1","spec":{"prompt":"p"}}`), 0600)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	job, marker, sha, err := Read(root)
	if err != nil || job == nil || job.ID != "j1" || !job.Paused || job.Spec.Deliver != "telegram" || marker == nil || marker.JobID != "j1" || sha != Desired("").ScriptSHA {
		t.Fatalf("read: %+v %+v %q %v", job, marker, sha, err)
	}
	empty, _ := os.OpenRoot(t.TempDir())
	if job, marker, sha, err := Read(empty); job != nil || marker != nil || sha != "" || err != nil {
		t.Fatalf("absent state: %+v %+v %q %v", job, marker, sha, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/boardwatch/ -run 'TestDecide|TestReadHostState' -v`
Expected: FAIL to compile (`undefined: Desired`).

- [ ] **Step 3: Implement** (append to `internal/boardwatch/boardwatch.go`)

```go
import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

const (
	JobName    = "repokit-board-watch"
	ScriptName = "repokit-board-watch.py"
	MarkerName = "repokit-board-watch.json"
	Schedule   = "every 5m"
	Workdir    = "/workspace"
)

// Prompt is what default reads when the watch wakes it; the monitor digest
// follows it in the run context.
const Prompt = `RepoKit board watch woke you: this board is idle and the digest below names open goal cards (title "Goal:"), cards in triage and blocked cards. Do one bounded pass, then stop.
1. A triage card whose cause is now resolved (an owner decision recorded on it, a capability added): return it with "hermes kanban specify <id>", then immediately restore its original title and body with "hermes kanban edit <id> --title <original title> --body <original body>". Otherwise leave it.
2. Each goal card: read it and its comments. If its acceptance is met with evidence, complete it with that evidence. If its budget is spent, block it with one question for the owner. Otherwise create the next card toward it (after checking the board for one that already covers the work), then link it as a parent of the goal card with kanban_link.
3. Blocked cards waiting on the owner: once a day, report them in one short message, skipping any whose comments snooze them past today.
Never create work that no goal card asks for. If nothing needs doing or saying, reply exactly [SILENT].`

// Spec is what RepoKit manages on the job and its script.
type Spec struct {
	Prompt        string `json:"prompt"`
	Schedule      string `json:"schedule"`
	MonitorScript string `json:"monitor_script"`
	Workdir       string `json:"workdir"`
	Deliver       string `json:"deliver"`
	ScriptSHA     string `json:"script_sha256"`
}

// Job is the native cron job named JobName.
type Job struct {
	ID     string
	Name   string
	Paused bool
	Spec   Spec
}

// Marker records what RepoKit last wrote; its presence means RepoKit created
// the job, so a missing job is the owner's removal.
type Marker struct {
	JobID string `json:"job_id"`
	Spec  Spec   `json:"spec"`
}

type Decision string

const (
	Create        Decision = "create"
	Upgrade       Decision = "upgrade"
	Current       Decision = "current"
	Paused        Decision = "paused"
	OwnerModified Decision = "owner-modified"
	OptedOut      Decision = "opted-out"
	OwnerJob      Decision = "owner-job"
)

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// Desired is the job this release installs, delivering to deliver ("local"
// when the deployment has no messaging platform).
func Desired(deliver string) Spec {
	return Spec{Prompt: Prompt, Schedule: Schedule, MonitorScript: ScriptName, Workdir: Workdir, Deliver: deliver, ScriptSHA: sha(Script)}
}

// Decide never rewrites what the owner changed: a job RepoKit did not create,
// a paused or edited job, an edited script, or a removed job.
func Decide(job *Job, marker *Marker, diskSHA string, want Spec) Decision {
	switch {
	case marker == nil && job == nil:
		return Create
	case marker == nil:
		return OwnerJob
	case job == nil || job.ID != marker.JobID:
		return OptedOut
	case job.Paused:
		return Paused
	}
	installed := job.Spec
	installed.ScriptSHA = diskSHA
	if installed != marker.Spec {
		return OwnerModified
	}
	if marker.Spec != want {
		return Upgrade
	}
	return Current
}

// Read observes the job, marker and script from the host's .hermes root.
func Read(state *os.Root) (*Job, *Marker, string, error) {
	var job *Job
	if data, err := state.ReadFile("cron/jobs.json"); err == nil {
		var store struct {
			Jobs []struct {
				ID, Name, Prompt, Workdir, Deliver, State string
				ScheduleDisplay string `json:"schedule_display"`
				MonitorScript   string `json:"monitor_script"`
				Enabled         *bool
			}
		}
		if err := json.Unmarshal(data, &store); err != nil {
			return nil, nil, "", errors.New("cron job store unreadable")
		}
		for _, j := range store.Jobs {
			if j.Name == JobName {
				paused := j.State == "paused" || (j.Enabled != nil && !*j.Enabled)
				job = &Job{ID: j.ID, Name: j.Name, Paused: paused, Spec: Spec{Prompt: j.Prompt, Schedule: j.ScheduleDisplay, MonitorScript: j.MonitorScript, Workdir: j.Workdir, Deliver: j.Deliver}}
				break
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	var marker *Marker
	if data, err := state.ReadFile(MarkerName); err == nil {
		marker = &Marker{}
		if json.Unmarshal(data, marker) != nil {
			return nil, nil, "", errors.New("board watch marker unreadable")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	disk := ""
	if data, err := state.ReadFile("scripts/" + ScriptName); err == nil {
		disk = sha(data)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	if job != nil {
		job.Spec.ScriptSHA = disk
	}
	return job, marker, disk, nil
}
```

Note: `Job.Spec.ScriptSHA` is filled from disk so `Decide`'s comparison sees the installed script; `Decide` overwrites it with `diskSHA` anyway.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/boardwatch/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/boardwatch/boardwatch.go internal/boardwatch/state_test.go
git commit -m "feat: decide board watch install, upgrade and owner opt-out"
```

### Task 3: Apply the decision in the container

**Files:**
- Create: `internal/native/boardwatch.go`
- Create: `internal/native/boardwatch_test.go`

**Interfaces:**
- Consumes: `boardwatch.Read`, `boardwatch.Decide`, `boardwatch.Desired`, `boardwatch.Script`, consts; `chatPlatforms`, `nativeTeamCLI`, `InputRunner` (existing in `internal/native`).
- Produces: `func EnsureBoardWatch(ctx context.Context, id target.Identity, dc string, r InputRunner) (boardwatch.Decision, error)`.

- [ ] **Step 1: Write the failing test** (`internal/native/boardwatch_test.go`)

```go
package native

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/boardwatch"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// watchContainer simulates the container: file writes land in the host
// .hermes, and `cron create` records a job in jobs.json.
type watchContainer struct {
	state string
	calls []string
}

func (w *watchContainer) RunInput(_ context.Context, in io.Reader, _ string, args ...string) process.Result {
	line := strings.Join(args, " ")
	w.calls = append(w.calls, line)
	switch {
	case slices.Contains(args, "send"):
		return process.Result{Output: "Telegram:\n  telegram:Owner (dm)\n"}
	case slices.Contains(args, "sh"):
		data, _ := io.ReadAll(in)
		path := args[len(args)-1]
		os.MkdirAll(filepath.Dir(filepath.Join(w.state, strings.TrimPrefix(path, "/opt/data/"))), 0700)
		os.WriteFile(filepath.Join(w.state, strings.TrimPrefix(path, "/opt/data/")), data, 0600)
	case slices.Contains(args, "create"):
		os.MkdirAll(filepath.Join(w.state, "cron"), 0700)
		want := boardwatch.Desired("telegram")
		os.WriteFile(filepath.Join(w.state, "cron", "jobs.json"), []byte(`{"jobs":[{"id":"j1","name":"repokit-board-watch","prompt":`+quoteJSON(want.Prompt)+`,"schedule_display":"every 5m","monitor_script":"repokit-board-watch.py","workdir":"/workspace","deliver":"telegram","enabled":true,"state":"scheduled"}]}`), 0600)
	}
	return process.Result{}
}

func TestEnsureBoardWatchCreatesOnceThenLeavesIt(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hermes"), 0700)
	id := target.Identity{Root: root, Compose: filepath.Join(root, ".hermes", "compose.yaml")}
	c := &watchContainer{state: filepath.Join(root, ".hermes")}
	got, err := EnsureBoardWatch(context.Background(), id, "default", c)
	if err != nil || got != boardwatch.Create {
		t.Fatalf("first run: %s %v\n%s", got, err, strings.Join(c.calls, "\n"))
	}
	created := strings.Join(c.calls, "\n")
	for _, want := range []string{"cron create every 5m", "--name repokit-board-watch", "--monitor-script repokit-board-watch.py", "--workdir /workspace", "--deliver telegram", "--failure-deliver local"} {
		if !strings.Contains(created, want) {
			t.Errorf("create call missing %q:\n%s", want, created)
		}
	}
	c.calls = nil
	if got, err := EnsureBoardWatch(context.Background(), id, "default", c); err != nil || got != boardwatch.Current || strings.Contains(strings.Join(c.calls, "\n"), "cron") {
		t.Fatalf("second run must change nothing: %s %v %v", got, err, c.calls)
	}
	os.Remove(filepath.Join(root, ".hermes", "cron", "jobs.json"))
	if got, _ := EnsureBoardWatch(context.Background(), id, "default", c); got != boardwatch.OptedOut {
		t.Fatalf("a removed job is the owner's opt-out, got %s", got)
	}
}
```

Add a helper in the same test file:

```go
func quoteJSON(s string) string { b, _ := json.Marshal(s); return string(b) }
```
(with `"encoding/json"` imported).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/native/ -run TestEnsureBoardWatch -v`
Expected: FAIL to compile (`undefined: EnsureBoardWatch`).

- [ ] **Step 3: Implement** (`internal/native/boardwatch.go`)

```go
package native

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/boardwatch"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// EnsureBoardWatch installs or upgrades RepoKit's board watch cron job on
// default through public Hermes commands, and leaves alone a job the owner
// created, paused, edited or removed.
func EnsureBoardWatch(ctx context.Context, id target.Identity, dc string, r InputRunner) (boardwatch.Decision, error) {
	state, err := os.OpenRoot(filepath.Join(id.Root, ".hermes"))
	if err != nil {
		return "", err
	}
	defer state.Close()
	job, marker, disk, err := boardwatch.Read(state)
	if err != nil {
		return "", err
	}
	run := nativeTeamCLI(ctx, id, dc, r)
	deliver := "local"
	if platforms := chatPlatforms(run); len(platforms) > 0 {
		deliver = platforms[0]
	}
	want := boardwatch.Desired(deliver)
	decision := boardwatch.Decide(job, marker, disk, want)
	if decision != boardwatch.Create && decision != boardwatch.Upgrade {
		return decision, nil
	}
	if err := writeState(ctx, id, dc, r, "/opt/data/scripts/"+boardwatch.ScriptName, boardwatch.Script); err != nil {
		return "", err
	}
	fields := []string{"--prompt", want.Prompt, "--monitor-script", want.MonitorScript, "--workdir", want.Workdir, "--deliver", want.Deliver, "--failure-deliver", "local"}
	if decision == boardwatch.Create {
		args := append([]string{"-p", "default", "cron", "create", want.Schedule, "--name", boardwatch.JobName}, fields[2:]...)
		if _, err := run(append(args, want.Prompt)...); err != nil {
			return "", errors.New("board watch job not created")
		}
		if job, _, _, err = boardwatch.Read(state); err != nil || job == nil {
			return "", errors.New("board watch job not found after creation")
		}
	} else {
		args := append([]string{"-p", "default", "cron", "edit", job.ID, "--schedule", want.Schedule}, fields...)
		if _, err := run(args...); err != nil {
			return "", errors.New("board watch job not upgraded")
		}
	}
	data, _ := json.Marshal(boardwatch.Marker{JobID: job.ID, Spec: want})
	if err := writeState(ctx, id, dc, r, "/opt/data/"+boardwatch.MarkerName, data); err != nil {
		return "", err
	}
	return decision, nil
}

// writeState writes one private file under default's state, 0600, atomically.
func writeState(ctx context.Context, id target.Identity, dc string, r InputRunner, path string, content []byte) error {
	script := `umask 077; mkdir -p "$(dirname "$1")" && cat > "$1.tmp" && chmod 0600 "$1.tmp" && mv -f "$1.tmp" "$1"`
	res := r.RunInput(ctx, bytes.NewReader(content), "docker", "--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes", "hermes", "sh", "-c", script, "sh", path)
	if res.Err != nil {
		return errors.New("could not write " + path)
	}
	return nil
}
```

Note the create argv: `cron create every 5m --name repokit-board-watch --monitor-script … --workdir … --deliver … --failure-deliver local <prompt>` (`fields[2:]` drops `--prompt <prompt>`, which `create` takes positionally).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/native/ -v -run TestEnsureBoardWatch` then `go test ./internal/native/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/native/boardwatch.go internal/native/boardwatch_test.go
git commit -m "feat: install and upgrade the board watch job through Hermes cron"
```

### Task 4: Wire install and verify

**Files:**
- Modify: `internal/cli/install.go` (after `u.team(teamReport)`)
- Create: `internal/verify/boardwatch.go`
- Create: `internal/verify/boardwatch_test.go`
- Modify: `internal/verify/verify.go` (`Inspect` appends the probe)

**Interfaces:**
- Consumes: `native.EnsureBoardWatch`, `boardwatch.Read/Decide/Desired`.
- Produces: `func BoardWatch(id target.Identity) Probe` in package verify.

- [ ] **Step 1: Write the failing test** (`internal/verify/boardwatch_test.go`)

```go
package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestBoardWatchProbe(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hermes"), 0700)
	id := target.Identity{Root: root}
	if p := BoardWatch(id); p.Component != "board-watch" || p.Status != Unknown {
		t.Fatalf("absent: %+v", p)
	}
	os.WriteFile(filepath.Join(root, ".hermes", "repokit-board-watch.json"), []byte(`{"job_id":"j1","spec":{}}`), 0600)
	if p := BoardWatch(id); p.Status != Unknown || p.Detail != "owner removed the board watch job; RepoKit leaves it off" {
		t.Fatalf("opted out: %+v", p)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/verify/ -run TestBoardWatchProbe -v`
Expected: FAIL to compile.

- [ ] **Step 3: Implement** (`internal/verify/boardwatch.go`)

```go
package verify

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/boardwatch"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// BoardWatch reports RepoKit's board watch job from host state. The owner's
// choice to pause, edit or remove it is reported, never treated as a failure.
func BoardWatch(id target.Identity) Probe {
	state, err := os.OpenRoot(filepath.Join(id.Root, ".hermes"))
	if err != nil {
		return Probe{"board-watch", Unknown, "native state unavailable"}
	}
	defer state.Close()
	job, marker, disk, err := boardwatch.Read(state)
	if err != nil {
		return Probe{"board-watch", Unknown, err.Error()}
	}
	deliver := "local"
	if job != nil {
		deliver = job.Spec.Deliver
	}
	switch boardwatch.Decide(job, marker, disk, boardwatch.Desired(deliver)) {
	case boardwatch.Current:
		return Probe{"board-watch", Healthy, "wakes default every 5 minutes only when the board is quiet with an open goal or stuck card"}
	case boardwatch.Upgrade:
		return Probe{"board-watch", Unknown, "an earlier release's board watch; install upgrades it"}
	case boardwatch.Paused:
		return Probe{"board-watch", Unknown, "owner paused the board watch job"}
	case boardwatch.OwnerModified:
		return Probe{"board-watch", Unknown, "owner changed the board watch job or script; RepoKit leaves it as is"}
	case boardwatch.OptedOut:
		return Probe{"board-watch", Unknown, "owner removed the board watch job; RepoKit leaves it off"}
	case boardwatch.OwnerJob:
		return Probe{"board-watch", Unknown, "a job named repokit-board-watch that RepoKit did not create"}
	}
	return Probe{"board-watch", Unknown, "not installed yet; install adds it once the team is configured"}
}
```

In `internal/verify/verify.go`, at the end of `Inspect` before its return, append `BoardWatch(id)` to the probes slice it returns (read the function first and follow its variable name).

In `internal/cli/install.go`, right after `u.team(teamReport)`:

```go
	if decision, err := native.EnsureBoardWatch(context.Background(), id, dockerContext, runner); err != nil {
		u.note("board watch not installed: " + err.Error())
	} else if decision == boardwatch.Create || decision == boardwatch.Upgrade {
		u.ok("Board watch", "default wakes when the board is quiet with an open goal or a stuck card")
	}
```
(import `github.com/TrebuchetDynamics/hermes-repokit/internal/boardwatch`).

- [ ] **Step 4: Run the full gate**

Run: `go vet ./... && test -z "$(gofmt -l $(git ls-files '*.go'))" && go test ./...`
Expected: PASS (update any install test that pins the exact command sequence to accept the new board-watch calls).

- [ ] **Step 5: Commit**

```bash
git add internal/cli/install.go internal/verify/boardwatch.go internal/verify/boardwatch_test.go internal/verify/verify.go
git commit -m "feat: install the board watch with the team and report it in verify"
```

### Task 5: Goal cards and the goal budget

**Files:**
- Modify: `internal/team/souls/default.md`
- Modify: `internal/team/team.go` (`coordinator` settings)
- Modify: `internal/native/team_go_test.go` (reset key list)
- Modify: `docs/team-model.md`

- [ ] **Step 1: Update the pinned reset test** — in `TestResetDefaultRestoresIdentityAndGrantedSettingsOnly` change the expected keys to `"approvals.mode,goals.max_turns,kanban.dispatch_interval_seconds,security.protected_instruction_files"`.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/native/ -run TestResetDefaultRestoresIdentityAndGrantedSettingsOnly`
Expected: FAIL (keys differ).

- [ ] **Step 3: Implement**

`internal/team/team.go`:
```go
var coordinator = settings(noApprovals, map[string]any{"kanban.dispatch_interval_seconds": 10, "goals.max_turns": 100})
```
and extend its comment: "A chat /goal gets 100 continuations instead of Hermes's 20, so a long program does not silently expire."

`internal/team/souls/default.md`, after the "Never wait on work that is not on this team's board" paragraph:
```markdown
Record each goal the owner states as a goal card, so it outlives this chat:
kanban_create with a title beginning "Goal:", assignee default and
initial_status "blocked", whose body holds the outcome, acceptance, a budget
(a number of cards or an end date) and the owner's constraints. Never
dispatch it. Link each card that works toward it as its parent
(kanban_link parent=<card> child=<goal>). Record later constraints and snoozes
as comments on it. RepoKit's board watch wakes you when the board is quiet and
a goal can advance.
```

`docs/team-model.md`: add a "Board watch" paragraph after the settings paragraph describing goal cards, the 5-minute watch, `busy` suppression, back-off buckets, opt-out by pausing/editing/removing the job, and `goals.max_turns: 100`.

- [ ] **Step 4: Run the full gate**

Run: `go vet ./... && test -z "$(gofmt -l $(git ls-files '*.go'))" && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/team/souls/default.md internal/team/team.go internal/native/team_go_test.go docs/team-model.md
git commit -m "feat: record owner goals as goal cards and give chat goals 100 turns"
```

### Task 6: Live proof on the dogfood deployment, then rollout

- [ ] **Step 1:** Push, build `scratchpad/repokit-next`, run `repokit-next install --reset-profile default` in `/home/xel/git/hermes-repokit`. Expected: `Board watch` line; `hermes -p default cron list` shows `repokit-board-watch`; `.hermes/scripts/repokit-board-watch.py` and `.hermes/repokit-board-watch.json` are 0600; `repokit-next verify` reports `board-watch healthy`.
- [ ] **Step 2:** Busy suppression: with no goal cards, wait two ticks; `cron runs` shows the job, the job's output dir shows `no_change (agent run suppressed)` after the baseline.
- [ ] **Step 3:** Goal flow: create a throwaway goal card from the CLI with no assignee, so it is never dispatched (`hermes kanban create "Goal: RepoKit watch proof (ignore)" --body "Acceptance: one card titled 'Watch proof child (ignore)' exists, assigned to researcher, whose body is 'Reply DONE and complete.'. Budget: 1 card."`, then `hermes kanban block <id> "goal card"`). Do not shorten the job's schedule (that would mark it owner-modified); wait for the 30m bucket. Expected: the woken coordinator creates the child, links it, the child completes, and the next wake completes the goal card. Archive both afterwards.
- [ ] **Step 4:** Opt-out: `hermes cron pause <id>`; `repokit-next install` reports nothing new and `verify` says paused; `hermes cron resume <id>`.
- [ ] **Step 5:** Roll out (`install --reset-profile default` per deployment, waiting for idle) to wing, polymarket-mega-bot and sdrhf; `verify` shows `board-watch healthy` on each.

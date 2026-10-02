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

package native

import (
	"context"
	"encoding/json"
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
		os.WriteFile(filepath.Join(w.state, "cron", "jobs.json"), []byte(`{"jobs":[{"id":"j1","name":"repokit-board-watch","prompt":`+quoteJSON(want.Prompt)+`,"schedule_display":"every 5m","monitor_script":"repokit-board-watch.py","workdir":null,"deliver":"telegram","enabled":true,"state":"scheduled"}]}`), 0600)
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
	if _, create, _ := strings.Cut(created, "cron create"); strings.Contains(create, "--workdir") {
		t.Errorf("the job must not load the repository's context files:\n%s", created)
	}
	for _, want := range []string{"cron create every 5m", "--name repokit-board-watch", "--monitor-script repokit-board-watch.py", "--deliver telegram", "--failure-deliver local"} {
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

func quoteJSON(s string) string { b, _ := json.Marshal(s); return string(b) }

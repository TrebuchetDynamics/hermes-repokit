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

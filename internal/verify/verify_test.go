package verify

import (
	"context"
	"errors"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fake struct {
	calls  [][]string
	result process.Result
}

func (f *fake) Run(_ context.Context, p string, a ...string) process.Result {
	f.calls = append(f.calls, append([]string{p}, a...))
	return f.result
}
func TestMissingStateAndDockerFailureRemainIndependent(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	r := &fake{result: process.Result{Err: errors.New("unavailable")}}
	out := Inspect(context.Background(), id, r)
	if out[0].Status != PendingSetup || out[1].Status != Unknown {
		t.Fatalf("%+v", out)
	}
	for _, c := range r.calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "hermes kanban") || strings.Contains(joined, " up ") {
			t.Fatal("mutating probe")
		}
	}
}
func TestVerifyRejectsSymlinkWithoutReadingReceipt(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	os.Mkdir(filepath.Join(id.Root, ".hermes"), 0700)
	os.Symlink("missing", id.Compose)
	os.WriteFile(filepath.Join(id.Root, ".hermes/repokit-install.json"), []byte("corrupt"), 0600)
	out := Inspect(context.Background(), id, &fake{})
	if out[0].Status != Degraded {
		t.Fatalf("%+v", out)
	}
}
func TestRuntimeStateDoesNotClaimModelOrMemoryOperation(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	data, e := launcher.Render(id, "default")
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(filepath.Dir(id.Launcher), 0700)
	os.WriteFile(id.Launcher, data, 0700)
	r := &fake{result: process.Result{Output: `{"status":"running","project":"` + id.Project + `","workspace":"` + id.Root + `","home":"` + filepath.Join(id.Root, ".hermes") + `"}`}}
	out := Inspect(context.Background(), id, r)
	if out[1].Status != Healthy {
		t.Fatalf("%+v", out)
	}
	for _, p := range out[2:] {
		if p.Component == "openviking" || p.Component == "nerve-laya" || p.Component == "superpowers" {
			t.Fatalf("unproved integration: %+v", p)
		}
	}
}

func TestVerifyUsesDeploymentContextInsteadOfAmbientContext(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	data, e := launcher.Render(id, "repo-local")
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(filepath.Dir(id.Launcher), 0700)
	os.WriteFile(id.Launcher, data, 0700)
	r := &fake{}
	Inspect(context.Background(), id, r)
	if len(r.calls) != 1 || !strings.HasPrefix(strings.Join(r.calls[0], " "), "docker --context repo-local container inspect") {
		t.Fatalf("ambient Docker used: %v", r.calls)
	}
}

func TestKanbanMetadataReportsPendingWithoutInitializing(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	os.Mkdir(filepath.Join(id.Root, ".hermes"), 0700)
	probe := func() Probe {
		for _, p := range Inspect(context.Background(), id, &fake{}) {
			if p.Component == "kanban" {
				return p
			}
		}
		t.Fatal("missing Kanban observation")
		return Probe{}
	}
	if p := probe(); p.Status != PendingSetup {
		t.Fatalf("missing board: %+v", p)
	}
	board := filepath.Join(id.Root, ".hermes/kanban.db")
	if _, err := os.Stat(board); !os.IsNotExist(err) {
		t.Fatal("verify created board")
	}
	os.WriteFile(board, nil, 0600)
	if p := probe(); p.Status != Degraded {
		t.Fatalf("empty board: %+v", p)
	}
	os.WriteFile(board, []byte("metadata-only fixture; not a qualified DB"), 0600)
	if p := probe(); p.Status != Healthy || !strings.Contains(p.Detail, "not opened") {
		t.Fatalf("metadata scope: %+v", p)
	}
}

package verify

import (
	"context"
	"errors"
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
	r := &fake{result: process.Result{Output: `{"status":"running","project":"` + id.Project + `","workspace":"` + id.Root + `","home":"` + filepath.Join(id.Root, ".hermes") + `"}`}}
	out := Inspect(context.Background(), id, r)
	if out[1].Status != Healthy {
		t.Fatalf("%+v", out)
	}
	for _, p := range out[2:] {
		if p.Status == Healthy {
			t.Fatalf("unproved integration: %+v", p)
		}
	}
}

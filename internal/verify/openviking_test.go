package verify

import (
	"context"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memoryObservation struct {
	output string
	calls  []string
}

func (r *memoryObservation) Run(_ context.Context, p string, args ...string) process.Result {
	call := p + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	if strings.Contains(call, "container ls") {
		return process.Result{Output: "abcdef123456"}
	}
	if strings.Contains(call, "container inspect") {
		return process.Result{Output: r.output}
	}
	return process.Result{Err: fmt.Errorf("not a metadata command")}
}
func TestOpenVikingObservationNeverProbesModels(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	os.MkdirAll(filepath.Dir(id.Launcher), 0700)
	launch, _ := launcher.Render(id, "local")
	os.WriteFile(id.Launcher, launch, 0700)
	os.Mkdir(filepath.Join(id.Root, ".hermes/openviking"), 0700)
	r := &memoryObservation{output: fmt.Sprintf(`{"status":"running","health":"healthy","image":%q,"project":%q,"service":"openviking","home":%q}`, projectmemory.Image, id.Project, filepath.Join(id.Root, ".hermes/openviking"))}
	for _, configured := range []bool{false, true} {
		if configured {
			os.WriteFile(filepath.Join(id.Root, ".hermes/openviking/ov.conf"), []byte("secret configuration must never be read"), 0600)
		}
		probes := OpenViking(context.Background(), id, r)
		for _, probe := range probes {
			if probe.Component == "openviking-config" {
				want := PendingSetup
				if configured {
					want = Healthy
				}
				if probe.Status != want {
					t.Fatal(probe)
				}
			} else if probe.Status != PendingSetup {
				t.Fatal("legacy sidecar accepted as embedded memory", probe)
			}
		}
	}
	if len(r.calls) != 0 {
		t.Fatal("legacy sidecar was probed instead of reporting topology drift")
	}
}

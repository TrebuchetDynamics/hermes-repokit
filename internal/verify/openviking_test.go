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
		status := map[string]Status{}
		for _, p := range probes {
			status[p.Component] = p.Status
		}
		if status["openviking"] != Unknown {
			t.Fatal("metadata promoted live memory acceptance")
		}
		want := PendingSetup
		if configured {
			want = Healthy
		}
		if status["openviking-config"] != want || status["openviking-runtime"] != Healthy {
			t.Fatalf("unexpected observation: %v", probes)
		}
	}
	for _, call := range r.calls {
		if strings.Contains(call, " exec ") || strings.Contains(call, "/ready") || strings.Contains(call, "ov.conf") {
			t.Fatal("verification executed native service or read private config")
		}
	}
	for _, health := range []string{"starting", "unhealthy"} {
		badHealth := *r
		badHealth.output = strings.ReplaceAll(r.output, `"health":"healthy"`, `"health":"`+health+`"`)
		states := map[string]Status{}
		for _, p := range OpenViking(context.Background(), id, &badHealth) {
			states[p.Component] = p.Status
		}
		if states["openviking-container"] != Healthy || states["openviking-runtime"] != Degraded {
			t.Fatalf("matching running service must allow native diagnosis despite health=%s: %v", health, states)
		}
	}
	for _, change := range [][2]string{
		{`"status":"running"`, `"status":"exited"`},
		{id.Project, "different-project"},
		{filepath.Join(id.Root, ".hermes/openviking"), "/owner/other-memory"},
	} {
		other := *r
		other.output = strings.ReplaceAll(r.output, change[0], change[1])
		for _, p := range OpenViking(context.Background(), id, &other) {
			if (p.Component == "openviking-container" || p.Component == "openviking-runtime") && p.Status != Degraded {
				t.Fatalf("mismatched or stopped service accepted: %+v", p)
			}
		}
	}
	r.output = strings.ReplaceAll(r.output, projectmemory.Image, "owner:latest")
	for _, p := range OpenViking(context.Background(), id, r) {
		if p.Component == "openviking-runtime" && p.Status != Degraded {
			t.Fatal("wrong image accepted")
		}
	}
}

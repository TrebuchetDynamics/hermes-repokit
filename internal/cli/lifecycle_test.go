package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// liveCompose simulates Compose changing the container the fake runner reports.
func liveCompose(a *App, r *foundationRunner) {
	a.ComposeExec = func(_, _ io.Writer, args ...string) error {
		call := strings.Join(args, " ")
		r.composeCalls = append(r.composeCalls, call)
		switch {
		case strings.HasSuffix(call, " up -d --build hermes"):
			r.runtime = developmentRuntimeFixture(r.id)
		case strings.HasSuffix(call, " stop"):
			r.runtime = strings.Replace(developmentRuntimeFixture(r.id), `"running"`, `"exited"`, 1)
		}
		return nil
	}
}

// statsInput answers the public CLI reads used by start/install.
type statsInput struct {
	*foundationRunner
	running bool
}

func (s *statsInput) RunInput(ctx context.Context, input io.Reader, p string, args ...string) process.Result {
	if input == nil && strings.HasSuffix(strings.Join(args, " "), "kanban stats --json") {
		if s.running {
			return process.Result{Output: `{"by_status":{"running":1}}`}
		}
		return process.Result{Output: `{"by_status":{}}`}
	}
	return s.foundationRunner.RunInput(ctx, input, p, args...)
}

func TestInstallBuildsStartsAndInitializes(t *testing.T) {
	a, r := foundationApp(t)
	liveCompose(&a, r)
	code, out, diag := invoke(t, a, "install")
	if code != 0 || len(r.composeCalls) != 1 || !strings.Contains(out, "hermes-test-project running") || !strings.Contains(out, "not set up yet") {
		t.Fatalf("install did not start and initialize: code=%d calls=%v out=%s diag=%s", code, r.composeCalls, out, diag)
	}
	if _, err := os.Stat(filepath.Join(a.Directory, ".hermes/kanban.db")); err != nil {
		t.Fatal("Kanban not initialized after start")
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 || len(r.composeCalls) != 1 {
		t.Fatalf("rerun touched a running current deployment: %v %s", r.composeCalls, diag)
	}
}

func TestInstallDefersRecreateWhileWorkRuns(t *testing.T) {
	a, r := foundationApp(t)
	liveCompose(&a, r)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	// Running container on an older generated image, with a card in flight.
	o, _ := compose.DevelopmentSelected(r.id)
	current := development.ImageName(r.id.Container, *o.Development)
	r.runtime = strings.Replace(developmentRuntimeFixture(r.id), current, "repokit/hermes-test-project:0123456789abcdef01234567", 1)
	a.Initializer = &statsInput{foundationRunner: r, running: true}
	r.composeCalls = nil
	code, out, diag := invoke(t, a, "install")
	if len(r.composeCalls) != 0 || !strings.Contains(out, "a Kanban card is running") {
		t.Fatalf("recreated over running work: code=%d calls=%v out=%s diag=%s", code, r.composeCalls, out, diag)
	}
}

func TestStopAndStartRoundTrip(t *testing.T) {
	a, r := foundationApp(t)
	liveCompose(&a, r)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	a.Initializer = &statsInput{foundationRunner: r}
	code, out, diag := invoke(t, a, "stop")
	if code != 0 || !strings.HasSuffix(r.composeCalls[len(r.composeCalls)-1], "-f "+r.id.Compose+" --profile docker-tests stop") || !strings.Contains(out, "state is preserved") {
		t.Fatalf("stop: code=%d calls=%v out=%s diag=%s", code, r.composeCalls, out, diag)
	}
	if _, err := os.Stat(filepath.Join(a.Directory, ".hermes/compose.yaml")); err != nil {
		t.Fatal("stop deleted state")
	}
	code, out, diag = invoke(t, a, "start")
	if code != 0 || !strings.HasSuffix(r.composeCalls[len(r.composeCalls)-1], " up -d --build hermes") || !strings.Contains(out, "answering") {
		t.Fatalf("start: code=%d calls=%v out=%s diag=%s", code, r.composeCalls, out, diag)
	}
	// The team was never set up, so there is no gateway to check yet.
	if !strings.Contains(out, "setup  set up the team") || strings.Contains(out, "gateway status") {
		t.Fatalf("start on a deployment without a team did not point at setup:\n%s", out)
	}
	calls := len(r.composeCalls)
	if code, out, _ := invoke(t, a, "start"); code != 0 || len(r.composeCalls) != calls || !strings.Contains(out, "already running") {
		t.Fatalf("start on a running deployment changed it: %v %s", r.composeCalls, out)
	}
}

func TestLifecycleRefusesForeignDeployments(t *testing.T) {
	for _, command := range []string{"start", "stop"} {
		a, r := foundationApp(t)
		liveCompose(&a, r)
		if code, _, diag := invoke(t, a, "install"); code != 0 {
			t.Fatal(diag)
		}
		r.composeCalls = nil
		r.runtime = strings.ReplaceAll(developmentRuntimeFixture(r.id), r.id.Project, "someone-else")
		if code, _, diag := invoke(t, a, command); code == 0 || len(r.composeCalls) != 0 {
			t.Fatalf("%s acted on a foreign container: %v %s", command, r.composeCalls, diag)
		}
		data, _ := os.ReadFile(r.id.Compose)
		os.WriteFile(r.id.Compose, append(data, []byte("# owner edit\n")...), 0600)
		r.runtime = ""
		if code, _, diag := invoke(t, a, command); code == 0 || len(r.composeCalls) != 0 || !strings.Contains(diag, "not a RepoKit-generated Compose") {
			t.Fatalf("%s acted on an edited Compose file: %v %s", command, r.composeCalls, diag)
		}
	}
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "stop"); code == 0 || len(r.composeCalls) != 0 || !strings.Contains(diag, "no RepoKit deployment") {
		t.Fatalf("stop without a deployment: %s", diag)
	}
}

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// gatewayInput simulates the public Hermes CLI and the locked bootstrap shell.
// Scripts return team; a gateway restart script replaces the gateway PID.
type gatewayInput struct {
	calls   int
	kanban  string
	pid     int
	team    string
	scripts []string
	err     error
}

func (r *gatewayInput) RunInput(_ context.Context, input io.Reader, program string, args ...string) process.Result {
	r.calls++
	joined := strings.Join(args, " ")
	if input != nil {
		body, _ := io.ReadAll(input)
		r.scripts = append(r.scripts, string(body))
		if !strings.Contains(joined, "/usr/bin/flock -n /workspace/.hermes-repokit.lock") {
			return process.Result{Err: fmt.Errorf("unlocked mutation")}
		}
		if strings.Contains(string(body), "'gateway' 'restart'") {
			if r.err != nil {
				return process.Result{Err: r.err, Output: "secret native output"}
			}
			r.pid++
		}
		return process.Result{Output: r.team}
	}
	switch {
	case strings.HasSuffix(joined, "config get kanban --json"):
		return process.Result{Output: r.kanban}
	case strings.HasSuffix(joined, "gateway status"):
		return process.Result{Output: fmt.Sprintf("✓ Gateway is running (PID: %d)", r.pid)}
	case strings.HasSuffix(joined, "kanban stats --json"):
		return process.Result{Output: `{"by_status":{}}`}
	case strings.Contains(joined, " config get "):
		return process.Result{Output: "null"}
	}
	return process.Result{Err: fmt.Errorf("unexpected native call")}
}

func TestGatewayFinalizationNeverRunsAfterFailedStage(t *testing.T) {
	app, r := foundationApp(t)
	if code, _, diag := invoke(t, app, "install"); code != 0 {
		t.Fatal(diag)
	}
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10}
	app.Initializer = input
	var out, diag bytes.Buffer
	if got := app.finishSetup(r.id, r.context, 7, &out, &diag); got != 7 || input.calls != 0 {
		t.Fatal("failed stage finalized")
	}
	if got := app.finishSetup(r.id, r.context, 0, &out, &diag); got != 0 || !strings.Contains(out.String(), "restarted the gateway") {
		t.Fatalf("finish: %d %s %s", got, &out, &diag)
	}
	if strings.Contains(out.String(), "canary completed") || !strings.Contains(out.String(), "No worker has been exercised") {
		t.Fatalf("setup claimed unexercised work: %s", &out)
	}
	input.kanban, input.err = `{"dispatch_in_gateway":false}`, fmt.Errorf("private-native-error")
	diag.Reset()
	if got := app.finishSetup(r.id, r.context, 0, &out, &diag); got == 0 {
		t.Fatal("failed gateway restart certified")
	}
	if strings.Contains(diag.String(), "secret") || strings.Contains(diag.String(), "private-native-error") {
		t.Fatalf("native output leaked: %s", &diag)
	}
}

func TestSetupPreservesOwnerChangedDispatchPolicy(t *testing.T) {
	app, r := foundationApp(t)
	if code, _, diag := invoke(t, app, "install"); code != 0 {
		t.Fatal(diag)
	}
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":true,"max_in_progress":5}`, pid: 10}
	app.Initializer = input
	var out, diag bytes.Buffer
	if got := app.finishSetup(r.id, r.context, 0, &out, &diag); got == 0 || len(input.scripts) != 0 {
		t.Fatal("owner dispatch policy overwritten")
	}
}

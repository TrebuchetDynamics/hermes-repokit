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
	starts  int
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
		if strings.Contains(string(body), "'gateway' 'start'") {
			r.starts++
			r.pid = 500
		}
		return process.Result{Output: r.team}
	}
	switch {
	case strings.HasSuffix(joined, "config get kanban --json"):
		return process.Result{Output: r.kanban}
	case strings.HasSuffix(joined, "gateway status"):
		if r.pid == 0 {
			return process.Result{Output: "✗ Gateway is not running"}
		}
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
	if got := app.finishSetup(r.id, r.context, 0, &out, &diag); got != 0 || !strings.Contains(out.String(), "native automatic dispatch configured") {
		t.Fatalf("finish: %d %s %s", got, &out, &diag)
	}
	if strings.Contains(out.String(), "canary completed") || !strings.Contains(out.String(), "no worker has run yet") {
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

func TestSetupStartsStoppedGatewayButInstallOnlyPrintsCommand(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 0, team: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	a.Initializer = input
	a.Stdin = strings.NewReader("")
	code, out, diag := invoke(t, a, "setup", "--team")
	if code != 0 || input.starts != 1 || input.pid == 0 || !strings.Contains(out, "default gateway started") {
		t.Fatalf("setup --team left a fresh gateway stopped: code=%d starts=%d out=%s diag=%s", code, input.starts, out, diag)
	}
	var buf, errs bytes.Buffer
	input.pid, input.starts = 0, 0
	if got := a.finishSetup(r.id, r.context, 0, &buf, &errs); got != 0 || input.starts != 0 ||
		!strings.Contains(buf.String(), r.id.Container+" -p default gateway start") {
		t.Fatalf("install path must not start an owner-stopped gateway and must print the command: code=%d starts=%d out=%s", got, input.starts, &buf)
	}
}

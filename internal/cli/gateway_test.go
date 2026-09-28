package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

type gatewayInput struct {
	calls int
	state string
	err   error
}

func (r *gatewayInput) RunInput(_ context.Context, input io.Reader, program string, args ...string) process.Result {
	r.calls++
	body, _ := io.ReadAll(input)
	if strings.Contains(string(body), "REPOKIT_MAINTENANCE_PY") {
		return process.Result{Output: "REPOKIT_MAINTENANCE=configured"}
	}
	if program != "docker" || !strings.Contains(string(body), "REPOKIT_GATEWAY_PY") || !strings.Contains(strings.Join(args, " "), "/usr/bin/flock -n /workspace/.hermes-repokit.lock") {
		return process.Result{Err: fmt.Errorf("unexpected mutation route")}
	}
	return process.Result{Output: "REPOKIT_GATEWAY=" + r.state + "\nREPOKIT_CANARY=researcher-done\nREPOKIT_DISPATCH=prepared", Err: r.err}
}
func TestGatewayFinalizationNeverRunsAfterFailedStage(t *testing.T) {
	app, r := foundationApp(t)
	if code, _, diag := invoke(t, app, "install"); code != 0 {
		t.Fatal(diag)
	}
	input := &gatewayInput{state: "current"}
	app.Initializer = input
	var out, diag bytes.Buffer
	if got := app.finishSetup(r.id, r.context, 7, &out, &diag); got != 7 || input.calls != 0 {
		t.Fatal("failed stage finalized")
	}
	if got := app.finishSetup(r.id, r.context, 0, &out, &diag); got != 0 || input.calls != 2 || !strings.Contains(diag.String(), "Optional memory") || !strings.Contains(out.String(), "operational") {
		t.Fatalf("finish: %d %s %s", got, &out, &diag)
	}
	input.err = fmt.Errorf("restart failed")
	input.state = "current\nREPOKIT_DISPATCH_FAILURE=RuntimeError|dispatch_main:400>switch_gateway:121"
	if got := app.finishSetup(r.id, r.context, 0, &out, &diag); got == 0 {
		t.Fatal("failed gateway certified")
	}
	if !strings.Contains(diag.String(), "switch_gateway:121") || strings.Contains(diag.String(), "restart failed") {
		t.Fatalf("safe stage diagnostic not propagated: %s", &diag)
	}
}

func TestActivationRequiresCanaryAndLiveGatewayReceipts(t *testing.T) {
	app, r := foundationApp(t)
	if code, _, diag := invoke(t, app, "install"); code != 0 {
		t.Fatal(diag)
	}
	input := &gatewayInput{state: "current"}
	if state, err := native.ConvergeGateway(context.Background(), r.id, r.context, input); err != nil || state != "current" {
		t.Fatal(state, err)
	}
	input.state = "not-running"
	if _, err := native.ConvergeGateway(context.Background(), r.id, r.context, input); err == nil {
		t.Fatal("stopped gateway accepted")
	}
}

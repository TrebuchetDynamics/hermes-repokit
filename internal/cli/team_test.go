package cli

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

type teamResumeInput struct {
	result string
	calls  int
}

func (r *teamResumeInput) RunInput(_ context.Context, input io.Reader, _ string, _ ...string) process.Result {
	r.calls++
	data, _ := io.ReadAll(input)
	if strings.Contains(string(data), "REPOKIT_MAINTENANCE_PY") {
		return process.Result{Output: "REPOKIT_MAINTENANCE=configured"}
	}
	if strings.Contains(string(data), "REPOKIT_GATEWAY_PY") {
		return process.Result{Output: "REPOKIT_GATEWAY=not-running\nREPOKIT_DISPATCH=prepared"}
	}
	return process.Result{Output: r.result}
}

func TestTeamResumeWithoutPrivateWizard(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	input := &teamResumeInput{result: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	a.Initializer = input
	a.Stdin = strings.NewReader("")
	if code, out, diag := invoke(t, a, "setup", "--team"); code == 0 || input.calls != 3 || !strings.Contains(diag, "mandatory") {
		t.Fatalf("team-only resume: code=%d calls=%d out=%s diag=%s", code, input.calls, out, diag)
	}
	input.result = `REPOKIT_TEAM={"status":"pending-setup","drift":[]}`
	if code, _, _ := invoke(t, a, "setup", "--team"); code == 0 {
		t.Fatal("missing default model accepted")
	}
	input.calls = 0
	r.runtime = strings.Replace(r.runtime, "running", "exited", 1)
	if code, _, _ := invoke(t, a, "setup", "--team"); code == 0 || input.calls != 0 {
		t.Fatal("stopped runtime accepted")
	}
}

func TestTeamResumeFlagsAreExclusive(t *testing.T) {
	for _, other := range []string{"--memory"} {
		if code, _, _ := invoke(t, App{}, "setup", "--team", other); code != 2 {
			t.Fatal("conflicting setup stages accepted")
		}
	}
}

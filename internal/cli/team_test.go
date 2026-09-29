package cli

import (
	"strings"
	"testing"
)

func TestTeamResumeWithoutPrivateWizard(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, team: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	a.Initializer = input
	a.Stdin = strings.NewReader("")
	if code, out, diag := invoke(t, a, "setup", "--team"); code != 0 || input.pid != 11 || !strings.Contains(diag, "Optional memory") {
		t.Fatalf("team-only resume: code=%d calls=%d out=%s diag=%s", code, input.calls, out, diag)
	}
	input.team = `REPOKIT_TEAM={"status":"pending-setup","drift":[]}`
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

func TestTeamSetupReportsPreservedOwnerProfilesAndActivates(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, team: `REPOKIT_TEAM={"status":"configured","drift":[],"customized":["executor"],"deferred":["reviewer"]}`}
	a.Initializer = input
	a.Stdin = strings.NewReader("")
	code, out, diag := invoke(t, a, "setup", "--team")
	if code != 0 || input.pid != 11 {
		t.Fatalf("owner customization blocked activation: code=%d out=%s diag=%s", code, out, diag)
	}
	if !strings.Contains(out, "Owner-customized profiles preserved: executor.") || !strings.Contains(out, "deferred while a card is running: reviewer.") {
		t.Fatalf("preserved profiles not reported:\n%s", out)
	}
}

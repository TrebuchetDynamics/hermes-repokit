package cli

import (
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
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
	if code, out, diag := invoke(t, a, "setup", "--team"); code != 0 || input.pid != 11 || strings.Contains(strings.ToLower(diag), "memory") {
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
	if !strings.Contains(out, "owner-customized, preserved as is: executor") || !strings.Contains(out, "deferred while a card is running: reviewer") {
		t.Fatalf("preserved profiles not reported:\n%s", out)
	}
}

func TestPlanPreviewsTeamOnlyForRunningDeployment(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`}
	a.Initializer = input
	var report Plan
	code, out, diag := invoke(t, a, "plan")
	if code != 0 || json.Unmarshal([]byte(out), &report) != nil || report.Team == nil || report.Team.Status != "runtime-not-running" || input.calls != 0 {
		t.Fatalf("stopped runtime preview: code=%d calls=%d out=%s diag=%s", code, input.calls, out, diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	code, out, diag = invoke(t, a, "plan", "--reset-profile", "executor")
	if code != 0 || json.Unmarshal([]byte(out), &report) != nil || report.Team == nil || report.Team.Status != "pending-setup" || report.Team.Detail == "" {
		t.Fatalf("running runtime preview: code=%d out=%s diag=%s", code, out, diag)
	}
	for _, s := range input.scripts {
		t.Fatalf("plan wrote native state:\n%s", s)
	}
}

func TestResetProfileFlag(t *testing.T) {
	a, r := foundationApp(t)
	for _, args := range [][]string{{"plan", "--reset-profile", "flutter-specialist"}, {"install", "--reset-profile", "../default"}, {"setup", "--reset-profile", "executor"}} {
		if code, _, _ := invoke(t, a, args...); code != 2 {
			t.Fatalf("%v accepted", args)
		}
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	// A stopped runtime cannot apply an explicit reset; that is a failure.
	if code, _, diag := invoke(t, a, "install", "--reset-profile", "executor"); code == 0 || !strings.Contains(diag, "reset not applied") {
		t.Fatalf("unapplied reset reported success: %s", diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	a.Initializer = &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, team: `REPOKIT_TEAM={"status":"configured","drift":[],"reset":["executor"]}`}
	code, out, diag := invoke(t, a, "install", "--reset-profile", "executor")
	if code != 0 || !strings.Contains(out, "executor returned to RepoKit baseline") || !strings.Contains(out, ".before-reset-") {
		t.Fatalf("reset not reported: code=%d out=%s diag=%s", code, out, diag)
	}
}

func TestCustomizedProfilesDoNotHoldBackInstallActivation(t *testing.T) {
	probe := func(name string, status verify.Status) verify.Probe {
		return verify.Probe{Component: "profile:" + name, Status: status}
	}
	complete, pending, unverified := profileProgress([]verify.Probe{probe("default", verify.Healthy), probe("executor", verify.Customized)})
	if !complete || pending || len(unverified) != 0 {
		t.Fatalf("customized profile held back activation: %v %v %v", complete, pending, unverified)
	}
	complete, pending, unverified = profileProgress([]verify.Probe{probe("executor", verify.Customized), probe("tester", verify.PendingSetup), probe("reviewer", verify.Degraded)})
	if complete || !pending || strings.Join(unverified, ",") != "reviewer" {
		t.Fatalf("pending or unverified profile not reported: %v %v %v", complete, pending, unverified)
	}
}

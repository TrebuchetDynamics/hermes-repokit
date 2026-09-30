package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// wizardMarker makes the native launcher's docker call record that the
// private Hermes wizard was started.
func wizardMarker(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	marker := filepath.Join(bin, "wizard-started")
	t.Setenv("REPOKIT_TEST_WIZARD_MARKER", marker)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n: > \"$REPOKIT_TEST_WIZARD_MARKER\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return marker
}

// The happy path is install, then setup. A fresh install points only at setup.
func TestFreshInstallPointsOnlyAtSetup(t *testing.T) {
	a, r := foundationApp(t)
	a.ComposeExec = func(_, _ io.Writer, args ...string) error {
		r.runtime = developmentRuntimeFixture(r.id)
		return nil
	}
	a.Initializer = &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, team: `REPOKIT_TEAM={"status":"pending-setup","drift":[]}`}
	code, out, diag := invoke(t, a, "install")
	if code != 0 || !strings.Contains(out, "Setup is still required.") || !strings.Contains(out, "repokit setup") {
		t.Fatalf("fresh install must end by pointing at setup: code=%d out=%s diag=%s", code, out, diag)
	}
	for _, choreography := range []string{"--team", "rerun", "docker --context", "compose up"} {
		if strings.Contains(out+diag, choreography) {
			t.Fatalf("fresh install exposed %q:\n%s%s", choreography, out, diag)
		}
	}
}

// With default's model already chosen, setup skips the private wizard and
// converges straight to a proven, ready team.
func TestSetupSkipsWizardWhenConfiguredAndEndsReady(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	marker := wizardMarker(t)
	a.Initializer = &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, model: "provider/model", team: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	canaries := 0
	a.Canary = func(target.Identity, string) (string, error) { canaries++; return "t_canary", nil }
	code, out, diag := invoke(t, a, "setup")
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("private wizard started although default already has a model")
	}
	if code != 0 || canaries != 1 || !strings.Contains(out, "private setup skipped") || !strings.Contains(out, "t_canary") ||
		!strings.Contains(out, "RepoKit ready.") || !strings.Contains(out, r.id.Container+"  open the project team") {
		t.Fatalf("setup did not converge to ready: code=%d canaries=%d out=%s diag=%s", code, canaries, out, diag)
	}
	if strings.Contains(out, "provider/model") {
		t.Fatal("setup printed the configured model")
	}
}

// setup starts a stopped deployment itself instead of printing Compose.
func TestSetupStartsAStoppedDeployment(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.composeCalls = nil
	a.ComposeExec = func(_, _ io.Writer, args ...string) error {
		r.composeCalls = append(r.composeCalls, strings.Join(args, " "))
		r.runtime = developmentRuntimeFixture(r.id)
		return nil
	}
	a.Initializer = &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, model: "provider/model", team: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	code, out, diag := invoke(t, a, "setup")
	if code != 0 || len(r.composeCalls) != 1 || !strings.HasSuffix(r.composeCalls[0], "up -d --build hermes") {
		t.Fatalf("setup did not start the deployment: code=%d calls=%v out=%s diag=%s", code, r.composeCalls, out, diag)
	}
	if strings.Contains(out+diag, "docker --context") {
		t.Fatalf("setup printed a Compose command:\n%s%s", out, diag)
	}
}

// A failed canary leaves the team configured but never claims readiness.
func TestSetupCanaryFailureIsNotReady(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	a.Initializer = &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, model: "provider/model", team: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	a.Canary = func(target.Identity, string) (string, error) { return "", errors.New("card not claimed within 150s") }
	code, out, diag := invoke(t, a, "setup")
	if code == 0 || strings.Contains(out, "RepoKit ready.") || !strings.Contains(diag, "canary failed: card not claimed") {
		t.Fatalf("failed canary reported ready: code=%d out=%s diag=%s", code, out, diag)
	}
	a.Canary = func(target.Identity, string) (string, error) { t.Fatal("--no-canary ran the canary"); return "", nil }
	if code, out, diag := invoke(t, a, "setup", "--no-canary"); code != 0 || !strings.Contains(out, "skipped (--no-canary)") {
		t.Fatalf("--no-canary: code=%d out=%s diag=%s", code, out, diag)
	}
}

// Converging an existing team prints one line per profile, in roster order.
func TestTeamReportPrintsOneLinePerProfile(t *testing.T) {
	var out strings.Builder
	report := native.TeamReport{Roles: []native.RoleStatus{
		{Profile: "default", State: "current"}, {Profile: "executor", State: "customized"},
		{Profile: "tester", State: "missing"}, {Profile: "reviewer", State: "reset"},
	}}
	newUI(&out, io.Discard).team(report)
	want := []string{"default", "current", "executor", "owner-customized; preserved", "tester", "created", "reviewer", "reset to RepoKit's baseline"}
	text := out.String()
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Fatalf("missing %q in:\n%s", w, text)
		}
	}
	if strings.Count(text, "\n") != len(report.Roles) {
		t.Fatalf("expected one line per profile:\n%s", text)
	}
}

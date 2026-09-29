package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
)

// On an SELinux-enabled host, plan must expose the relabel decision and install
// must generate private Z relabeling for both repository bind mounts.
func TestPlanAndInstallReflectSELinuxRelabel(t *testing.T) {
	a, r := foundationApp(t)
	a.HostSELinux = selinux.Enforcing
	code, out, diag := invoke(t, a, "plan")
	var plan Plan
	if code != 0 || json.Unmarshal([]byte(out), &plan) != nil {
		t.Fatalf("plan: %d %s %s", code, out, diag)
	}
	if plan.HostSecurity.SELinux != selinux.Enforcing || plan.HostSecurity.RelabelMode != "Z" || plan.HostSecurity.BindRelabeling != "enabled" {
		t.Fatalf("plan host security = %+v", plan.HostSecurity)
	}
	code, out, diag = invoke(t, a, "install")
	if code != 0 {
		t.Fatalf("install: %d %s", code, diag)
	}
	if !strings.Contains(out, "SELinux enforcing; private Z relabeling on repository mounts") {
		t.Fatalf("install diagnostics missing host security: %s", out)
	}
	data, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "selinux: Z"); got != 2 {
		t.Fatalf("expected 2 private relabel options, got %d\n%s", got, data)
	}
}

// An existing generated deployment without relabel metadata upgrades in place
// when the host has SELinux enabled.
func TestExistingDeploymentUpgradesToSELinuxRelabel(t *testing.T) {
	a, r := foundationApp(t)
	a.HostSELinux = selinux.Disabled
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("first install: %d %s", code, diag)
	}
	before, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(before), "selinux:") {
		t.Fatal("precondition: disabled host already relabels")
	}
	a.HostSELinux = selinux.Enforcing
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("upgrade install: %d %s", code, diag)
	}
	after, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(after), "selinux: Z"); got != 2 {
		t.Fatalf("expected upgraded relabel, got %d\n%s", got, after)
	}
	if _, err := os.Stat(filepath.Join(a.Directory, ".hermes", "compose.before-selinux.yaml")); err != nil {
		t.Fatalf("missing preimage backup: %v", err)
	}
}
func TestDisabledSELinuxKeepsComposeUnchanged(t *testing.T) {
	a, r := foundationApp(t)
	a.HostSELinux = selinux.Disabled
	if code, out, diag := invoke(t, a, "plan"); code != 0 {
		t.Fatalf("plan: %d %s %s", code, out, diag)
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("install: %d %s", code, diag)
	}
	data, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "selinux:") {
		t.Fatalf("disabled SELinux emitted relabel option\n%s", data)
	}
}

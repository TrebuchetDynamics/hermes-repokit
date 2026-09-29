package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
)

func TestFullSetupActivatesCoreWhenMemoryStageFails(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10, team: `REPOKIT_TEAM={"status":"configured","drift":[]}`}
	a.Initializer = input
	terminal, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	if !native.InteractiveInput(terminal) {
		t.Fatal("fixture is not a terminal")
	}
	a.Stdin = terminal
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	// The runtime/team commands succeed, but profiles are absent on disk, so
	// the real memory stage refuses its prerequisites without starting a wizard.
	var output bytes.Buffer
	code := a.Run([]string{"setup"}, &output, &output)
	out := output.String()
	core := strings.Index(out, "Configured native automatic dispatch")
	memory := strings.Index(out, "Complete native default setup")
	if code == 0 || core < 0 || memory < core || !strings.Contains(out, "Optional memory setup incomplete") {
		t.Fatalf("core must activate before optional memory, preserving its failure: code=%d out=%s", code, out)
	}
}

func TestMemoryOnlyFailureDoesNotSuspendCoreDispatch(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	input := &gatewayInput{kanban: `{"dispatch_in_gateway":false}`, pid: 10}
	a.Initializer = input
	terminal, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	a.Stdin = terminal
	code, _, diag := invoke(t, a, "setup", "--memory")
	if code == 0 || input.calls != 0 {
		t.Fatalf("memory prerequisites must fail without suspending gateway: code=%d mutations=%d diag=%s", code, input.calls, diag)
	}
}

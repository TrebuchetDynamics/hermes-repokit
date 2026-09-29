package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
)

// Memory providers are Hermes features: full setup completes the core team
// and dispatch and never runs, gates on or mentions a memory stage.
func TestFullSetupHasNoMemoryStage(t *testing.T) {
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
	var output bytes.Buffer
	code := a.Run([]string{"setup"}, &output, &output)
	out := output.String()
	if code != 0 || !strings.Contains(out, "Configured native automatic dispatch") || strings.Contains(strings.ToLower(out), "memory") || strings.Contains(strings.ToLower(out), "openviking") {
		t.Fatalf("core setup must complete with no memory stage: code=%d out=%s", code, out)
	}
}

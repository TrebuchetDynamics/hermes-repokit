package native

import (
	"bytes"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMemorySetupStopsAfterFailedNativeDoctor(t *testing.T) {
	id := target.Identity{Compose: "/repo/.hermes/compose.yaml", Launcher: "/repo/.hermes/bin/hermes-repo"}
	var calls []string
	var output bytes.Buffer
	code := memorySetup(id, "local", false, func(cmd *exec.Cmd) int {
		call := strings.Join(cmd.Args, " ")
		calls = append(calls, call)
		if strings.HasSuffix(call, " doctor") {
			return 23
		}
		return 0
	}, func() error { t.Fatal("activated memory after doctor failure"); return nil }, &output)
	if code != 23 || len(calls) != 2 {
		t.Fatalf("%d %v", code, calls)
	}
	if !strings.Contains(calls[0], "--context local compose --env-file /dev/null -f /repo/.hermes/compose.yaml exec openviking openviking-server init") {
		t.Fatal(calls)
	}
}
func TestMemorySetupPreservesExistingConfigAndPropagatesLinkFailure(t *testing.T) {
	id := target.Identity{Compose: "/repo/.hermes/compose.yaml", Launcher: "/repo/.hermes/bin/hermes-repo"}
	var calls []string
	var output bytes.Buffer
	code := memorySetup(id, "local", true, func(cmd *exec.Cmd) int { calls = append(calls, strings.Join(cmd.Args, " ")); return 0 }, func() error { return fmt.Errorf("connection not configured") }, &output)
	if code == 0 || len(calls) != 2 {
		t.Fatalf("%d %v", code, calls)
	}
	if !strings.HasSuffix(calls[1], "-p default memory setup openviking") {
		t.Fatal(calls)
	}
	for _, call := range calls {
		if strings.HasSuffix(call, " init") {
			t.Fatal("reopened existing server wizard")
		}
	}
}
func TestMemoryConnectionFilesystem(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python development tests need python3")
	}
	cmd := exec.Command(python, "memory_test.py")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

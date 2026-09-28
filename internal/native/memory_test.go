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
	if !strings.Contains(calls[0], "--context local compose --env-file /dev/null -f /repo/.hermes/compose.yaml exec --user hermes hermes repokit-openviking server init") {
		t.Fatal(calls)
	}
}
func TestMemorySetupPreservesExistingConfigAndPropagatesLinkFailure(t *testing.T) {
	id := target.Identity{Compose: "/repo/.hermes/compose.yaml", Launcher: "/repo/.hermes/bin/hermes-repo"}
	var calls []string
	var output bytes.Buffer
	code := memorySetup(id, "local", true, func(cmd *exec.Cmd) int { calls = append(calls, strings.Join(cmd.Args, " ")); return 0 }, func() error { return fmt.Errorf("connection not configured") }, &output)
	if code == 0 || len(calls) != 5 {
		t.Fatalf("%d %v", code, calls)
	}
	if !strings.HasSuffix(calls[4], "-p default memory setup openviking") {
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
	for _, script := range []string{"memory_test.py", "memory_server_test.py"} {
		cmd := exec.Command(python, script)
		cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v %s", script, err, out)
		}
	}
}

func TestMemorySetupGatesActivationOnServerChecks(t *testing.T) {
	id := target.Identity{Compose: "/repo/.hermes/compose.yaml", Launcher: "/repo/.hermes/bin/hermes-repo"}
	for _, failure := range []string{"validate", "restart", "health", ""} {
		t.Run(failure, func(t *testing.T) {
			var calls []string
			activated := false
			var output bytes.Buffer
			code := memorySetup(id, "local", true, func(cmd *exec.Cmd) int {
				call := strings.Join(cmd.Args, " ")
				stage := "wizard"
				switch {
				case strings.HasSuffix(call, " doctor"):
					stage = "doctor"
				case strings.HasSuffix(call, " validate"):
					stage = "validate"
				case strings.HasSuffix(call, " repokit-openviking restart"):
					stage = "restart"
				case strings.HasSuffix(call, " health"):
					stage = "health"
				}
				calls = append(calls, stage)
				if stage == failure {
					return 19
				}
				return 0
			}, func() error { activated = true; return nil }, &output)
			if failure == "" {
				if code != 0 || !activated || strings.Join(calls, ",") != "doctor,validate,restart,health,wizard" {
					t.Fatalf("%d %v %v", code, activated, calls)
				}
			} else if code != 19 || activated || calls[len(calls)-1] != failure {
				t.Fatalf("failed %s: %d %v %v", failure, code, activated, calls)
			}
		})
	}
}

package native

import (
	"bytes"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"os/exec"
	"path/filepath"
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
	}, func() error { t.Fatal("validated after doctor failure"); return nil }, func() error { t.Fatal("activated memory after doctor failure"); return nil }, &output)
	if code != 23 || len(calls) != 2 {
		t.Fatalf("%d %v", code, calls)
	}
	if !strings.Contains(calls[0], "--context local compose --env-file /dev/null -f /repo/.hermes/compose.yaml exec --user hermes hermes repokit-openviking server init") {
		t.Fatal(calls)
	}
}

func TestServerConfigValidationUsesPrivateRepositoryFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".hermes", "openviking")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(path, "ov.conf")
	good := `{"storage":{"workspace":"/opt/data/openviking/data"},"server":{"host":"127.0.0.1","port":1933,"root_api_key":"private-test-key"},"memory":{"extraction_enabled":true}}`
	if err := os.WriteFile(file, []byte(good), 0600); err != nil {
		t.Fatal(err)
	}
	id := target.Identity{Root: root}
	if err := validateServerConfig(id); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateServerConfig(id); err == nil {
		t.Fatal("accepted public secret-bearing config")
	}
	if err := os.Chmod(file, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(strings.Replace(good, "/opt/data/openviking/data", "/tmp/data", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateServerConfig(id); err == nil {
		t.Fatal("accepted nonpersistent workspace")
	}
}
func TestMemorySetupPreservesExistingConfigAndPropagatesLinkFailure(t *testing.T) {
	id := target.Identity{Compose: "/repo/.hermes/compose.yaml", Launcher: "/repo/.hermes/bin/hermes-repo"}
	var calls []string
	var output bytes.Buffer
	code := memorySetup(id, "local", true, func(cmd *exec.Cmd) int { calls = append(calls, strings.Join(cmd.Args, " ")); return 0 }, func() error { return nil }, func() error { return fmt.Errorf("connection not configured") }, &output)
	if code == 0 || len(calls) != 4 {
		t.Fatalf("%d %v", code, calls)
	}
	if !strings.HasSuffix(calls[3], "-p default memory setup openviking") {
		t.Fatal(calls)
	}
	for _, call := range calls {
		if strings.HasSuffix(call, " init") {
			t.Fatal("reopened existing server wizard")
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
			}, func() error {
				calls = append(calls, "validate")
				if failure == "validate" {
					return fmt.Errorf("invalid")
				}
				return nil
			}, func() error { activated = true; return nil }, &output)
			if failure == "" {
				if code != 0 || !activated || strings.Join(calls, ",") != "doctor,validate,restart,health,wizard" {
					t.Fatalf("%d %v %v", code, activated, calls)
				}
			} else if code != map[bool]int{true: 1, false: 19}[failure == "validate"] || activated || calls[len(calls)-1] != failure {
				t.Fatalf("failed %s: %d %v %v", failure, code, activated, calls)
			}
		})
	}
}

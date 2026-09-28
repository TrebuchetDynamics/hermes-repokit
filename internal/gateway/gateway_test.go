package gateway

import (
	"os/exec"
	"testing"
)

func TestGenerationAndRestartSafety(t *testing.T) {
	for _, script := range []string{"state_test.py", "catalog_test.py", "channels_test.py", "dispatch_test.py"} {
		out, err := exec.Command("python3", "-B", script).CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", script, err, out)
		}
	}
}

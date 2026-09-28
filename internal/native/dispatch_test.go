package native

import (
	"os/exec"
	"testing"
)

func TestOperationalActivationGates(t *testing.T) {
	if out, err := exec.Command("python3", "-B", "dispatch_test.py").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

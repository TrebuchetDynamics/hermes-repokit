package native

import (
	"os"
	"os/exec"
	"testing"
)

func TestSupervisionConfiguration(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python development tests need python3")
	}
	cmd := exec.Command(python, "supervision_test.py")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

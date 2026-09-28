package acceptance

import (
	"os"
	"os/exec"
	"testing"
)

func TestLiveKanbanEvidenceValidator(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python required for live acceptance harness development tests")
	}
	cmd := exec.Command(python, "-m", "unittest", "kanban_live_test.py")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("live acceptance validator: %v\n%s", err, out)
	}
}

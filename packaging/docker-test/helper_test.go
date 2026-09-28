package dockertest

import (
	_ "embed"
	"os"
	"os/exec"
	"testing"
)

// Embedding makes Go test caching track the Python regression suite too.
//
//go:embed test_helper.py
var helperTests string

func TestHelperIsolationAndCleanup(t *testing.T) {
	cmd := exec.Command("python3", "-c", "__file__ = 'test_helper.py'\n"+helperTests)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("helper tests: %v\n%s", err, out)
	}
}

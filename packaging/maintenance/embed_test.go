package maintenance

import (
	"os/exec"
	"testing"
)

func TestNativeBrokerSafety(t *testing.T) {
	output, err := exec.Command("python3", "-B", "test_bridge.py").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
}

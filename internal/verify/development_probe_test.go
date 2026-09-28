package verify

import (
	_ "embed"
	"os/exec"
	"testing"
)

//go:embed development_probe_test.py
var developmentTestSource string

func TestDevelopmentProbeIsObservational(t *testing.T) {
	if out, err := exec.Command("python3", "-B", "development_probe_test.py").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

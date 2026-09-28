package supervision

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
)

func TestPinnedCheckoutCannotExecuteFiltersOrTrustIndexFlags(t *testing.T) {
	if os.Getenv("REPOKIT_TEST_VERIFY_DOCKER") != "1" {
		t.Skip("set REPOKIT_TEST_VERIFY_DOCKER=1 with cached pinned Hermes image")
	}
	fixture, err := os.ReadFile("testdata/checkout_readonly.py")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--pull=never", "--network", "none", "--read-only", "--tmpfs", "/tmp:rw,nosuid,nodev,size=16m", "--entrypoint", "/opt/hermes/.venv/bin/python", qualification.FoundationImage, "-I", "-B", "-c", string(fixture), CheckoutScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("checkout safety fixture: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "checkout safety fixtures passed" {
		t.Fatalf("unexpected fixture output: %s", out)
	}
}

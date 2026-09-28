package native

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestMaintenanceProvisioningFilesystem(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python required for source fixture tests")
	}
	cmd := exec.Command(python, "-B", "maintenance_test.py")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

func TestMaintenanceProvisioningRequiresNativeReceiptUnderLock(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	os.Mkdir(filepath.Join(root, ".hermes"), 0700)
	os.WriteFile(filepath.Join(root, ".hermes-repokit.lock"), nil, 0600)
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []process.Result{{}, {Output: "secret diagnostics"}, {Err: fmt.Errorf("secret failure")}, {Output: "REPOKIT_MAINTENANCE=configured", Truncated: true}} {
		r := &bootstrapInput{result: result}
		err := ProvisionMaintenance(context.Background(), id, "local", r)
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("invalid receipt accepted or diagnostic leaked: %v", err)
		}
	}
	r := &bootstrapInput{result: process.Result{Output: "REPOKIT_MAINTENANCE=configured\n"}}
	if err := ProvisionMaintenance(context.Background(), id, "local", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(r.args, " "), "/usr/bin/flock -n /workspace/.hermes-repokit.lock") || !strings.Contains(r.data, "REPOKIT_MAINTENANCE_PY") || !strings.Contains(r.data, "--no-enable") {
		t.Fatal("maintenance provisioning lost native bootstrap lock or installer")
	}
}

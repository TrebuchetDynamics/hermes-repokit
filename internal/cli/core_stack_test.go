package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreStackHasNoOwnedSupervision(t *testing.T) {
	a, _ := foundationApp(t)
	code, out, diag := invoke(t, a, "plan")
	if code != 0 {
		t.Fatal(diag)
	}
	var plan map[string]any
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatal(err)
	}
	// The temporary repository path includes the test name; it is not policy.
	delete(plan, "target")
	policy, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, removed := range []string{"laya", "nerve", "supervision"} {
		if strings.Contains(strings.ToLower(string(policy)), removed) {
			t.Fatalf("core plan retains %s", removed)
		}
	}
	if code, _, diag = invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	state := filepath.Join(a.Directory, ".hermes")
	for _, name := range []string{"laya", "laya-image"} {
		if _, err := os.Stat(filepath.Join(state, name)); !os.IsNotExist(err) {
			t.Fatalf("generated removed component %s", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(state, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "  laya:") || strings.Contains(string(data), "  openviking:") || !strings.Contains(string(data), "REPOKIT_OPENVIKING: \"1\"") {
		t.Fatal("core topology differs", string(data))
	}
}

func TestRemovedSupervisionFlagsAreRejected(t *testing.T) {
	a, _ := foundationApp(t)
	for _, args := range [][]string{{"setup", "--supervision"}, {"install", "--laya-image", "sha256:" + strings.Repeat("a", 64)}, {"plan", "--laya-image", "sha256:" + strings.Repeat("a", 64)}} {
		if code, _, _ := invoke(t, a, args...); code != 2 {
			t.Fatalf("removed flag accepted %v: %d", args, code)
		}
	}
}

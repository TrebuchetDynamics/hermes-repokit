package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallTrustsProjectSkillsBeforeRuntimeStarts(t *testing.T) {
	a, _ := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("install: %s", diag)
	}
	path := filepath.Join(a.Directory, ".hermes/config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "skills:\n  trusted_project_dirs: [/workspace]\n") {
		t.Fatal("fresh native config does not trust the selected repository")
	}
	owner := []byte("skills:\n  project_discovery: false\n  trusted_project_dirs: [/owner]\n")
	if err := os.WriteFile(path, owner, 0600); err != nil {
		t.Fatal(err)
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("rerun: %s", diag)
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(owner) {
		t.Fatal("stopped install rerun replaced owner skill settings")
	}
}

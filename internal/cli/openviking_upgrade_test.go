package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
)

// An OpenViking-era deployment (embedded-memory recipe and Compose marker) is
// RepoKit's own output: install upgrades it in place, backs up both the old
// Compose and the old recipe, and leaves memory data untouched.
func TestInstallUpgradesOpenVikingEraDeployment(t *testing.T) {
	a, r := foundationApp(t)
	a.HostSELinux = selinux.Disabled
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	state := filepath.Join(r.id.Root, ".hermes")
	current, err := os.ReadFile(filepath.Join(state, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// Replace the deployment with the OpenViking-era generation.
	recipeDir := filepath.Join(state, "development-image")
	if err := os.RemoveAll(recipeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(recipeDir, 0700); err != nil {
		t.Fatal(err)
	}
	old := map[string][]byte{}
	for _, name := range []string{"Dockerfile-go", ".dockerignore", "repokit-docker-test", "repokit-openviking", "openviking-run", "openviking-finish", "patch-openviking-entrypoint.py"} {
		data, err := os.ReadFile(filepath.Join("../development/testdata/e0246ef", name))
		if err != nil {
			t.Fatal(err)
		}
		old[strings.TrimSuffix(name, "-go")] = data
	}
	fingerprint, ok := development.GeneratedRecipe(old)
	if !ok {
		t.Fatal("historical recipe no longer self-certifies")
	}
	for name, data := range old {
		if err := os.WriteFile(filepath.Join(recipeDir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	req := development.Requirements{}
	previous, err := compose.OlderRecipe(r.id, compose.Options{HermesImage: qualification.FoundationImage, HistoricalOpenViking: true, Development: &req, UID: os.Getuid(), GID: os.Getgid(), SELinux: selinux.Disabled}, fingerprint)
	if err != nil || !bytes.Contains(previous, []byte("REPOKIT_OPENVIKING")) {
		t.Fatalf("historical Compose not rendered: %v", err)
	}
	if err := os.WriteFile(filepath.Join(state, "compose.yaml"), previous, 0600); err != nil {
		t.Fatal(err)
	}
	memory := filepath.Join(state, "openviking")
	if err := os.MkdirAll(memory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memory, "ov.conf"), []byte("owner memory data"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := compose.DevelopmentInstallSelected(r.id); !ok {
		t.Fatal("OpenViking-era Compose not recognized as RepoKit-generated")
	}

	if code, out, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("OpenViking-era deployment not upgraded: out=%s diag=%s", out, diag)
	}
	got, _ := os.ReadFile(filepath.Join(state, "compose.yaml"))
	if !bytes.Equal(got, current) {
		t.Fatalf("Compose not upgraded to the current render:\n%s", got)
	}
	backup := "compose.before-openviking-recipe-" + fingerprint[:12] + ".yaml"
	if saved, err := os.ReadFile(filepath.Join(state, backup)); err != nil || !bytes.Equal(saved, previous) {
		t.Fatalf("old Compose not backed up as %s: %v", backup, err)
	}
	if _, err := os.Stat(filepath.Join(recipeDir, "repokit-openviking")); !os.IsNotExist(err) {
		t.Fatal("OpenViking recipe helper survived the upgrade")
	}
	if saved, err := os.ReadFile(filepath.Join(state, install.RecipeBackupName(backup), "repokit-openviking")); err != nil || !bytes.Equal(saved, old["repokit-openviking"]) {
		t.Fatalf("old recipe not backed up: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(memory, "ov.conf")); err != nil || string(data) != "owner memory data" {
		t.Fatal("memory data touched by the upgrade")
	}
}

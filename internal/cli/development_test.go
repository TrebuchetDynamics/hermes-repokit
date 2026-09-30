package cli

import (
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func developmentRuntimeFixture(id target.Identity) string {
	o, ok := compose.DevelopmentSelected(id)
	if !ok {
		panic("fixture requires generated development Compose")
	}
	mounts := []map[string]any{{"Type": "bind", "Source": id.Root, "Destination": "/workspace", "RW": true}, {"Type": "bind", "Source": filepath.Join(id.Root, ".hermes"), "Destination": "/opt/data", "RW": true}}
	// Current renders hide .hermes inside /workspace behind an empty tmpfs.
	mounts = append(mounts, map[string]any{"Type": "tmpfs", "Destination": "/workspace/.hermes", "RW": true})
	if o.DockerTests {
		mounts = append(mounts, map[string]any{"Type": "volume", "Name": id.Project + "_docker-test-run", "Destination": "/docker-test/run", "RW": false}, map[string]any{"Type": "volume", "Name": id.Project + "_docker-test-work", "Destination": "/docker-tests", "RW": true})
	}
	b, _ := json.Marshal(map[string]any{"id": strings.Repeat("a", 64), "status": "running", "service": "hermes", "unexpectedMounts": "", "image": development.ImageName(id.Container, *o.Development), "imageID": "sha256:" + strings.Repeat("d", 64), "project": id.Project, "workspace": id.Root, "home": filepath.Join(id.Root, ".hermes"), "mounts": mounts})
	return string(b)
}
func TestDevelopmentInstallDetectsGoAndPreservesRecipeDrift(t *testing.T) {
	a, r := foundationApp(t)
	os.WriteFile(filepath.Join(a.Directory, "go.mod"), []byte("module example.test/demo\n\ngo 1.26.0\n"), 0600)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	o, ok := compose.DevelopmentSelected(r.id)
	if !ok || !o.Development.Go || o.DockerTests {
		t.Fatal("wrong default development selection")
	}
	path := filepath.Join(a.Directory, ".hermes/development-image/Dockerfile")
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "go"+development.GoVersion) {
		t.Fatal("missing Go toolchain")
	}
	os.WriteFile(path, append(data, []byte("# owner change\n")...), 0600)
	if code, _, _ := invoke(t, a, "install"); code == 0 {
		t.Fatal("overwrote owner recipe")
	}
	got, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(got), "# owner change\n") {
		t.Fatal("lost owner edit")
	}
}
func TestDockerTestOptInPersistsAcrossInstallRerun(t *testing.T) {
	a, r := foundationApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	if c, out, d := invoke(t, a, "install", "--docker-tests"); c != 0 || !strings.Contains(out, "--profile docker-tests") {
		t.Fatal(c, out, d)
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	o, ok := compose.DevelopmentSelected(r.id)
	if !ok || !o.DockerTests {
		t.Fatal("opt-in selection lost")
	}
}

func TestEnablingDockerTestsDefersUntilExistingRuntimeRecreated(t *testing.T) {
	a, r := foundationApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	if c, out, d := invoke(t, a, "install", "--docker-tests"); c != 0 || !strings.Contains(out, "initialization pending") {
		t.Fatal(c, out, d)
	}
}

// Opting a current deployment into the Docker test daemon, then rerunning
// install without the flag, republishes in place and keeps the opt-in.
func TestDockerTestsOptInOnACurrentDeployment(t *testing.T) {
	a, _ := foundationApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	if c, _, d := invoke(t, a, "install", "--docker-tests"); c != 0 {
		t.Fatal(d)
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
}

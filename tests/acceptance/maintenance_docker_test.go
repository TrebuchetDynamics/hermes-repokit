//go:build docker

package acceptance

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/packaging/maintenance"
)

//go:embed fixtures/maintenance.py
var maintenanceFixture string

func TestDockerMaintenanceNativePackage(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" {
		t.Skip("requires authorized local Docker")
	}
	dc := os.Getenv("REPOKIT_DOCKER_CONTEXT")
	if dc == "" {
		dc = "default"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	teamSource, err := os.ReadFile("../../internal/native/team.py")
	if err != nil {
		t.Fatal(err)
	}
	provisioner, err := os.ReadFile("../../internal/native/maintenance.py")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(struct {
		Name  string            `json:"name"`
		Files map[string]string `json:"files"`
	}{maintenance.Name, maintenance.Files()})
	if err != nil {
		t.Fatal(err)
	}
	script := team.KanbanPolicy + "\n" + string(teamSource) + "\n" + string(provisioner) + "\n" + maintenanceFixture +
		"\nimport base64\nmaintenance_acceptance(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\n"
	// No mounts, credentials, network, live gateway or persisted runtime state.
	cmd := exec.CommandContext(ctx, "docker", "--context", dc, "run", "--rm", "--network", "none", "--user", "1000:1000", "--entrypoint", "/opt/hermes/.venv/bin/python", qualification.FoundationImage, "-B", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "NATIVE_MAINTENANCE_PACKAGE_PASS") {
		t.Fatalf("native maintenance scanner/install/registry fixture: %v\n%s", err, out)
	}
}

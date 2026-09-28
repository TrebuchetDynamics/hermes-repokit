//go:build docker

package acceptance

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

//go:embed fixtures/default_kanban.py
var defaultKanbanFixture string

func TestDockerDefaultKanbanChannels(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" {
		t.Skip("requires authorized local Docker")
	}
	dc := os.Getenv("REPOKIT_DOCKER_CONTEXT")
	if dc == "" {
		dc = "default"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	source, err := os.ReadFile("../../internal/native/team.py")
	if err != nil {
		t.Fatal(err)
	}
	script := team.KanbanPolicy + "\n" + string(source) + "\n" + defaultKanbanFixture
	cmd := exec.CommandContext(ctx, "docker", "--context", dc, "run", "--rm", "--network", "none", "--user", "1000:1000", "--entrypoint", "/opt/hermes/.venv/bin/python", qualification.FoundationImage, "-B", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "NATIVE_KANBAN_MEMORY_CHANNELS_PASS") {
		t.Fatalf("native schema fixture: %v\n%s", err, out)
	}
}

//go:build docker

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// Config-only: exercises the real Compose resolver without starting, stopping,
// building, or inspecting any service or requiring a running Docker daemon.
func TestDockerComposeCoexistenceConfig(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker Compose CLI unavailable")
	}
	root := t.TempDir()
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	owner := filepath.Join(root, "docker-compose.yml")
	original := []byte("name: owner-stack\nservices:\n  owner:\n    image: busybox:1.37\n    command: [sleep, infinity]\n")
	if err := os.WriteFile(owner, original, 0600); err != nil {
		t.Fatal(err)
	}
	// Invalid automatic overrides and ambient selectors must never be merged.
	if err := os.WriteFile(filepath.Join(root, "compose.override.yml"), []byte("services: [invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("COMPOSE_FILE=missing-owner.yml\nCOMPOSE_PROJECT_NAME=wrong-project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMPOSE_FILE", owner)
	t.Setenv("COMPOSE_PROJECT_NAME", "wrong-project")
	runner := process.Runner{Timeout: 30 * time.Second}
	resolve := func(file string) []byte {
		t.Helper()
		result := runner.Run(context.Background(), "docker", "compose", "--env-file", "/dev/null", "-f", file, "config", "--format", "json")
		if result.Err != nil || result.Truncated {
			t.Fatalf("Compose config: %v %s", result.Err, result.Output)
		}
		return []byte(result.Output)
	}
	before := resolve(owner)
	if err := os.Mkdir(filepath.Dir(id.Compose), 0700); err != nil {
		t.Fatal(err)
	}
	generated, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: 1000, GID: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(id.Compose, generated, 0600); err != nil {
		t.Fatal(err)
	}
	var resolved struct {
		Name     string                     `json:"name"`
		Services map[string]json.RawMessage `json:"services"`
	}
	if err := json.Unmarshal(resolve(id.Compose), &resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.Name != id.Project || len(resolved.Services) != 1 || resolved.Services["hermes"] == nil {
		t.Fatalf("owner service or selector entered RepoKit project: %+v", resolved)
	}
	if !bytes.Equal(before, resolve(owner)) {
		t.Fatal("owner Compose resolution changed")
	}
	if actual, err := os.ReadFile(owner); err != nil || !bytes.Equal(actual, original) {
		t.Fatal("owner file changed")
	}
}

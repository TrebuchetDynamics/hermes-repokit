package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

type integrationRunner struct {
	hermes, service, image, config, health string
	calls                                  [][]string
}

func (r *integrationRunner) Run(_ context.Context, p string, args ...string) process.Result {
	r.calls = append(r.calls, append([]string{p}, args...))
	call := strings.Join(args, " ")
	switch {
	case strings.Contains(call, "container ls"):
		return process.Result{Output: "123456abcdef"}
	case strings.Contains(call, "container inspect"):
		if args[len(args)-1] == "123456abcdef" {
			return process.Result{Output: r.service}
		}
		return process.Result{Output: r.hermes}
	case strings.Contains(call, "image inspect"):
		return process.Result{Output: r.image}
	case strings.Contains(call, integrationsProbe):
		return process.Result{Output: r.config}
	case strings.Contains(call, kanbanProbe):
		return process.Result{Output: r.config}
	default:
		return process.Result{Err: fmt.Errorf("unexpected command")}
	}
}

func integrationFixture(t *testing.T) (target.Identity, *integrationRunner) {
	t.Helper()
	id, err := target.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(id.Launcher), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := launcher.Render(id, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(id.Launcher, data, 0700); err != nil {
		t.Fatal(err)
	}
	data, err = compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(id.Compose, data, 0600); err != nil {
		t.Fatal(err)
	}
	r := &integrationRunner{
		hermes: fmt.Sprintf(`{"id":"%s","status":"running","image":%q,"project":%q,"service":"hermes","workspace":%q,"home":%q}`, strings.Repeat("a", 64), qualification.FoundationImage, id.Project, id.Root, filepath.Join(id.Root, ".hermes")),
		config: `{"memory":"active"}`, health: "healthy\n",
	}
	return id, r
}

func TestRuntimeIntegrationsKeepReadinessAndAcceptanceIndependent(t *testing.T) {
	id, r := integrationFixture(t)
	probes := RuntimeIntegrations(context.Background(), id, r)
	want := []Status{Active, Unqualified}
	if len(probes) != len(want) {
		t.Fatalf("unexpected integration requirements: %+v", probes)
	}
	for i, p := range probes {
		if p.Status != want[i] {
			t.Fatalf("%+v", probes)
		}
	}
	for _, call := range r.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, " exec ") && !strings.Contains(joined, "/opt/hermes/.venv/bin/python -I -B -c") {
			t.Fatal("exec was not isolated Python")
		}
		for _, forbidden := range []string{"hermes kanban", "discover_plugins", "validate_auth", "/v1/systemone", "registry.dispatch", "plugins enable"} {
			if strings.Contains(joined, forbidden) {
				t.Fatalf("unsafe probe: %s", forbidden)
			}
		}
	}
	r.config = `{"memory":"degraded"}`
	probes = RuntimeIntegrations(context.Background(), id, r)
	if probes[0].Status != Degraded || probes[1].Status != Unqualified {
		t.Fatalf("dependent statuses: %+v", probes)
	}
}

func TestRuntimeIntegrationsRefuseExecOnIdentityMismatch(t *testing.T) {
	for _, field := range []string{"image", "project", "workspace", "home", "service", "status", "id", "unexpectedMounts"} {
		t.Run(field, func(t *testing.T) {
			id, r := integrationFixture(t)
			var state map[string]string
			json.Unmarshal([]byte(r.hermes), &state)
			state[field] = "different"
			data, _ := json.Marshal(state)
			r.hermes = string(data)
			RuntimeIntegrations(context.Background(), id, r)
			for _, call := range r.calls {
				if strings.Contains(strings.Join(call, " "), " exec ") {
					t.Fatalf("exec after %s mismatch", field)
				}
			}
		})
	}
}

// Runs the actual parser with pinned PyYAML/dotenv in an isolated disposable
// container. Only test fixtures write /tmp; the verifier never creates state.
func TestPinnedReadOnlyIntegrationParser(t *testing.T) {
	if os.Getenv("REPOKIT_TEST_VERIFY_DOCKER") != "1" {
		t.Skip("set REPOKIT_TEST_VERIFY_DOCKER=1 with cached pinned Hermes image")
	}
	fixture, err := os.ReadFile("testdata/integrations_fixture.py")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--pull=never", "--network", "none", "--read-only", "--tmpfs", "/tmp:rw,nosuid,nodev,size=16m", "--entrypoint", "/opt/hermes/.venv/bin/python", qualification.FoundationImage, "-I", "-B", "-c", string(fixture), integrationsProbe)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("parser fixture: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "read-only parser fixtures passed" {
		t.Fatalf("unexpected output: %s", output)
	}
}

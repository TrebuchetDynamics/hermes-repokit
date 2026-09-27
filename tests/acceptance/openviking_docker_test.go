//go:build docker

package acceptance

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// The official pending-config server is real. No embedding, memory response,
// provider credentials or live acceptance result is synthesized by this test.
func TestDockerOpenVikingPending(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" {
		t.Skip("set REPOKIT_DOCKER_TESTS=1 with an authorized local Docker daemon")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dc := os.Getenv("REPOKIT_DOCKER_CONTEXT")
	if dc == "" {
		dc = "default"
	}
	host, err := exec.CommandContext(ctx, "docker", "context", "inspect", dc, "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil || !strings.HasPrefix(string(host), "unix://") {
		t.Fatal("local Docker context required")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := exec.CommandContext(ctx, "git", "-C", root, "init", "--quiet").Run(); err != nil {
		t.Fatal(err)
	}
	installer, removeInstaller := disposableCLI(t)
	install := func() {
		t.Helper()
		cmd := exec.CommandContext(ctx, installer, "install")
		cmd.Dir = root
		cmd.Env = append(process.CleanEnvironment(os.Environ()), "DOCKER_CONTEXT="+dc)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("install: %v %s", err, out)
		}
	}
	install()
	base := []string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose}
	docker := func(args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, "docker", append(append([]string{}, base...), args...)...)
		cmd.Env = process.CleanEnvironment(os.Environ())
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Docker %v: %v %s", args, err, out)
		}
		return out
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cleanup, "docker", append(base, "down")...)
		cmd.Env = process.CleanEnvironment(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("cleanup: %v %s", err, out)
		}
	})
	docker("config", "--quiet")
	docker("up", "-d", "--pull", "never", "openviking")
	probe := `import urllib.request, urllib.error
try:
    urllib.request.urlopen('http://127.0.0.1:1933/health', timeout=2)
except urllib.error.HTTPError as e:
    assert e.code == 503
else:
    raise AssertionError('unconfigured service claimed ready')
`
	deadline := time.Now().Add(30 * time.Second)
	for {
		args := append(append([]string{}, base...), "exec", "-T", "openviking", "python", "-c", probe)
		if exec.CommandContext(ctx, "docker", args...).Run() == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("official unconfigured endpoint did not return 503")
		}
		time.Sleep(time.Second)
	}
	if got := strings.TrimSpace(string(docker("exec", "-T", "openviking", "id", "-u"))); got != strconv.Itoa(os.Getuid()) {
		t.Fatalf("sidecar UID = %s", got)
	}
	// Qualify native init's path and writer without invoking its private wizard.
	docker("exec", "-T", "openviking", "python", "-c", `from openviking_cli.setup_wizard import _config_path, _workspace_path
assert str(_config_path()) == '/app/.openviking/ov.conf'
assert _workspace_path() == '/app/.openviking/data'
from pathlib import Path
p = Path('/app/.openviking/persistence-fixture')
p.write_text('nonsecret persistent fixture')
p.chmod(0o600)
`)
	install()
	for _, p := range verify.OpenViking(ctx, id, process.Runner{}) {
		if p.Component == "openviking-container" {
			if p.Status != verify.Healthy {
				t.Fatalf("container identity: %+v", p)
			}
			continue
		}
		if p.Component == "openviking" && p.Status != verify.Unknown {
			t.Fatalf("live gate: %+v", p)
		}
		if p.Component != "openviking" && p.Status != verify.PendingSetup {
			t.Fatalf("pending probe: %+v", p)
		}
	}
	removeInstaller()
	docker("up", "-d", "--pull", "never", "--force-recreate", "openviking")
	docker("exec", "-T", "openviking", "python", "-c", `from pathlib import Path
assert Path('/app/.openviking/persistence-fixture').read_text() == 'nonsecret persistent fixture'
assert not Path('/app/.openviking/ov.conf').exists()
`)
	info, err := os.Stat(filepath.Join(root, ".hermes/openviking"))
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatal("sidecar directory is not private")
	}
	if _, err := os.Stat(installer); !os.IsNotExist(err) {
		t.Fatal("installer survived removal")
	}
	t.Log("PASS: pinned nonroot OpenViking pending server returns 503; native config/workspace paths use private mount; install rerun and Compose recreation preserve data after installer removal. Live memory remains unproven.")
}

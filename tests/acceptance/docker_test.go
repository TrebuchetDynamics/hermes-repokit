//go:build docker

package acceptance

import (
	"context"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This proves credential-free foundation behavior only. It is deliberately not
// named the removal-first release gate: no paid chat, review or memory is faked.
func TestDockerFoundation(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" {
		t.Skip("set REPOKIT_DOCKER_TESTS=1 with an authorized local Docker daemon")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dc := os.Getenv("REPOKIT_DOCKER_CONTEXT")
	if dc == "" {
		dc = "default"
	}
	inspect := exec.CommandContext(ctx, "docker", "context", "inspect", dc, "--format", "{{.Endpoints.docker.Host}}")
	host, e := inspect.Output()
	if e != nil || !strings.HasPrefix(string(host), "unix://") {
		t.Fatal("local Docker context required")
	}
	root, e := os.MkdirTemp("", "repokit-acceptance-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(root)
	os.Chmod(root, 0700)
	id, e := target.Resolve(root)
	if e != nil {
		t.Fatal(e)
	}
	installer, removeInstaller := disposableCLI(t)
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git: %v %s", err, out)
	}
	runCLI := func(command string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, installer, command)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "DOCKER_CONTEXT="+dc)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("CLI %s: %v %s", command, err, out)
		}
	}
	runCLI("plan")
	runCLI("install")
	base := []string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose}
	docker := func(args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, "docker", append(append([]string(nil), base...), args...)...)
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("Docker %v failed: %v\n%s", args, e, out)
		}
		return out
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cleanup, "docker", append(base, "down")...)
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Errorf("cleanup: %v %s", e, out)
		}
	}()
	docker("config", "--quiet")
	docker("up", "-d", "hermes")
	// Bounded readiness polling does not imply authentication or inference readiness.
	ready := false
	for i := 0; i < 60; i++ {
		cmd := exec.CommandContext(ctx, id.Launcher, "--version")
		if out, e := cmd.CombinedOutput(); e == nil && strings.Contains(string(out), "v0.21.5") {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		t.Fatal("native exec did not become ready")
	}
	runCLI("verify")
	runCLI("install") // A running owned container must not block a safe no-op.
	removeInstaller()
	// There is no receipt in a foundation install. Removing source and binary
	// before native commands proves those artifacts cannot be runtime dependencies.
	if _, err := os.Stat(installer); !os.IsNotExist(err) {
		t.Fatal("installer still accessible")
	}
	cmd := exec.CommandContext(ctx, id.Launcher, "kanban", "init")
	cmd.Dir = t.TempDir()
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("native board initialization: %v %s", e, out)
	}
	info, e := os.Stat(filepath.Join(root, ".hermes/kanban.db"))
	if e != nil || info.Size() == 0 {
		t.Fatal("persistent native board absent")
	}
	before := info
	docker("restart", "hermes")
	after, e := os.Stat(filepath.Join(root, ".hermes/kanban.db"))
	if e != nil || !os.SameFile(before, after) {
		t.Fatal("restart replaced board")
	}
	cmd = exec.CommandContext(ctx, id.Launcher, "--version")
	cmd.Dir = t.TempDir()
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "v0.21.5") {
		t.Fatalf("native exec after restart/removal: %v %s", err, out)
	}
	t.Log("PASS: real CLI plan/install/verify/rerun; disposable installer binary and source removed; native exec and Kanban from unrelated cwd; raw Compose restart and persistent board. No inference or full v1 gate claimed.")
}

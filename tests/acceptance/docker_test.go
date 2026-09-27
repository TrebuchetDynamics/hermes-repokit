//go:build docker

package acceptance

import (
	"context"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
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
	data, e := compose.Render(id, compose.Options{HermesImage: "nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db", UID: os.Getuid(), GID: os.Getgid()})
	if e != nil {
		t.Fatal(e)
	}
	script, e := launcher.Render(id, dc)
	if e != nil {
		t.Fatal(e)
	}
	_, e = install.Publish(id, map[string]install.Artifact{"compose.yaml": {Data: data, Mode: 0600}, "config.yaml": {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n"), Mode: 0600}, "bin/" + id.Container: {Data: script, Mode: 0700}}, nil)
	if e != nil {
		t.Fatal(e)
	}
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
	t.Log("PASS: generated Compose parsed; official Hermes created; native exec worked from unrelated cwd; Kanban persisted through raw Compose restart. No inference or full removal gate claimed.")
}

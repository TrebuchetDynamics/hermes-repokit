//go:build docker

package acceptance

import (
	"context"
	"crypto/rand"
	_ "embed"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//go:embed fixtures/worker_terminal.py
var workerTerminalFixture string

// Real image + compiler proof, no credential/model calls. This never mounts the
// host socket or native state. Daemon access is the same opt-in acceptance gate.
func TestDockerDevelopmentRuntime(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" {
		t.Skip("set REPOKIT_DOCKER_TESTS=1 with an authorized Docker daemon")
	}
	dc := os.Getenv("REPOKIT_DOCKER_CONTEXT")
	if dc == "" {
		dc = "default"
	}
	root := t.TempDir()
	files, err := development.Recipe(development.Requirements{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	image := "repokit-development-test-" + strings.ToLower(rand.Text()) + ":fixture"
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "docker", append([]string{"--context", dc}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Docker development fixture: %v\n%s", err, out)
		}
	}
	run("build", "--tag", image, root)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(cleanup, "docker", "--context", dc, "image", "rm", image).CombinedOutput(); err != nil {
			t.Errorf("fixture image cleanup: %v %s", err, out)
		}
	}()
	run("run", "--rm", "--network", "none", "--user", "1000:1000", "--env", "HERMES_DISABLE_LAZY_INSTALLS=1", "--entrypoint", "/opt/hermes/.venv/bin/python", image, "-B", "-c", workerTerminalFixture)
	run("run", "--rm", "--network", "none", "--user", "1000:1000", "--entrypoint", "/bin/sh", image, "-ec", `
 test ! -S /var/run/docker.sock
 test ! -S /docker-test/run/docker.sock
 test "$(pwd)" = /workspace
 git --version
 jq --version
 python3 --version
 node --version
 npm --version
 rg --version
 docker compose version
 docker buildx version
 work=$(mktemp -d /tmp/repokit-development-XXXXXX)
 trap 'rm -rf "$work"' EXIT
 cd "$work"
 printf 'module fixture\n\ngo 1.26.0\n' > go.mod
 printf 'package fixture\nimport "testing"\nfunc TestCompiler(t *testing.T){if 2+2!=4{t.Fatal("bad")}}\n' > fixture_test.go
 GOCACHE="$work/cache" GOMODCACHE="$work/mod" GOTOOLCHAIN=local go test -race ./...
 `)
}

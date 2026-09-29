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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
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
	installerEnv := installerEnvironment(t)
	install := func() {
		t.Helper()
		cmd := exec.CommandContext(ctx, installer, "install")
		cmd.Dir = root
		cmd.Env = append(process.CleanEnvironment(installerEnv), "DOCKER_CONTEXT="+dc)
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
	if services := string(docker("config", "--services")); strings.Contains(services, "openviking") {
		t.Fatal("OpenViking rendered as an independent service")
	}
	docker("up", "-d", "--build", "hermes")
	probe := `[ "$(/usr/bin/curl --silent --output /dev/null --write-out '%{http_code}' --max-time 2 --noproxy '*' http://127.0.0.1:1933/health)" = 503 ]`
	// The embedded service only starts after the base image's stage-2 hook has
	// remapped the runtime user and prepared the data volume, so waiting for the
	// pending endpoint also establishes that Hermes identity/mount setup is done.
	waitPending := func() {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for {
			args := append(append([]string{}, base...), "exec", "-T", "--user", "hermes", "hermes", "/bin/sh", "-c", probe)
			if exec.CommandContext(ctx, "docker", args...).Run() == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Fatal("official unconfigured endpoint did not return 503")
			}
			time.Sleep(time.Second)
		}
	}
	waitPending()
	if got := strings.TrimSpace(string(docker("exec", "-T", "--user", "hermes", "hermes", "id", "-u"))); got != strconv.Itoa(os.Getuid()) {
		t.Fatalf("embedded process UID = %s", got)
	}
	// Verify the private mount persists without invoking the native wizard.
	docker("exec", "-T", "--user", "hermes", "hermes", "/bin/sh", "-c", `printf '%s' 'nonsecret persistent fixture' > /opt/data/openviking/persistence-fixture && chmod 600 /opt/data/openviking/persistence-fixture`)
	install()
	for _, p := range verify.OpenViking(ctx, id, process.Runner{}) {
		if p.Component == "openviking-container" {
			if p.Status != verify.Healthy {
				t.Fatalf("container identity: %+v", p)
			}
			continue
		}
		if p.Component == "openviking" && (p.Status != verify.PendingSetup || !strings.Contains(p.Detail, "acceptance unqualified")) {
			t.Fatalf("live gate: %+v", p)
		}
		if p.Component != "openviking" && p.Status != verify.PendingSetup {
			t.Fatalf("pending probe: %+v", p)
		}
	}
	removeInstaller()
	docker("up", "-d", "--no-build", "--pull", "never", "--force-recreate", "hermes")
	waitPending()
	docker("exec", "-T", "--user", "hermes", "hermes", "/bin/sh", "-c", `[ "$(cat /opt/data/openviking/persistence-fixture)" = 'nonsecret persistent fixture' ] && [ ! -e /opt/data/openviking/ov.conf ]`)
	info, err := os.Stat(filepath.Join(root, ".hermes/openviking"))
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatal("memory directory is not private")
	}
	if _, err := os.Stat(installer); !os.IsNotExist(err) {
		t.Fatal("installer survived removal")
	}
	t.Log("PASS: embedded pinned nonroot OpenViking pending server returns 503; install rerun and Compose recreation preserve private mount data after installer removal. Live memory remains unproven.")
}

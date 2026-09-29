//go:build docker

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/dockertest"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// This separately authorized privileged fixture proves the generated DinD
// topology, shared scratch bind mounts and baked helper. Compiling it is only a
// source check; no live qualification is claimed until this explicit gate runs.
// Shared memory stays unconfigured; this uses no provider credentials/model calls.
func TestDockerIsolatedAcceptanceDaemon(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" || os.Getenv("REPOKIT_DIND_TESTS") != "1" {
		t.Skip("requires REPOKIT_DOCKER_TESTS=1 and REPOKIT_DIND_TESTS=1 to authorize privileged disposable DinD")
	}
	dc := os.Getenv("REPOKIT_DOCKER_CONTEXT")
	if dc == "" {
		dc = "default"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	endpoint, err := exec.CommandContext(ctx, "docker", "context", "inspect", dc, "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(endpoint)), "unix://") {
		t.Fatal("authorized local Unix Docker context required")
	}
	root, err := os.MkdirTemp(t.TempDir(), "dind-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	source := map[string][]byte{
		"go.mod":          []byte("module dindfixture\n\ngo 1.26.0\n"),
		"fixture_test.go": []byte("package dindfixture\nimport \"testing\"\nfunc TestLocalCompiler(t *testing.T){if 2+2!=4{t.Fatal(\"compiler\")}}\n"),
		"source.txt":      []byte("original source\n"),
	}
	for name, data := range source {
		if err = os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "--quiet"}, {"add", "go.mod", "fixture_test.go", "source.txt"}} {
		if out, err := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("fixture Git: %v %s", err, out)
		}
	}
	source["source.txt"] = []byte("dirty source\n")
	if err = os.WriteFile(filepath.Join(root, "source.txt"), source["source.txt"], 0600); err != nil {
		t.Fatal(err)
	}
	installer, removeInstaller := disposableCLI(t)
	install := exec.CommandContext(ctx, installer, "install", "--docker-tests")
	install.Dir = root
	install.Env = append(installerEnvironment(t), "DOCKER_CONTEXT="+dc)
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("generated opt-in install: %v\n%s", err, out)
	}
	removeInstaller() // Compose and its baked helper must stand alone thereafter.
	base := []string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose, "--profile", "docker-tests"}
	compose := func(args ...string) []byte {
		t.Helper()
		out, err := exec.CommandContext(ctx, "docker", append(append([]string{}, base...), args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Compose %v: %v\n%s", args, err, out)
		}
		return out
	}
	// This project is derived from a fresh random repository path. Cleanup targets
	// only its generated services and named volumes; no prune or owner runtime.
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 90*time.Second)
		defer stop()
		args := append(append([]string{}, base...), "down", "--volumes", "--remove-orphans", "--timeout", "10")
		if out, err := exec.CommandContext(cleanup, "docker", args...).CombinedOutput(); err != nil {
			t.Errorf("disposable project cleanup: %v\n%s", err, out)
		}
	}()
	compose("config", "--quiet")
	compose("up", "-d", "--build", "hermes", "docker-test")
	daemon := strings.TrimSpace(string(compose("ps", "--quiet", "docker-test")))
	if len(daemon) != 64 || strings.Trim(daemon, "0123456789abcdef") != "" {
		t.Fatal("expected one exact fixture daemon ID")
	}
	deadline := time.Now().Add(2 * time.Minute)
	ready := false
	for time.Now().Before(deadline) && ctx.Err() == nil {
		probe, stop := context.WithTimeout(ctx, 5*time.Second)
		health, err := exec.CommandContext(probe, "docker", "--context", dc, "container", "inspect", "--format", "{{.State.Health.Status}}", daemon).Output()
		stop()
		if err == nil && strings.TrimSpace(string(health)) == "healthy" {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		t.Fatal("disposable DinD did not become healthy within two minutes")
	}
	const topology = `{"privileged":{{json .HostConfig.Privileged}},"entrypoint":{{json .Config.Entrypoint}},"command":{{json .Config.Cmd}},"mounts":{{json .Mounts}},"ports":{{json .HostConfig.PortBindings}},"networks":{{json .NetworkSettings.Networks}}}`
	observed, err := exec.CommandContext(ctx, "docker", "--context", dc, "container", "inspect", "--format", topology, daemon).Output()
	var state struct {
		Privileged          bool
		Entrypoint, Command []string
		Mounts              []struct{ Type, Name, Destination string }
		Ports, Networks     map[string]json.RawMessage
	}
	if err != nil || json.Unmarshal(observed, &state) != nil {
		t.Fatal("cannot read bounded fixture daemon topology")
	}
	if !state.Privileged || !reflect.DeepEqual(state.Entrypoint, []string{"sh", "-ec"}) || !reflect.DeepEqual(state.Command, []string{dockertest.DaemonCommand(os.Getuid(), os.Getgid())}) || len(state.Ports) != 0 || len(state.Networks) != 1 {
		t.Fatal("daemon command, privilege, network or port boundary differs")
	}
	if _, ok := state.Networks[id.Project+"_docker-test"]; !ok {
		t.Fatal("daemon joined an unexpected network")
	}
	mounts := map[string]string{"/docker-test/run": "docker-test-run", "/docker-tests": "docker-test-work", "/var/lib/docker": "docker-test-data"}
	if len(state.Mounts) != len(mounts) {
		t.Fatal("unexpected daemon mounts")
	}
	for _, m := range state.Mounts {
		suffix, ok := mounts[m.Destination]
		if !ok || m.Type != "volume" || m.Name != id.Project+"_"+suffix {
			t.Fatal("daemon exposes a mount outside its test volumes")
		}
	}
	// These are synthetic credentials, deliberately outside .hermes and unignored.
	// The helper must exclude known private names without touching their originals.
	for name, data := range map[string]string{".env": "SYNTHETIC_KEY=fixture-only\n", "auth.json": "{\"synthetic\":true}\n"} {
		source[name] = []byte(data)
		if err = os.WriteFile(filepath.Join(root, name), source[name], 0600); err != nil {
			t.Fatal(err)
		}
	}
	nativeConfig, err := os.ReadFile(filepath.Join(root, ".hermes/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	compose("exec", "-T", "--user", "hermes", "hermes", "sh", "-ec", `test ! -S /var/run/docker.sock; test -z "${DOCKER_HOST:-}"`)
	const nested = `
 case "$PWD" in /docker-tests/fixture-*/source) ;; *) exit 20;; esac
 case "$TMPDIR" in /docker-tests/fixture-*/tmp) ;; *) exit 21;; esac
 test -d .git
 test ! -e .hermes
 test ! -e .env
 test ! -e auth.json
 test "$(cat source.txt)" = 'dirty source'
 test "$DOCKER_HOST" = unix:///docker-test/run/docker.sock
 test "$(docker context inspect default --format '{{.Endpoints.docker.Host}}')" = "$DOCKER_HOST"
 GOTOOLCHAIN=local go test ./...
 docker pull "$1"
 docker run --rm --network none --user "$(id -u):$(id -g)" --entrypoint sh \
   --mount "type=bind,source=$PWD,target=/fixture,readonly" \
   --mount "type=bind,source=$TMPDIR,target=/fixture-tmp" "$1" -ec '
     test "$(cat /fixture/source.txt)" = "dirty source"
     test ! -e /fixture/.hermes
     test ! -e /fixture/.env
     test ! -e /fixture/auth.json
     printf nested-bind-pass > /fixture-tmp/nested-marker
   '
 test "$(cat "$TMPDIR/nested-marker")" = nested-bind-pass
 printf DIND_HELPER_ACCEPTANCE_PASS
 `
	out := compose("exec", "-T", "--user", "hermes", "--workdir", "/workspace", "hermes", "repokit-docker-test", "--timeout", "300", "--", "sh", "-ec", nested, "fixture", dockertest.Image)
	if !bytes.Contains(out, []byte("DIND_HELPER_ACCEPTANCE_PASS")) {
		t.Fatal("nested helper fixture did not complete")
	}
	for name, want := range source {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("helper modified original source: %s", name)
		}
	}
	got, err := os.ReadFile(filepath.Join(root, ".hermes/config.yaml"))
	if err != nil || !bytes.Equal(got, nativeConfig) {
		t.Fatal("helper modified native configuration")
	}
	compose("exec", "-T", "--user", "hermes", "hermes", "sh", "-ec", `test -z "$(find /docker-tests -mindepth 1 -maxdepth 1 -name 'fixture-*' -print -quit)"`)
}

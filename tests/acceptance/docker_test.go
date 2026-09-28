//go:build docker

package acceptance

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/locking"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//go:embed fixtures/team_lifecycle.py
var teamLifecycle string

//go:embed fixtures/memory_link.py
var memoryLinkFixture string

// This proves credential-free foundation behavior only. It is deliberately not
// named the removal-first release gate: no paid chat, review or memory is faked.
func TestDockerFoundation(t *testing.T) {
	if os.Getenv("REPOKIT_DOCKER_TESTS") != "1" {
		t.Skip("set REPOKIT_DOCKER_TESTS=1 with an authorized local Docker daemon")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
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
	runCLI := func(command ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, installer, command...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "DOCKER_CONTEXT="+dc)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("CLI %v: %v %s", command, err, out)
		}
	}
	// Team scaffolding is credential-free, but operational dispatch now requires
	// private memory setup. Prove the exact pending boundary,
	// rather than treating an arbitrary setup error as fixture success.
	runPendingCLI := func(command ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, installer, command...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "DOCKER_CONTEXT="+dc)
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "Operational dispatch pending: shared OpenViking memory is mandatory.") {
			t.Fatalf("expected explicit incomplete-integration gate for %v: %v %s", command, err, out)
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
	docker("up", "-d", "--build", "hermes")
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
	// Docker exec work can outlive its client: prove the container-held flock
	// still excludes a second host installer after the client is killed.
	marker := filepath.Join(root, ".hermes/lock-ready")
	lockArgs := append(append([]string{}, base...), "exec", "-T", "--user", "hermes", "hermes", "/usr/bin/flock", "-n", "/workspace/.hermes-repokit.lock", "sh", "-c", "printf ready > /opt/data/lock-ready; sleep 3")
	client := exec.CommandContext(ctx, "docker", lockArgs...)
	if err := client.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			client.Process.Kill()
			client.Wait()
			t.Fatal("container lock did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	client.Process.Kill()
	client.Wait()
	lockPath := filepath.Join(root, ".hermes-repokit.lock")
	if lock, err := locking.Acquire(lockPath); err == nil {
		lock.Close()
		t.Fatal("Docker client death released active native lock")
	}
	for {
		lock, err := locking.Acquire(lockPath)
		if err == nil {
			lock.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("native lock failed to release")
		}
		time.Sleep(20 * time.Millisecond)
	}
	os.Remove(marker)
	runCLI("install") // Initialize only through native CLI in the existing runtime.
	pending := exec.CommandContext(ctx, installer, "verify")
	pending.Dir = root
	if out, err := pending.CombinedOutput(); err == nil || !strings.Contains(string(out), "pending-setup") {
		t.Fatalf("team before setup: %v %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".hermes/kanban.db")); err != nil {
		t.Fatal("install did not initialize native Kanban")
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".hermes/profiles")); err == nil && len(entries) > 0 {
		t.Fatal("default install created extra profiles")
	}
	// Synthetic provider configuration tests native cloning without borrowing
	// credentials or claiming interactive setup/authenticated inference success.
	if err := os.WriteFile(filepath.Join(root, ".hermes/.env"), []byte("REPOKIT_TEST_SENTINEL=nonsecret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "hermes", "config", "set", "model.default", "fixture-model")
	docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "hermes", "config", "set", "model.provider", "custom")
	os.MkdirAll(filepath.Join(root, ".hermes/memories"), 0700)
	os.WriteFile(filepath.Join(root, ".hermes/memories/MEMORY.md"), []byte("default-only history"), 0600)
	os.WriteFile(filepath.Join(root, ".hermes/memories/USER.md"), []byte("default-only user history"), 0600)

	runPendingCLI("setup", "--team") // Scaffold without claiming private integrations or live dispatch.
	scaffoldVerify := exec.CommandContext(ctx, installer, "verify")
	scaffoldVerify.Dir = root
	output, verifyErr := scaffoldVerify.CombinedOutput()
	var probes []verify.Probe
	if verifyErr == nil || json.Unmarshal(output, &probes) != nil {
		t.Fatalf("full acceptance falsely certified: %v %s", verifyErr, output)
	}
	for _, p := range probes {
		pending := map[string]verify.Status{
			"kanban:dispatch":            verify.Inactive,
			"kanban:dispatch-configured": verify.Inactive,
			"kanban:dispatch-live":       verify.Inactive,
			"kanban:dispatcher-canary":   verify.Unqualified,
			"gateway-inputs":             verify.Unknown,
			"maintenance:live":           verify.Unqualified,
			"channel:cli:route":          verify.Unknown,
			"channel:cli:authorization":  verify.Unqualified,
			"memory:default:fallback":    verify.Inactive,
		}
		if want, ok := pending[p.Component]; ok {
			if p.Status != want {
				t.Fatalf("credential-free boundary: %+v; want %s", p, want)
			}
			continue
		}
		if p.Component == "gateway-generation" {
			if p.Status != verify.Inactive {
				t.Fatalf("unexpected gateway: %+v", p)
			}
			continue
		}

		if strings.HasPrefix(p.Component, "openviking") || p.Component == "memory" || p.Component == "review" {
			if p.Status == verify.Healthy {
				t.Fatal("integration falsely certified")
			}
			continue
		}
		if p.Status != verify.Healthy {
			t.Fatalf("scaffold probe: %+v", p)
		}
	}
	for _, role := range team.ForRepository(id) {
		dir := filepath.Join(root, ".hermes")
		if role.Name != "default" {
			dir = filepath.Join(dir, "profiles", role.Name)
		}
		soul, err := os.ReadFile(filepath.Join(dir, "SOUL.md"))
		if err != nil || string(soul) != role.Soul {
			t.Fatalf("SOUL %s: %v", role.Name, err)
		}
		docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "hermes", "profile", "show", role.Name)
		desc := docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "hermes", "profile", "describe", role.Name)
		if !strings.Contains(string(desc), role.Description) {
			t.Fatalf("description %s", role.Name)
		}
		if role.Name != "default" {
			env, _ := os.ReadFile(filepath.Join(dir, ".env"))
			if !strings.Contains(string(env), "REPOKIT_TEST_SENTINEL=nonsecret") {
				t.Fatal("native config clone missing environment")
			}
			for _, file := range []string{"MEMORY.md", "USER.md"} {
				if _, err := os.Stat(filepath.Join(dir, "memories", file)); !os.IsNotExist(err) {
					t.Fatal("inherited default memory")
				}
			}
		}
	}
	preserved := map[string][]byte{}
	for _, name := range []string{"researcher", "planner", "executor", "reviewer", "steward"} {
		path := filepath.Join(root, ".hermes/profiles", name, "memories/MEMORY.md")
		data := []byte("role lesson must survive rerun")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		preserved[path] = data
	}
	runPendingCLI("install")
	for path, want := range preserved {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("role memory changed")
		}
	}
	// Exercise the actual pinned native config setters and per-profile resolver.
	// This fixture never contacts a memory service or claims authenticated recall.
	memorySource, err := os.ReadFile("../../internal/native/memory.py")
	if err != nil {
		t.Fatal(err)
	}
	docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "python", "-c", string(memorySource)+"\n"+memoryLinkFixture)
	lifecycle := func(action, profile string) {
		home := "/opt/data"
		if profile != "default" {
			home += "/profiles/" + profile
		}
		docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--env", "HERMES_HOME="+home, "--env", "HERMES_PROFILE_NAME="+profile, "hermes", "python", "-c", teamLifecycle, action)
	}
	for _, step := range [][2]string{{"create", "default"}, {"research", "researcher"}, {"plan", "planner"}, {"execute", "executor"}, {"changes", "reviewer"}, {"revise", "executor"}, {"approve", "reviewer"}} {
		lifecycle(step[0], step[1])
	}
	removeInstaller()
	// There is no receipt in a foundation install. Removing source and binary
	// before native commands proves those artifacts cannot be runtime dependencies.
	if _, err := os.Stat(installer); !os.IsNotExist(err) {
		t.Fatal("installer still accessible")
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
	lifecycle("persist", "reviewer")
	docker("up", "-d", "--force-recreate", "hermes")
	ready = false
	for i := 0; i < 60; i++ {
		args := append(append([]string{}, base...), "exec", "-T", "--user", "hermes", "hermes", "test", "-w", "/opt/data/kanban.db.init.lock")
		if exec.CommandContext(ctx, "docker", args...).Run() == nil {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		t.Fatal("recreated Hermes identity not ready")
	}
	lifecycle("persist", "executor")
	cmd := exec.CommandContext(ctx, id.Launcher, "--version")
	cmd.Dir = t.TempDir()
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "v0.21.5") {
		t.Fatalf("native exec after restart/removal: %v %s", err, out)
	}
	t.Log("PASS: real CLI plan/install/verify/rerun; disposable installer binary and source removed; native generic profiles cloned with distinct identities and fresh memories; existing profile edits preserved; raw Compose restart and persistent board. No inference or full v1 gate claimed.")
}

// This runner is used ONLY in the credential-free fixture, to expose native
// compatibility failures while production continues to sanitize captured output.
type fixtureInitializer struct{ t *testing.T }

func (r fixtureInitializer) RunInput(ctx context.Context, input io.Reader, program string, args ...string) process.Result {
	data, _ := io.ReadAll(input)
	script := strings.ReplaceAll(string(data), "stdout=subprocess.DEVNULL", "stdout=None")
	script = strings.ReplaceAll(script, "stderr=subprocess.DEVNULL", "stderr=None")
	script = strings.ReplaceAll(script, "except Exception:\n    print(", "except Exception:\n    import traceback; traceback.print_exc()\n    print(")
	timeout := 2 * time.Minute

	result := (process.Runner{Timeout: timeout}).RunInput(ctx, strings.NewReader(script), program, args...)
	if result.Err != nil {
		r.t.Log(result.Output)
	}
	return result
}

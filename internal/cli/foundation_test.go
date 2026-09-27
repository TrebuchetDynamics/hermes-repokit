package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

type foundationRunner struct {
	id                      target.Identity
	context, names, runtime string
	lists                   int
	onList                  func(int)
}

func (r *foundationRunner) Run(ctx context.Context, program string, args ...string) process.Result {
	if program == "git" {
		return (process.Runner{}).Run(ctx, program, args...)
	}
	call := strings.Join(args, " ")
	switch {
	case call == "context show":
		return process.Result{Output: r.context}
	case strings.Contains(call, "context inspect"):
		return process.Result{Output: "unix:///var/run/docker.sock"}
	case strings.Contains(call, "container ls"):
		r.lists++
		if r.onList != nil {
			r.onList(r.lists)
		}
		return process.Result{Output: r.names}
	case strings.HasPrefix(call, "--context "+r.context+" container inspect"):
		return process.Result{Output: r.runtime}
	default:
		return process.Result{Err: fmt.Errorf("unexpected command %s", call)}
	}
}
func foundationApp(t *testing.T) (App, *foundationRunner) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Test Project")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git: %v %s", err, out)
	}
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	r := &foundationRunner{id: id, context: "local-test"}
	return App{Directory: root, Runner: r}, r
}
func invoke(t *testing.T, a App, args ...string) (int, string, string) {
	t.Helper()
	var out, diag bytes.Buffer
	code := a.Run(args, &out, &diag)
	return code, out.String(), diag.String()
}
func TestFoundationInstallAndVerifyWithoutOptionalIntegrations(t *testing.T) {
	a, r := foundationApp(t)
	code, out, diag := invoke(t, a, "plan")
	var plan Plan
	if code != 0 || json.Unmarshal([]byte(out), &plan) != nil || len(plan.Plugins) != 0 || len(plan.Unsupported) != 0 {
		t.Fatalf("plan: %d %s %s", code, out, diag)
	}
	code, out, diag = invoke(t, a, "install")
	if code != 0 {
		t.Fatalf("install: %d %s", code, diag)
	}
	if !strings.Contains(out, "--context 'local-test'") || !strings.Contains(out, "up -d hermes") {
		t.Fatalf("missing standalone handoff: %s", out)
	}
	for _, name := range []string{"compose.yaml", "config.yaml", "bin/hermes-test-project"} {
		if _, err := os.Stat(filepath.Join(a.Directory, ".hermes", name)); err != nil {
			t.Fatal(err)
		}
	}
	if r.lists < 2 {
		t.Fatal("inventory was not rechecked under lock")
	}
	data, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "openviking") || strings.Contains(string(data), "laya") {
		t.Fatal("unexpected sidecar")
	}
	r.runtime = fmt.Sprintf(`{"status":"running","project":%q,"workspace":%q,"home":%q}`, r.id.Project, r.id.Root, filepath.Join(r.id.Root, ".hermes"))
	code, out, diag = invoke(t, a, "verify")
	if code != 0 {
		t.Fatalf("foundation verify: %d %s %s", code, out, diag)
	}
	var probes []verify.Probe
	if err := json.Unmarshal([]byte(out), &probes); err != nil {
		t.Fatal(err)
	}
	for _, p := range probes {
		if p.Component == "openviking" || p.Component == "nerve-laya" || p.Component == "superpowers" {
			t.Fatalf("unselected probe: %+v", p)
		}
	}
	config := filepath.Join(a.Directory, ".hermes/config.yaml")
	edited := []byte("# native owner edit\nkanban:\n  dispatch_in_gateway: false\n")
	if err := os.WriteFile(config, edited, 0600); err != nil {
		t.Fatal(err)
	}
	r.names = r.id.Container
	// Current context changed: rerun must inspect and preserve captured context.
	r.context = "local-test"
	a.Runner = contextSwitchRunner{r}
	if code, _, diag = invoke(t, a, "install"); code != 0 {
		t.Fatalf("rerun: %s", diag)
	}
	got, _ := os.ReadFile(config)
	if !bytes.Equal(got, edited) {
		t.Fatal("native config overwritten")
	}
	r.runtime = strings.Replace(r.runtime, "running", "exited", 1)
	if code, _, _ = invoke(t, a, "verify"); code == 0 {
		t.Fatal("stopped runtime passed verification")
	}
}

type contextSwitchRunner struct{ *foundationRunner }

func (r contextSwitchRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	if strings.Join(args, " ") == "context show" {
		return process.Result{Output: "different-context"}
	}
	return r.foundationRunner.Run(ctx, p, args...)
}
func TestFoundationRefusesForeignContainerBeforePublication(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprint(late), func(t *testing.T) {
			a, r := foundationApp(t)
			if late {
				r.onList = func(n int) {
					if n == 2 {
						r.names = r.id.Container
					}
				}
			} else {
				r.names = r.id.Container
			}
			if code, _, _ := invoke(t, a, "install"); code == 0 {
				t.Fatal("foreign container accepted")
			}
			if _, err := os.Lstat(filepath.Join(a.Directory, ".hermes")); !os.IsNotExist(err) {
				t.Fatal("state published despite collision")
			}
		})
	}
}

func TestFoundationRefusesStateExposedToGit(t *testing.T) {
	for _, exposure := range []string{"removed-ignore", "tracked-state", "ignore-exception"} {
		t.Run(exposure, func(t *testing.T) {
			a, r := foundationApp(t)
			if code, _, diag := invoke(t, a, "install"); code != 0 {
				t.Fatal(diag)
			}
			r.runtime = fmt.Sprintf(`{"status":"running","project":%q,"workspace":%q,"home":%q}`, r.id.Project, r.id.Root, filepath.Join(r.id.Root, ".hermes"))
			ignore := filepath.Join(r.id.Root, ".hermes/.gitignore")
			switch exposure {
			case "removed-ignore":
				if err := os.Remove(ignore); err != nil {
					t.Fatal(err)
				}
			case "ignore-exception":
				if err := os.WriteFile(ignore, []byte("*\n!config.yaml\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "tracked-state":
				if out, err := exec.Command("git", "-C", r.id.Root, "add", "--force", ".hermes/config.yaml").CombinedOutput(); err != nil {
					t.Fatalf("git add: %v %s", err, out)
				}
			}
			for _, command := range []string{"install", "verify"} {
				if code, out, diag := invoke(t, a, command); code == 0 {
					t.Fatalf("%s accepted exposed state: %s %s", command, out, diag)
				}
			}
			if exposure == "removed-ignore" {
				if _, err := os.Lstat(ignore); !os.IsNotExist(err) {
					t.Fatal("owner file repaired implicitly")
				}
			}
		})
	}
}

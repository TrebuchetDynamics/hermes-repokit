package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"io"
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
	composeCalls            []string
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
	case strings.Contains(call, "image inspect"):
		if args[len(args)-1] == qualification.FoundationImage {
			return process.Result{Output: `["sha256:` + strings.Repeat("e", 64) + `"]`}
		}
		o, ok := compose.DevelopmentSelected(r.id)
		if !ok {
			return process.Result{Err: fmt.Errorf("no derived recipe")}
		}
		data, _ := json.Marshal(map[string]any{"id": "sha256:" + strings.Repeat("d", 64), "os": "linux", "arch": "amd64", "recipe": development.Fingerprint(*o.Development), "base": qualification.FoundationImage, "layers": []string{"sha256:" + strings.Repeat("e", 64), "sha256:" + strings.Repeat("f", 64)}})
		return process.Result{Output: string(data)}
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
	home := t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
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
	// Record Compose calls without starting anything; tests that need a
	// running deployment set r.runtime themselves.
	compose := func(_, _ io.Writer, args ...string) error {
		r.composeCalls = append(r.composeCalls, strings.Join(args, " "))
		return nil
	}
	return App{Directory: root, Runner: r, Initializer: r, ComposeExec: compose}, r
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
	if len(r.composeCalls) == 0 || !strings.Contains(r.composeCalls[len(r.composeCalls)-1], "--context local-test compose --env-file /dev/null -f "+r.id.Compose+" up -d --build hermes") {
		t.Fatalf("install did not build and start the deployment: %v\n%s", r.composeCalls, out)
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
	if strings.Contains(string(data), "  laya:") || !strings.Contains(string(data), "container_name:") {
		t.Fatal("unexpected sidecar selection")
	}
	r.runtime = developmentRuntimeFixture(r.id)
	if code, _, _ = invoke(t, a, "verify"); code == 0 {
		t.Fatal("uninitialized native Kanban passed verify")
	}
	if code, _, diag = invoke(t, a, "install"); code != 0 {
		t.Fatalf("initialize native board: %s", diag)
	}
	code, out, diag = invoke(t, a, "verify")
	if code != 1 || !strings.Contains(out, "pending-setup") {
		t.Fatalf("unconfigured team verify: %d %s %s", code, out, diag)
	}
	var probes []verify.Probe
	if err := json.Unmarshal([]byte(out), &probes); err != nil {
		t.Fatal(err)
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
			r.runtime = developmentRuntimeFixture(r.id)
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

func TestEngineeringProfilesCanBeSelectedBeforeSidecars(t *testing.T) {
	a, r := foundationApp(t)
	code, out, diag := invoke(t, a, "plan", "--engineering")
	var plan Plan
	if code != 0 || json.Unmarshal([]byte(out), &plan) != nil || len(plan.Unsupported) > 0 || len(plan.Profiles) != 7 {
		t.Fatalf("engineering plan: %d %s %s", code, out, diag)
	}
	if code, out, diag = invoke(t, a, "install", "--engineering"); code != 0 {
		t.Fatalf("engineering artifact install: %d %s %s", code, out, diag)
	}
	if _, err := os.Stat(filepath.Join(r.id.Root, ".hermes/profiles")); !os.IsNotExist(err) {
		t.Fatal("profiles created without native runtime")
	}
	if !strings.Contains(out, "pending") {
		t.Fatal("missing native initialization status")
	}
}

func (r *foundationRunner) RunInput(_ context.Context, input io.Reader, _ string, args ...string) process.Result {
	if input == nil && strings.Contains(strings.Join(args, " "), " config get ") {
		return process.Result{Output: "null"} // public CLI read: nothing configured yet
	}
	return process.Result{Output: `REPOKIT_TEAM={"status":"pending-setup","drift":[]}`, Err: os.WriteFile(filepath.Join(r.id.Root, ".hermes/kanban.db"), []byte("native board fixture"), 0600)}
}

func TestExistingPlanDisclosesNativeInitialization(t *testing.T) {
	a, _ := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	code, out, diag := invoke(t, a, "plan", "--engineering")
	var p Plan
	if code != 0 || json.Unmarshal([]byte(out), &p) != nil {
		t.Fatalf("plan: %s %s", out, diag)
	}
	proposed := strings.Join(p.ProposedChanges, " ")
	if !strings.Contains(proposed, "Kanban") || !strings.Contains(proposed, "profiles") {
		t.Fatalf("hidden native effects: %s", proposed)
	}
}

func TestGenericTeamIsDefaultPlan(t *testing.T) {
	a, _ := foundationApp(t)
	code, out, diag := invoke(t, a, "plan")
	var p Plan
	if code != 0 || json.Unmarshal([]byte(out), &p) != nil {
		t.Fatalf("%s %s", out, diag)
	}
	if strings.Join(p.Profiles, ",") != "default,researcher,planner,executor,tester,reviewer,steward" {
		t.Fatalf("roster %v", p.Profiles)
	}
	if p.Kanban["orchestrator_profile"] != "default" || p.Kanban["max_in_progress"] != float64(1) {
		t.Fatalf("defaults %v", p.Kanban)
	}
}

func TestVerifyDoesNotCertifyUnqualifiedIntegrations(t *testing.T) {
	a, _ := foundationApp(t)
	_, out, _ := invoke(t, a, "verify")
	var probes []verify.Probe
	if json.Unmarshal([]byte(out), &probes) != nil {
		t.Fatal(out)
	}
	pending := map[string]bool{}
	for _, p := range probes {
		switch p.Component {
		case "memory", "review:evidence", "CORE_READY", "MEMORY":
			pending[p.Component] = p.Status != verify.Healthy
		}
	}
	if !pending["memory"] || !pending["review:evidence"] || !pending["CORE_READY"] || !pending["MEMORY"] {
		t.Fatal("missing explicit integration gates")
	}
}

func TestLegacyEngineeringIsOnlyAnAlias(t *testing.T) {
	a, _ := foundationApp(t)
	for _, installed := range []bool{false, true} {
		if installed {
			if code, _, diag := invoke(t, a, "install"); code != 0 {
				t.Fatal(diag)
			}
		}
		_, plain, _ := invoke(t, a, "plan")
		_, legacy, _ := invoke(t, a, "plan", "--engineering")
		if plain != legacy {
			t.Fatal("legacy flag changes universal team plan")
		}
	}
}

func TestSetupRefusesEditedLauncherBeforeExecution(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	marker := filepath.Join(a.Directory, "unexpected-execution")
	if err := os.WriteFile(r.id.Launcher, []byte("#!/bin/sh\ntouch '"+marker+"'\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	code, _, diag := invoke(t, a, "setup")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("executed an unverified launcher")
	}
	if code != 1 || !strings.Contains(diag, "routing") {
		t.Fatalf("%d %s", code, diag)
	}
}

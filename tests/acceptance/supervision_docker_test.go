//go:build docker

package acceptance

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

//go:embed fixtures/supervision_probe.py
var supervisionProbe string

// This is real upstream Nerve/Laya inference, independent of main-model auth.
// It intentionally does not claim model-driven Kanban work or live memory.
func TestDockerSupervisionStack(t *testing.T) {
	image := os.Getenv("REPOKIT_LAYA_IMAGE")
	if os.Getenv("REPOKIT_NERVE_DOCKER_TESTS") != "1" || image == "" {
		t.Skip("set REPOKIT_NERVE_DOCKER_TESTS=1 and REPOKIT_LAYA_IMAGE to a qualified local image ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	root, err := os.MkdirTemp("", "repokit-supervision-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	installer, removeInstaller := disposableCLI(t)
	run := func(program string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, program, args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "DOCKER_CONTEXT=default")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("disposable native command failed: %v\n%s", err, output)
		}
		return output
	}
	run("git", "init", "--quiet")
	run(installer, "install", "--laya-image", image)
	base := []string{"--context", "default", "compose", "--env-file", "/dev/null", "-f", id.Compose}
	docker := func(args ...string) []byte { return run("docker", append(append([]string{}, base...), args...)...) }
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if out, err := exec.CommandContext(cleanup, "docker", append(base, "down")...).CombinedOutput(); err != nil {
			t.Errorf("cleanup: %v %s", err, out)
		}
	}()
	docker("up", "-d", "hermes", "laya")
	for i := 0; i < 90; i++ {
		cmd := exec.CommandContext(ctx, "docker", append(append([]string{}, base...), "exec", "-T", "--user", "hermes", "hermes", "test", "-w", "/opt/data/config.yaml")...)
		if cmd.Run() == nil {
			break
		}
		if i == 89 {
			t.Fatal("Hermes native home never writable")
		}
		time.Sleep(time.Second)
	}
	// Auth-free synthetic baseline qualifies native profile provisioning only.
	docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "hermes", "config", "set", "model.default", "fixture-model")
	docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "hermes", "hermes", "config", "set", "model.provider", "custom")
	if err := native.Initialize(ctx, id, "default", true, fixtureInitializer{t}); err != nil {
		t.Fatal(err)
	}
	// Exercise the public setup path, including first native admission,
	// owned-runtime preflight and
	// coordinated gateway/Laya recreation, before removing the bootstrapper.
	run(installer, "setup", "--supervision")
	verifyCommand := exec.CommandContext(ctx, installer, "verify")
	verifyCommand.Dir = root
	verifyOutput, verifyErr := verifyCommand.Output()
	var probes []struct{ Component, Status string }
	if verifyErr == nil || json.Unmarshal(verifyOutput, &probes) != nil {
		t.Fatalf("verify must report remaining memory/review gates: %v %s", verifyErr, verifyOutput)
	}
	observed := map[string]string{}
	for _, p := range probes {
		observed[p.Component] = p.Status
	}
	if observed["nerve"] != "healthy" || observed["laya"] != "healthy" || observed["review"] != "unqualified" {
		t.Fatalf("native read-only component reporting differs: %s", verifyOutput)
	}
	// Repeat after source/binary removal and external network disconnect. All
	// native packages/configuration persist in the ordinary Compose deployment.
	removeInstaller()
	network := id.Project + "_default"
	run("docker", "--context", "default", "network", "disconnect", network, id.Container)
	for _, role := range team.Roster() {
		home := "/opt/data"
		if role.Name != "default" {
			home += "/profiles/" + role.Name
		}
		docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--env", "HERMES_HOME="+home, "--env", "HERMES_PROFILE="+role.Name, "hermes", "python", "-c", supervisionProbe)
	}
	docker("stop", "laya")
	outage := `import json
from hermes_cli.plugins import discover_plugins,get_plugin_manager
from tools.registry import registry
discover_plugins()
r=registry.dispatch('nerve_decide',{'state':'Synthetic outage','instructions':'Choose','choices':['a','b']},scope=get_plugin_manager().scope_key)
if isinstance(r,str):r=json.loads(r)
assert r.get('ok') is False and 'Laya sidecar connection failed' in r.get('error','')
print('local outage failed closed')`
	out := docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--env", "HERMES_HOME=/opt/data/profiles/executor", "--env", "HERMES_PROFILE=executor", "hermes", "python", "-c", outage)
	if !strings.Contains(string(out), "local outage failed closed") {
		t.Fatal("outage not observed")
	}
	// Reconnect only to let Compose restore its declared network topology.
	run("docker", "--context", "default", "network", "connect", network, id.Container)
	docker("up", "-d", "--force-recreate", "hermes", "laya")
	for i := 0; i < 90; i++ {
		output, err := exec.CommandContext(ctx, "docker", "--context", "default", "inspect", "--format", "{{json .State.Health.Status}}", id.Project+"-laya-1").Output()
		var health string
		if err == nil && json.Unmarshal(output, &health) == nil && health == "healthy" {
			break
		}
		if i == 89 {
			t.Fatal("recreated Laya not healthy")
		}
		time.Sleep(time.Second)
	}
	for _, role := range team.Roster() {
		home := "/opt/data"
		if role.Name != "default" {
			home += "/profiles/" + role.Name
		}
		docker("exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--env", "HERMES_HOME="+home, "--env", "HERMES_PROFILE="+role.Name, "hermes", "python", "-c", supervisionProbe)
	}
	if _, err := os.Stat(filepath.Join(root, ".hermes/kanban.db")); err != nil {
		t.Fatal("Kanban state lost")
	}
	t.Log("six native profile plugins, hooks, LOCAL_ONLY decisions, offline outage and Compose recreation passed; main-model work and memory acceptance remain unqualified")
}

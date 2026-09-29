package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

func (a App) install(id target.Identity, report Plan, engineering bool, stdout, stderr io.Writer) int {
	if len(report.Unsupported) > 0 {
		fmt.Fprintln(stderr, "installation qualification incomplete:", strings.Join(report.Unsupported, "; "))
		return 1
	}
	if len(report.Collisions) > 0 {
		fmt.Fprintln(stderr, "target collisions:", strings.Join(report.Collisions, "; "))
		return 1
	}
	// An old sidecar and embedded server must never share the same writable
	// database concurrently. Do not stop/delete owner runtime from install.
	oldMemory := a.Runner.Run(context.Background(), "docker", "--context", report.DockerContext, "container", "ls", "--filter", "label=com.docker.compose.project="+id.Project, "--filter", "label=com.docker.compose.service=openviking", "--format", "{{.ID}}")
	if oldMemory.Err != nil || oldMemory.Truncated || strings.TrimSpace(oldMemory.Output) != "" {
		fmt.Fprintln(stderr, "installation refused: verify that the previous OpenViking sidecar is stopped through its original Compose file before embedding memory; preserve .hermes/openviking data")
		return 1
	}
	data, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &report.Development, DockerTests: report.DockerTests, UID: os.Getuid(), GID: os.Getgid(), SELinux: a.selinuxState()})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	script, err := launcher.Render(id, report.DockerContext)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	artifacts := map[string]install.Artifact{
		"compose.yaml":          {Data: data, Mode: 0600},
		"config.yaml":           {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n  orchestrator_profile: default\n  max_in_progress: 1\ntoolsets: [kanban, memory]\nplatform_toolsets:\n  cli: [kanban, memory]\nterminal:\n  backend: local\n  cwd: /workspace\nskills:\n  trusted_project_dirs: [/workspace]\n"), Mode: 0600},
		"bin/" + id.Container:   {Data: script, Mode: 0700},
		"openviking/.gitignore": {Data: []byte("*\n"), Mode: 0600},
	}
	recipe, err := development.Recipe(report.Development)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for name, data := range recipe {
		artifacts["development-image/"+name] = install.Artifact{Data: data, Mode: 0600}
	}
	var previous []install.StackUpgrade
	priorID := target.PreviousNames(id)
	priorLauncher := ""
	if priorID.Container != id.Container {
		priorLauncher = priorID.Container
	}
	// Any older generated recipe self-certifies through its fingerprint label;
	// an owner-edited recipe does not and is never upgraded. An interrupted
	// upgrade is recognized from the exact recipe copy saved beside its backup.
	type olderRecipe struct {
		files       map[string][]byte
		fingerprint string
	}
	var olderRecipes []olderRecipe
	candidates := []string{filepath.Join(id.Root, ".hermes", "development-image")}
	saved, _ := filepath.Glob(filepath.Join(id.Root, ".hermes", "compose.before-*.development-image"))
	for _, dir := range append(candidates, saved...) {
		files, fingerprint, ok := development.ReadGeneratedRecipe(dir)
		if ok && fingerprint != development.Fingerprint(report.Development) {
			olderRecipes = append(olderRecipes, olderRecipe{files, fingerprint})
		}
	}
	for _, memory := range []string{"", projectmemory.Image} {
		for _, tests := range []bool{false, true} {
			if tests && !report.DockerTests {
				continue
			}
			for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
				opts := compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: memory, Development: &report.Development, DockerTests: tests, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}
				for _, older := range olderRecipes {
					old, err := compose.LegacyDevelopment(id, opts, older.fingerprint)
					if err != nil {
						fmt.Fprintln(stderr, err)
						return 1
					}
					previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-path.yaml", PreviousRecipe: older.files, PreviousLauncher: priorLauncher})
					old, err = compose.OlderRecipe(id, opts, older.fingerprint)
					if err != nil {
						fmt.Fprintln(stderr, err)
						return 1
					}
					previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-recipe-" + older.fingerprint[:12] + ".yaml", PreviousRecipe: older.files})
				}
				old, err := compose.PreviousNames(id, opts)
				if err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-names.yaml", PreviousLauncher: priorLauncher})
			}
		}
	}
	legacy, err := compose.LegacyLayaBuild(priorID, os.Getuid(), os.Getgid())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	previous = append(previous, install.StackUpgrade{Compose: legacy, BackupName: "compose.before-core.yaml", PreviousLauncher: priorLauncher})
	for _, memory := range []string{"", projectmemory.Image} {
		old, err := compose.PreviousNames(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: memory, UID: os.Getuid(), GID: os.Getgid()})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, PrepareMemory: memory == "", PreviousLauncher: priorLauncher})
	}
	if a.selinuxState().Enabled() {
		// Recognize a previously generated development runtime without SELinux
		// relabeling so an existing deployment upgrades in place.
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &report.Development, DockerTests: report.DockerTests, UID: os.Getuid(), GID: os.Getgid()})
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-selinux.yaml"})
	}
	if report.DockerTests {
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &report.Development, UID: os.Getuid(), GID: os.Getgid()})
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-docker-tests.yaml"})
	}
	created, err := install.PublishStackChecked(id, artifacts, previous, func() error {
		current := a.plan(id, false)
		if len(current.Unsupported) > 0 {
			return fmt.Errorf("selected integration became unqualified")
		}
		if len(current.Collisions) > 0 {
			return fmt.Errorf("target changed: %s", strings.Join(current.Collisions, "; "))
		}
		if current.Development.Go != report.Development.Go {
			return fmt.Errorf("repository toolchain requirements changed during installation")
		}
		if current.DockerContext != report.DockerContext {
			return fmt.Errorf("Docker context changed during installation")
		}
		return nil
	})
	if err != nil {
		if created {
			fmt.Fprintln(stderr, "artifacts published, but durability confirmation failed; preserve .hermes and inspect before retrying")
		}
		fmt.Fprintln(stderr, "installation refused:", err)
		return 1
	}
	if created {
		fmt.Fprintln(stdout, "Created Hermes with embedded OpenViking bootstrap artifacts; native setup is pending.")
	} else {
		fmt.Fprintln(stdout, "Preserved existing Hermes deployment and native configuration.")
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		fmt.Fprintf(stderr, "Warning: host command unavailable: %v. Use %s directly.\n", homeErr, id.Launcher)
	} else if command, err := launcher.Expose(id, home); err != nil {
		fmt.Fprintf(stderr, "Warning: host command unavailable: %v. Use %s directly.\n", err, id.Launcher)
	} else {
		fmt.Fprintf(stdout, "Host command: %s -> %s\n", command, id.Launcher)
		if !launcher.OnPath(command, a.Path) {
			fmt.Fprintf(stderr, "Warning: %s is not on PATH as an absolute directory; use %s directly or add that directory to your shell environment.\n", filepath.Dir(command), command)
		}
	}
	state := a.selinuxState()
	relabel := "disabled"
	if state.Enabled() {
		relabel = "enabled (private Z)"
	}
	fmt.Fprintf(stdout, "Host security: SELinux %s; bind relabeling %s.\n", state, relabel)
	if code := a.initialize(id, report.DockerContext, false, stdout, stderr); code != 0 {
		return code
	}
	// Only a complete scaffold can yield a generation. Initial publication and
	// pending native setup still report pending without starting any service.
	complete := true
	for _, probe := range verify.Profiles(id) {
		complete = complete && probe.Status == verify.Healthy
	}
	if ready, _ := a.nativeRuntimeReady(id, report.DockerContext); complete && ready {
		if code := a.finishSetup(id, report.DockerContext, 0, stdout, stderr); code != 0 {
			return code
		}
	}
	fmt.Fprintln(stdout, "Start Hermes with ordinary Compose:")
	fmt.Fprintln(stdout, strings.TrimSuffix(launcher.StartCommand(id.Compose, report.DockerContext), " up -d hermes)")+" up -d --build hermes)")
	fmt.Fprintln(stdout, "OpenViking runs inside Hermes; native memory setup remains pending until configured.")
	if report.DockerTests {
		fmt.Fprintln(stdout, "Docker acceptance is opt-in and privileged; its daemon owns only disposable test storage, not the host Docker socket.")
		fmt.Fprintln(stdout, strings.TrimSuffix(launcher.StartCommand(id.Compose, report.DockerContext), " up -d hermes)")+" --profile docker-tests up -d docker-test)")
		fmt.Fprintln(stdout, "After activation, authorized workers can run repokit-docker-test from /workspace; normal coding never needs this daemon.")
	}
	fmt.Fprintln(stdout, "Run hermes-repokit setup in your private terminal for native default, team and shared memory setup.")
	fmt.Fprintln(stdout, "Resume individual stages with setup --team or setup --memory. Rerun install to reconcile existing generated profiles.")
	return 0
}

func (a App) initialize(id target.Identity, dockerContext string, afterSetup bool, stdout, stderr io.Writer) int {
	ready, err := a.nativeRuntimeReady(id, dockerContext)
	if err != nil {
		fmt.Fprintln(stderr, "native initialization refused:", err)
		return 1
	}
	if !ready {
		fmt.Fprintln(stdout, "Native initialization pending: start the existing Compose service, then rerun install.")
		if afterSetup {
			return 1
		}
		return 0
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 2 * time.Minute}
	}
	if err := a.waitForNativeCLI(id, dockerContext); err != nil {
		fmt.Fprintln(stderr, "native initialization deferred:", err)
		return 1
	}
	if err := native.Initialize(context.Background(), id, dockerContext, afterSetup, runner); err != nil {
		if errors.Is(err, native.ErrTeamPending) && !afterSetup {
			fmt.Fprintln(stdout, "Native shared Kanban checked; team setup pending. After native default setup, run hermes-repokit setup --team without repeating login.")
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Native shared Kanban and seven-profile team reconciled. Memory and model-driven acceptance are separate stages.")
	return 0
}

// nativeReadyTimeout bounds the wait for a just-started container's Hermes CLI.
var nativeReadyTimeout = 90 * time.Second

func (a App) waitForNativeCLI(id target.Identity, dockerContext string) error {
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 30 * time.Second}
	}
	return native.WaitForCLI(context.Background(), id, dockerContext, runner, nativeReadyTimeout, time.Second)
}

package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
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
	u := newUI(stdout, stderr)
	u.title("install", id.Name, id.Root)
	if len(report.Unsupported) > 0 {
		fmt.Fprintln(stderr, "installation qualification incomplete:", strings.Join(report.Unsupported, "; "))
		return 1
	}
	if len(report.Collisions) > 0 {
		fmt.Fprintln(stderr, "target collisions:", strings.Join(report.Collisions, "; "))
		return 1
	}
	data, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, Development: &report.Development, DockerTests: report.DockerTests, UID: os.Getuid(), GID: os.Getgid(), SELinux: a.selinuxState()})
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
		"compose.yaml":        {Data: data, Mode: 0600},
		"config.yaml":         {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n  orchestrator_profile: default\n  max_in_progress: 1\ntoolsets: [kanban, memory]\nplatform_toolsets:\n  cli: [kanban, memory]\nterminal:\n  backend: local\n  cwd: /workspace\nskills:\n  trusted_project_dirs: [/workspace]\n"), Mode: 0600},
		"bin/" + id.Container: {Data: script, Mode: 0700},
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
	for _, tests := range []bool{false, true} {
		if tests && !report.DockerTests {
			continue
		}
		for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
			opts := compose.Options{HermesImage: qualification.FoundationImage, Development: &report.Development, DockerTests: tests, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}
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
	legacy, err := compose.LegacyLayaBuild(priorID, os.Getuid(), os.Getgid())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	previous = append(previous, install.StackUpgrade{Compose: legacy, BackupName: "compose.before-core.yaml", PreviousLauncher: priorLauncher})
	old, err := compose.PreviousNames(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	previous = append(previous, install.StackUpgrade{Compose: old, PreviousLauncher: priorLauncher})
	if a.selinuxState().Enabled() {
		// Recognize a previously generated development runtime without SELinux
		// relabeling so an existing deployment upgrades in place.
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, Development: &report.Development, DockerTests: report.DockerTests, UID: os.Getuid(), GID: os.Getgid()})
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-selinux.yaml"})
	}
	if report.DockerTests {
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, Development: &report.Development, UID: os.Getuid(), GID: os.Getgid()})
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
	if err := a.excludeInstallLock(context.Background(), id); err != nil {
		u.warn(".hermes-repokit.lock may appear in git status (%v); add %s to .git/info/exclude", err, lockExcludeEntry)
	}
	if created {
		u.ok("Deployment", "created private .hermes state, Compose file and launcher")
	} else {
		u.ok("Deployment", "existing deployment and native configuration preserved")
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		u.warn("host command unavailable (%v); use %s directly", homeErr, tildePath(id.Launcher))
	} else if command, err := launcher.Expose(id, home); err != nil {
		u.warn("host command unavailable (%v); use %s directly", err, tildePath(id.Launcher))
	} else {
		u.ok("Host command", tildePath(command)+" → "+tildePath(id.Launcher))
		if !launcher.OnPath(command, a.Path) {
			u.warn("%s is not on PATH; add it to your shell's PATH or run %s directly", tildePath(filepath.Dir(command)), tildePath(command))
		}
	}
	if state := a.selinuxState(); state.Enabled() {
		u.ok("Host security", "SELinux "+string(state)+"; private Z relabeling on repository mounts")
	} else {
		u.ok("Host security", "SELinux "+string(state)+"; no bind relabeling needed")
	}
	if code := a.startDeployment(id, report.DockerContext, stdout, stderr); code != 0 {
		return code
	}
	if code := a.initialize(id, report.DockerContext, false, stdout, stderr); code != 0 {
		return code
	}
	// Only a complete scaffold can yield a generation.
	complete := true
	for _, probe := range verify.Profiles(id) {
		complete = complete && probe.Status == verify.Healthy
	}
	ready, _ := a.nativeRuntimeReady(id, report.DockerContext)
	if complete && ready {
		if code := a.finishSetup(id, report.DockerContext, 0, stdout, stderr); code != 0 {
			return code
		}
	}
	if report.DockerTests {
		u.pending("Docker tests", "opt-in privileged test daemon (no host Docker socket) not started")
		u.note("start it: " + strings.TrimSuffix(launcher.StartCommand(id.Compose, report.DockerContext), " up -d hermes)") + " --profile docker-tests up -d docker-test)")
	}
	switch {
	case !ready:
		u.next([2]string{"repokit start", "start the deployment, then rerun repokit install"})
	case !complete:
		u.next([2]string{"repokit setup", "choose the model provider and create the team (in your own terminal)"})
	default:
		u.next([2]string{"repokit verify", "check readiness"}, [2]string{id.Container, "talk to your team"})
	}
	return 0
}

func (a App) initialize(id target.Identity, dockerContext string, afterSetup bool, stdout, stderr io.Writer) int {
	u := newUI(stdout, stderr)
	ready, err := a.nativeRuntimeReady(id, dockerContext)
	if err != nil {
		u.fail("native initialization refused: %v", err)
		return 1
	}
	if !ready {
		u.pending("Kanban", "initialization pending: the container is not running")
		if a.resetProfile != "" {
			u.fail("profile reset not applied: the container is not running")
			return 1
		}
		if afterSetup {
			return 1
		}
		return 0
	}
	runner := a.Initializer
	if runner == nil {
		// Provisioning seven native profiles is a long chain of hermes calls;
		// on a freshly booted container it exceeded two minutes and was killed.
		runner = process.Runner{Timeout: 10 * time.Minute}
	}
	if err := a.waitForNativeCLI(id, dockerContext); err != nil {
		u.fail("native initialization deferred: %v", err)
		return 1
	}
	teamReport, err := native.Initialize(context.Background(), id, dockerContext, afterSetup, a.resetProfile, runner)
	if err != nil {
		if errors.Is(err, native.ErrTeamPending) && !afterSetup && a.resetProfile == "" {
			u.ok("Kanban", "native board ready")
			u.pending("Team", "not set up yet")
			return 0
		}
		u.fail("%v", err)
		return 1
	}
	u.ok("Kanban", "native board ready")
	u.ok("Team", "seven profiles reconciled")
	if len(teamReport.Customized) > 0 {
		u.note("owner-customized profiles preserved: " + strings.Join(teamReport.Customized, ", ") + " (RepoKit does not overwrite them or apply newer defaults)")
	}
	if len(teamReport.Reset) > 0 {
		u.ok("Reset", strings.Join(teamReport.Reset, ", ")+" returned to RepoKit baseline")
		u.note("prior SOUL.md, config.yaml and profile.yaml kept beside them as *.before-reset-<UTC time>; start a fresh conversation to use the new identity")
	} else if a.resetProfile != "" {
		u.ok("Reset", a.resetProfile+" had no existing profile to reset; it now starts from RepoKit's baseline")
	}
	if len(teamReport.Deferred) > 0 {
		u.pending("Team", "SOUL upgrades deferred while a card is running: "+strings.Join(teamReport.Deferred, ", ")+"; rerun install when the board is idle")
	}
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

package cli

import (
	"bytes"
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

func (a App) install(id target.Identity, report Plan, stdout, stderr io.Writer) int {
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
	// The seed keeps Hermes's native CLI preset and opts into Kanban, which
	// Hermes leaves off by default. Memory and every other native tool stay as
	// Hermes ships them; RepoKit neither adds nor removes them.
	artifacts := map[string]install.Artifact{
		"compose.yaml":        {Data: data, Mode: 0600},
		"config.yaml":         {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n  orchestrator_profile: default\n  max_in_progress: 1\ntoolsets: [hermes-cli, kanban]\nplatform_toolsets:\n  cli: [hermes-cli, kanban]\nterminal:\n  backend: local\n  cwd: /workspace\nskills:\n  trusted_project_dirs: [/workspace]\n"), Mode: 0600},
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
	// Reconfiguring a current deployment republishes it in place: a changed
	// recipe (such as a repository gaining a go.mod), opting into the Docker
	// test daemon, or a host that enabled SELinux. Each exact preimage is
	// backed up first. The only earlier release recognized is v0.2.0's Go
	// render, which predates the toolchain-cache volume.
	var previous []install.StackUpgrade
	// A generated recipe self-certifies through its fingerprint label; an
	// owner-edited recipe does not and is never replaced. An interrupted change
	// is recognized from the exact recipe copy saved beside its backup.
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
			for _, older := range olderRecipes {
				// The older Compose was rendered for the older recipe's
				// toolchains: a repository that gained a go.mod or Cargo.toml
				// had no toolchain-cache volume before.
				olderReq := development.RecipeRequirements(older.files)
				opts := compose.Options{HermesImage: qualification.FoundationImage, Development: &olderReq, DockerTests: tests, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}
				old, err := compose.OlderRecipe(id, opts, older.fingerprint)
				if err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-recipe-" + older.fingerprint[:12] + ".yaml", PreviousRecipe: older.files})
				// The previous release, before .hermes was hidden inside /workspace.
				v023, err := compose.BeforeStateMask(id, opts, older.fingerprint)
				if err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				previous = append(previous, install.StackUpgrade{Compose: v023, BackupName: "compose.before-state-mask-" + older.fingerprint[:12] + ".yaml", PreviousRecipe: older.files})
				if olderReq.Go {
					v020, err := compose.BeforeToolchainCache(id, opts, older.fingerprint)
					if err != nil {
						fmt.Fprintln(stderr, err)
						return 1
					}
					previous = append(previous, install.StackUpgrade{Compose: v020, BackupName: "compose.before-toolchain-cache-" + older.fingerprint[:12] + ".yaml", PreviousRecipe: older.files})
				}
			}
		}
	}
	if a.selinuxState().Enabled() {
		// The same deployment rendered before the host enabled SELinux.
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, Development: &report.Development, DockerTests: report.DockerTests, UID: os.Getuid(), GID: os.Getgid()})
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-selinux.yaml"})
	}
	if report.DockerTests {
		// The same deployment before it opted into the Docker test daemon.
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, Development: &report.Development, UID: os.Getuid(), GID: os.Getgid(), SELinux: a.selinuxState()})
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-docker-tests.yaml"})
	}
	// Every deployment generated before Compose carried the stop grace period
	// lacks that one line: the current render and each reconfiguration above.
	for _, prior := range previous {
		if stripped := compose.WithoutStopGrace(prior.Compose); !bytes.Equal(stripped, prior.Compose) {
			previous = append(previous, install.StackUpgrade{Compose: stripped, BackupName: prior.BackupName, PreviousRecipe: prior.PreviousRecipe})
		}
	}
	previous = append(previous, install.StackUpgrade{Compose: compose.WithoutStopGrace(data), BackupName: "compose.before-stop-grace.yaml"})
	created, err := install.PublishStackChecked(id, artifacts, previous, func() error {
		current := a.plan(id)
		if len(current.Unsupported) > 0 {
			return fmt.Errorf("selected integration became unqualified")
		}
		if len(current.Collisions) > 0 {
			return fmt.Errorf("target changed: %s", strings.Join(current.Collisions, "; "))
		}
		if current.Development.Go != report.Development.Go || current.Development.Rust != report.Development.Rust || current.Development.Flutter != report.Development.Flutter {
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
		if _, current := compose.DevelopmentInstallSelected(id); report.ExistingState && !current {
			// RepoKit recognizes only its current generation, and never adopts
			// an edited, foreign or earlier deployment.
			fmt.Fprintln(stderr, "This .hermes is not RepoKit's current deployment (edited, foreign or from an earlier release). To start over, stop it with docker compose -f .hermes/compose.yaml down, move .hermes aside, then rerun install.")
		}
		return 1
	}
	if err := a.excludeInstallLock(context.Background(), id); err != nil {
		u.warn(".hermes-repokit.lock may appear in git status (%v); add %s to .git/info/exclude", err, lockExcludeEntry)
	}
	// created means files were written; over existing state that is an
	// in-place upgrade of recognized generated files, never a fresh start.
	switch {
	case created && report.ExistingState:
		u.ok("Deployment", "upgraded the generated Compose file and launcher; previous Compose backed up, native configuration preserved")
	case created:
		u.ok("Deployment", "created private .hermes state, Compose file and launcher")
	default:
		u.ok("Deployment", "existing deployment and native configuration preserved")
	}
	// Earlier recipes kept Go's caches in .hermes/development, inside the
	// repository, where whole-tree tools trip over third-party sources. The
	// cache is regenerable but unproven as RepoKit's alone, so it is reported,
	// never deleted.
	if info, err := os.Lstat(filepath.Join(id.Root, ".hermes", "development")); err == nil && info.IsDir() {
		old := tildePath(filepath.Join(id.Root, ".hermes", "development"))
		u.note(old + " holds Go caches from an earlier RepoKit and is no longer used; the new cache is the project's " + compose.ToolchainCacheVolume + " volume. Remove it when convenient: chmod -R u+w " + old + " && rm -rf " + old)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		u.warn("host command unavailable (%v); use %s directly", homeErr, tildePath(id.Launcher))
	} else if command, err := launcher.Expose(id, home); err != nil && target.RepoKitBootstrap(command) {
		u.pending("Host command", tildePath(command)+" is the RepoKit bootstrap; kept")
		u.note("open this team with " + tildePath(id.Launcher) + ", or remove that copy (`repokit` stays the bootstrap) and rerun install")
	} else if err != nil {
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
	complete, teamPending, differs := profileProgress(verify.Profiles(id))
	ready, _ := a.nativeRuntimeReady(id, report.DockerContext)
	gateway := ""
	if complete && ready {
		code, state := a.finishSetup(id, report.DockerContext, 0, stdout, stderr)
		if code != 0 {
			return code
		}
		gateway = state
	}
	if report.DockerTests {
		u.pending("Docker tests", "opt-in privileged test daemon (no host Docker socket) not started")
		u.note("start it: " + strings.TrimSuffix(launcher.StartCommand(id.Compose, report.DockerContext), " up -d hermes)") + " --profile docker-tests up -d docker-test)")
	}
	switch {
	case !ready:
		u.next([2]string{self() + " start", "recreate the container once the running card finishes"})
	case teamPending:
		u.headline("Setup is still required.",
			"If Hermes has no model yet, setup opens Hermes's own private setup in your terminal; RepoKit never sees it.",
			"RepoKit then creates the team, turns on dispatch and proves it with one canary card.")
		u.next([2]string{self() + " setup", "the only remaining step"})
	case !complete:
		u.headline("Some profiles could not be verified: "+strings.Join(differs, ", ")+".",
			"RepoKit changed nothing in them, so dispatch was not re-checked by this run.")
		u.next([2]string{self() + " plan", "see what differs"}, [2]string{self() + " verify", "check readiness"})
	case gateway == "not-running":
		// install never starts a gateway the owner stopped; setup does.
		u.next([2]string{self() + " setup", "start the gateway and prove dispatch"})
	default:
		u.ready(a.teamCommand(id))
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
		u.pending("Kanban", "initialization pending: the container is not running the current deployment")
		if a.resetProfile != "" {
			u.fail("profile reset not applied: the container is not running the current deployment")
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
		if errors.Is(err, native.ErrWorkStarted) && !afterSetup && a.resetProfile == "" {
			u.pending("Team", "a card started running during install; nothing was changed")
			u.note("rerun install when the board is idle")
			return 0
		}
		u.fail("%v", err)
		return 1
	}
	u.ok("Kanban", "native board ready")
	u.team(teamReport)
	// Per-profile lines already show each state; these add what to do next.
	perRole := len(teamReport.Roles) > 0
	if len(teamReport.Reset) > 0 {
		if !perRole {
			u.ok("Reset", strings.Join(teamReport.Reset, ", ")+" returned to RepoKit baseline")
		}
		u.note("prior SOUL.md, config.yaml and profile.yaml kept beside them as *.before-reset-<UTC time>; start a fresh conversation to use the new identity")
	} else if a.resetProfile != "" {
		u.ok("Reset", a.resetProfile+" had no existing profile to reset; it now starts from RepoKit's baseline")
	}
	// A chat keeps the identity it started with; only the owner can start a
	// fresh one, so say so whenever default's identity changed.
	for _, role := range teamReport.Roles {
		if role.Profile == "default" && role.State == "upgrade" {
			u.note("default's identity changed: send /new in each chat (Telegram and others) so the conversation uses it")
		}
	}
	return 0
}

// nativeReadyTimeout bounds the wait for a just-started container's Hermes CLI.
// A first boot remaps the hermes user and fixes ownership of the state tree,
// which took over 90 seconds on an SELinux-enforcing Fedora host.
var nativeReadyTimeout = 5 * time.Minute

func (a App) waitForNativeCLI(id target.Identity, dockerContext string) error {
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 30 * time.Second}
	}
	return native.WaitForCLI(context.Background(), id, dockerContext, runner, nativeReadyTimeout, time.Second)
}

// profileProgress decides whether install may finish activation. Only a
// complete scaffold can yield a generation. An owner-customized profile is
// complete: it is preserved as is and does not hold back the team. A pending
// profile needs setup (never provisioned, or a RepoKit upgrade); any other
// state is a profile RepoKit cannot verify, which setup cannot fix.
func profileProgress(probes []verify.Probe) (complete, teamPending bool, unverified []string) {
	complete = true
	for _, probe := range probes {
		switch probe.Status {
		case verify.Healthy, verify.Customized, verify.Upgradable:
		case verify.PendingSetup:
			complete, teamPending = false, true
		default:
			complete = false
			unverified = append(unverified, strings.TrimPrefix(probe.Component, "profile:"))
		}
	}
	return complete, teamPending, unverified
}

// teamCommand is how the owner opens this team: the host command when PATH
// resolves it to this launcher, otherwise the launcher path itself.
func (a App) teamCommand(id target.Identity) string {
	for _, dir := range filepath.SplitList(a.Path) {
		p := filepath.Join(dir, id.Container)
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(p); err == nil && resolved == id.Launcher {
			return id.Container
		}
		break // the first match shadows the rest
	}
	return tildePath(id.Launcher)
}

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var commands = [...]string{"plan", "install", "setup", "verify", "start", "stop", "remove"}

func Commands() []string { return append([]string(nil), commands[:]...) }

type App struct {
	Directory, Path string
	DockerTests     bool
	Runner          verify.Runner
	Stdin           io.Reader
	Initializer     native.InputRunner
	// HostSELinux overrides host detection in tests; empty detects the live host.
	HostSELinux selinux.State
	// startGateway lets explicit setup start a stopped default gateway.
	startGateway bool
	// Confirm replaces the interactive remove confirmation in tests.
	Confirm func(prompt string) (string, error)
	// ComposeExec replaces the streamed `docker compose` invocation in tests.
	ComposeExec func(stdout, stderr io.Writer, args ...string) error
}

func Run(args []string, stdout, stderr io.Writer) int {
	dir, e := os.Getwd()
	if e != nil {
		fmt.Fprintln(stderr, "cannot resolve working directory")
		return 1
	}
	// Native CLI calls through docker exec start Python; 5s flaps under load.
	return (App{Directory: dir, Path: os.Getenv("PATH"), Runner: process.Runner{Timeout: 30 * time.Second}, Stdin: os.Stdin}).Run(args, stdout, stderr)
}
func (a App) Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		usage(stdout)
		return 0
	}
	if len(args) == 0 || !recognized(args[0]) {
		return usageError(stderr)
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	engineering := false
	teamSetup := false
	dispatchCheck := false
	if args[0] == "verify" {
		flags.BoolVar(&dispatchCheck, "dispatch-check", false, "create one researcher card and require automatic gateway completion")
	}
	if args[0] == "plan" || args[0] == "install" {
		flags.BoolVar(&a.DockerTests, "docker-tests", false, "publish opt-in privileged isolated Docker acceptance service; never the host socket")
	}
	if args[0] != "setup" {
		flags.BoolVar(&engineering, "engineering", false, "legacy alias; generic team is the default")
	} else {
		flags.BoolVar(&teamSetup, "team", false, "resume team provisioning using the saved default model; no private wizard")
	}
	err := flags.Parse(args[1:])
	if errors.Is(err, flag.ErrHelp) && len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
		usage(stdout)
		return 0
	}
	if err != nil || flags.NArg() != 0 {
		return usageError(stderr)
	}
	for _, arg := range args[1:] {
		if arg == "--" {
			return usageError(stderr)
		}
	}
	if dispatchCheck {
		return a.dispatchCheck(stdout)
	}
	id, err := target.Resolve(a.Directory)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if selected, ok := compose.DevelopmentInstallSelected(id); ok && selected.DockerTests {
		a.DockerTests = true
	}
	if a.Runner == nil {
		a.Runner = process.Runner{}
	}
	switch args[0] {
	case "setup":
		// Explicit setup activates the team, so it starts a stopped gateway.
		a.startGateway = true
		if issues := target.Inspect(id, ""); len(issues) > 0 {
			fmt.Fprintln(stderr, "unsafe native state:", strings.Join(issues, "; "))
			return 1
		}
		if issues := a.gitIssues(context.Background(), id); len(issues) > 0 {
			fmt.Fprintln(stderr, "setup refused: private state protection is unverified:", strings.Join(issues, "; "))
			return 1
		}
		dockerContext, routingErr := launcher.Context(id)
		if routingErr != nil {
			fmt.Fprintln(stderr, "setup refused: deployment routing cannot be verified")
			return 1
		}
		for _, p := range verify.Inspect(context.Background(), id, a.Runner) {
			if p.Component == "compose" && p.Status != verify.Healthy {
				fmt.Fprintln(stderr, "setup refused: generated Compose cannot be verified")
				return 1
			}
		}
		ready, runtimeErr := a.nativeRuntimeReady(id, dockerContext)
		if runtimeErr != nil {
			fmt.Fprintln(stderr, "setup refused:", runtimeErr)
			return 1
		}
		if !ready {
			fmt.Fprintf(stderr, "Setup requires the running pinned Hermes deployment. Inspect Docker access and service state. Start with: %s\n", launcher.StartCommand(id.Compose, dockerContext))
			return 1
		}
		if err := a.waitForNativeCLI(id, dockerContext); err != nil {
			fmt.Fprintln(stderr, "setup deferred:", err)
			return 1
		}
		if teamSetup {
			return a.finishSetup(id, dockerContext, a.initialize(id, dockerContext, true, stdout, stderr), stdout, stderr)
		}
		if code := native.Setup(id.Launcher, id.Compose, dockerContext, a.Stdin, stdout, stderr); code != 0 {
			return code
		}
		if !native.InteractiveInput(a.Stdin) {
			fmt.Fprintln(stderr, "Default setup was noninteractive; team provisioning remains pending. Run setup in your terminal.")
			return 1
		}
		if code := a.initialize(id, dockerContext, true, stdout, stderr); code != 0 {
			return code
		}
		return a.finishSetup(id, dockerContext, 0, stdout, stderr)
	case "verify":
		probes := verify.Inspect(context.Background(), id, a.Runner)
		if issues := a.gitIssues(context.Background(), id); len(issues) > 0 {
			probes = append(probes, verify.Probe{Component: "git", Status: verify.Degraded, Detail: strings.Join(issues, "; ")})
		}
		probes = append(probes, verify.Development(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.Profiles(id)...)
		probes = append(probes, verify.Gateway(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.DefaultKanban(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.RuntimeIntegrations(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.ReviewEvidence(context.Background(), id, a.Runner))
		probes = append(probes, verify.HostSecurity(context.Background(), id, a.Runner)...)
		readiness := verify.Readiness(probes)
		if err := json.NewEncoder(stdout).Encode(append(readiness, probes...)); err != nil {
			return 1
		}
		if !verify.CoreUsable(readiness) {
			return 1
		}
		return 0
	case "start":
		return a.start(id, stdout, stderr)
	case "stop":
		return a.stop(id, stdout, stderr)
	case "remove":
		if _, production := a.Runner.(process.Runner); production || a.Runner == nil {
			a.Runner = process.Runner{Timeout: 3 * time.Minute} // Compose down drains the gateway
		}
		return a.remove(id, stdout, stderr)
	case "plan", "install":
		report := a.plan(id, engineering)
		if args[0] == "plan" {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if e := enc.Encode(report); e != nil {
				return 1
			}
			return 0
		}
		return a.install(id, report, engineering, stdout, stderr)
	}
	return 2
}

type HostSecurity struct {
	SELinux        selinux.State `json:"selinux"`
	RelabelMode    string        `json:"relabel_mode"`
	BindRelabeling string        `json:"bind_relabeling"`
}

type Plan struct {
	Development                  development.Requirements `json:"development_requirements"`
	DockerTests                  bool                     `json:"docker_tests_opt_in"`
	Target                       target.Identity          `json:"target"`
	DockerContext                string                   `json:"docker_context"`
	ExistingState                bool                     `json:"existing_state"`
	Collisions                   []string                 `json:"collisions"`
	CandidateImages              map[string]string        `json:"candidate_images_not_release_qualified"`
	HostSecurity                 HostSecurity             `json:"host_security"`
	Profiles, Plugins            []string
	Kanban                       map[string]any `json:"kanban"`
	ProposedChanges, Unsupported []string
}

// selinuxState resolves the host SELinux state, allowing tests to override it.
func (a App) selinuxState() selinux.State {
	if a.HostSELinux != "" {
		return a.HostSELinux
	}
	return selinux.Detect()
}

func (a App) plan(id target.Identity, engineering bool) Plan {
	p := Plan{Target: id, Collisions: target.Inspect(id, a.Path), CandidateImages: map[string]string{"hermes": qualification.FoundationImage}, Profiles: []string{"default"}, Plugins: []string{}, Kanban: map[string]any{"dispatch_in_gateway": false, "auto_decompose": false, "orchestrator_profile": "default", "max_in_progress": 1}, ProposedChanges: []string{"private .hermes native state", "standalone Hermes Compose and launcher", "native safe-default config; operator starts Compose and runs setup"}}

	p.Development, _ = development.Detect(id.Root)
	if _, err := development.Detect(id.Root); err != nil {
		p.Unsupported = append(p.Unsupported, "repository toolchain manifests cannot be safely inspected")
	}
	state := a.selinuxState()
	decision := "disabled"
	if state.Enabled() {
		decision = "enabled"
		p.ProposedChanges = append(p.ProposedChanges, "apply private SELinux Z relabeling to repository bind mounts; SELinux policy and global host labels unchanged")
	}
	p.HostSecurity = HostSecurity{SELinux: state, RelabelMode: state.RelabelMode(), BindRelabeling: decision}
	p.DockerTests = a.DockerTests
	p.ProposedChanges = append(p.ProposedChanges, "build pinned Hermes development image; project compiler readiness is verified separately")
	if a.DockerTests {
		p.ProposedChanges = append(p.ProposedChanges, "opt-in privileged Docker test daemon with private scratch volumes, no host daemon socket; not a VM security boundary")
	}
	p.Profiles = nil
	for _, role := range team.Roster() {
		p.Profiles = append(p.Profiles, role.Name)
	}
	p.ProposedChanges = append(p.ProposedChanges, "generic team provisioned after successful native default setup; integration acceptance pending")

	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err == nil {
		p.ExistingState = true
		p.ProposedChanges = []string{"inspect and preserve existing native configuration; refuse ambiguous adoption", "initialize missing native Kanban in the running qualified container", "upgrade only recognized generated Hermes Compose; preserve native state and back up old Compose"}
		p.ProposedChanges = append(p.ProposedChanges, "reconcile the seven native team profiles after default setup; preserve user drift and unknown profiles; integrations remain pending")
	}
	p.ProposedChanges = append(p.ProposedChanges, "create or reuse ~/.local/bin/"+id.Container+" as a symlink to the generated launcher when safe; preserve conflicts and report missing PATH")
	p.ProposedChanges = append(p.ProposedChanges, "build and start "+id.Container+" with ordinary Docker Compose, then initialize native Kanban")
	p.ProposedChanges = append(p.ProposedChanges, "add "+lockExcludeEntry+" to the local, never-committed .git/info/exclude unless already ignored, so the installer lock stays out of git status")
	if compose.LegacyLayaBuildSelected(id) {
		p.ProposedChanges = append(p.ProposedChanges, "recognized legacy Hermes/Laya build stack: install backs up compose.before-core.yaml and generates the core runtime; all service data preserved")
	}
	ctx := context.Background()
	p.Collisions = append(p.Collisions, a.gitIssues(ctx, id)...)
	if p.ExistingState {
		captured, err := launcher.InstallContext(id)
		if err != nil {
			p.Collisions = append(p.Collisions, "existing launcher context cannot be verified")
			return p
		}
		p.DockerContext = captured
	} else {
		dc := a.Runner.Run(ctx, "docker", "context", "show")
		if dc.Err != nil || dc.Truncated {
			p.Collisions = append(p.Collisions, "Docker context unavailable")
			return p
		}
		p.DockerContext = strings.TrimSpace(dc.Output)
	}
	if _, err := launcher.Render(id, p.DockerContext); err != nil {
		p.Collisions = append(p.Collisions, "invalid Docker context")
		return p
	}
	host := a.Runner.Run(ctx, "docker", "context", "inspect", p.DockerContext, "--format", "{{.Endpoints.docker.Host}}")
	if host.Err != nil || host.Truncated || !strings.HasPrefix(strings.TrimSpace(host.Output), "unix://") {
		p.Collisions = append(p.Collisions, "only a qualified local Docker socket context is supported")
	}
	containers := a.Runner.Run(ctx, "docker", "--context", p.DockerContext, "container", "ls", "--all", "--format", "{{.Names}}")
	if containers.Err != nil || containers.Truncated {
		p.Collisions = append(p.Collisions, "container names could not be inspected")
	} else {
		for _, name := range strings.Fields(containers.Output) {
			if name == id.Container || (p.ExistingState && name == target.PreviousNames(id).Container) {
				observed := a.Runner.Run(ctx, "docker", "--context", p.DockerContext, "container", "inspect", "--format", verify.InspectFormat, name)
				var state verify.Runtime
				if !p.ExistingState || observed.Err != nil || observed.Truncated || json.Unmarshal([]byte(observed.Output), &state) != nil || state.Project != id.Project || state.Workspace != id.Root || state.Home != filepath.Join(id.Root, ".hermes") {
					p.Collisions = append(p.Collisions, "container name already exists (running or stopped); native deployment ownership requires verification")
				}
			}
		}
	}
	return p
}
func recognized(command string) bool {
	for _, c := range commands {
		if command == c {
			return true
		}
	}
	return false
}
func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: hermes-repokit <plan|install|setup|verify|start|stop|remove> [--engineering] [--help]")
	fmt.Fprintln(w, "       hermes-repokit setup [--team]")
	fmt.Fprintln(w, "       hermes-repokit verify [--dispatch-check] (one researcher card through automatic dispatch; model cost)")
	fmt.Fprintln(w, "       hermes-repokit <plan|install> [--docker-tests]")
	fmt.Fprintln(w, "       hermes-repokit start | stop (start or stop the deployment; state is kept)")
	fmt.Fprintln(w, "       hermes-repokit remove (deletes the deployment and .hermes after typed confirmation)")
}
func usageError(w io.Writer) int { fmt.Fprintln(w, "usage error"); usage(w); return 2 }

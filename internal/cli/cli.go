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

var commands = [...]string{"plan", "install", "setup", "verify", "start", "stop", "remove", "github-login"}

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
	// Interactive replaces the terminal check that lets install continue
	// straight into setup, in tests.
	Interactive func() bool
	// noSetup keeps install from continuing into setup.
	noSetup bool
	// Fetch and RunInstaller replace the network and the installer for
	// update in tests.
	Fetch        func(url string) ([]byte, error)
	RunInstaller func(script []byte, ref string, stdout, stderr io.Writer) error
	// FreeSpace replaces the Docker free-space probe in tests.
	FreeSpace func(dockerContext string) (uint64, bool)
	// GitHubLogin replaces gh's interactive sign-in in tests.
	GitHubLogin func(compose, dockerContext string, stdin io.Reader, stdout, stderr io.Writer) int
	// Canary replaces the setup canary card in tests.
	Canary func(id target.Identity, dockerContext string) (string, error)
	// resetProfile names one roster profile install returns to RepoKit's
	// baseline, and plan previews.
	resetProfile string
	// teamShape, when set, is the team shape install records for the
	// deployment ("single" or "seven"); later installs keep it.
	teamShape string
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
	if len(args) >= 1 && args[0] == "version" && len(args) == 1 {
		fmt.Fprintln(stdout, "repokit "+Version)
		return 0
	}
	if len(args) >= 1 && args[0] == "list" {
		switch {
		case len(args) == 1:
			return a.listDeployments(false, stdout, stderr)
		case len(args) == 2 && args[1] == "--json":
			return a.listDeployments(true, stdout, stderr)
		}
		return usageError(stderr)
	}
	if len(args) >= 1 && args[0] == "update" {
		switch {
		case len(args) == 1:
			return a.update(false, stdout, stderr)
		case len(args) == 2 && args[1] == "--main":
			return a.update(true, stdout, stderr)
		}
		return usageError(stderr)
	}
	if len(args) == 0 || !recognized(args[0]) {
		return usageError(stderr)
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	teamSetup := false
	noCanary := false
	dispatchCheck := false
	if args[0] == "verify" {
		flags.BoolVar(&dispatchCheck, "dispatch-check", false, "create one researcher card and require automatic gateway completion")
	}
	if args[0] == "plan" || args[0] == "install" {
		flags.BoolVar(&a.DockerTests, "docker-tests", false, "publish opt-in privileged isolated Docker acceptance service; never the host socket")
		if args[0] == "install" {
			flags.BoolVar(&a.noSetup, "no-setup", false, "stop after install even in a terminal; run setup yourself later")
			flags.StringVar(&a.teamShape, "team", "", "record the team shape: single (default runs and verifies every card itself) or seven (the seven-profile team); later installs keep it")
		}
		flags.StringVar(&a.resetProfile, "reset-profile", "", "return one roster profile to RepoKit's baseline SOUL, description and managed configuration; prior files are backed up")
	}
	if args[0] == "setup" {
		flags.BoolVar(&teamSetup, "team", false, "recovery: reconcile the team from the saved default model without the private wizard")
		flags.BoolVar(&noCanary, "no-canary", false, "skip the canary card (no model call); dispatch stays unproven")
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
	if a.resetProfile != "" && !rosterProfile(a.resetProfile) {
		fmt.Fprintf(stderr, "--reset-profile takes a RepoKit roster profile (%s); owner-created profiles are never reset\n", strings.Join(rosterNames(), ", "))
		return 2
	}
	if a.teamShape != "" && a.teamShape != string(team.Single) && a.teamShape != string(team.Seven) {
		fmt.Fprintln(stderr, "--team takes single or seven")
		return 2
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
		return a.setup(id, teamSetup, noCanary, stdout, stderr)
	case "verify":
		probes := verify.Inspect(context.Background(), id, a.Runner)
		if issues := a.gitIssues(context.Background(), id); len(issues) > 0 {
			probes = append(probes, verify.Probe{Component: "git", Status: verify.Degraded, Detail: strings.Join(issues, "; ")})
		}
		probes = append(probes, verify.Development(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.Profiles(id)...)
		probes = append(probes, verify.Gateway(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.DefaultKanban(context.Background(), id, a.Runner)...)
		probes = append(probes, verify.ChannelSessions(context.Background(), id, a.Runner)...)
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
	case "github-login":
		return a.githubLogin(id, stdout, stderr)
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
		report := a.plan(id)
		if args[0] == "plan" {
			report.Team = a.planTeam(id, report)
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if e := enc.Encode(report); e != nil {
				return 1
			}
			return 0
		}
		code := a.install(id, report, stdout, stderr)
		a.recordInstall(id, report, code)
		return code
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
	// Team previews per-profile convergence on an existing running deployment.
	Team *TeamPlan `json:"team,omitempty"`
}

// TeamPlan is the plan's per-profile table. Detail explains a status that
// prevents a preview or blocks the run.
type TeamPlan struct {
	native.TeamStatus
	Detail string `json:"detail,omitempty"`
}

func rosterNames() []string {
	names := []string{}
	for _, role := range team.Roster() {
		names = append(names, role.Name)
	}
	return names
}

func rosterProfile(name string) bool {
	for _, n := range rosterNames() {
		if n == name {
			return true
		}
	}
	return false
}

// planTeam reads the decision install would make, through the running
// deployment's public native commands only. Nothing is written.
func (a App) planTeam(id target.Identity, report Plan) *TeamPlan {
	if !report.ExistingState {
		return nil
	}
	unavailable := func(status, detail string) *TeamPlan {
		return &TeamPlan{TeamStatus: native.TeamStatus{Status: status, Profiles: []native.RoleStatus{}}, Detail: detail}
	}
	if report.DockerContext == "" {
		return unavailable("unavailable", "deployment routing cannot be verified")
	}
	if ready, err := a.nativeRuntimeReady(id, report.DockerContext); err != nil || !ready {
		// A running container that is not the current RepoKit runtime is
		// not previewed: team changes run only on the current runtime.
		if state, _ := a.containerState(id, report.DockerContext); state == "running" {
			return unavailable("runtime-outdated", "the running container is not the current RepoKit runtime; team convergence is previewed only on the current runtime")
		}
		return unavailable("runtime-not-running", "start the existing Compose service to preview team convergence")
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 30 * time.Second}
	}
	status, err := native.PlanTeam(context.Background(), id, report.DockerContext, a.resetProfile, runner)
	if err != nil {
		return unavailable("unavailable", err.Error())
	}
	plan := &TeamPlan{TeamStatus: status}
	plan.Detail = teamDetail(status)
	return plan
}

// selinuxState resolves the host SELinux state, allowing tests to override it.
func (a App) selinuxState() selinux.State {
	if a.HostSELinux != "" {
		return a.HostSELinux
	}
	return selinux.Detect()
}

func (a App) plan(id target.Identity) Plan {
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
	ctx := context.Background()
	p.Collisions = append(p.Collisions, a.gitIssues(ctx, id)...)
	if p.ExistingState {
		captured, err := launcher.Context(id)
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
			if name == id.Container {
				observed := a.Runner.Run(ctx, "docker", "--context", p.DockerContext, "container", "inspect", "--format", verify.InspectFormat, name)
				var state verify.Runtime
				leftover := observed.Err == nil && !observed.Truncated && json.Unmarshal([]byte(observed.Output), &state) == nil && state.Project == id.Project && state.Workspace == id.Root && state.Home == filepath.Join(id.Root, ".hermes")
				if !p.ExistingState && leftover {
					p.Collisions = append(p.Collisions, "container "+name+" is left from an earlier install of this repository, but .hermes is missing; clear it with `"+self()+" remove`, then install again")
				} else if !p.ExistingState || observed.Err != nil || observed.Truncated || json.Unmarshal([]byte(observed.Output), &state) != nil || state.Project != id.Project || state.Workspace != id.Root || state.Home != filepath.Join(id.Root, ".hermes") {
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

// usage names the command as it was invoked: in a repository named repokit,
// hermes-repokit is that repository's Hermes launcher, not the bootstrap.
func usage(w io.Writer) {
	me := self()
	pad := strings.Repeat(" ", len("usage: "))
	fmt.Fprintf(w, "usage: %s <plan|install|setup|verify|start|stop|remove> [--help]\n", me)
	fmt.Fprintf(w, "%s%s install [--no-setup] (the whole first-time path: continues into setup in a terminal)\n", pad, me)
	fmt.Fprintf(w, "%s%s setup [--no-canary] [--team] (--team: recovery without the private wizard)\n", pad, me)
	fmt.Fprintf(w, "%s%s verify [--dispatch-check] (one researcher card through automatic dispatch; model cost)\n", pad, me)
	fmt.Fprintf(w, "%s%s <plan|install> [--docker-tests] [--reset-profile <role>]\n", pad, me)
	fmt.Fprintf(w, "%s%s install --team <single|seven>\n", pad, me)
	fmt.Fprintf(w, "%s%s start | stop (start or stop the deployment; state is kept)\n", pad, me)
	fmt.Fprintf(w, "%s%s remove (deletes the deployment and .hermes after typed confirmation)\n", pad, me)
	fmt.Fprintf(w, "%s%s update [--main] (replace this binary with the latest release, or main) | version\n", pad, me)
	fmt.Fprintf(w, "%s%s list [--json] (every repository RepoKit installed into on this machine, with live state)\n", pad, me)
	fmt.Fprintf(w, "%s%s github-login (sign in to GitHub inside the deployment so agents can push; gh's own login)\n", pad, me)
}
func usageError(w io.Writer) int { fmt.Fprintln(w, "usage error"); usage(w); return 2 }

// teamDetail summarizes a team preview in one line for the plan output.
func teamDetail(status native.TeamStatus) string {
	switch status.Status {
	case "drift":
		return "managed configuration drift blocks every write in this run; inspect the drift profiles or reset one with --reset-profile"
	case "pending-setup":
		return "Hermes private setup has not chosen a default model yet; run repokit setup"
	}
	for _, row := range status.Profiles {
		if row.State == "reset" {
			return row.Profile + " will be replaced with RepoKit's baseline after its files are backed up; every other profile is preserved"
		}
	}
	return "owner-customized profiles are preserved; none will be overwritten"
}

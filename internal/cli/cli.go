package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/supervision"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var commands = [...]string{"plan", "install", "setup", "verify"}

func Commands() []string { return append([]string(nil), commands[:]...) }

type App struct {
	Directory, Path string
	Runner          verify.Runner
	Stdin           io.Reader
	Initializer     native.InputRunner
}

func Run(args []string, stdout, stderr io.Writer) int {
	dir, e := os.Getwd()
	if e != nil {
		fmt.Fprintln(stderr, "cannot resolve working directory")
		return 1
	}
	return (App{Directory: dir, Path: os.Getenv("PATH"), Runner: process.Runner{}, Stdin: os.Stdin}).Run(args, stdout, stderr)
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
	memorySetup := false
	if args[0] != "setup" {
		flags.BoolVar(&engineering, "engineering", false, "legacy alias; generic team is the default")
	} else {
		flags.BoolVar(&memorySetup, "memory", false, "private native OpenViking setup and shared profile connection")
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
	id, err := target.Resolve(a.Directory)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if a.Runner == nil {
		a.Runner = process.Runner{}
	}
	switch args[0] {
	case "setup":
		if memorySetup && !native.InteractiveInput(a.Stdin) {
			fmt.Fprintln(stderr, "Run setup --memory in your private terminal; credentials must remain in native setup.")
			return 1
		}
		if issues := target.Inspect(id, ""); len(issues) > 0 {
			fmt.Fprintln(stderr, "unsafe native state:", strings.Join(issues, "; "))
			return 1
		}
		dockerContext, _ := launcher.Context(id)
		if memorySetup {
			return a.setupMemory(id, dockerContext, stdout, stderr)
		}
		if code := native.Setup(id.Launcher, id.Compose, dockerContext, a.Stdin, stdout, stderr); code != 0 {
			return code
		}
		if !native.InteractiveInput(a.Stdin) {
			fmt.Fprintln(stderr, "Default setup was noninteractive; team provisioning remains pending. Run setup in your terminal.")
			return 1
		}
		return a.initialize(id, dockerContext, true, stdout, stderr)
	case "verify":
		probes := verify.Inspect(context.Background(), id, a.Runner)
		if issues := a.gitIssues(context.Background(), id); len(issues) > 0 {
			probes = append(probes, verify.Probe{Component: "git", Status: verify.Degraded, Detail: strings.Join(issues, "; ")})
		}
		probes = append(probes, verify.Profiles(id)...)
		probes = append(probes, verify.OpenViking(context.Background(), id, a.Runner)...)
		probes = append(probes,
			verify.Probe{Component: "nerve-laya", Status: verify.Unknown, Detail: "local inference fixture qualified; this deployment's all-profile supervision and sidecar lifecycle not established"},
		)
		if err := json.NewEncoder(stdout).Encode(probes); err != nil {
			return 1
		}
		for _, p := range probes {
			if p.Status != verify.Healthy {
				return 1
			}
		}
		return 0
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

type Plan struct {
	Target                       target.Identity   `json:"target"`
	DockerContext                string            `json:"docker_context"`
	ExistingState                bool              `json:"existing_state"`
	Collisions                   []string          `json:"collisions"`
	CandidateImages              map[string]string `json:"candidate_images_not_release_qualified"`
	Profiles, Plugins            []string
	ProposedMemoryConfig         map[string]any `json:"proposed_memory_config_not_activated"`
	Kanban                       map[string]any `json:"kanban"`
	ProposedNerveSettings        map[string]any `json:"proposed_nerve_settings_not_activated"`
	NerveRevision                string         `json:"candidate_nerve_revision"`
	OpenViking, NerveLaya        string
	ProposedChanges, Unsupported []string
}

func (a App) plan(id target.Identity, engineering bool) Plan {
	p := Plan{Target: id, Collisions: target.Inspect(id, a.Path), CandidateImages: map[string]string{"hermes": qualification.FoundationImage, "openviking": projectmemory.Image}, Profiles: []string{"default"}, Plugins: []string{}, Kanban: map[string]any{"dispatch_in_gateway": false, "auto_decompose": false, "orchestrator_profile": "default", "max_in_progress": 1}, OpenViking: "pending private native embedding/VLM setup and live memory qualification", NerveLaya: "pending qualified local sidecar deployment", ProposedChanges: []string{"private .hermes native state", "standalone Hermes/OpenViking Compose and launcher", "native safe-default config; operator starts Compose and runs setup"}}

	p.ProposedMemoryConfig, _ = projectmemory.NativeConfig(id.Project)
	p.ProposedNerveSettings = supervision.LocalLayaSettings()
	p.NerveRevision = supervision.NerveRevision
	p.ProposedChanges = append(p.ProposedChanges, "private persistent OpenViking service; native setup required before memory activation")
	p.Profiles = nil
	for _, role := range team.Roster() {
		p.Profiles = append(p.Profiles, role.Name)
	}
	p.ProposedChanges = append(p.ProposedChanges, "generic team provisioned after successful native default setup; integration acceptance pending")

	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err == nil {
		p.ExistingState = true
		p.ProposedChanges = []string{"inspect and preserve existing native configuration; refuse ambiguous adoption", "initialize missing native Kanban in the running qualified container", "upgrade only recognized Hermes-only Compose to include OpenViking; preserve native state and back up old Compose"}
		p.ProposedChanges = append(p.ProposedChanges, "reconcile the six native team profiles after default setup; preserve user drift and unknown profiles; integrations remain pending")
	}
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
				observed := a.Runner.Run(ctx, "docker", "--context", p.DockerContext, "container", "inspect", "--format", verify.InspectFormat, id.Container)
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
	fmt.Fprintln(w, "usage: hermes-repokit <plan|install|setup|verify> [--engineering] [--help]")
	fmt.Fprintln(w, "       hermes-repokit setup --memory")
}
func usageError(w io.Writer) int { fmt.Fprintln(w, "usage error"); usage(w); return 2 }

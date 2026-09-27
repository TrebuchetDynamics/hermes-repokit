package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
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
}

func Run(args []string, stdout, stderr io.Writer) int {
	dir, e := os.Getwd()
	if e != nil {
		fmt.Fprintln(stderr, "cannot resolve working directory")
		return 1
	}
	return (App{dir, os.Getenv("PATH"), process.Runner{}, os.Stdin}).Run(args, stdout, stderr)
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
	if args[0] != "setup" {
		flags.BoolVar(&engineering, "engineering", false, "select engineering preset")
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
		if issues := target.Inspect(id, ""); len(issues) > 0 {
			fmt.Fprintln(stderr, "unsafe native state:", strings.Join(issues, "; "))
			return 1
		}
		return native.Setup(id.Launcher, id.Compose, a.Stdin, stdout, stderr)
	case "verify":
		probes := verify.Inspect(context.Background(), id, a.Runner)
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
		// The release has no admitted full preset yet. Refuse before locks, pulls,
		// publication or native initialization; source candidates are not approval.
		fmt.Fprintln(stderr, "installation qualification incomplete:", strings.Join(report.Unsupported, "; "))
		if len(report.Collisions) > 0 {
			fmt.Fprintln(stderr, "target collisions:", strings.Join(report.Collisions, "; "))
		}
		return 1
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
	Kanban                       map[string]bool `json:"kanban"`
	OpenViking, NerveLaya        string
	ProposedChanges, Unsupported []string
}

func (a App) plan(id target.Identity, engineering bool) Plan {
	p := Plan{Target: id, Collisions: target.Inspect(id, a.Path), CandidateImages: map[string]string{
		"hermes":     "nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db",
		"openviking": "ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05"}, Profiles: []string{"default"}, Plugins: []string{"obra/superpowers@8ca22dba9a94f28898bbce59f2537ff4d87c747d"}, Kanban: map[string]bool{"dispatch_in_gateway": false, "auto_decompose": false}, OpenViking: "pending native setup and release qualification", NerveLaya: "not selected", ProposedChanges: []string{"private .hermes native state", "standalone Compose and launcher", "native default profile and Kanban", "upstream Superpowers", "official OpenViking service and native setup handoff"}, Unsupported: []string{"Superpowers CAUTION requires exact SHA/findings approval (docs/qualification/superpowers-8ca22dba-scan.txt)", "native exec/chat/setup, plugin loading and OpenViking release evidence incomplete", "no release-qualified preset: removal-first acceptance has not passed"}}
	if engineering {
		p.Profiles = append(p.Profiles, "researcher", "planner", "builder", "reviewer")
		p.Plugins = append(p.Plugins, "upstream nerve (not yet admitted)")
		p.NerveLaya = "selected, unsupported pending transport/checkpoint/inference qualification"
		p.Unsupported = append(p.Unsupported, "Nerve/Laya transport and same-card actor independence not qualified")
	}
	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err == nil {
		p.ExistingState = true
		p.ProposedChanges = []string{"inspect and preserve existing native configuration; refuse ambiguous adoption"}
	}
	ctx := context.Background()
	git := a.Runner.Run(ctx, "git", "-C", id.Root, "rev-parse", "--show-toplevel")
	if git.Err != nil || git.Truncated || strings.TrimSpace(git.Output) != id.Root {
		p.Collisions = append(p.Collisions, "target must be the canonical Git repository root")
	}
	tracked := a.Runner.Run(ctx, "git", "-C", id.Root, "ls-files", "-z", "--", ".hermes")
	if tracked.Err != nil || tracked.Truncated {
		p.Collisions = append(p.Collisions, "cannot inspect tracked private state")
	} else if tracked.Output != "" {
		p.Collisions = append(p.Collisions, "private .hermes state is tracked by Git")
	}
	dc := a.Runner.Run(ctx, "docker", "context", "show")
	if dc.Err != nil || dc.Truncated {
		p.Collisions = append(p.Collisions, "Docker context unavailable")
		return p
	}
	p.DockerContext = strings.TrimSpace(dc.Output)
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
				p.Collisions = append(p.Collisions, "container name already exists (running or stopped); native deployment ownership requires verification")
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
}
func usageError(w io.Writer) int { fmt.Fprintln(w, "usage error"); usage(w); return 2 }

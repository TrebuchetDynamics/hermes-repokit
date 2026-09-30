package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// self is the name the owner typed, so printed next steps match their command.
func self() string {
	if name := filepath.Base(os.Args[0]); name == "repokit" || name == "hermes-repokit" {
		return name
	}
	return "repokit"
}

// setup takes a deployment from "installed" to "team ready" in one command:
// it starts the container when needed, hands the terminal to Hermes's own
// private setup only while default has no model, then provisions the team,
// activates dispatch and proves it with one canary card. teamOnly skips the
// wizard (recovery); noCanary skips the paid proof (offline acceptance).
func (a App) setup(id target.Identity, teamOnly, noCanary bool, stdout, stderr io.Writer) int {
	u := newUI(stdout, stderr)
	u.title("setup", id.Name, id.Root)
	// Explicit setup activates the team, so it starts a stopped gateway.
	a.startGateway = true
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		u.fail("setup refused: unsafe native state: %s", strings.Join(issues, "; "))
		return 1
	}
	if issues := a.gitIssues(context.Background(), id); len(issues) > 0 {
		u.fail("setup refused: private state protection is unverified: %s", strings.Join(issues, "; "))
		return 1
	}
	dc, err := launcher.Context(id)
	if err != nil {
		u.fail("setup refused: deployment routing cannot be verified; run %s install", self())
		return 1
	}
	for _, p := range verify.Inspect(context.Background(), id, a.Runner) {
		if p.Component == "compose" && p.Status != verify.Healthy {
			u.fail("setup refused: generated Compose cannot be verified; run %s install", self())
			return 1
		}
	}
	if code := a.startDeployment(id, dc, stdout, stderr); code != 0 {
		return code
	}
	if ready, err := a.nativeRuntimeReady(id, dc); err != nil || !ready {
		u.fail("setup deferred: the running container is not the current deployment yet; run %s start when the running card finishes", self())
		return 1
	}
	if err := a.waitForNativeCLI(id, dc); err != nil {
		u.fail("setup deferred: %v", err)
		return 1
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 30 * time.Second}
	}
	if !teamOnly {
		configured, err := native.DefaultModelConfigured(context.Background(), id, dc, runner)
		if err != nil {
			u.fail("setup refused: %v", err)
			return 1
		}
		if configured {
			u.ok("Hermes", "default model already configured; private setup skipped")
			u.note("change the provider or add channels natively: " + id.Container + " -p default setup")
		} else {
			u.working("Hermes", "private setup: answer Hermes's own questions; RepoKit never sees them")
			if code := native.Setup(id.Launcher, id.Compose, dc, a.Stdin, stdout, stderr); code != 0 {
				u.fail("Hermes setup did not finish; nothing else changed. Run %s setup again", self())
				return code
			}
			if !native.InteractiveInput(a.Stdin) {
				u.fail("Hermes setup was noninteractive; team provisioning remains pending. Run %s setup in your terminal", self())
				return 1
			}
			if configured, err = native.DefaultModelConfigured(context.Background(), id, dc, runner); err != nil || !configured {
				u.fail("Hermes setup ended without a default model; run %s setup again", self())
				return 1
			}
			u.ok("Hermes", "private setup complete")
		}
	}
	if code := a.initialize(id, dc, true, stdout, stderr); code != 0 {
		return code
	}
	code, gateway := a.finishSetup(id, dc, 0, stdout, stderr)
	if code != 0 {
		return code
	}
	switch {
	case noCanary:
		u.pending("Canary", "skipped (--no-canary); prove the loop with "+self()+" verify --dispatch-check")
	case gateway == "not-running":
		u.pending("Canary", "skipped: the default gateway is not running")
		return 1
	default:
		u.working("Canary", "one researcher card through automatic dispatch (a small model call)")
		task, err := a.canary(id, dc)
		if err != nil {
			u.fail("canary failed: %v; the team is configured but automatic dispatch is unproven", err)
			u.next([2]string{self() + " verify", "inspect readiness"}, [2]string{self() + " verify --dispatch-check", "retry the canary"})
			return 1
		}
		u.ok("Canary", "gateway ran researcher card "+task+" automatically; card archived")
	}
	u.ready(id)
	return 0
}

// canary runs the dispatch check: one no-write researcher card that the
// running gateway must claim and complete by itself.
func (a App) canary(id target.Identity, dc string) (string, error) {
	if a.Canary != nil {
		return a.Canary(id, dc)
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 2 * time.Minute}
	}
	return native.DispatchCheck(context.Background(), id, dc, runner, 150*time.Second, 6*time.Minute, 3*time.Second)
}

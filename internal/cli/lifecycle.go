package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// composeProject runs one ordinary Compose command on RepoKit's own generated
// project, streaming output so long image builds stay visible.
func (a App) composeProject(id target.Identity, dc string, stdout, stderr io.Writer, args ...string) error {
	argv := append([]string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose}, args...)
	if a.ComposeExec != nil {
		return a.ComposeExec(stdout, stderr, argv...)
	}
	if _, real := a.Runner.(process.Runner); !real {
		// Injected runners (tests) never reach the real Docker daemon.
		return a.Runner.Run(context.Background(), "docker", argv...).Err
	}
	cmd := exec.Command("docker", argv...)
	cmd.Env = process.CleanEnvironment(os.Environ())
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

// containerState returns the deployment container's Docker state ("" when absent).
func (a App) containerState(id target.Identity, dc string) (string, error) {
	out := a.Runner.Run(context.Background(), "docker", "--context", dc, "container", "inspect", "--format", verify.InspectFormat, id.Container)
	if out.Err != nil || out.Truncated || strings.TrimSpace(out.Output) == "" {
		return "", nil // no such container
	}
	var state verify.Runtime
	if json.Unmarshal([]byte(out.Output), &state) != nil || state.Project != id.Project || state.Workspace != id.Root || state.Home != filepath.Join(id.Root, ".hermes") {
		return "", fmt.Errorf("container %s does not belong to this deployment", id.Container)
	}
	return state.Status, nil
}

// startDeployment builds (when needed) and starts the Hermes service. A
// running current deployment is left alone. Recreating a running deployment
// for an upgraded image is deferred while a Kanban card is running.
func (a App) startDeployment(id target.Identity, dc string, stdout, stderr io.Writer) int {
	ready, err := a.nativeRuntimeReady(id, dc)
	if err != nil {
		fmt.Fprintln(stderr, "start refused:", err)
		return 1
	}
	if ready {
		return 0
	}
	state, err := a.containerState(id, dc)
	if err != nil {
		fmt.Fprintln(stderr, "start refused:", err)
		return 1
	}
	if state == "running" {
		runner := a.Initializer
		if runner == nil {
			runner = process.Runner{Timeout: 30 * time.Second}
		}
		if busy, err := native.RunningWork(context.Background(), id, dc, runner); err == nil && busy {
			fmt.Fprintln(stdout, "An upgraded image is ready, but a Kanban card is running; the container was not recreated.")
			fmt.Fprintf(stdout, "Recreate it when the work finishes: hermes-repokit start (or %s)\n", buildCommand(id, dc))
			return 0
		}
	}
	fmt.Fprintf(stdout, "Building and starting %s (the first build can take several minutes)...\n", id.Container)
	if err := a.composeProject(id, dc, stdout, stderr, "up", "-d", "--build", "hermes"); err != nil {
		fmt.Fprintf(stderr, "Docker Compose could not start the deployment; native state is preserved. Retry with: %s\n", buildCommand(id, dc))
		return 1
	}
	fmt.Fprintf(stdout, "Started %s.\n", id.Container)
	return 0
}

func buildCommand(id target.Identity, dc string) string {
	s := launcher.StartCommand(id.Compose, dc)
	return s[:len(s)-len(" up -d hermes)")] + " up -d --build hermes)"
}

// lifecycleTarget proves a deployment is RepoKit-generated before stop/start.
func (a App) lifecycleTarget(id target.Identity) (string, error) {
	info, err := os.Lstat(filepath.Join(id.Root, ".hermes"))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("no RepoKit deployment here; run hermes-repokit install")
	}
	dc, err := launcher.Context(id)
	if err != nil {
		return "", fmt.Errorf("the launcher is not RepoKit-generated")
	}
	if _, ok := compose.DevelopmentInstallSelected(id); !ok {
		return "", fmt.Errorf(".hermes/compose.yaml is not a RepoKit-generated Compose file")
	}
	return dc, nil
}

// stop stops the deployment's containers; all state and the container stay,
// so `start` resumes where it left off.
func (a App) stop(id target.Identity, stdout, stderr io.Writer) int {
	dc, err := a.lifecycleTarget(id)
	if err == nil {
		_, err = a.containerState(id, dc)
	}
	if err != nil {
		fmt.Fprintln(stderr, "stop refused:", err)
		return 1
	}
	if err := a.composeProject(id, dc, stdout, stderr, "--profile", "docker-tests", "stop"); err != nil {
		fmt.Fprintln(stderr, "Docker Compose could not stop the deployment; inspect it with docker compose.")
		return 1
	}
	fmt.Fprintf(stdout, "Stopped %s. State is preserved; run hermes-repokit start to resume.\n", id.Container)
	return 0
}

// start brings a stopped deployment back and waits for Hermes to answer.
// Hermes restarts gateways that were running before the stop.
func (a App) start(id target.Identity, stdout, stderr io.Writer) int {
	dc, err := a.lifecycleTarget(id)
	if err != nil {
		fmt.Fprintln(stderr, "start refused:", err)
		return 1
	}
	if ready, err := a.nativeRuntimeReady(id, dc); err == nil && ready {
		fmt.Fprintf(stdout, "%s is already running.\n", id.Container)
		return 0
	}
	if code := a.startDeployment(id, dc, stdout, stderr); code != 0 {
		return code
	}
	if err := a.waitForNativeCLI(id, dc); err != nil {
		fmt.Fprintln(stderr, "The container started, but", err)
		return 1
	}
	fmt.Fprintf(stdout, "Hermes is answering. Check the team with: %s -p default gateway status\n", id.Container)
	return 0
}

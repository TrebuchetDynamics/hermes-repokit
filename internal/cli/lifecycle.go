package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
	// Compose's own progress display would break RepoKit's aligned status
	// lines; quiet mode still reports errors. Inherited COMPOSE_* are cleared.
	cmd.Env = append(process.CleanEnvironment(os.Environ()), "COMPOSE_PROGRESS=quiet")
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

// stopTimeout gives RepoKit's own stop and recreation the same shutdown time
// that the generated Compose's stop_grace_period gives every other stop.
var stopTimeout = strconv.Itoa(compose.StopGraceSeconds)

// startDeployment builds (when needed) and starts the Hermes service. A
// running current deployment is left alone. Recreating a running deployment
// for an upgraded image is deferred while a Kanban card is running.
func (a App) startDeployment(id target.Identity, dc string, stdout, stderr io.Writer) int {
	u := newUI(stdout, stderr)
	ready, err := a.nativeRuntimeReady(id, dc)
	if err != nil {
		u.fail("start refused: %v", err)
		return 1
	}
	if ready {
		u.ok("Container", id.Container+" running (up to date)")
		return 0
	}
	state, err := a.containerState(id, dc)
	if err != nil {
		u.fail("start refused: %v", err)
		return 1
	}
	if !a.imageBuilt(id, dc) {
		if free, known := a.dockerFree(dc); known && free < minBuildFree {
			u.fail("only %.1f GB free where Docker keeps images; a build needs about %d GB. %s", float64(free)/(1<<30), minBuildFree>>30, keepsRunning(id, state))
			u.note("free space (for example docker builder prune, or remove unused images), then rerun")
			return 1
		}
	}
	if state == "running" {
		// Build first while the running container keeps working, so the
		// idle check below is followed by a recreation of seconds, not by a
		// build of minutes during which a card could start and be cut off.
		if !a.imageBuilt(id, dc) {
			u.working("Container", "building the upgraded image while "+id.Container+" keeps running (this can take several minutes)")
			if err := a.composeProject(id, dc, stdout, stderr, "build", "hermes"); err != nil {
				u.fail("Docker Compose could not build the upgraded image; %s keeps running unchanged. Retry with: %s", id.Container, buildCommand(id, dc))
				return 1
			}
		}
		runner := a.Initializer
		if runner == nil {
			runner = process.Runner{Timeout: 30 * time.Second}
		}
		if busy, err := native.RunningWork(context.Background(), id, dc, runner); err == nil && busy {
			u.pending("Container", "upgraded image built; not recreated while a Kanban card is running")
			u.note("recreate it when the work finishes: " + self() + " start")
			return 0
		}
		// Recreating restarts the gateway, which would drop a reply the owner
		// is waiting for.
		if quiet, ok := native.ChatQuiet(context.Background(), id, dc, runner); ok && !quiet {
			u.pending("Container", "upgraded image built; not recreated while the owner is mid-conversation")
			u.note("recreate it once the chat has been quiet for 10 minutes: " + self() + " start")
			return 0
		}
	}
	if a.imageBuilt(id, dc) {
		u.working("Container", "starting "+id.Container)
	} else {
		u.working("Container", "building and starting "+id.Container+" (the first build can take several minutes)")
	}
	if err := a.composeProject(id, dc, stdout, stderr, "up", "-d", "--build", "--timeout", stopTimeout, "hermes"); err != nil {
		u.fail("Docker Compose could not start %s; native state is preserved. Retry with: %s", id.Container, buildCommand(id, dc))
		return 1
	}
	u.ok("Container", id.Container+" running")
	if n := a.removeSuperseded(id, dc); n > 0 {
		u.ok("Images", fmt.Sprintf("removed %d earlier image(s) this deployment no longer uses", n))
	}
	return 0
}

// minBuildFree is the free space a development image build needs; a Flutter
// or Godot image adds several gigabytes plus build cache.
const minBuildFree = 15 << 30

func keepsRunning(id target.Identity, state string) string {
	if state == "running" {
		return id.Container + " keeps running unchanged."
	}
	return "Nothing was built."
}

// dockerFree reports the free space on the filesystem holding a local Docker
// daemon's images. It is unknown for a remote daemon.
func (a App) dockerFree(dc string) (uint64, bool) {
	if a.FreeSpace != nil {
		return a.FreeSpace(dc)
	}
	ctx := context.Background()
	host := a.Runner.Run(ctx, "docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}", dc)
	if host.Err != nil || !strings.HasPrefix(strings.TrimSpace(host.Output), "unix://") {
		return 0, false
	}
	root := a.Runner.Run(ctx, "docker", "--context", dc, "info", "--format", "{{.DockerRootDir}}")
	dir := strings.TrimSpace(root.Output)
	var st syscall.Statfs_t
	if root.Err != nil || !filepath.IsAbs(dir) || syscall.Statfs(dir, &st) != nil {
		return 0, false
	}
	return st.Bavail * uint64(st.Bsize), true
}

// removeSuperseded removes this deployment's earlier images once the current
// one runs, so every upgrade does not leave a multi-gigabyte copy behind.
// Only RepoKit's own repository for this container is touched, and never an
// image any container (running or stopped) still uses.
func (a App) removeSuperseded(id target.Identity, dc string) int {
	data, err := os.ReadFile(id.Compose)
	if err != nil {
		return 0
	}
	m := composeImage.FindSubmatch(data)
	repo := "repokit/" + id.Container
	if m == nil || !strings.HasPrefix(string(m[1]), repo+":") {
		return 0
	}
	ctx := context.Background()
	used := a.Runner.Run(ctx, "docker", "--context", dc, "container", "ls", "--all", "--format", "{{.Image}}")
	list := a.Runner.Run(ctx, "docker", "--context", dc, "image", "ls", "--format", "{{.Repository}}:{{.Tag}}", repo)
	if used.Err != nil || used.Truncated || list.Err != nil || list.Truncated {
		return 0
	}
	inUse := map[string]bool{string(m[1]): true}
	for _, ref := range strings.Fields(used.Output) {
		inUse[ref] = true
	}
	n := 0
	for _, ref := range strings.Fields(list.Output) {
		if !strings.HasPrefix(ref, repo+":") || inUse[ref] {
			continue
		}
		if a.Runner.Run(ctx, "docker", "--context", dc, "image", "rm", ref).Err == nil {
			n++
		}
	}
	return n
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
	u := newUI(stdout, stderr)
	u.title("stop", id.Name, id.Root)
	dc, err := a.lifecycleTarget(id)
	if err == nil {
		_, err = a.containerState(id, dc)
	}
	if err != nil {
		u.fail("stop refused: %v", err)
		return 1
	}
	if err := a.composeProject(id, dc, stdout, stderr, "--profile", "docker-tests", "stop", "--timeout", stopTimeout); err != nil {
		u.fail("Docker Compose could not stop the deployment; inspect it with docker compose")
		return 1
	}
	u.ok("Container", id.Container+" stopped; state is preserved")
	u.next([2]string{self() + " start", "resume"})
	return 0
}

// start brings a stopped deployment back and waits for Hermes to answer.
// Hermes restarts gateways that were running before the stop.
func (a App) start(id target.Identity, stdout, stderr io.Writer) int {
	u := newUI(stdout, stderr)
	u.title("start", id.Name, id.Root)
	dc, err := a.lifecycleTarget(id)
	if err != nil {
		u.fail("start refused: %v", err)
		return 1
	}
	if ready, err := a.nativeRuntimeReady(id, dc); err == nil && ready {
		u.ok("Container", id.Container+" is already running")
		return 0
	}
	if code := a.startDeployment(id, dc, stdout, stderr); code != 0 {
		return code
	}
	if err := a.waitForNativeCLI(id, dc); err != nil {
		u.fail("the container started, but %v", err)
		return 1
	}
	u.ok("Hermes", "answering")
	// A deployment whose team was never set up has no gateway to check yet.
	if _, pending, _ := profileProgress(verify.Profiles(id)); pending {
		u.next([2]string{self() + " setup", "set up the team"})
		return 0
	}
	u.next([2]string{id.Container + " -p default gateway status", "check the team's gateway"})
	return 0
}

// imageBuilt reports whether the image named in the generated Compose file
// already exists locally, so the first-build warning is shown only when true.
func (a App) imageBuilt(id target.Identity, dc string) bool {
	data, err := os.ReadFile(id.Compose)
	if err != nil {
		return false
	}
	m := composeImage.FindSubmatch(data)
	if m == nil {
		return false
	}
	return a.Runner.Run(context.Background(), "docker", "--context", dc, "image", "inspect", "--format", "{{.Id}}", string(m[1])).Err == nil
}

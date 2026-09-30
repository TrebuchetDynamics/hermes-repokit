package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

var composeImage = regexp.MustCompile(`(?m)^    image: "([^"]+)"$`)

// removal lists exactly what `remove` deletes, all proven RepoKit-created.
type removal struct {
	dockerContext string
	container     bool   // the deployment's container exists and is ours
	image         string // generated image named in the generated Compose
	hostCommand   string // ~/.local/bin/hermes-<repo> symlink to this launcher
	stateBytes    int64
	stateFiles    int
}

// remove deletes a RepoKit deployment: its Compose project (container,
// network, docker-test volumes), its generated image, its host command, the
// installer lock and its exclude line, and the private .hermes state. It acts
// only on deployments it can prove it generated and only after the owner types
// the repository name in an interactive terminal.
func (a App) remove(id target.Identity, stdout, stderr io.Writer) int {
	plan, err := a.planRemoval(id)
	if err != nil {
		fmt.Fprintln(stderr, "remove refused:", err)
		return 1
	}
	fmt.Fprintf(stdout, "This permanently deletes the RepoKit deployment of %s:\n", id.Root)
	if plan.container {
		fmt.Fprintf(stdout, "  - container %s and its Compose network (running work stops immediately)\n", id.Container)
	}
	if plan.image != "" {
		fmt.Fprintf(stdout, "  - image %s\n", plan.image)
	}
	if plan.hostCommand != "" {
		fmt.Fprintf(stdout, "  - host command %s\n", plan.hostCommand)
	}
	fmt.Fprintf(stdout, "  - %s/.hermes (%d files, %.1f MB): profiles, Kanban board, provider logins,\n", id.Root, plan.stateFiles, float64(plan.stateBytes)/1e6)
	fmt.Fprintln(stdout, "    messaging tokens, sessions and memory. This cannot be undone.")
	fmt.Fprintln(stdout, "  - .hermes-repokit.lock and its line in the local Git exclude file")
	fmt.Fprintln(stdout, "Repository files and Git history are not touched.")
	answer, err := a.confirmation(stdout, fmt.Sprintf("Type the repository name (%s) to delete it: ", id.Name))
	if err != nil {
		fmt.Fprintln(stderr, "remove refused:", err)
		return 1
	}
	if answer != id.Name {
		fmt.Fprintln(stderr, "Confirmation did not match; nothing was removed.")
		return 1
	}
	lock, err := lockForRemoval(id)
	if err != nil {
		fmt.Fprintln(stderr, "remove refused:", err)
		return 1
	}
	defer lock.Close()

	ctx := context.Background()
	// Always run the generated project's `down`: it is idempotent and also
	// removes a network left behind when the container is already gone.
	down := a.Runner.Run(ctx, "docker", "--context", plan.dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose, "--profile", "docker-tests", "down", "--volumes", "--remove-orphans")
	if down.Err != nil {
		fmt.Fprintln(stderr, "remove stopped: Compose could not remove the deployment; nothing else was deleted. Inspect with docker compose, then retry.")
		return 1
	}
	fmt.Fprintf(stdout, "Removed the Compose deployment (%s).\n", id.Container)
	if plan.image != "" {
		if rm := a.Runner.Run(ctx, "docker", "--context", plan.dockerContext, "image", "rm", plan.image); rm.Err != nil {
			fmt.Fprintf(stderr, "Warning: image %s was not removed (still in use or already gone).\n", plan.image)
		} else {
			fmt.Fprintf(stdout, "Removed image %s.\n", plan.image)
		}
	}
	if plan.hostCommand != "" {
		if err := os.Remove(plan.hostCommand); err != nil {
			fmt.Fprintf(stderr, "Warning: host command %s was not removed: %v\n", plan.hostCommand, err)
		} else {
			fmt.Fprintf(stdout, "Removed host command %s.\n", plan.hostCommand)
		}
	}
	if err := removeState(filepath.Join(id.Root, ".hermes")); err != nil {
		fmt.Fprintf(stderr, "Warning: part of .hermes could not be deleted (%v); remove the rest manually.\n", err)
	} else {
		fmt.Fprintln(stdout, "Deleted .hermes.")
	}
	if err := a.dropLockExclude(ctx, id); err != nil {
		fmt.Fprintf(stderr, "Warning: %v\n", err)
	}
	lock.Close()
	os.Remove(filepath.Join(id.Root, ".hermes-repokit.lock"))
	fmt.Fprintln(stdout, "RepoKit deployment removed. Run hermes-repokit install to start over.")
	return 0
}

// planRemoval proves every target is RepoKit's before anything is deleted.
func (a App) planRemoval(id target.Identity) (removal, error) {
	var plan removal
	info, err := os.Lstat(filepath.Join(id.Root, ".hermes"))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return plan, fmt.Errorf("no RepoKit deployment here (.hermes is absent or not a directory)")
	}
	dc, err := launcher.Context(id)
	if err != nil {
		return plan, fmt.Errorf("the launcher is not RepoKit-generated; this .hermes was not created by RepoKit")
	}
	plan.dockerContext = dc
	if _, ok := compose.DevelopmentInstallSelected(id); !ok {
		return plan, fmt.Errorf(".hermes/compose.yaml is not a RepoKit-generated Compose file (edited or foreign); nothing was removed")
	}
	data, err := os.ReadFile(id.Compose)
	if err != nil {
		return plan, err
	}
	if m := composeImage.FindSubmatch(data); m != nil {
		image := string(m[1])
		if strings.HasPrefix(image, "repokit/"+id.Container+":") || strings.HasPrefix(image, id.Project+"-hermes-dev:") {
			plan.image = image
		}
	}
	ctx := context.Background()
	observed := a.Runner.Run(ctx, "docker", "--context", dc, "container", "inspect", "--format", verify.InspectFormat, id.Container)
	if observed.Err == nil && !observed.Truncated {
		var state verify.Runtime
		if json.Unmarshal([]byte(observed.Output), &state) != nil || state.Project != id.Project || state.Workspace != id.Root || state.Home != filepath.Join(id.Root, ".hermes") {
			return plan, fmt.Errorf("container %s does not belong to this deployment; nothing was removed", id.Container)
		}
		plan.container = true
	}
	if home, err := os.UserHomeDir(); err == nil {
		command := filepath.Join(home, ".local", "bin", id.Container)
		if dest, err := os.Readlink(command); err == nil && dest == id.Launcher {
			plan.hostCommand = command
		}
	}
	filepath.WalkDir(filepath.Join(id.Root, ".hermes"), func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			plan.stateFiles++
			if fi, e := d.Info(); e == nil {
				plan.stateBytes += fi.Size()
			}
		}
		return nil
	})
	return plan, nil
}

// confirmation reads one line from an interactive terminal only.
func (a App) confirmation(stdout io.Writer, prompt string) (string, error) {
	if a.Confirm != nil {
		return a.Confirm(prompt)
	}
	if !native.InteractiveInput(a.Stdin) {
		return "", fmt.Errorf("run remove in an interactive terminal to confirm")
	}
	fmt.Fprint(stdout, prompt)
	line, err := bufio.NewReader(a.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("no confirmation read")
	}
	return strings.TrimSpace(line), nil
}

// lockForRemoval takes the installer lock so install/setup cannot run
// concurrently; a running install or setup makes remove refuse.
func lockForRemoval(id target.Identity) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(id.Root, ".hermes-repokit.lock"), os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, fmt.Errorf("installer lock unavailable")
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("install or setup is running for this repository; retry when it finishes")
	}
	return f, nil
}

// dropLockExclude removes only the exact entry (and its comment) that install
// added to the local exclude file; every other line is preserved.
func (a App) dropLockExclude(ctx context.Context, id target.Identity) error {
	out := a.Runner.Run(ctx, "git", "-C", id.Root, "rev-parse", "--git-path", "info/exclude")
	path := strings.TrimSpace(out.Output)
	if out.Err != nil || path == "" {
		return nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(id.Root, path)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	kept := lines[:0]
	for _, line := range lines {
		if line == lockExcludeEntry || line == "# Hermes RepoKit installer lock (local only)" {
			continue
		}
		kept = append(kept, line)
	}
	result := strings.TrimRight(strings.Join(kept, "\n"), "\n")
	if result != "" {
		result += "\n"
	}
	if result == string(data) {
		return nil
	}
	if err := os.WriteFile(path, []byte(result), info.Mode().Perm()); err != nil {
		return fmt.Errorf("could not remove %s from %s", lockExcludeEntry, path)
	}
	return nil
}

// removeState deletes the private state tree. Go marks its module cache
// read-only, and earlier recipes kept that cache in .hermes/development, so a
// failed first pass makes directories inside the tree writable and retries.
// WalkDir never follows symbolic links, so nothing outside the tree changes.
func removeState(dir string) error {
	if err := os.RemoveAll(dir); err == nil {
		return nil
	}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			if info, e := d.Info(); e == nil && info.Mode().Perm()&0200 == 0 {
				os.Chmod(path, info.Mode().Perm()|0700)
			}
		}
		return nil
	})
	return os.RemoveAll(dir)
}

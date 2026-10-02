package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
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
	volumes       []string // existing project volumes the Compose file declares
}

// remove deletes a RepoKit deployment: its Compose project (container,
// network, docker-test volumes), its generated image, its host command, the
// installer lock and its exclude line, and the private .hermes state. It acts
// only on deployments it can prove it generated and only after the owner types
// the repository name in an interactive terminal.
func (a App) remove(id target.Identity, stdout, stderr io.Writer) int {
	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); os.IsNotExist(err) {
		return a.removeLeftovers(id, stdout, stderr)
	}
	plan, err := a.planRemoval(id)
	if err != nil {
		fmt.Fprintln(stderr, "remove refused:", err)
		var foreign foreignState
		if errors.As(err, &foreign) {
			a.foreignGuidance(id, foreign, stderr)
		}
		return 1
	}
	fmt.Fprintf(stdout, "This permanently deletes the RepoKit deployment of %s:\n", id.Root)
	if plan.container {
		fmt.Fprintf(stdout, "  - container %s and its Compose network (running work stops immediately)\n", id.Container)
	}
	if len(plan.volumes) > 0 {
		fmt.Fprintf(stdout, "  - volumes %s (toolchain caches and Docker test data)\n", strings.Join(plan.volumes, ", "))
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
	fmt.Fprintf(stdout, "RepoKit deployment removed. Run %s install to start over.\n", self())
	return 0
}

// foreignState refuses state RepoKit cannot prove it generated.
type foreignState struct {
	reason         string
	earlierRelease bool // a RepoKit launcher with a Compose RepoKit no longer recognizes
}

func (f foreignState) Error() string { return f.reason }

// foreignGuidance tells the owner what refused state contains and how to
// remove it themselves; RepoKit never deletes state it cannot prove it made.
func (a App) foreignGuidance(id target.Identity, f foreignState, w io.Writer) {
	state := filepath.Join(id.Root, ".hermes")
	if f.earlierRelease {
		fmt.Fprintf(w, "If an earlier RepoKit release made it, run %s install to upgrade it, then %s remove.\n", self(), self())
	}
	fmt.Fprintln(w, "RepoKit never deletes state it cannot prove it created. To remove it yourself:")
	step := 0
	say := func(format string, args ...any) {
		step++
		fmt.Fprintf(w, "  %d. "+format+"\n", append([]any{step}, args...)...)
	}
	switch containers, err := a.containersMounting(state); {
	case err != nil:
		say("Check for containers that use it: docker ps -a (stop and remove any that mount %s)", shellQuote(state))
	case len(containers) > 0:
		say("Stop and remove its containers: docker rm -f %s", strings.Join(containers, " "))
	}
	if links := hostLinksInto(state); len(links) > 0 {
		say("Remove its host commands: rm %s", strings.Join(quoteAll(links), " "))
	}
	files, size := 0, int64(0)
	filepath.WalkDir(state, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files++
			if fi, e := d.Info(); e == nil {
				size += fi.Size()
			}
		}
		return nil
	})
	say("Copy out anything you want to keep, then delete it: rm -rf %s (%d files, %.1f MB, including provider logins, sessions and memory)", shellQuote(state), files, float64(size)/1e6)
	fmt.Fprintf(w, "Then start again with %s install.\n", self())
}

// containersMounting names the containers whose mounts include dir, read
// through the default Docker context.
func (a App) containersMounting(dir string) ([]string, error) {
	ctx := context.Background()
	listed := a.Runner.Run(ctx, "docker", "ps", "-a", "--format", "{{.Names}}")
	if listed.Err != nil || listed.Truncated {
		return nil, errors.New("docker unavailable")
	}
	var found []string
	for _, name := range strings.Fields(listed.Output) {
		mounts := a.Runner.Run(ctx, "docker", "inspect", "--format", "{{range .Mounts}}{{.Source}}\n{{end}}", name)
		if mounts.Err != nil {
			continue
		}
		for _, source := range strings.Split(mounts.Output, "\n") {
			if source == dir || strings.HasPrefix(source, dir+"/") {
				found = append(found, name)
				break
			}
		}
	}
	return found, nil
}

// hostLinksInto lists ~/.local/bin symlinks that point inside dir.
func hostLinksInto(dir string) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	bin := filepath.Join(home, ".local", "bin")
	entries, err := os.ReadDir(bin)
	if err != nil {
		return nil
	}
	var links []string
	for _, e := range entries {
		path := filepath.Join(bin, e.Name())
		if dest, err := os.Readlink(path); err == nil && strings.HasPrefix(dest, dir+"/") {
			links = append(links, path)
		}
	}
	return links
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func quoteAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = shellQuote(s)
	}
	return out
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
		return plan, foreignState{reason: "the launcher is not RepoKit-generated; this .hermes was not created by RepoKit"}
	}
	plan.dockerContext = dc
	// `down --volumes` runs with this file, so only RepoKit's exact current
	// Compose qualifies: an edited or earlier one could name owner volumes.
	if _, ok := compose.DevelopmentInstallSelected(id); !ok {
		return plan, foreignState{reason: ".hermes/compose.yaml is not RepoKit's current generated Compose (edited, foreign or from an earlier release); nothing was removed", earlierRelease: true}
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
	// `down --volumes` deletes the volumes the generated Compose declares;
	// list those that exist so the confirmation names everything deleted.
	for _, name := range declaredVolumes(data) {
		volume := id.Project + "_" + name
		if inspected := a.Runner.Run(ctx, "docker", "--context", dc, "volume", "inspect", volume); inspected.Err == nil {
			plan.volumes = append(plan.volumes, volume)
		}
	}
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

// declaredVolumes reads the top-level volumes block of a generated Compose
// file: two-space-indented names under "volumes:" until the next key.
func declaredVolumes(compose []byte) []string {
	var names []string
	in := false
	for _, line := range strings.Split(string(compose), "\n") {
		switch {
		case line == "volumes:":
			in = true
		case in && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && strings.HasSuffix(line, ":"):
			names = append(names, strings.TrimSuffix(strings.TrimSpace(line), ":"))
		case in:
			in = false
		}
	}
	return names
}

// leftovers are what an install of this repository left behind once its
// .hermes is gone: without the generated Compose file, each is proved by the
// Compose project label (a hash of this exact repository path), the mount
// sources, RepoKit's image name or the host link's exact destination.
type leftovers struct {
	dockerContext     string
	container, image  string
	volumes, networks []string
	hostCommand       string
}

func (a App) planLeftovers(ctx context.Context, id target.Identity) (leftovers, error) {
	var l leftovers
	dc := a.Runner.Run(ctx, "docker", "context", "show")
	if dc.Err != nil || dc.Truncated {
		return l, fmt.Errorf("Docker context unavailable")
	}
	l.dockerContext = strings.TrimSpace(dc.Output)
	docker := func(args ...string) process.Result {
		return a.Runner.Run(ctx, "docker", append([]string{"--context", l.dockerContext}, args...)...)
	}
	if observed := docker("container", "inspect", "--format", verify.InspectFormat, id.Container); observed.Err == nil && !observed.Truncated {
		var state verify.Runtime
		if json.Unmarshal([]byte(observed.Output), &state) != nil || state.Project != id.Project || state.Workspace != id.Root || state.Home != filepath.Join(id.Root, ".hermes") {
			return l, fmt.Errorf("container %s does not belong to this repository; nothing was removed", id.Container)
		}
		l.container = id.Container
		if image := docker("container", "inspect", "--format", "{{.Config.Image}}", id.Container); image.Err == nil {
			if name := strings.TrimSpace(image.Output); strings.HasPrefix(name, "repokit/"+id.Container+":") {
				l.image = name
			}
		}
	}
	label := "label=com.docker.compose.project=" + id.Project
	if out := docker("volume", "ls", "--quiet", "--filter", label); out.Err == nil {
		l.volumes = strings.Fields(out.Output)
	}
	if out := docker("network", "ls", "--quiet", "--filter", label); out.Err == nil {
		l.networks = strings.Fields(out.Output)
	}
	if home, err := os.UserHomeDir(); err == nil {
		command := filepath.Join(home, ".local", "bin", id.Container)
		if dest, err := os.Readlink(command); err == nil && dest == id.Launcher {
			l.hostCommand = command
		}
	}
	return l, nil
}

// removeLeftovers clears an install whose .hermes is gone, so install can
// start over. It never touches anything it cannot prove belongs here.
func (a App) removeLeftovers(id target.Identity, stdout, stderr io.Writer) int {
	ctx := context.Background()
	l, err := a.planLeftovers(ctx, id)
	if err != nil {
		fmt.Fprintln(stderr, "remove refused:", err)
		return 1
	}
	if l.container == "" && l.image == "" && len(l.volumes)+len(l.networks) == 0 && l.hostCommand == "" {
		fmt.Fprintln(stderr, "remove refused: no RepoKit deployment here (.hermes is absent and nothing from an earlier install remains)")
		return 1
	}
	fmt.Fprintf(stdout, "%s/.hermes is gone; these are left from an earlier install of this repository:\n", id.Root)
	if l.container != "" {
		fmt.Fprintf(stdout, "  - container %s (running work stops immediately)\n", l.container)
	}
	if len(l.volumes) > 0 {
		fmt.Fprintf(stdout, "  - volumes %s\n", strings.Join(l.volumes, ", "))
	}
	if len(l.networks) > 0 {
		fmt.Fprintf(stdout, "  - networks %s\n", strings.Join(l.networks, ", "))
	}
	if l.image != "" {
		fmt.Fprintf(stdout, "  - image %s\n", l.image)
	}
	if l.hostCommand != "" {
		fmt.Fprintf(stdout, "  - host command %s\n", l.hostCommand)
	}
	fmt.Fprintln(stdout, "Repository files and Git history are not touched.")
	answer, err := a.confirmation(stdout, fmt.Sprintf("Type the repository name (%s) to remove them: ", id.Name))
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
	docker := func(args ...string) error {
		return a.Runner.Run(ctx, "docker", append([]string{"--context", l.dockerContext}, args...)...).Err
	}
	if l.container != "" {
		if err := docker("container", "rm", "--force", l.container); err != nil {
			fmt.Fprintf(stderr, "remove stopped: container %s could not be removed; nothing else was deleted.\n", l.container)
			return 1
		}
		fmt.Fprintf(stdout, "Removed container %s.\n", l.container)
	}
	for _, network := range l.networks {
		if docker("network", "rm", network) != nil {
			fmt.Fprintf(stderr, "Warning: network %s was not removed.\n", network)
		}
	}
	for _, volume := range l.volumes {
		if docker("volume", "rm", volume) != nil {
			fmt.Fprintf(stderr, "Warning: volume %s was not removed.\n", volume)
		}
	}
	if l.image != "" && docker("image", "rm", l.image) != nil {
		fmt.Fprintf(stderr, "Warning: image %s was not removed (still in use or already gone).\n", l.image)
	}
	if l.hostCommand != "" {
		if err := os.Remove(l.hostCommand); err != nil {
			fmt.Fprintf(stderr, "Warning: host command %s was not removed: %v\n", l.hostCommand, err)
		} else {
			fmt.Fprintf(stdout, "Removed host command %s.\n", l.hostCommand)
		}
	}
	if err := a.dropLockExclude(ctx, id); err != nil {
		fmt.Fprintf(stderr, "Warning: %v\n", err)
	}
	lock.Close()
	os.Remove(filepath.Join(id.Root, ".hermes-repokit.lock"))
	fmt.Fprintf(stdout, "Leftovers removed. Run %s install to start over.\n", self())
	return 0
}

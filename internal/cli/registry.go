package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// Deployment is RepoKit's local record of one repository it installed into,
// kept so every deployment on this machine can be listed and debugged from
// one place. install writes it and remove deletes it; nothing reads it while a
// deployment runs.
type Deployment struct {
	Name          string    `json:"name"`
	Root          string    `json:"root"`
	Container     string    `json:"container"`
	Project       string    `json:"project"`
	Launcher      string    `json:"launcher"`
	DockerContext string    `json:"docker_context"`
	Image         string    `json:"image,omitempty"`
	Toolchains    []string  `json:"toolchains"`
	DockerTests   bool      `json:"docker_tests"`
	RepoKit       string    `json:"repokit"`
	LastInstall   time.Time `json:"last_install"`
	LastResult    string    `json:"last_result"`
}

// registryPath is $XDG_STATE_HOME/repokit/deployments.json, or
// ~/.local/state/repokit/deployments.json.
func registryPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" || !filepath.IsAbs(base) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "repokit", "deployments.json"), nil
}

func loadRegistry() (map[string]Deployment, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	records := map[string]Deployment{}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return records, nil
	}
	if err != nil {
		return nil, err
	}
	var list []Deployment
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("deployment record %s is unreadable", path)
	}
	for _, d := range list {
		records[d.Root] = d
	}
	return records, nil
}

func saveRegistry(records map[string]Deployment) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	if err := safeStateDir(filepath.Dir(path)); err != nil {
		return err
	}
	list := make([]Deployment, 0, len(records))
	for _, d := range records {
		list = append(list, d)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Root < list[j].Root })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// safeStateDir creates dir and its missing parents below the user's home,
// refusing any existing component that is a symlink or not owned by this user,
// like the host command's directory: the record never follows a redirect.
func safeStateDir(dir string) error {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		info, err := os.Lstat(d)
		if os.IsNotExist(err) {
			missing = append([]string{d}, missing...)
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is not a plain directory", d)
		}
		if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Geteuid()) {
			return fmt.Errorf("%s is not owned by you", d)
		}
		break
	}
	for _, d := range missing {
		if err := os.Mkdir(d, 0700); err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// recordInstall notes an install run, successful or not, once the deployment
// exists; a refused install leaves no record, and a failure to record never
// fails the install.
func (a App) recordInstall(id target.Identity, report Plan, code int) {
	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err != nil {
		return
	}
	records, err := loadRegistry()
	if err != nil {
		return
	}
	d := Deployment{Name: id.Name, Root: id.Root, Container: id.Container, Project: id.Project, Launcher: id.Launcher,
		DockerContext: report.DockerContext, Toolchains: []string{}, DockerTests: report.DockerTests, RepoKit: Version,
		LastInstall: time.Now().UTC().Truncate(time.Second), LastResult: "ok"}
	if code != 0 {
		d.LastResult = "failed"
	}
	for name, on := range map[string]bool{"go": report.Development.Go, "rust": report.Development.Rust, "flutter": report.Development.Flutter, "flutter-linux": report.Development.FlutterLinux} {
		if on {
			d.Toolchains = append(d.Toolchains, name)
		}
	}
	if report.Development.Godot != "" {
		d.Toolchains = append(d.Toolchains, "godot-"+report.Development.Godot)
		for _, platform := range report.Development.GodotExport {
			d.Toolchains = append(d.Toolchains, "godot-export-"+platform)
		}
	}
	sort.Strings(d.Toolchains)
	if data, err := os.ReadFile(id.Compose); err == nil {
		if m := composeImage.FindSubmatch(data); m != nil {
			d.Image = string(m[1])
		}
	}
	records[d.Root] = d
	_ = saveRegistry(records)
}

// forgetDeployment drops a removed deployment's record.
func forgetDeployment(root string) {
	if records, err := loadRegistry(); err == nil {
		if _, ok := records[root]; ok {
			delete(records, root)
			_ = saveRegistry(records)
		}
	}
}

// listDeployments prints every recorded deployment with its live state: the
// container's status and whether its .hermes still exists.
func (a App) listDeployments(asJSON bool, stdout, stderr io.Writer) int {
	records, err := loadRegistry()
	if err != nil {
		fmt.Fprintln(stderr, "list:", err)
		return 1
	}
	type row struct {
		Deployment
		Container string `json:"container_state"`
		State     string `json:"state_dir"`
	}
	rows := []row{}
	for _, d := range records {
		r := row{Deployment: d, Container: "unknown", State: "present"}
		if _, err := os.Lstat(filepath.Join(d.Root, ".hermes")); err != nil {
			r.State = "missing"
		}
		if a.Runner != nil {
			ctx := context.Background()
			args := []string{"container", "inspect", "--format", "{{.State.Status}}", d.Container}
			if d.DockerContext != "" {
				args = append([]string{"--context", d.DockerContext}, args...)
			}
			if out := a.Runner.Run(ctx, "docker", args...); out.Err == nil {
				r.Container = strings.TrimSpace(out.Output)
			} else {
				r.Container = "absent"
			}
		}
		rows = append(rows, r)
	}
	// Containers RepoKit built (repokit/ images) that predate the record show
	// up too, so the list is complete before every repository reinstalls.
	if a.Runner != nil {
		for _, d := range a.unrecordedContainers(records) {
			r := row{Deployment: d, Container: d.LastResult, State: "present"}
			r.LastResult = "not recorded"
			if _, err := os.Lstat(filepath.Join(d.Root, ".hermes")); err != nil {
				r.State = "missing"
			}
			rows = append(rows, r)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Root < rows[j].Root })
	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if enc.Encode(rows) != nil {
			return 1
		}
		return 0
	}
	if len(rows) == 0 {
		fmt.Fprintln(stdout, "No RepoKit deployments recorded yet; each repokit install records its repository.")
		return 0
	}
	w := tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCONTAINER\tSTATE\tTOOLCHAINS\tREPOKIT\tLAST INSTALL\tROOT")
	for _, r := range rows {
		tools := strings.Join(r.Toolchains, ",")
		if tools == "" {
			tools = "-"
		}
		when, version := "-", r.RepoKit
		if !r.LastInstall.IsZero() {
			when = r.LastInstall.Local().Format("2006-01-02 15:04")
		}
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(w, "%s\t%s (%s)\t.hermes %s\t%s\t%s\t%s %s\t%s\n", r.Name, r.Deployment.Container, r.Container, r.State, tools, version, when, r.LastResult, tildePath(r.Root))
	}
	w.Flush()
	return 0
}

// unrecordedContainers finds containers running a RepoKit-built image whose
// repository has no record yet, on the default Docker context. LastResult
// carries the container's status for the caller.
func (a App) unrecordedContainers(records map[string]Deployment) []Deployment {
	ctx := context.Background()
	names := a.Runner.Run(ctx, "docker", "ps", "-a", "--format", "{{.Names}}")
	if names.Err != nil {
		return nil
	}
	known := map[string]bool{}
	for _, d := range records {
		known[d.Container] = true
	}
	var found []Deployment
	for _, name := range strings.Fields(names.Output) {
		if known[name] || !strings.HasPrefix(name, "hermes-") {
			continue
		}
		out := a.Runner.Run(ctx, "docker", "container", "inspect", "--format", `{{.Config.Image}}|{{.State.Status}}|{{index .Config.Labels "com.docker.compose.project"}}|{{range .Mounts}}{{if eq .Destination "/workspace"}}{{.Source}}{{end}}{{end}}`, name)
		parts := strings.Split(strings.TrimSpace(out.Output), "|")
		if out.Err != nil || len(parts) != 4 || !strings.HasPrefix(parts[0], "repokit/") || parts[3] == "" {
			continue
		}
		if _, ok := records[parts[3]]; ok {
			continue
		}
		found = append(found, Deployment{Name: strings.TrimPrefix(name, "hermes-"), Root: parts[3], Container: name, Project: parts[2], Image: parts[0], Toolchains: []string{}, LastResult: parts[1]})
	}
	return found
}

package verify

import (
	"context"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

var extensionModule = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)(\.[^.]+)?\.so$`)

// ImportCollisions lists repository-root entries that Python would import in
// place of a pinned Hermes module when a worker runs from /workspace. It only
// reads directory entries; it never renames or modifies repository files.
func ImportCollisions(id target.Identity) ([]string, error) {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, err
	}
	hermes := map[string]bool{}
	for _, name := range qualification.HermesImportNames {
		hermes[name] = true
	}
	var found []string
	for _, entry := range entries {
		name := entry.Name()
		info, err := root.Stat(name) // follows in-root symlinks, as Python does
		if err != nil {
			continue
		}
		if info.IsDir() {
			// A directory without __init__ is a namespace package, which never
			// shadows a regular Hermes package later on sys.path.
			if !hermes[name] {
				continue
			}
			for _, init := range []string{"__init__.py", "__init__.pyc"} {
				if f, err := root.Stat(name + "/" + init); err == nil && f.Mode().IsRegular() {
					found = append(found, "/workspace/"+name+"/")
					break
				}
			}
			continue
		}
		module := ""
		switch {
		case strings.HasSuffix(name, ".py"):
			module = strings.TrimSuffix(name, ".py")
		case strings.HasSuffix(name, ".pyc"):
			module = strings.TrimSuffix(name, ".pyc")
		case extensionModule.MatchString(name):
			module = extensionModule.FindStringSubmatch(name)[1]
		}
		if hermes[module] {
			found = append(found, "/workspace/"+name)
		}
	}
	sort.Strings(found)
	return found, nil
}

// WorkerLauncher is the console script the development image pins as
// HERMES_BIN, so Hermes starts workers without /workspace on their import path.
const WorkerLauncher = "/opt/hermes/.venv/bin/hermes"

// PythonImports reports worker import collisions as a core blocker. A running
// container whose Hermes launches workers through its console script is not
// affected; that is observed live, never assumed from the recipe.
func PythonImports(id target.Identity) Probe {
	return pythonImports(id, false)
}

// PythonImportsLive is PythonImports with the running container's launcher
// observed: only that container's HERMES_BIN is read.
func PythonImportsLive(ctx context.Context, id target.Identity, r Runner) Probe {
	found, err := ImportCollisions(id)
	if err != nil || len(found) == 0 {
		return pythonImports(id, false)
	}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return pythonImports(id, false)
	}
	out := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", container, "printenv", "HERMES_BIN")
	return pythonImports(id, out.Err == nil && !out.Truncated && strings.TrimSpace(out.Output) == WorkerLauncher)
}

func pythonImports(id target.Identity, launcherSafe bool) Probe {
	found, err := ImportCollisions(id)
	if err != nil {
		return Probe{"python-imports", Unknown, "repository root could not be listed for Python import collisions"}
	}
	if len(found) == 0 {
		return Probe{"python-imports", Healthy, "no repository-root Python module shadows the pinned Hermes runtime's imports"}
	}
	if launcherSafe {
		return Probe{"python-imports", Healthy, "repository-root " + strings.Join(found, ", ") + " shares a Hermes module name, but Hermes starts workers through its console script (HERMES_BIN), which keeps /workspace off their import path"}
	}
	return Probe{"python-imports", Degraded, "Python import collision: " + strings.Join(found, ", ") +
		". Hermes workers run from /workspace and may import these repository modules instead of Hermes' internal ones, so capabilities such as file tools can disappear. RepoKit will not rename or modify repository files; the current development image pins HERMES_BIN to avoid this, so rerun repokit install and recreate the container."}
}

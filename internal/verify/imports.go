package verify

import (
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

// PythonImports reports worker import collisions as a core blocker.
func PythonImports(id target.Identity) Probe {
	found, err := ImportCollisions(id)
	if err != nil {
		return Probe{"python-imports", Unknown, "repository root could not be listed for Python import collisions"}
	}
	if len(found) == 0 {
		return Probe{"python-imports", Healthy, "no repository-root Python module shadows the pinned Hermes runtime's imports"}
	}
	return Probe{"python-imports", Degraded, "Python import collision: " + strings.Join(found, ", ") +
		". Hermes workers run from /workspace and may import these repository modules instead of Hermes' internal ones, so capabilities such as file tools can disappear. RepoKit will not rename or modify repository files; see docs/qualification/python-import-collision.md for the upstream Hermes issue."}
}

package verify

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func importFixture(t *testing.T, files ...string) target.Identity {
	t.Helper()
	dir := t.TempDir()
	for _, rel := range files {
		path := filepath.Join(dir, rel)
		if strings.HasSuffix(rel, "/") {
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# fixture\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return target.Identity{Root: dir}
}

func TestImportCollisionsFollowPythonPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		want  []string
	}{
		{"root tools package", []string{"tools/__init__.py", "tools/x.py"}, []string{"/workspace/tools/"}},
		{"other runtime packages", []string{"plugins/__init__.py", "providers/__init__.py", "agent/__init__.pyc"}, []string{"/workspace/agent/", "/workspace/plugins/", "/workspace/providers/"}},
		{"root modules", []string{"utils.py", "cli.py", "toolsets.pyc", "run_agent.cpython-313-x86_64-linux-gnu.so"}, []string{"/workspace/cli.py", "/workspace/run_agent.cpython-313-x86_64-linux-gnu.so", "/workspace/toolsets.pyc", "/workspace/utils.py"}},
		{"nested package is not on the root path", []string{"src/tools/__init__.py", "python-finrl/utils.py"}, nil},
		{"namespace directory without __init__", []string{"tools/", "tools/README.md", "plugins/x.py"}, nil},
		{"unrelated names and documents", []string{"tools.md", "setup.py", "evals/__init__.py", "main.py", "README.md", "tool/__init__.py"}, nil},
		{"clean repository", []string{"go.mod", "main.go", "docs/tools.md"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ImportCollisions(importFixture(t, tc.files...))
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v %v, want %v", got, err, tc.want)
			}
		})
	}
}

func TestPythonImportsProbeIsObservational(t *testing.T) {
	id := importFixture(t, "tools/__init__.py", "utils.py", "docs/a.md")
	before := snapshotTree(t, id.Root)
	p := PythonImports(id)
	if p.Status != Degraded || !strings.Contains(p.Detail, "/workspace/tools/") || !strings.Contains(p.Detail, "will not rename or modify") {
		t.Fatalf("collision not reported: %+v", p)
	}
	if after := snapshotTree(t, id.Root); !reflect.DeepEqual(before, after) {
		t.Fatal("verify changed repository files")
	}
	if clean := PythonImports(importFixture(t, "main.go")); clean.Status != Healthy {
		t.Fatalf("clean repository flagged: %+v", clean)
	}
}

func TestImportCollisionBlocksCoreReadiness(t *testing.T) {
	probes := []Probe{{"compose", Healthy, ""}, {"gateway", Healthy, ""}, {"kanban:dispatch", Healthy, ""}, {"review:evidence", Healthy, ""},
		PythonImports(importFixture(t, "tools/__init__.py"))}
	got := Readiness(probes)
	if got[0].Status != Degraded || !strings.Contains(got[0].Detail, "python-imports") || CoreUsable(got) {
		t.Fatalf("CORE_READY must not be healthy with a worker import collision: %+v", got[0])
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data := ""
		if d.Type().IsRegular() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			data = string(b)
		}
		out[path] = info.Mode().String() + info.ModTime().String() + data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

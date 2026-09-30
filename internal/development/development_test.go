package development

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func manifest(t *testing.T, root, name, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDetectRootManifests(t *testing.T) {
	root := t.TempDir()
	manifest(t, root, "go.mod", "module example.org/repo\ngo 1.26.0\n")
	manifest(t, root, "package.json", `{"engines":{"node":">=22","npm":">=10"}}`)
	manifest(t, root, "pyproject.toml", "[project]\nrequires-python = \">=3.11,<3.14\"\n")
	manifest(t, root, "requirements-dev.txt", "pytest\n")
	manifest(t, root, "Makefile", "test:\n\tgo test ./...\n")
	r, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Go || len(r.Unsupported) != 0 {
		t.Fatalf("%+v", r)
	}
	if !reflect.DeepEqual(r.Detected, []string{"go", "make", "node", "python"}) {
		t.Fatalf("%+v", r)
	}
}

func TestDetectRejectsUnsafeManifests(t *testing.T) {
	for _, mode := range []string{"symlink", "directory", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "go.mod")
			switch mode {
			case "symlink":
				other := filepath.Join(t.TempDir(), "go.mod")
				if err := os.WriteFile(other, []byte("go 1.26.0"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				manifest(t, root, "go.mod", strings.Repeat("x", maxManifestBytes+1))
			}
			if _, err := Detect(root); err == nil {
				t.Fatal("unsafe manifest accepted")
			}
		})
	}
}

func TestDetectReportsUnqualifiedRequirements(t *testing.T) {
	for _, tc := range []struct{ name, content string }{
		{"go.mod", "go 1.99.0\n"},
		{"go.mod", "go 1.26.0\ntoolchain go1.99.0\n"},
		{"package.json", `{"engines":{"node":"^20.0.0"}}`},
		{"package.json", `{"engines":{"node":"latest"}}`},
		{"package.json", `{"packageManager":"pnpm@10.0.0"}`},
		{"pyproject.toml", "[project]\nrequires-python = \">=3.14\"\n"},
		{"pyproject.toml", "[project]\nrequires-python = \"==3.13\"\n"},
		{"pyproject.toml", "[project]\nrequires-python = \"^3.0\"\n"},
		{"package.json", `{"engines":{"bun":"1"}}`},
		{"Cargo.toml", "[package]\nname='example'\n"},
		{"pom.xml", "<project/>"},
		{"build.gradle.kts", "plugins {}"},
	} {
		t.Run(tc.name+tc.content, func(t *testing.T) {
			root := t.TempDir()
			manifest(t, root, tc.name, tc.content)
			r, err := Detect(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Unsupported) == 0 {
				t.Fatalf("requirements unexpectedly qualified: %+v", r)
			}
		})
	}
}

func TestDetectLimitsRootManifestCount(t *testing.T) {
	root := t.TempDir()
	for i := 0; i <= maxManifests; i++ {
		manifest(t, root, fmt.Sprintf("requirements-%d.txt", i), "")
	}
	if _, err := Detect(root); err == nil {
		t.Fatal("unbounded manifests accepted")
	}
}

func TestDockerfileRunSyntax(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not installed")
	}
	files, err := Recipe(Requirements{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	// Validate complete continued RUN instructions without executing downloads.
	var instruction string
	for _, line := range strings.Split(string(files["Dockerfile"]), "\n") {
		if instruction == "" {
			if strings.HasPrefix(line, "RUN ") {
				instruction = strings.TrimPrefix(line, "RUN ")
			} else {
				continue
			}
		} else {
			instruction += line
		}
		if strings.HasSuffix(instruction, "\\") {
			instruction = strings.TrimSuffix(instruction, "\\") + "\n"
			continue
		}
		cmd := exec.Command("sh", "-n")
		cmd.Stdin = strings.NewReader(instruction)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("RUN syntax: %s %v", output, err)
		}
		instruction = ""
	}
	if instruction != "" {
		t.Fatal("unfinished RUN")
	}
}

func TestDetectIgnoresNestedProjectsAndDoesNotExecuteManifests(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest(t, filepath.Join(root, "nested"), "go.mod", "go 1.26.0\n")
	manifest(t, root, "package.json", `{"scripts":{"postinstall":"touch OWNED"}}`)
	r, err := Detect(root)
	if err != nil || r.Go {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(root, "OWNED")); !os.IsNotExist(err) {
		t.Fatal("manifest executed")
	}
}

func TestRecipePinsInputsAndChangesForGo(t *testing.T) {
	base, err := Recipe(Requirements{})
	if err != nil {
		t.Fatal(err)
	}
	goRecipe, err := Recipe(Requirements{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(base["Dockerfile"]), "go.dev/dl") {
		t.Fatal("Go installed without manifest")
	}
	for _, want := range []string{"@sha256:", "sha256sum -c", "go" + GoVersion + ".linux-", "GOTOOLCHAIN=local", "WORKDIR /workspace", "org.repokit.development.recipe=" + Fingerprint(Requirements{Go: true}), "/usr/local/lib/docker/cli-plugins/docker-compose", "/usr/local/lib/docker/cli-plugins/docker-buildx", "docker compose version --short", "docker buildx version"} {
		if !strings.Contains(string(goRecipe["Dockerfile"]), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, forbidden := range []string{"apt-get", ":latest", "docker.sock", "npm install", "pip install"} {
		if strings.Contains(string(goRecipe["Dockerfile"]), forbidden) {
			t.Fatalf("unexpected %s", forbidden)
		}
	}
	if ImageName("repo", Requirements{}) == ImageName("repo", Requirements{Go: true}) {
		t.Fatal("recipe identity did not change")
	}
	if Fingerprint(Requirements{Go: true}) != Fingerprint(Requirements{Go: true, Detected: []string{"go"}}) {
		t.Fatal("manifest inventory changed same image recipe")
	}
}

func TestGoCachesLiveOnTheToolchainVolume(t *testing.T) {
	files, err := Recipe(Requirements{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	dockerfile := string(files["Dockerfile"])
	for _, want := range []string{"install -d -m 1777 /var/cache/repokit", "GOCACHE=/var/cache/repokit/go-build", "GOMODCACHE=/var/cache/repokit/go-mod"} {
		if !strings.Contains(dockerfile, want) {
			t.Errorf("recipe missing %q", want)
		}
	}
	if strings.Contains(dockerfile, "/opt/data/development") {
		t.Error("Go caches still point into the repository's .hermes")
	}
	if _, ok := GeneratedRecipe(files); !ok {
		t.Error("recipe no longer self-certifies")
	}
}

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

// A monorepo's nested go.mod provisions Go. The walk is bounded, skips
// vendored, generated and hidden trees, never follows symlinks, never executes
// a manifest, and a nested problem is a note with its path, not a failure.
func TestDetectNestedProjectsWithinBounds(t *testing.T) {
	put := func(root, rel, value string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		manifest(t, filepath.Dir(path), filepath.Base(path), value)
	}
	root := t.TempDir()
	put(root, "rig/go.mod", "go 1.26.0\n")
	put(root, "app/android/build.gradle.kts", "plugins {}\n")
	put(root, "web/package.json", `{"scripts":{"postinstall":"touch OWNED"}}`)
	r, err := Detect(root)
	if err != nil || !r.Go || len(r.Unsupported) != 0 || strings.Join(r.Detected, ",") != "go,jvm,node" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(root, "web", "OWNED")); !os.IsNotExist(err) {
		t.Fatal("manifest executed")
	}
	for _, hidden := range []string{"node_modules/x/go.mod", "vendor/x/go.mod", "third_party/x/go.mod", ".cache/x/go.mod", "testdata/go.mod", "a/b/c/d/go.mod"} {
		skip := t.TempDir()
		put(skip, hidden, "go 1.26.0\n")
		if r, err := Detect(skip); err != nil || r.Go {
			t.Errorf("%s provisioned Go: %+v %v", hidden, r, err)
		}
	}
	linked := t.TempDir()
	put(linked, "real/go.mod", "go 1.26.0\n")
	outside := t.TempDir()
	put(outside, "go.mod", "go 1.26.0\n")
	if err := os.Symlink(outside, filepath.Join(linked, "elsewhere")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "go.mod"), filepath.Join(linked, "real", "go.sum")); err != nil {
		t.Fatal(err)
	}
	if r, err := Detect(linked); err != nil || !r.Go || len(r.Unsupported) != 0 {
		t.Fatalf("symlinked tree changed detection: %+v %v", r, err)
	}
	newer := t.TempDir()
	put(newer, "svc/go.mod", "go 1.99.0\n")
	if r, err := Detect(newer); err != nil || !r.Go || len(r.Unsupported) != 1 || !strings.HasPrefix(r.Unsupported[0], "svc/go.mod: ") {
		t.Fatalf("nested unqualified Go not reported with its path: %+v %v", r, err)
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
	if strings.Contains(string(base["Dockerfile"]), "go.dev/dl") || strings.Contains(string(base["Dockerfile"]), "staticcheck") {
		t.Fatal("Go tools installed without manifest")
	}
	for _, want := range []string{"@sha256:", "sha256sum -c", "go" + GoVersion + ".linux-", "GOTOOLCHAIN=local", "WORKDIR /workspace", "org.repokit.development.recipe=" + Fingerprint(Requirements{Go: true}), "/usr/local/lib/docker/cli-plugins/docker-compose", "/usr/local/lib/docker/cli-plugins/docker-buildx", "docker compose version --short", "docker buildx version"} {
		if !strings.Contains(string(goRecipe["Dockerfile"]), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, forbidden := range []string{"apt-get", ":latest", "docker.sock", "npm install"} {
		if strings.Contains(string(goRecipe["Dockerfile"]), forbidden) {
			t.Fatalf("unexpected %s", forbidden)
		}
	}
	// A Python install is admitted only when every file is hash-verified.
	for _, line := range strings.Split(string(goRecipe["Dockerfile"]), "\n") {
		if strings.Contains(line, "pip install") && !strings.Contains(line, "--require-hashes") {
			t.Fatalf("unpinned pip install: %s", line)
		}
	}
	for _, want := range []string{"ast-grep/releases/download/0.45.3/", "ast-grep --version | grep -F 0.45.3", "ddgs --help", "sheeki03/tirith/releases/download/v0.4.2/", "tirith --version | grep -F 0.4.2", "go-tools/releases/download/" + StaticcheckVersion + "/", "staticcheck -version | grep -F '" + StaticcheckVersion + "'", "shellcheck/releases/download/v0.11.0/", "shellcheck --version | grep -F 'version: 0.11.0'"} {
		if !strings.Contains(string(goRecipe["Dockerfile"]), want) {
			t.Fatalf("missing %s", want)
		}
	}
	requirements := string(goRecipe["repokit-ddgs-requirements.txt"])
	for _, line := range strings.Split(requirements, "\n") {
		if strings.Contains(line, "==") && !strings.HasSuffix(strings.TrimSpace(line), "\\") {
			t.Fatalf("ddgs requirement without hashes: %s", line)
		}
	}
	if !strings.Contains(requirements, "ddgs==") || !strings.Contains(requirements, "--hash=sha256:") {
		t.Fatal("ddgs requirements not pinned by hash")
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

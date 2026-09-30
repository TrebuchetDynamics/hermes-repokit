package compose

import (
	"bytes"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentComposeKeepsCodingSeparateFromDaemon(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	req := development.Requirements{Go: true}
	opts := Options{HermesImage: qualification.FoundationImage, UID: 1000, GID: 1000, Development: &req}
	body, err := Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, word := range []string{"context: ./development-image", "working_dir: /workspace", "target: /opt/data"} {
		if !strings.Contains(s, word) {
			t.Fatal("missing", word)
		}
	}
	for _, word := range []string{"privileged:", "docker.sock", "DOCKER_HOST", "pi-coding", "codex"} {
		if strings.Contains(s, word) {
			t.Fatal("unexpected", word)
		}
	}
	opts.DockerTests = true
	body, err = Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	s = string(body)
	if !strings.Contains(s, "docker-test:") || !strings.Contains(s, "profiles: [docker-tests]") {
		t.Fatal(s)
	}
	if strings.Contains(s, "/var/run/docker.sock") || strings.Contains(s, "ports:") {
		t.Fatal("host daemon exposed")
	}
}
func TestDevelopmentSelectionRequiresExactGeneratedDocument(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	req := development.Requirements{Go: true}
	opts := Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid(), Development: &req}
	body, err := Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(id.Compose), 0700)
	os.WriteFile(id.Compose, body, 0600)
	got, ok := DevelopmentSelected(id)
	if !ok || got.Development == nil || !got.Development.Go {
		t.Fatal("not recovered")
	}
	os.WriteFile(id.Compose, append(body, []byte("# owner change\n")...), 0600)
	if _, ok := DevelopmentSelected(id); ok {
		t.Fatal("adopted owner file")
	}
}

func TestToolchainCacheVolumeKeepsGoCachesOutOfTheRepository(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	goReq := development.Requirements{Go: true}
	mount := "      - type: volume\n        source: toolchain-cache\n        target: /var/cache/repokit\n"
	for _, tests := range []bool{false, true} {
		body, err := Render(id, Options{HermesImage: qualification.FoundationImage, Development: &goReq, DockerTests: tests, UID: 1000, GID: 1000})
		if err != nil {
			t.Fatal(err)
		}
		s := string(body)
		if !strings.Contains(s, mount) || strings.Count(s, "\nvolumes:\n") != 1 || !strings.Contains(s, "\nvolumes:\n  toolchain-cache:\n") {
			t.Fatalf("docker-tests=%v: toolchain cache volume missing or duplicated:\n%s", tests, s)
		}
	}
	// Without Go there is no cache volume.
	noGo := development.Requirements{}
	for _, o := range []Options{{HermesImage: qualification.FoundationImage, Development: &noGo, UID: 1000, GID: 1000}} {
		body, err := Render(id, o)
		if err != nil || strings.Contains(string(body), "toolchain-cache") {
			t.Fatalf("unexpected cache volume: %v\n%s", err, body)
		}
	}
}

// v0.2.0 generated Go deployments without the toolchain-cache volume. The
// goldens are that release's exact output (rendered by the v0.2.0 source for
// this identity), so install can keep recognizing and upgrading them.
func TestBeforeToolchainCacheReproducesV020GoRender(t *testing.T) {
	id := target.Identity{Root: "/srv/atlas", Name: "atlas", Container: "hermes-atlas", Project: "repokit-0123456789abcdef01234567", Compose: "/srv/atlas/.hermes/compose.yaml", Launcher: "/srv/atlas/.hermes/bin/hermes-atlas"}
	const v020GoRecipe = "6341646efb2340d19e3453bb9abce1800b74f24137ad9b70fd71cb36cd2b8718"
	req := development.Requirements{Go: true}
	for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
		for _, tests := range []bool{false, true} {
			o := Options{HermesImage: qualification.FoundationImage, UID: 1000, GID: 1000, Development: &req, DockerTests: tests, SELinux: state}
			want, err := os.ReadFile(filepath.Join("testdata", fmt.Sprintf("v020-go-tests%v-%s.yaml", tests, state)))
			if err != nil {
				t.Fatal(err)
			}
			got, err := BeforeToolchainCache(id, o, v020GoRecipe)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("tests=%v selinux=%s: v0.2.0 render not reproduced (%v)\n%s", tests, state, err, got)
			}
			current, _ := Render(id, o)
			if bytes.Contains(got, []byte(ToolchainCacheTarget)) || !bytes.Contains(current, []byte(ToolchainCacheTarget)) {
				t.Fatal("only the current render mounts the toolchain cache")
			}
		}
	}
	plain := development.Requirements{}
	if _, err := BeforeToolchainCache(id, Options{HermesImage: qualification.FoundationImage, UID: 1000, GID: 1000, Development: &plain}, v020GoRecipe); err == nil {
		t.Fatal("pre-toolchain-cache render accepted for a non-Go deployment")
	}
}

func TestCurrentRenderHidesStateInsideWorkspace(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	req := development.Requirements{}
	o := Options{HermesImage: qualification.FoundationImage, Development: &req, UID: 1000, GID: 1000}
	mask := "      - type: tmpfs\n        target: /workspace/.hermes\n        tmpfs:\n          size: 4096\n          mode: 0555\n"
	current, err := Render(id, o)
	if err != nil || !strings.Contains(string(current), mask) {
		t.Fatalf("current render does not hide .hermes in /workspace: %v\n%s", err, current)
	}
	previous, err := BeforeStateMask(id, o, strings.Repeat("ab", 32))
	if err != nil || strings.Contains(string(previous), "/workspace/.hermes") {
		t.Fatalf("previous-release render gained the mask: %v", err)
	}
}

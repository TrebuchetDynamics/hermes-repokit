package cli

import (
	"bytes"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
)

func TestOlderGeneratedImageIsRecreatePendingNotRefused(t *testing.T) {
	a, r := foundationApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	current := developmentRuntimeFixture(r.id)
	o, _ := compose.DevelopmentSelected(r.id)
	image := development.ImageName(r.id.Container, *o.Development)
	for older, wantErr := range map[string]bool{
		"repokit/" + r.id.Container + ":0123456789abcdef01234567": false,
		r.id.Project + "-hermes-dev:0123456789abcdef01234567":     true,
		"foreign/" + r.id.Container + ":0123456789abcdef01234567": true,
	} {
		r.runtime = string(bytes.Replace([]byte(current), []byte(image), []byte(older), 1))
		ready, err := a.nativeRuntimeReady(r.id, r.context)
		if ready || (err != nil) != wantErr {
			t.Fatalf("%s: ready=%v err=%v", older, ready, err)
		}
	}
}

// A repository that gained a go.mod: its running container predates the Go
// selection (older generated image, only the two binds) and awaits
// recreation; the current image without the cache volume is not qualified.
func TestContainerFromBeforeGoWasSelectedAwaitsRecreation(t *testing.T) {
	a, r := foundationApp(t)
	if err := os.WriteFile(filepath.Join(a.Directory, "go.mod"), []byte("module example.test/demo\n\ngo 1.26.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	o, ok := compose.DevelopmentSelected(r.id)
	if !ok || !compose.ToolchainCacheMounted(o) {
		t.Fatal("current Go install does not mount the toolchain cache")
	}
	image := development.ImageName(r.id.Container, *o.Development)
	binds := []map[string]any{{"Type": "bind", "Source": r.id.Root, "Destination": "/workspace", "RW": true}, {"Type": "bind", "Source": filepath.Join(r.id.Root, ".hermes"), "Destination": "/opt/data", "RW": true}}
	runtime := func(image string) string {
		b, _ := json.Marshal(map[string]any{"status": "running", "service": "hermes", "unexpectedMounts": "", "image": image, "imageID": "sha256:" + strings.Repeat("d", 64), "project": r.id.Project, "workspace": r.id.Root, "home": filepath.Join(r.id.Root, ".hermes"), "mounts": binds})
		return string(b)
	}
	r.runtime = runtime("repokit/" + r.id.Container + ":0123456789abcdef01234567")
	if ready, err := a.nativeRuntimeReady(r.id, r.context); ready || err != nil {
		t.Fatalf("pre-volume container refused instead of awaiting recreation: ready=%v err=%v", ready, err)
	}
	r.runtime = runtime(image)
	if ready, err := a.nativeRuntimeReady(r.id, r.context); ready || err == nil {
		t.Fatalf("current image without the toolchain volume accepted: ready=%v err=%v", ready, err)
	}
}

// The previous release left .hermes visible inside /workspace; install
// upgrades that exact layout in place and backs it up.
func TestInstallUpgradesThePreMaskLayout(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	state := filepath.Join(r.id.Root, ".hermes")
	current, _ := os.ReadFile(r.id.Compose)
	// A self-certified older recipe on disk, as a previous release left it.
	recipeDir := filepath.Join(state, "development-image")
	if err := os.RemoveAll(recipeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(recipeDir, 0700); err != nil {
		t.Fatal(err)
	}
	old := map[string][]byte{}
	for _, name := range []string{"Dockerfile-base", ".dockerignore", "repokit-docker-test", "repokit-openviking", "openviking-run", "openviking-finish", "patch-openviking-entrypoint.py"} {
		data, err := os.ReadFile(filepath.Join("../development/testdata/e0246ef", name))
		if err != nil {
			t.Fatal(err)
		}
		old[strings.TrimSuffix(name, "-base")] = data
	}
	fingerprint, ok := development.GeneratedRecipe(old)
	if !ok {
		t.Fatal("fixture recipe does not self-certify")
	}
	for name, data := range old {
		if err := os.WriteFile(filepath.Join(recipeDir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	req := development.Requirements{}
	previous, err := compose.BeforeStateMask(r.id, compose.Options{HermesImage: qualification.FoundationImage, Development: &req, UID: os.Getuid(), GID: os.Getgid(), SELinux: a.selinuxState()}, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.id.Compose, previous, 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("pre-mask deployment not upgraded: out=%s diag=%s", out, diag)
	}
	if got, _ := os.ReadFile(r.id.Compose); !bytes.Equal(got, current) {
		t.Fatalf("Compose not upgraded to the masked render:\n%s", got)
	}
	if saved, err := os.ReadFile(filepath.Join(state, "compose.before-state-mask-"+fingerprint[:12]+".yaml")); err != nil || !bytes.Equal(saved, previous) {
		t.Fatalf("pre-mask Compose not backed up: %v", err)
	}
}

// A deployment installed without a toolchain whose repository then gains a
// go.mod or Cargo.toml (at the root or in a nested project) upgrades in place:
// the older Compose was rendered for the older recipe, without the
// toolchain-cache volume.
func TestInstallUpgradesWhenARepositoryGainsAToolchain(t *testing.T) {
	for _, path := range []string{"go.mod", "svc/go.mod", "Cargo.toml", "crate/Cargo.toml"} {
		t.Run(path, func(t *testing.T) {
			a, r := foundationApp(t)
			if c, _, d := invoke(t, a, "install"); c != 0 {
				t.Fatal(d)
			}
			if o, _ := compose.DevelopmentSelected(r.id); o.Development.Go || o.Development.Rust {
				t.Fatal("toolchain selected before any manifest")
			}
			rust := strings.HasSuffix(path, "Cargo.toml")
			content := "module example.test/demo\n\ngo 1.26.0\n"
			if rust {
				content = "[package]\nname = \"demo\"\n"
			}
			file := filepath.Join(a.Directory, path)
			os.MkdirAll(filepath.Dir(file), 0700)
			if err := os.WriteFile(file, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if c, out, d := invoke(t, a, "install"); c != 0 {
				t.Fatalf("gaining Go refused: %s %s", out, d)
			}
			o, ok := compose.DevelopmentSelected(r.id)
			if !ok || o.Development.Go == rust || o.Development.Rust != rust || !compose.ToolchainCacheMounted(o) {
				t.Fatal("upgraded Compose does not select the toolchain with its cache")
			}
			if backups, _ := filepath.Glob(filepath.Join(r.id.Root, ".hermes", "compose.before-recipe-*.yaml")); len(backups) != 1 {
				t.Fatalf("prior Compose not backed up: %v", backups)
			}
		})
	}
}

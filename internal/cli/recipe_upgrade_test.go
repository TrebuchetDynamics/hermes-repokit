package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
)

// Read the frozen e0246ef bytes directly, independently of the migration helpers.
func priorRecipeFixture(t *testing.T, goTool bool) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	for _, name := range []string{"Dockerfile", ".dockerignore", "repokit-docker-test", "repokit-openviking", "openviking-run", "openviking-finish", "patch-openviking-entrypoint.py"} {
		source := name
		if name == "Dockerfile" {
			source = "Dockerfile-base"
			if goTool {
				source = "Dockerfile-go"
			}
		}
		data, err := os.ReadFile(filepath.Join("../development/testdata/e0246ef", source))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = data
	}
	return files
}

func TestPriorDevelopmentRecipeUpgrade(t *testing.T) {
	for _, goTool := range []bool{false, true} {
		for _, dockerTests := range []bool{false, true} {
			{
				for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
					t.Run(fmt.Sprintf("go=%v/docker=%v/selinux=%v", goTool, dockerTests, state), func(t *testing.T) {
						a, r := foundationApp(t)
						a.HostSELinux = state
						if goTool {
							if err := os.WriteFile(filepath.Join(a.Directory, "go.mod"), []byte("module example.test/demo\n\ngo 1.26.0\n"), 0600); err != nil {
								t.Fatal(err)
							}
						}
						if c, _, d := invoke(t, a, "install"); c != 0 {
							t.Fatal(d)
						}
						req := development.Requirements{Go: goTool}
						old, err := compose.Render(r.id, compose.Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: dockerTests, UID: os.Getuid(), GID: os.Getgid(), SELinux: state})
						if err != nil {
							t.Fatal(err)
						}
						hash := "8ec197a291d0200317e88b6d7654fd0cc91ca9dec342ad0b03ba50d070a591d9"
						if goTool {
							hash = "617671ef605ee7a176cf809ae663eec87419d5f094897f9fabcf0120422e3335"
						}
						old = bytes.ReplaceAll(old, []byte(development.ImageName(r.id.Container, req)), []byte(r.id.Project+"-hermes-dev:"+hash[:24]))
						write := func(name string, data []byte) {
							t.Helper()
							if err := os.WriteFile(filepath.Join(a.Directory, ".hermes", name), data, 0600); err != nil {
								t.Fatal(err)
							}
						}
						write("compose.yaml", old)
						for name, data := range priorRecipeFixture(t, goTool) {
							write("development-image/"+name, data)
						}
						native := []byte("# owner native configuration\n")
						write("config.yaml", native)
						if c, _, d := invoke(t, a, "install"); c != 0 {
							t.Fatal(d)
						}
						selected, ok := compose.DevelopmentInstallSelected(r.id)
						if !ok || selected.DockerTests != dockerTests || selected.Development.Go != goTool {
							t.Fatal("selection lost", selected)
						}
						upgraded, _ := os.ReadFile(filepath.Join(a.Directory, ".hermes/compose.yaml"))
						if bytes.Contains(bytes.ToLower(upgraded), []byte("openviking")) {
							t.Fatal("upgrade kept a RepoKit-owned OpenViking")
						}
						current, _ := development.Recipe(req)
						for name, want := range current {
							got, _ := os.ReadFile(filepath.Join(a.Directory, ".hermes/development-image", name))
							if !bytes.Equal(got, want) {
								t.Fatalf("recipe %s not upgraded", name)
							}
						}
						for name, want := range map[string][]byte{"compose.before-path.yaml": old, "config.yaml": native} {
							got, _ := os.ReadFile(filepath.Join(a.Directory, ".hermes", name))
							if !bytes.Equal(got, want) {
								t.Fatalf("lost %s", name)
							}
						}
						if c, _, d := invoke(t, a, "install"); c != 0 {
							t.Fatal("rerun", d)
						}
					})
				}
			}
		}
	}
}

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
		r.id.Project + "-hermes-dev:0123456789abcdef01234567":     false,
		"foreign/" + r.id.Container + ":0123456789abcdef01234567": true,
	} {
		r.runtime = string(bytes.Replace([]byte(current), []byte(image), []byte(older), 1))
		ready, err := a.nativeRuntimeReady(r.id, r.context)
		if ready || (err != nil) != wantErr {
			t.Fatalf("%s: ready=%v err=%v", older, ready, err)
		}
	}
}

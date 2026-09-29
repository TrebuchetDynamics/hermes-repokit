package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestPriorRecipeSelectionIsInstallOnlyAndExact(t *testing.T) {
	id, err := target.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(id.Root, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	req := development.Requirements{Go: true}
	recipe := filepath.Join(id.Root, ".hermes", "development-image")
	if err := os.Mkdir(recipe, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Dockerfile-go", ".dockerignore", "repokit-docker-test", "repokit-openviking", "openviking-run", "openviking-finish", "patch-openviking-entrypoint.py"} {
		data, err := os.ReadFile(filepath.Join("../development/testdata/e0246ef", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(recipe, strings.TrimSuffix(name, "-go")), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	const fingerprint = "617671ef605ee7a176cf809ae663eec87419d5f094897f9fabcf0120422e3335"
	for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
		old, err := LegacyDevelopment(id, Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: true, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}, fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(id.Root, ".hermes/compose.yaml")
		if err := os.WriteFile(path, old, 0600); err != nil {
			t.Fatal(err)
		}
		if selected, ok := DevelopmentInstallSelected(id); !ok || !selected.DockerTests || !selected.Development.Go {
			t.Fatal("lost exact prior selection")
		}
		if _, ok := DevelopmentSelected(id); ok {
			t.Fatal("old recipe accepted by current runtime selector")
		}
		if err := os.WriteFile(path, append(old, []byte("# owner change\n")...), 0600); err != nil {
			t.Fatal(err)
		}
		if _, ok := DevelopmentInstallSelected(id); ok {
			t.Fatal("owner Compose drift selected")
		}
		if err := os.WriteFile(path, old, 0600); err != nil {
			t.Fatal(err)
		}
		dockerfile := filepath.Join(recipe, "Dockerfile")
		original, _ := os.ReadFile(dockerfile)
		if err := os.WriteFile(dockerfile, append(append([]byte{}, original...), "# owner\n"...), 0600); err != nil {
			t.Fatal(err)
		}
		if _, ok := DevelopmentInstallSelected(id); ok {
			t.Fatal("owner-edited recipe recognized as generated")
		}
		if err := os.WriteFile(dockerfile, original, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

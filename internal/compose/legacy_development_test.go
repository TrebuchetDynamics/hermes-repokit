package compose

import (
	"os"
	"path/filepath"
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
	for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
		old, err := LegacyDevelopment(id, Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: true, UID: os.Getuid(), GID: os.Getgid(), SELinux: state})
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
	}
}

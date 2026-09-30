package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func namingApp(t *testing.T) (App, *foundationRunner) {
	t.Helper()
	a, r := foundationApp(t)
	path := filepath.Join(filepath.Dir(a.Directory), "hermes-repokit")
	if err := os.Rename(a.Directory, path); err != nil {
		t.Fatal(err)
	}
	a.Directory = path
	var err error
	r.id, err = target.Resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	return a, r
}

func TestInstallReadableRuntimeNames(t *testing.T) {
	a, r := namingApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	data, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("container_name: \"hermes-repokit\"")) || !bytes.Contains(data, []byte("image: \"repokit/hermes-repokit:")) {
		t.Fatalf("runtime names are not readable:\n%s", data)
	}
	link := filepath.Join(os.Getenv("HOME"), ".local/bin/hermes-repokit")
	if got, err := os.Readlink(link); err != nil || got != filepath.Join(a.Directory, ".hermes/bin/hermes-repokit") {
		t.Fatalf("host command: %q %v", got, err)
	}
}

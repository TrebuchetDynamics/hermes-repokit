package compose

import (
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayaUsesPinnedImageAndHermesLoopback(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	opts := Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid(), LayaImage: "sha256:" + strings.Repeat("b", 64)}
	got, err := Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"  laya:\n", "network_mode: service:hermes", "pull_policy: never", "read_only: true", "restart: true"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("missing %s", want)
		}
	}
	os.Mkdir(filepath.Join(id.Root, ".hermes"), 0700)
	os.WriteFile(id.Compose, got, 0600)
	if found := SelectedLaya(id); found != opts.LayaImage {
		t.Fatalf("standalone selection not recovered: %s", found)
	}
	os.WriteFile(id.Compose, append(got, []byte("# owner edit\n")...), 0600)
	if SelectedLaya(id) != "" {
		t.Fatal("adopted edited Compose")
	}
	opts.LayaImage = "owner:latest"
	if _, err := Render(id, opts); err == nil {
		t.Fatal("mutable Laya image accepted")
	}
}

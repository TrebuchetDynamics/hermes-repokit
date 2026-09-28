package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestDefaultLayaBuildIsStandaloneAndPersistent(t *testing.T) {
	id, err := target.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := Render(id, Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, LayaBuild: true, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"  hermes:\n", "  openviking:\n", "  laya:\n", "context: ./laya-image", "platform: linux/amd64", "network_mode: service:hermes", "source: \"./laya\"", "target: /cache", "HF_HOME: /cache/huggingface", "TORCHINDUCTOR_CACHE_DIR: /cache/torchinductor"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing standalone build contract %q", want)
		}
	}
	for _, forbidden := range []string{"ports:", "docker.sock", "repokit install", "pull_policy: never"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("unexpected build contract %q", forbidden)
		}
	}
	if err := os.Mkdir(filepath.Join(id.Root, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(id.Compose, data, 0600); err != nil {
		t.Fatal(err)
	}
	if !DefaultLayaSelected(id) {
		t.Fatal("default build stack not recognized")
	}
	if err := os.WriteFile(id.Compose, append(data, []byte("# owner edit\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if DefaultLayaSelected(id) {
		t.Fatal("owner edited Compose adopted")
	}
}

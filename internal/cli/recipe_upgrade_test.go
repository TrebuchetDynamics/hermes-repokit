package cli

import (
	"bytes"
	"encoding/json"
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

package compose

import (
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"strings"
	"testing"
)

func TestRenderIsPinnedIsolatedAndStandalone(t *testing.T) {
	id := target.Identity{Project: "repokit-abc", Container: "hermes-my-project"}
	out, err := Render(id, Options{HermesImage: "org/hermes@sha256:" + strings.Repeat("a", 64), OpenVikingImage: "org/openviking@sha256:" + strings.Repeat("b", 64), UID: 1000, GID: 1000})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"name: \"repokit-abc\"", "container_name: \"hermes-my-project\"", "source: \"..\"", "target: /workspace", "target: /opt/data", "HERMES_HOME: /opt/data", "HERMES_WRITE_SAFE_ROOT: /opt/data:/workspace", "target: /app/.openviking", "OPENVIKING_WITH_BOT: \"0\"", "create_host_path: false"} {
		if !strings.Contains(string(out), part) {
			t.Errorf("missing %q\n%s", part, out)
		}
	}
	for _, part := range []string{"ports:", "latest", "privileged:", "docker.sock", "user:"} {
		if strings.Contains(string(out), part) {
			t.Errorf("unexpected %q", part)
		}
	}
}
func TestRenderRejectsUnpinnedImagesAndBadIdentity(t *testing.T) {
	id := target.Identity{Project: "repokit-abc", Container: "hermes-good"}
	for _, image := range []string{"", "x:latest", "x:v1"} {
		if _, e := Render(id, Options{HermesImage: image, UID: 1000, GID: 1000}); e == nil {
			t.Fatal("accepted tag")
		}
	}
	if _, e := Render(target.Identity{}, Options{HermesImage: "x@sha256:" + strings.Repeat("a", 64)}); e == nil {
		t.Fatal("accepted incomplete identity")
	}
}

package compose

import (
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"strings"
	"testing"
)

func TestOpenVikingRunsAsHostOwnerWithPersistentConfiguration(t *testing.T) {
	id := target.Identity{Project: "repokit-abc", Container: "hermes-example"}
	out, err := Render(id, Options{HermesImage: "h@sha256:" + strings.Repeat("a", 64), OpenVikingImage: "ov@sha256:" + strings.Repeat("b", 64), UID: 1234, GID: 2345})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(out), "  openviking:\n", 2)
	if len(parts) != 2 {
		t.Fatal("memory service missing")
	}
	for _, want := range []string{"user: \"1234:2345\"", "HOME: /app/.openviking", "OPENVIKING_CONFIG_FILE: /app/.openviking/ov.conf", "OPENVIKING_WITH_BOT: \"0\"", "source: \"./openviking\"", "target: /app/.openviking", "healthcheck:", "openviking-entrypoint", "--healthcheck"} {
		if !strings.Contains(parts[1], want) {
			t.Errorf("missing service contract %q", want)
		}
	}
	if strings.Contains(parts[0], "    user:") {
		t.Fatal("Hermes privilege-dropping entrypoint was overridden")
	}
	for _, forbidden := range []string{"ports:", "api_key:", "OPENVIKING_CONF_CONTENT", "docker.sock"} {
		if strings.Contains(string(out), forbidden) {
			t.Errorf("unsafe generated service: %s", forbidden)
		}
	}
}

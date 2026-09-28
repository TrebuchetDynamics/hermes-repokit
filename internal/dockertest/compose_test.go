package dockertest

import (
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"strconv"
	"strings"
	"testing"
)

func TestOptInDaemonOnlySharesPrivateSocketAndScratch(t *testing.T) {
	data, err := EmitServices(1001, 1002)
	if err != nil {
		t.Fatal(err)
	}
	if !qualification.ImmutableImage(Image) {
		t.Fatal("daemon image is not pinned")
	}
	for _, want := range []string{Image, "profiles: [docker-tests]", "privileged: true", "unix:///docker-test/run/docker.sock", "--group=1002", "chown 1001:1002 /docker-tests", "dockerd-entrypoint.sh dockerd", "io.repokit.docker-test=1"} {
		if !strings.Contains(data, want) {
			t.Fatal("missing", want)
		}
	}
	if !strings.Contains(data, "      - "+strconv.Quote(DaemonCommand(1001, 1002))) {
		t.Fatal("daemon command differs from shared identity contract")
	}
	for _, forbidden := range []string{"tcp://", "ports:", "/workspace", "/opt/data", "/var/run/docker.sock", "type: bind", "network_mode: host"} {
		if strings.Contains(data, forbidden) {
			t.Fatal("unexpected daemon access", forbidden)
		}
	}
	if strings.Count(Mounts(), "type: volume") != 2 || !strings.Contains(Mounts(), "source: docker-test-run") || !strings.Contains(Mounts(), "source: docker-test-work") {
		t.Fatal("unexpected Hermes mounts")
	}
	for _, name := range []string{"docker-test-run:", "docker-test-work:", "docker-test-data:"} {
		if !strings.Contains(Resources(), name) {
			t.Fatal("missing resource", name)
		}
	}
	if _, err = EmitServices(0, 1000); err == nil {
		t.Fatal("root owner allowed")
	}
	if _, err = EmitServices(1000, -1); err == nil {
		t.Fatal("bad group allowed")
	}
}

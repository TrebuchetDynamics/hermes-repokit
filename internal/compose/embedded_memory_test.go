package compose

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestDevelopmentEmbedsMemoryInHermesContainer(t *testing.T) {
	id := target.Identity{Project: "repokit-example", Container: "hermes-example"}
	req := development.Requirements{Go: true}
	b, err := Render(id, Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &req, UID: 1000, GID: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "\n  openviking:") {
		t.Fatal("OpenViking must live inside the Hermes container")
	}
	for _, want := range []string{"target: /opt/data", "REPOKIT_OPENVIKING: \"1\""} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, forbidden := range []string{"/app/.openviking", "source: \"./openviking\""} {
		if strings.Contains(string(b), forbidden) {
			t.Fatalf("OpenViking state must live under /opt/data, found %s", forbidden)
		}
	}
}

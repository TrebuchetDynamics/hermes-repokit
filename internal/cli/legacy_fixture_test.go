package cli

import (
	"os"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
)

func legacyMemoryFixture(t *testing.T) (App, []byte) {
	t.Helper()
	a, r := foundationApp(t)
	old, err := compose.Render(r.id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	launch, err := launcher.Render(r.id, r.context)
	if err != nil {
		t.Fatal(err)
	}
	_, err = install.Publish(r.id, map[string]install.Artifact{
		"compose.yaml":          {Data: old, Mode: 0600},
		"config.yaml":           {Data: []byte("owner-config\n"), Mode: 0600},
		"bin/" + r.id.Container: {Data: launch, Mode: 0700},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return a, old
}

package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
)

func TestEmbeddedMemoryObservesHermesMountAndLoopbackHealth(t *testing.T) {
	id, base := integrationFixture(t)
	req := development.Requirements{}
	data, _ := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &req, UID: os.Getuid(), GID: os.Getgid()})
	os.WriteFile(id.Compose, data, 0600)
	os.Mkdir(filepath.Join(id.Root, ".hermes/development-image"), 0700)
	os.Mkdir(filepath.Join(id.Root, ".hermes/openviking"), 0700)
	recipe, _ := development.Recipe(req)
	for name, content := range recipe {
		os.WriteFile(filepath.Join(id.Root, ".hermes/development-image", name), content, 0600)
	}
	imageID := "sha256:" + strings.Repeat("d", 64)
	r := &devRunner{integrationRunner: base, derived: fmt.Sprintf(`{"id":%q,"os":"linux","arch":"amd64","recipe":%q,"base":%q,"layers":["sha256:%s","sha256:%s"]}`, imageID, development.Fingerprint(req), qualification.FoundationImage, strings.Repeat("e", 64), strings.Repeat("f", 64))}
	var state map[string]any
	json.Unmarshal([]byte(r.hermes), &state)
	state["image"], state["imageID"] = development.ImageName(id.Project, req), imageID
	mounts := []RuntimeMount{{Type: "bind", Source: id.Root, Destination: "/workspace", RW: true}, {Type: "bind", Source: id.Root + "/.hermes", Destination: "/opt/data", RW: true}}
	update := func() { state["mounts"] = mounts; b, _ := json.Marshal(state); r.hermes = string(b) }
	update()
	check := func(container, runtime Status) {
		t.Helper()
		p := OpenViking(context.Background(), id, r)
		if p[1].Status != container || p[2].Status != runtime {
			t.Fatal(p)
		}
	}
	check(Healthy, PendingSetup)
	// Only metadata may be read: this deliberately is not parseable YAML/JSON.
	os.WriteFile(filepath.Join(id.Root, ".hermes/openviking/ov.conf"), []byte("private opaque bytes"), 0600)
	r.health = "healthy\n"
	check(Healthy, Healthy)
	r.health = "degraded\n"
	check(Healthy, Degraded)
	r.health = "healthy\n"
	mounts[1].Source = "/another-repo/.hermes"
	update()
	check(Degraded, Degraded)
	for _, call := range r.calls {
		command := strings.Join(call, " ")
		if strings.Contains(command, "service=openviking") || strings.Contains(command, "/ready") || strings.Contains(command, "ov.conf") {
			t.Fatalf("unsafe/legacy observation: %s", command)
		}
	}
}

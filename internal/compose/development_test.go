package compose

import (
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentComposeKeepsCodingSeparateFromDaemon(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	req := development.Requirements{Go: true}
	opts := Options{HermesImage: qualification.FoundationImage, UID: 1000, GID: 1000, Development: &req}
	body, err := Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, word := range []string{"context: ./development-image", "working_dir: /workspace", "target: /opt/data"} {
		if !strings.Contains(s, word) {
			t.Fatal("missing", word)
		}
	}
	for _, word := range []string{"privileged:", "docker.sock", "DOCKER_HOST", "pi-coding", "codex"} {
		if strings.Contains(s, word) {
			t.Fatal("unexpected", word)
		}
	}
	opts.DockerTests = true
	body, err = Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	s = string(body)
	if !strings.Contains(s, "docker-test:") || !strings.Contains(s, "profiles: [docker-tests]") {
		t.Fatal(s)
	}
	if strings.Contains(s, "/var/run/docker.sock") || strings.Contains(s, "ports:") {
		t.Fatal("host daemon exposed")
	}
}
func TestDevelopmentSelectionRequiresExactGeneratedDocument(t *testing.T) {
	id, _ := target.Resolve(t.TempDir())
	req := development.Requirements{Go: true}
	opts := Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid(), Development: &req}
	body, err := Render(id, opts)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(id.Compose), 0700)
	os.WriteFile(id.Compose, body, 0600)
	got, ok := DevelopmentSelected(id)
	if !ok || got.Development == nil || !got.Development.Go {
		t.Fatal("not recovered")
	}
	os.WriteFile(id.Compose, append(body, []byte("# owner change\n")...), 0600)
	if _, ok := DevelopmentSelected(id); ok {
		t.Fatal("adopted owner file")
	}
}

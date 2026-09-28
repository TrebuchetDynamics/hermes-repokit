package verify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/dockertest"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

type dockerAcceptanceRunner struct {
	*devRunner
	observedCalls [][]string
}

func (r *dockerAcceptanceRunner) Run(ctx context.Context, program string, args ...string) process.Result {
	r.observedCalls = append(r.observedCalls, append([]string{program}, args...))
	return r.devRunner.Run(ctx, program, args...)
}

func encodeAcceptanceState(t *testing.T, state map[string]any) string {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func isolatedDockerAcceptanceFixture(t *testing.T) (target.Identity, *dockerAcceptanceRunner, map[string]any, map[string]any) {
	t.Helper()
	id, base := integrationFixture(t)
	req := development.Requirements{Go: true}
	data, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: true, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(id.Compose, data, 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(id.Root, ".hermes/development-image")
	if err = os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	files, err := development.Recipe(req)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err = os.WriteFile(filepath.Join(dir, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	imageID := "sha256:" + strings.Repeat("d", 64)
	hermes := map[string]any{
		"id": strings.Repeat("a", 64), "status": "running", "service": "hermes", "project": id.Project,
		"image": development.ImageName(id.Project, req), "imageID": imageID,
		"workspace": id.Root, "home": filepath.Join(id.Root, ".hermes"), "unexpectedMounts": "xx",
		"mounts": []RuntimeMount{
			{Type: "bind", Source: id.Root, Destination: "/workspace", RW: true},
			{Type: "bind", Source: filepath.Join(id.Root, ".hermes"), Destination: "/opt/data", RW: true},
			{Type: "volume", Name: id.Project + "_docker-test-run", Destination: "/docker-test/run", RW: false},
			{Type: "volume", Name: id.Project + "_docker-test-work", Destination: "/docker-tests", RW: true},
		},
	}
	daemon := map[string]any{
		"status": "running", "image": dockertest.Image, "project": id.Project, "service": "docker-test",
		"health": "healthy", "privileged": true, "ports": map[string]any{},
		"networks":    map[string]any{id.Project + "_docker-test": map[string]any{}},
		"networkMode": id.Project + "_docker-test", "entrypoint": []string{"sh", "-ec"},
		"command": []string{dockertest.DaemonCommand(os.Getuid(), os.Getgid())},
		"mounts": []RuntimeMount{
			{Type: "volume", Name: id.Project + "_docker-test-run", Destination: "/docker-test/run", RW: true},
			{Type: "volume", Name: id.Project + "_docker-test-work", Destination: "/docker-tests", RW: true},
			{Type: "volume", Name: id.Project + "_docker-test-data", Destination: "/var/lib/docker", RW: true},
		},
	}
	built := map[string]any{"id": imageID, "os": "linux", "arch": "amd64", "recipe": development.Fingerprint(req), "base": qualification.FoundationImage,
		"layers": []string{"sha256:" + strings.Repeat("e", 64), "sha256:" + strings.Repeat("f", 64)}}
	r := &dockerAcceptanceRunner{devRunner: &devRunner{integrationRunner: base, derived: encodeAcceptanceState(t, built)}}
	r.hermes, r.service = encodeAcceptanceState(t, hermes), encodeAcceptanceState(t, daemon)
	return id, r, hermes, daemon
}

func TestDockerAcceptanceRequiresExactIsolatedRuntime(t *testing.T) {
	id, r, _, _ := isolatedDockerAcceptanceFixture(t)
	selection, ok := compose.DevelopmentSelected(id)
	if !ok || !selection.DockerTests || !selection.Development.Go {
		t.Fatal("fixture did not select the generated development/test stack")
	}
	got := dockerAcceptance(context.Background(), id, r)
	if got.Status != Healthy || !strings.Contains(got.Detail, "real Docker test suite remains separate") {
		t.Fatalf("%+v", got)
	}
	if len(r.observedCalls) == 0 {
		t.Fatal("runtime was not inspected")
	}
	for _, call := range r.observedCalls {
		if call[0] != "docker" {
			t.Fatalf("unexpected executable: %v", call)
		}
		joined := strings.Join(call, " ")
		if !strings.Contains(joined, " image inspect ") && !strings.Contains(joined, " container inspect ") && !strings.Contains(joined, " container ls ") {
			t.Fatalf("non-observational operation: %v", call)
		}
	}
}

func TestDockerAcceptanceRejectsDaemonIsolationDrift(t *testing.T) {
	for _, variant := range []string{"host_socket", "checkout_bind", "public_port", "extra_network", "host_network", "tcp_command", "foreign_image", "foreign_volume", "unhealthy"} {
		t.Run(variant, func(t *testing.T) {
			id, r, _, daemon := isolatedDockerAcceptanceFixture(t)
			mounts := daemon["mounts"].([]RuntimeMount)
			switch variant {
			case "host_socket":
				mounts[0] = RuntimeMount{Type: "bind", Source: "/var/run/docker.sock", Destination: "/docker-test/run", RW: true}
			case "checkout_bind":
				mounts[1] = RuntimeMount{Type: "bind", Source: id.Root, Destination: "/docker-tests", RW: true}
			case "public_port":
				daemon["ports"] = map[string]any{"2375/tcp": []map[string]string{{"HostIp": "0.0.0.0", "HostPort": "2375"}}}
			case "extra_network":
				daemon["networks"].(map[string]any)[id.Project+"_default"] = map[string]any{}
			case "host_network":
				daemon["networkMode"] = "host"
			case "tcp_command":
				daemon["command"] = []string{"dockerd --host=tcp://0.0.0.0:2375"}
			case "foreign_image":
				daemon["image"] = "docker:latest"
			case "foreign_volume":
				mounts[2].Name = "another-project_docker-test-data"
			case "unhealthy":
				daemon["health"] = "unhealthy"
			}
			daemon["mounts"] = mounts
			r.service = encodeAcceptanceState(t, daemon)
			if got := dockerAcceptance(context.Background(), id, r); got.Status != Degraded {
				t.Fatalf("isolation drift accepted: %+v", got)
			}
		})
	}
}

func TestDockerAcceptanceRefusesUnsafeHermesMountsBeforeDaemonProbe(t *testing.T) {
	for _, variant := range []string{"writable_socket", "host_socket", "extra_bind", "missing_scratch", "foreign_image"} {
		t.Run(variant, func(t *testing.T) {
			id, r, hermes, _ := isolatedDockerAcceptanceFixture(t)
			mounts := hermes["mounts"].([]RuntimeMount)
			switch variant {
			case "writable_socket":
				mounts[2].RW = true
			case "host_socket":
				mounts[2] = RuntimeMount{Type: "bind", Source: "/var/run/docker.sock", Destination: "/docker-test/run", RW: false}
			case "extra_bind":
				mounts = append(mounts, RuntimeMount{Type: "bind", Source: "/home", Destination: "/host-home", RW: true})
			case "missing_scratch":
				mounts = mounts[:3]
			case "foreign_image":
				hermes["imageID"] = "sha256:" + strings.Repeat("9", 64)
			}
			hermes["mounts"] = mounts
			r.hermes = encodeAcceptanceState(t, hermes)
			if got := dockerAcceptance(context.Background(), id, r); got.Status != Unknown {
				t.Fatalf("unsafe Hermes runtime accepted: %+v", got)
			}
			for _, call := range r.observedCalls {
				if strings.Contains(strings.Join(call, " "), " container ls ") {
					t.Fatal("continued to daemon inspection after unqualified Hermes identity")
				}
			}
		})
	}
}

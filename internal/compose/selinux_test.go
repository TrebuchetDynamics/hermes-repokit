package compose

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func renderForSELinux(t *testing.T, state selinux.State) string {
	t.Helper()
	id := target.Identity{Project: "repokit-abc", Container: "hermes-my-project"}
	req := development.Requirements{Go: true}
	b, err := Render(id, Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &req, UID: 1000, GID: 1000, SELinux: state})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The private one-container topology must request Z relabeling for both
// repository-owned bind mounts on SELinux-enabled hosts.
func TestEnabledSELinuxRelabelsRepositoryMountsPrivately(t *testing.T) {
	out := renderForSELinux(t, selinux.Enforcing)
	if got := strings.Count(out, "selinux: Z"); got != 2 {
		t.Fatalf("expected 2 private relabel options, got %d\n%s", got, out)
	}
	for _, forbidden := range []string{"selinux: z", "selinux: Z ", "label=disable"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("unexpected relabel option %q", forbidden)
		}
	}
	for _, mount := range []string{"target: /workspace", "target: /opt/data"} {
		idx := strings.Index(out, mount)
		if idx < 0 {
			t.Fatalf("missing mount %s", mount)
		}
		end := idx + len(mount) + 80
		if end > len(out) {
			end = len(out)
		}
		if !strings.Contains(out[idx:end], "selinux: Z") {
			t.Fatalf("mount %s lacks private relabel\n%s", mount, out)
		}
	}
}

func TestPermissiveSELinuxAlsoRelabels(t *testing.T) {
	if !strings.Contains(renderForSELinux(t, selinux.Permissive), "selinux: Z") {
		t.Fatal("permissive SELinux must still relabel")
	}
}

func TestDisabledOrAbsentSELinuxLeavesMountsUnchanged(t *testing.T) {
	for _, state := range []selinux.State{selinux.Disabled, selinux.Unavailable, ""} {
		out := renderForSELinux(t, state)
		if strings.Contains(out, "selinux:") {
			t.Fatalf("state %q emitted SELinux option\n%s", state, out)
		}
		if !strings.Contains(out, "target: /workspace") || !strings.Contains(out, "target: /opt/data") {
			t.Fatalf("state %q removed normal mounts", state)
		}
	}
}

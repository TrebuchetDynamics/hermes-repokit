package verify

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestHostSecurityReportsSELinuxState(t *testing.T) {
	old := detectSELinux
	t.Cleanup(func() { detectSELinux = old })
	id := target.Identity{Root: t.TempDir(), Project: "repokit-abc", Container: "hermes-x"}

	for _, tc := range []struct {
		state  selinux.State
		detail string
	}{
		{selinux.Enforcing, "private Z relabeling"},
		{selinux.Permissive, "private Z relabeling"},
		{selinux.Disabled, "need no relabel"},
		{selinux.Unavailable, "need no relabel"},
	} {
		detectSELinux = func() selinux.State { return tc.state }
		probes := HostSecurity(context.Background(), id, nil)
		if len(probes) != 2 {
			t.Fatalf("%s: probes = %+v", tc.state, probes)
		}
		if p := probes[0]; p.Component != "selinux" || p.Status != Healthy || !strings.Contains(p.Detail, string(tc.state)) || !strings.Contains(p.Detail, tc.detail) {
			t.Fatalf("%s: selinux probe = %+v", tc.state, p)
		}
		// No running container in this fixture: access is unknown, never a false healthy.
		if p := probes[1]; p.Component != "access" || p.Status != Unknown {
			t.Fatalf("%s: access probe = %+v", tc.state, p)
		}
	}
}

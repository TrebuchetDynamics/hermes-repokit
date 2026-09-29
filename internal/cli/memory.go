package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

func (a App) setupMemory(id target.Identity, dockerContext string, stdout, stderr io.Writer) int {
	if dockerContext == "" || len(a.gitIssues(context.Background(), id)) > 0 {
		fmt.Fprintln(stderr, "memory setup refused: deployment routing or private state protection is unverified")
		return 1
	}
	for _, p := range verify.Inspect(context.Background(), id, a.Runner) {
		if (p.Component == "compose" || p.Component == "hermes") && p.Status != verify.Healthy {
			fmt.Fprintln(stderr, "memory setup refused: generated Compose and running Hermes runtime are required")
			return 1
		}
	}
	for _, p := range verify.Profiles(id) {
		if p.Status != verify.Healthy {
			fmt.Fprintln(stderr, "Complete native default setup and reconcile the seven-role scaffold before setup --memory.")
			return 1
		}
	}
	for _, p := range verify.OpenViking(context.Background(), id, a.Runner) {
		if p.Component == "openviking-container" && p.Status != verify.Healthy {
			fmt.Fprintln(stderr, "memory setup refused: pinned OpenViking runtime or persistent mount is not verified")
			return 1
		}
		if p.Component == "openviking-config" && (p.Status == verify.Degraded || p.Status == verify.Unknown) {
			fmt.Fprintln(stderr, "memory setup refused: inspect native ov.conf ownership and private regular-file permissions")
			return 1
		}
	}
	// Uses the established pinned-runtime/mount check and container-held lock.
	if code := a.initialize(id, dockerContext, false, stdout, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Shared OpenViking identity: account repokit, repository user %s. Use a normal user key, not a root/admin key.\n", id.Project)
	return native.SetupMemory(id, dockerContext, a.Stdin, stdout, stderr)
}

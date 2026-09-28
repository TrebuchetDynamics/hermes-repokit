package verify

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// detectSELinux is a test seam; production reads the live host state.
var detectSELinux = selinux.Detect

// mountAccessScript proves read/execute on the repository and native-state
// mounts and write access where the runtime expects it. It observes only.
const mountAccessScript = `ok=1
for p in /workspace /opt/data; do
  [ -d "$p" ] || { echo "$p missing"; ok=0; continue; }
  [ -r "$p" ] || { echo "$p read-denied"; ok=0; }
  [ -x "$p" ] || { echo "$p search-denied"; ok=0; }
done
for p in /workspace /opt/data; do
  [ -w "$p" ] || { echo "$p write-denied"; ok=0; }
done
[ "$ok" = 1 ] && echo MOUNT_ACCESS_OK
exit 0`

// HostSecurity reports the host SELinux state and proves the repository-owned
// bind mounts are usable inside the runtime. It never disables SELinux, changes
// policy, or writes to the mounted state.
func HostSecurity(ctx context.Context, id target.Identity, r Runner) []Probe {
	state := detectSELinux()
	detail := "SELinux " + string(state) + "; repository bind mounts need no relabel"
	if state.Enabled() {
		detail = "SELinux " + string(state) + "; private Z relabeling for repository bind mounts"
	}
	sel := Probe{"selinux", Healthy, detail}
	return append([]Probe{sel}, mountAccess(ctx, id, r)...)
}

func mountAccess(ctx context.Context, id target.Identity, r Runner) []Probe {
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return []Probe{{"access", Unknown, "runtime unavailable; in-container repository and native-state access not observed"}}
	}
	out := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", container, "/bin/sh", "-c", mountAccessScript)
	if out.Err == nil && !out.Truncated && strings.Contains(out.Output, "MOUNT_ACCESS_OK") {
		return []Probe{{"access", Healthy, "/workspace and /opt/data are readable and writable from inside the runtime"}}
	}
	reason := strings.TrimSpace(out.Output)
	if reason == "" {
		reason = "access command failed"
	}
	return []Probe{{"access", Degraded, "repository or native-state bind mount is not usable inside the runtime (SELinux relabeling or ownership pending): " + reason}}
}

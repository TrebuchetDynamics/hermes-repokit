package verify

import (
	"context"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var containerID = regexp.MustCompile(`^[a-f0-9]{12,64}$`)

const memoryInspectFormat = `{"status":{{json .State.Status}},"health":{{if .State.Health}}{{json .State.Health.Status}}{{else}}"unknown"{{end}},"image":{{json .Config.Image}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"service":{{json (index .Config.Labels "com.docker.compose.service")}},"home":{{range .Mounts}}{{if eq .Destination "/app/.openviking"}}{{json .Source}}{{end}}{{end}}}`

// OpenViking observes Docker and filesystem metadata only. In particular it
// never calls /ready, which performs an embedding request in the selected image.
func OpenViking(ctx context.Context, id target.Identity, r Runner) []Probe {
	config := Probe{"openviking-config", PendingSetup, "native ov.conf absent; run setup --memory in your terminal"}
	root, err := os.OpenRoot(id.Root)
	if err == nil {
		defer root.Close()
		info, e := root.Lstat(".hermes/openviking/ov.conf")
		if e == nil {
			config.Status = Degraded
			config.Detail = "native memory configuration is not a nonempty private regular file"
			if info.Mode().IsRegular() && info.Size() > 0 && info.Mode().Perm()&0077 == 0 {
				config.Status = Healthy
				config.Detail = "native memory configuration exists; contents, auth and models not inspected"
			}
		} else if !os.IsNotExist(e) {
			config.Status = Unknown
			config.Detail = "native memory configuration metadata unavailable"
		}
	}
	runtime := Probe{"openviking-runtime", Unknown, "service metadata unavailable"}
	container := Probe{"openviking-container", Unknown, "pinned running container identity unavailable"}
	contextName, e := launcher.Context(id)
	if e == nil {
		list := r.Run(ctx, "docker", "--context", contextName, "container", "ls", "--all", "--filter", "label=com.docker.compose.project="+id.Project, "--filter", "label=com.docker.compose.service=openviking", "--format", "{{.ID}}")
		ids := strings.Fields(list.Output)
		if list.Err == nil && !list.Truncated && len(ids) == 0 {
			container.Status = PendingSetup
			container.Detail = "OpenViking service absent; start it with ordinary Compose"
			runtime.Status = PendingSetup
			runtime.Detail = "OpenViking service absent; start it with ordinary Compose"
		}
		if list.Err == nil && !list.Truncated && len(ids) == 1 && containerID.MatchString(ids[0]) {
			result := r.Run(ctx, "docker", "--context", contextName, "container", "inspect", "--format", memoryInspectFormat, ids[0])
			var state struct{ Status, Health, Image, Project, Service, Home string }
			if result.Err == nil && !result.Truncated && json.Unmarshal([]byte(result.Output), &state) == nil {
				container.Status = Degraded
				runtime.Status = Degraded
				runtime.Detail = "OpenViking image, identity or persistent mount differs"
				if state.Image == projectmemory.Image && state.Project == id.Project && state.Service == "openviking" && state.Home == filepath.Join(id.Root, ".hermes/openviking") {
					if state.Status == "running" {
						container.Status = Healthy
						container.Detail = "running pinned image, Compose identity and persistent mount match; health separate"
					}
					runtime.Detail = "OpenViking service stopped or unhealthy; inspect native setup"
					if state.Status == "running" && state.Health == "healthy" {
						runtime.Status = Healthy
						runtime.Detail = "pinned service and persistent mount match; Docker health is not memory acceptance"
					} else if state.Status == "running" && config.Status == PendingSetup {
						runtime.Status = PendingSetup
						runtime.Detail = "service running in unconfigured mode; native setup required"
					}
				}
			}
		}
	}
	return []Probe{config, container, runtime, {"openviking", Unknown, "cross-profile recall, cross-repository denial and memory persistence acceptance not established"}}
}

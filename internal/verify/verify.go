// Package verify performs metadata-only observations. It never invokes Hermes.
package verify

import (
	"context"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
)

type Status string

const (
	Healthy      Status = "healthy"
	Degraded     Status = "degraded"
	PendingSetup Status = "pending-setup"
	Unsupported  Status = "unsupported"
	Unknown      Status = "unknown"
)

type Probe struct {
	Component string `json:"component"`
	Status    Status `json:"status"`
	Detail    string `json:"detail"`
}
type Runner interface {
	Run(context.Context, string, ...string) process.Result
}
type Runtime struct{ Status, Project, Workspace, Home string }

// InspectFormat selects only nonsecret Docker metadata, never Config.Env.
const InspectFormat = `{"status":{{json .State.Status}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"workspace":{{range .Mounts}}{{if eq .Destination "/workspace"}}{{json .Source}}{{end}}{{end}},"home":{{range .Mounts}}{{if eq .Destination "/opt/data"}}{{json .Source}}{{end}}{{end}}}`

func Inspect(ctx context.Context, id target.Identity, r Runner) []Probe {
	artifact := Probe{"compose", PendingSetup, "Compose file is absent"}
	if info, e := os.Lstat(id.Compose); e == nil {
		if info.Mode().IsRegular() {
			artifact = Probe{"compose", Unknown, "Compose exists; owner configuration is authoritative"}
		} else {
			artifact = Probe{"compose", Degraded, "Compose path is not a regular file"}
		}
	} else if !os.IsNotExist(e) {
		artifact = Probe{"compose", Unknown, "Compose metadata unavailable"}
	}
	runtime := Probe{"hermes", Unknown, "Docker runtime metadata unavailable"}
	result := r.Run(ctx, "docker", "container", "inspect", "--format", InspectFormat, id.Container)
	var state Runtime
	if result.Err == nil && !result.Truncated && json.Unmarshal([]byte(result.Output), &state) == nil {
		runtime.Status = Degraded
		runtime.Detail = "container identity or mounts do not match this repository"
		if state.Project == id.Project && state.Workspace == id.Root && state.Home == filepath.Join(id.Root, ".hermes") {
			if state.Status == "running" {
				runtime.Status = Healthy
				runtime.Detail = "container running with expected project and mounts; inference not probed"
			} else {
				runtime.Detail = "container is stopped; use ordinary Compose to start it"
			}
		}
	}
	return []Probe{artifact, runtime, {"kanban", Unknown, "database not opened: native queries may initialize or migrate it"}, {"superpowers", Unknown, "fresh native session loading not probed"}, {"openviking", PendingSetup, "native init/doctor and actual recall evidence required; no extraction triggered"}, {"nerve-laya", Unsupported, "sidecar transport, scanner admission and inference qualification pending"}}
}

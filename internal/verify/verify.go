// Package verify performs metadata-only observations. It never invokes Hermes.
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"io"
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
	result := process.Result{}
	dockerContext, contextErr := launcher.Context(id)
	if contextErr == nil {
		result = r.Run(ctx, "docker", "--context", dockerContext, "container", "inspect", "--format", InspectFormat, id.Container)
	} else {
		result.Err = contextErr
		runtime.Detail = "deployment Docker context unknown; launcher absent or owner-edited"
	}
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
	config := Probe{"config", PendingSetup, "native configuration is absent"}
	if info, err := os.Lstat(filepath.Join(id.Root, ".hermes/config.yaml")); err == nil {
		config = Probe{"config", Degraded, "native configuration is not a nonempty regular file"}
		if info.Mode().IsRegular() && info.Size() > 0 {
			config = Probe{"config", Healthy, "native configuration exists; credentials and semantic contents not inspected"}
		}
	}
	launch := Probe{"launcher", Unknown, "standalone launcher absent, edited or unusable"}
	if info, err := os.Lstat(id.Launcher); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0100 != 0 && contextErr == nil {
		launch = Probe{"launcher", Healthy, "recognized standalone launcher; native authentication not probed"}
	}
	// Compare only bounded public generated Compose, never credentials or receipt.
	if artifact.Status == Unknown {
		expected, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
		if err == nil && matchesCompose(id, expected) {
			artifact = Probe{"compose", Healthy, "generated Hermes-only Compose matches this repository"}
		}
	}
	probes := []Probe{artifact, runtime, config, launch}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		probes = append(probes, Probe{"filesystem", Degraded, "unsafe or ambiguous repository/native state"})
	}
	return probes
}

func matchesCompose(id target.Identity, expected []byte) bool {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return false
	}
	defer root.Close()
	f, err := root.Open(".hermes/compose.yaml")
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	return err == nil && len(data) <= 65536 && bytes.Equal(data, expected)
}

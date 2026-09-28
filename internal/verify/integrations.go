package verify

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed integrations_probe.py
var integrationsProbe string

const integrationInspect = `{"id":{{json .Id}},"status":{{json .State.Status}},"image":{{json .Config.Image}},"imageID":{{json .Image}},"mounts":{{json .Mounts}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"service":{{json (index .Config.Labels "com.docker.compose.service")}},"workspace":{{range .Mounts}}{{if eq .Destination "/workspace"}}{{json .Source}}{{end}}{{end}},"home":{{range .Mounts}}{{if eq .Destination "/opt/data"}}{{json .Source}}{{end}}{{end}},"unexpectedMounts":"{{range .Mounts}}{{if or (ne .Type "bind") (and (ne .Destination "/workspace") (ne .Destination "/opt/data"))}}x{{end}}{{end}}"}`

func integrationRuntime(ctx context.Context, id target.Identity, r Runner) (string, string, error) {
	if info, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("native home is not a real directory")
	}
	dc, err := launcher.Context(id)
	if err != nil {
		return "", "", err
	}
	out := r.Run(ctx, "docker", "--context", dc, "container", "inspect", "--format", integrationInspect, id.Container)
	var s struct {
		ID, Status, Image, ImageID, Project, Service, Workspace, Home, UnexpectedMounts string
		Mounts                                                                          []RuntimeMount
	}
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &s) != nil || !containerID.MatchString(s.ID) || len(s.ID) != 64 || s.Status != "running" || !HermesImageMatches(ctx, id, dc, s.Image, s.ImageID, r) || s.Project != id.Project || s.Service != "hermes" || s.Workspace != id.Root || s.Home != filepath.Join(id.Root, ".hermes") || !RuntimeMountsMatch(id, s.UnexpectedMounts, s.Mounts) {
		return "", "", fmt.Errorf("pinned running Hermes identity and mounts unavailable")
	}
	return dc, s.ID, nil
}

// RuntimeIntegrations reads native configuration without importing Hermes,
// discovering plugins, opening the board, or invoking any model. Active memory
// requires matching native bindings and authenticated user identity from the
// read-only /health endpoint; model-backed acceptance stays separate.
func RuntimeIntegrations(ctx context.Context, id target.Identity, r Runner) []Probe {
	memory := Probe{"memory", Unknown, "native memory configuration unavailable; runtime identity must match"}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err == nil {
		memoryService := "memory-service-unavailable"
		for _, p := range OpenViking(ctx, id, r) {
			if p.Component == "openviking-container" && p.Status == Healthy {
				memoryService = "memory-service-matched"
			}
		}
		result := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", container, "/opt/hermes/.venv/bin/python", "-I", "-B", "-c", integrationsProbe, "/opt/data", id.Project, memoryService)
		var observed struct{ Memory string }
		if result.Err == nil && !result.Truncated && json.Unmarshal([]byte(result.Output), &observed) == nil {
			switch observed.Memory {
			case "active":
				memory = Probe{"memory", Active, "all six native profiles authenticate as the project user through read-only /health; recall, extraction and persistence acceptance unqualified"}
			case "inactive":
				memory = Probe{"memory", Inactive, "native OpenViking memory, user key or matching service is absent or disabled"}
			case "degraded":
				memory = Probe{"memory", Degraded, "native memory configuration, linked identity or profile override differs; private values withheld"}
			}
		}
	}
	return []Probe{memory, {"review", Unqualified, "distinct-actor same-card review acceptance is not established by configuration or service health"}}
}

// IntegrationCheckSource lets mutating setup recheck the same passive policy
// inside its writer lock. Execute in a separate namespace to isolate helpers;
// __name__ must differ from __main__ so the CLI entry point is not invoked.
func IntegrationCheckSource() string { return integrationsProbe }

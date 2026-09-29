package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

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

// RuntimeIntegrations reports only what the pinned runtime and public service
// checks establish. Configured memory identity and recall behavior require an
// authenticated user-level operation, so passive verification does not certify
// them from files or service health.
func RuntimeIntegrations(ctx context.Context, id target.Identity, r Runner) []Probe {
	memory := Probe{"memory", Unknown, "native memory identity and effective profile configuration unavailable"}
	if _, _, err := integrationRuntime(ctx, id, r); err == nil {
		memory = Probe{"memory", Unqualified, "pinned runtime observed; all-profile user identity, recall, extraction and persistence unqualified"}
		for _, p := range OpenViking(ctx, id, r) {
			if p.Component == "openviking-config" && p.Status == PendingSetup {
				memory = Probe{"memory", Inactive, "native OpenViking configuration absent; run setup --memory"}
			}
		}
	}
	return []Probe{memory}
}

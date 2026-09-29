package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

var containerID = regexp.MustCompile(`^[a-f0-9]{12,64}$`)

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

// RuntimeIntegrations reports only what the pinned runtime establishes. Memory
// is user-managed, so passive verification does not certify it from files or
// service health.
func RuntimeIntegrations(ctx context.Context, id target.Identity, r Runner) []Probe {
	memory := Probe{"memory", Inactive, "memory is user-managed; RepoKit does not configure or verify it"}
	if _, _, err := integrationRuntime(ctx, id, r); err != nil {
		memory = Probe{"memory", Unknown, "pinned runtime unavailable; memory remains user-managed"}
	}
	return []Probe{memory}
}

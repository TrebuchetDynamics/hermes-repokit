package verify

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/supervision"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed integrations_probe.py
var integrationsProbeBody string

var integrationsProbe = supervision.CheckoutScript + "\n" + integrationsProbeBody

const integrationInspect = `{"id":{{json .Id}},"status":{{json .State.Status}},"image":{{json .Config.Image}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"service":{{json (index .Config.Labels "com.docker.compose.service")}},"workspace":{{range .Mounts}}{{if eq .Destination "/workspace"}}{{json .Source}}{{end}}{{end}},"home":{{range .Mounts}}{{if eq .Destination "/opt/data"}}{{json .Source}}{{end}}{{end}},"unexpectedMounts":"{{range .Mounts}}{{if or (ne .Type "bind") (and (ne .Destination "/workspace") (ne .Destination "/opt/data"))}}x{{end}}{{end}}"}`

func integrationRuntime(ctx context.Context, id target.Identity, r Runner) (string, string, error) {
	if info, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("native home is not a real directory")
	}
	dc, err := launcher.Context(id)
	if err != nil {
		return "", "", err
	}
	out := r.Run(ctx, "docker", "--context", dc, "container", "inspect", "--format", integrationInspect, id.Container)
	var s struct{ ID, Status, Image, Project, Service, Workspace, Home, UnexpectedMounts string }
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &s) != nil || !containerID.MatchString(s.ID) || len(s.ID) != 64 || s.Status != "running" || s.Image != qualification.FoundationImage || s.Project != id.Project || s.Service != "hermes" || s.Workspace != id.Root || s.Home != filepath.Join(id.Root, ".hermes") || s.UnexpectedMounts != "" {
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
	nerve := Probe{"nerve", Unknown, "native Nerve configuration unavailable; runtime identity must match"}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err == nil {
		memoryService := "memory-service-unavailable"
		for _, p := range OpenViking(ctx, id, r) {
			if p.Component == "openviking-container" && p.Status == Healthy {
				memoryService = "memory-service-matched"
			}
		}
		result := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", container, "/opt/hermes/.venv/bin/python", "-I", "-B", "-c", integrationsProbe, "/opt/data", id.Project, supervision.NerveRevision, memoryService)
		var observed struct{ Memory, Nerve string }
		if result.Err == nil && !result.Truncated && json.Unmarshal([]byte(result.Output), &observed) == nil {
			switch observed.Memory {
			case "active":
				memory = Probe{"memory", Active, "all six native profiles authenticate as the project user through read-only /health; recall, extraction and persistence acceptance unqualified"}
			case "inactive":
				memory = Probe{"memory", Inactive, "native OpenViking memory, user key or matching service is absent or disabled"}
			case "degraded":
				memory = Probe{"memory", Degraded, "native memory configuration, linked identity or profile override differs; private values withheld"}
			}
			switch observed.Nerve {
			case "configured":
				nerve = Probe{"nerve", Healthy, "all six profiles enable the pinned native Nerve install with local Laya settings; plugin loading and decisions not probed"}
			case "inactive":
				nerve = Probe{"nerve", Inactive, "Nerve absent or disabled in one or more native profiles"}
			case "degraded":
				nerve = Probe{"nerve", Degraded, "Nerve checkout, local settings or profile overrides differ; private values withheld"}
			}
		}
	}
	return []Probe{memory, nerve, Laya(ctx, id, r), {"review", Unqualified, "distinct-actor same-card review acceptance is not established by configuration or service health"}}
}

const layaInspect = `{"status":{{json .State.Status}},"image":{{json .Config.Image}},"imageID":{{json .Image}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"service":{{json (index .Config.Labels "com.docker.compose.service")}},"network":{{json .HostConfig.NetworkMode}},"readonly":{{json .HostConfig.ReadonlyRootfs}},"cache":{{$cache := ""}}{{range .Mounts}}{{if eq .Destination "/cache"}}{{$cache = .Source}}{{end}}{{end}}{{json $cache}},"unexpectedMounts":"{{range .Mounts}}{{if not (or (and (eq .Destination "/tmp") (eq .Type "tmpfs")) (and (eq .Destination "/cache") (eq .Type "bind")))}}x{{end}}{{end}}"}`

// Laya qualifies service identity and GET /healthz only, never inference.
func Laya(ctx context.Context, id target.Identity, r Runner) Probe {
	p := Probe{"laya", Unknown, "pinned running Hermes identity and mounts unavailable"}
	dc, hermes, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return p
	}
	image := compose.SelectedLaya(id)
	build := compose.DefaultLayaSelected(id)
	if image == "" && !build {
		return Probe{"laya", PendingSetup, "qualified local Laya image not selected in generated Compose"}
	}
	list := r.Run(ctx, "docker", "--context", dc, "container", "ls", "--all", "--filter", "label=com.docker.compose.project="+id.Project, "--filter", "label=com.docker.compose.service=laya", "--format", "{{.ID}}")
	ids := strings.Fields(list.Output)
	if list.Err != nil || list.Truncated {
		return p
	}
	if len(ids) == 0 {
		return Probe{"laya", PendingSetup, "Laya service absent; start it with ordinary Compose"}
	}
	p = Probe{"laya", Degraded, "Laya container identity, pinned image or shared network namespace differs"}
	if len(ids) != 1 || !containerID.MatchString(ids[0]) {
		return p
	}
	out := r.Run(ctx, "docker", "--context", dc, "container", "inspect", "--format", layaInspect, ids[0])
	var s struct {
		Status, Image, ImageID, Project, Service, Network, Cache, UnexpectedMounts string
		Readonly                                                                   bool
	}
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &s) != nil || s.Status != "running" || !compose.LocalImageID(s.ImageID) || s.Project != id.Project || s.Service != "laya" || s.Network != "container:"+hermes || !s.Readonly || s.UnexpectedMounts != "" {
		return p
	}
	if build {
		if s.Cache != filepath.Join(id.Root, ".hermes/laya") {
			return p
		}
	} else if s.Image != image || s.ImageID != image || s.Cache != "" {
		return p
	}
	if supervision.CheckImage(ctx, dc, s.ImageID, r) != nil {
		return p
	}
	result := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", hermes, "/opt/hermes/.venv/bin/python", "-I", "-B", "-c", layaHealthProbe)
	if result.Err != nil || result.Truncated || strings.TrimSpace(result.Output) != "healthy" {
		return Probe{"laya", Degraded, "qualified Laya /healthz unavailable or response differs; inference not probed"}
	}
	return Probe{"laya", Healthy, "qualified local Laya responds to read-only /healthz from Hermes; inference acceptance not probed"}
}

const layaHealthProbe = `import json, urllib.request
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None
try:
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    with opener.open('http://127.0.0.1:8765/healthz', timeout=3) as response:
        data = response.read(4097)
        assert response.status == 200 and len(data) <= 4096
        assert json.loads(data) == {'ok': True, 'provider': 'Laya', 'model': '/model', 'transport': 'laya-local-http'}
    print('healthy')
except Exception:
    print('degraded')
`

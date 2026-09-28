package verify

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/dockertest"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"reflect"
	"sort"
	"strings"
)

//go:embed development_probe.py
var developmentProbe string

func Development(ctx context.Context, id target.Identity, r Runner) []Probe {
	result := []Probe{{"development_environment", Unknown, "qualified runtime unavailable; tool presence alone is not coding acceptance"}}
	req, err := development.Detect(id.Root)
	if err != nil {
		result[0] = Probe{"development_environment", Degraded, "repository manifests cannot be safely inspected"}
		return append(result, dockerAcceptance(ctx, id, r))
	}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return append(result, dockerAcceptance(ctx, id, r))
	}
	mode := "base"
	if req.Go {
		mode = "go"
	}
	out := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/workspace", container, "/opt/hermes/.venv/bin/python", "-I", "-B", "-c", developmentProbe, mode)
	var observed struct {
		Tools map[string]struct {
			OK      bool `json:"ok"`
			Version string
		}
		Workspace, Certificates bool
		Profiles                map[string]bool
	}
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &observed) != nil {
		return append(result, dockerAcceptance(ctx, id, r))
	}
	missing := append([]string(nil), req.Unsupported...)
	expected := []string{"git", "bash", "curl", "jq", "rg", "python", "node", "npm", "make", "gcc", "g++", "docker", "compose", "buildx"}
	if req.Go {
		expected = append(expected, "go")
	}
	for _, name := range expected {
		tool := observed.Tools[name]
		status := Healthy
		if !tool.OK {
			status = Degraded
			missing = append(missing, name+" unavailable")
		}
		if name == "compose" && tool.OK && strings.TrimPrefix(tool.Version, "v") != development.ComposeVersion {
			status = Degraded
			missing = append(missing, "Docker Compose plugin version differs")
		}
		if name == "buildx" && tool.OK && !containsVersionToken(tool.Version, "v"+development.BuildxVersion) {
			status = Degraded
			missing = append(missing, "Docker Buildx plugin version differs")
		}
		if name == "go" && tool.OK && !strings.HasPrefix(tool.Version, "go version go"+development.GoVersion+" ") {
			status = Degraded
			missing = append(missing, "Go version differs from qualified recipe")
		}
		result = append(result, Probe{"development:" + name, status, tool.Version})
	}
	if !observed.Workspace {
		missing = append(missing, "/workspace is not the writable workdir")
	}
	if !observed.Certificates {
		missing = append(missing, "CA bundle absent")
	}
	for _, name := range []string{"default", "researcher", "planner", "executor", "reviewer", "steward"} {
		if !observed.Profiles[name] {
			missing = append(missing, name+" terminal not configured for local /workspace")
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		result[0] = Probe{"development_environment", Degraded, strings.Join(missing, "; ")}
	} else {
		result[0] = Probe{"development_environment", Healthy, "required tools execute and six profile terminals select local /workspace; real coding/review acceptance separate"}
	}
	return append(result, dockerAcceptance(ctx, id, r))
}

func dockerAcceptance(ctx context.Context, id target.Identity, r Runner) Probe {
	o, ok := compose.DevelopmentSelected(id)
	if !ok || !o.DockerTests {
		return Probe{"docker_acceptance", Healthy, "disabled; normal coding does not require a Docker daemon"}
	}
	dc, _, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return Probe{"docker_acceptance", Unknown, "selected test deployment identity unavailable"}
	}
	list := r.Run(ctx, "docker", "--context", dc, "container", "ls", "--all", "--filter", "label=com.docker.compose.project="+id.Project, "--filter", "label=com.docker.compose.service=docker-test", "--format", "{{.ID}}")
	ids := strings.Fields(list.Output)
	if list.Err != nil || list.Truncated || len(ids) != 1 {
		return Probe{"docker_acceptance", Inactive, "selected optional daemon is not uniquely running; activate the docker-tests Compose profile"}
	}
	if !containerID.MatchString(ids[0]) {
		return Probe{"docker_acceptance", Unknown, "test daemon identity unavailable"}
	}
	const format = `{"status":{{json .State.Status}},"image":{{json .Config.Image}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"service":{{json (index .Config.Labels "com.docker.compose.service")}},"health":{{if .State.Health}}{{json .State.Health.Status}}{{else}}"absent"{{end}},"privileged":{{json .HostConfig.Privileged}},"ports":{{json .HostConfig.PortBindings}},"mounts":{{json .Mounts}},"networks":{{json .NetworkSettings.Networks}},"networkMode":{{json .HostConfig.NetworkMode}},"entrypoint":{{json .Config.Entrypoint}},"command":{{json .Config.Cmd}}}`
	out := r.Run(ctx, "docker", "--context", dc, "container", "inspect", "--format", format, ids[0])
	var s struct {
		Status, Image, Project, Service, Health string
		Privileged                              bool
		Ports                                   map[string]any
		Mounts                                  []RuntimeMount
		Networks                                map[string]json.RawMessage
		NetworkMode                             string
		Entrypoint, Command                     []string
	}
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &s) != nil {
		return Probe{"docker_acceptance", Unknown, "test daemon metadata unavailable"}
	}
	healthy := s.Status == "running" && s.Image == dockertest.Image && s.Project == id.Project && s.Service == "docker-test" && s.Health == "healthy" && s.Privileged && len(s.Ports) == 0 && len(s.Mounts) == 3
	_, networkOK := s.Networks[id.Project+"_docker-test"]
	healthy = healthy && networkOK && len(s.Networks) == 1 && s.NetworkMode == id.Project+"_docker-test" && reflect.DeepEqual(s.Entrypoint, []string{"sh", "-ec"}) && reflect.DeepEqual(s.Command, []string{dockertest.DaemonCommand(os.Getuid(), os.Getgid())})
	expected := map[string]string{"/docker-test/run": "docker-test-run", "/docker-tests": "docker-test-work", "/var/lib/docker": "docker-test-data"}
	for _, m := range s.Mounts {
		suffix, ok := expected[m.Destination]
		if !ok || m.Type != "volume" || m.Name != id.Project+"_"+suffix {
			healthy = false
		}
		delete(expected, m.Destination)
	}
	if !healthy || len(expected) != 0 {
		return Probe{"docker_acceptance", Degraded, "test daemon health, image or isolation metadata differs"}
	}
	return Probe{"docker_acceptance", Healthy, "available: separate privileged test daemon is healthy, no host socket or published ports; real Docker test suite remains separate"}
}

func containsVersionToken(output, want string) bool {
	for _, token := range strings.Fields(output) {
		if token == want {
			return true
		}
	}
	return false
}

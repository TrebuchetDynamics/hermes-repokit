package verify

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/dockertest"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

var developmentVersionText = regexp.MustCompile(`^[A-Za-z0-9_ .()+,/:~\-]+$`)

func Development(ctx context.Context, id target.Identity, r Runner) []Probe {
	result := []Probe{{"development_environment", Unknown, "qualified coding runtime unavailable"}}
	req, err := development.Detect(id.Root)
	if err != nil {
		result[0] = Probe{"development_environment", Degraded, "repository manifests cannot be safely inspected"}
		return append(result, dockerAcceptance(ctx, id, r))
	}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return append(result, dockerAcceptance(ctx, id, r))
	}
	// Resolve tools by name in a login shell, as worker terminals do, rather
	// than trusting an absolute path that workers might not discover.
	tools := []struct{ name, command string }{
		{"git", "git --version"}, {"bash", "bash --version"}, {"curl", "curl --version"},
		{"jq", "jq --version"}, {"rg", "rg --version"}, {"python", "python3 --version"},
		{"node", "node --version"}, {"npm", "npm --version"}, {"make", "make --version"},
		{"gcc", "gcc --version"}, {"g++", "g++ --version"}, {"docker", "docker --version"},
		{"compose", "docker compose version --short"}, {"buildx", "docker buildx version"},
		{"shellcheck", "shellcheck --version | grep -F version:"},
		{"browser-use", "browser-use --version"}, {"chromium", "\"$AGENT_BROWSER_EXECUTABLE_PATH\" --version"},
	}
	if req.Go {
		tools = append(tools, struct{ name, command string }{"go", "go version"}, struct{ name, command string }{"staticcheck", "staticcheck -version"})
	}
	if req.Rust {
		tools = append(tools, struct{ name, command string }{"rustc", "rustc --version"}, struct{ name, command string }{"cargo", "cargo --version"}, struct{ name, command string }{"clippy", "cargo clippy --version"})
	}
	missing := append([]string(nil), req.Unsupported...)
	for _, tool := range tools {
		out := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/workspace", container, "/usr/bin/bash", "-lc", tool.command)
		version := ""
		if out.Err == nil && !out.Truncated {
			version = strings.TrimSpace(strings.SplitN(out.Output, "\n", 2)[0])
			if len(version) > 160 {
				version = version[:160] // e.g. curl lists every linked library
			}
		}
		status := Healthy
		if version == "" || !developmentVersionText.MatchString(version) {
			status, version = Degraded, "unavailable"
		} else if tool.name == "compose" && strings.TrimPrefix(version, "v") != development.ComposeVersion ||
			tool.name == "buildx" && !containsVersionToken(version, "v"+development.BuildxVersion) ||
			tool.name == "go" && !strings.HasPrefix(version, "go version go"+development.GoVersion+" ") ||
			tool.name == "staticcheck" && !strings.HasPrefix(version, "staticcheck "+development.StaticcheckVersion+" ") ||
			tool.name == "rustc" && !strings.HasPrefix(version, "rustc "+development.RustVersion+" ") {
			status = Degraded
		}
		if status == Degraded {
			missing = append(missing, tool.name+" unavailable or version differs")
		}
		result = append(result, Probe{"development:" + tool.name, status, version})
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		result[0] = Probe{"development_environment", Degraded, strings.Join(missing, "; ")}
	} else {
		result[0] = Probe{"development_environment", Unqualified, "required tools resolve by name in a worker-style login shell; live coding behavior is proved by reviewed work"}
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

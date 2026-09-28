package verify

import (
	"context"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"regexp"
	"strings"
)

var containerID = regexp.MustCompile(`^[a-f0-9]{12,64}$`)

// OpenViking observes runtime identity, filesystem metadata and loopback health. In particular it
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
	if o, selected := compose.DevelopmentSelected(id); selected {
		return embeddedOpenViking(ctx, id, r, config, o.OpenVikingImage != "")
	}
	return []Probe{
		config,
		{"openviking-container", PendingSetup, "embedded OpenViking not selected; reconcile the generated Hermes deployment"},
		{"openviking-runtime", PendingSetup, "independent sidecars do not satisfy the embedded memory contract"},
		{"openviking", PendingSetup, "embedded memory setup and live acceptance unqualified"},
	}
}

// The memory process shares Hermes's image, mount namespace and lifecycle.
// Component readiness is observed independently of gateway/container readiness.
func embeddedOpenViking(ctx context.Context, id target.Identity, r Runner, config Probe, selected bool) []Probe {
	container := Probe{"openviking-container", Degraded, "embedded memory runtime is not selected or Hermes identity/mounts differ"}
	runtime := Probe{"openviking-runtime", Degraded, "embedded memory health is unavailable"}
	if selected {
		dc, hermes, err := integrationRuntime(ctx, id, r)
		if err == nil {
			container = Probe{"openviking-container", Healthy, "OpenViking packaged inside the verified Hermes image with persistent private storage"}
			if config.Status == PendingSetup {
				runtime = Probe{"openviking-runtime", PendingSetup, "embedded OpenViking awaiting private native setup --memory"}
			} else if config.Status == Healthy {
				out := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", hermes, "/usr/local/bin/repokit-openviking", "health")
				if out.Err == nil && !out.Truncated && strings.TrimSpace(out.Output) == "healthy" {
					runtime = Probe{"openviking-runtime", Healthy, "embedded loopback /health responds; no model or memory operation performed"}
				}
			}
		}
	}
	service := runtime
	service.Component = "openviking"
	service.Detail += "; recall, extraction, cross-repository denial and persistence acceptance unqualified"
	return []Probe{config, container, runtime, service}
}

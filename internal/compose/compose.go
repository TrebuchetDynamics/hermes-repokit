// Package compose renders ordinary Compose files without runtime callbacks.
package compose

import (
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/dockertest"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"regexp"
	"strings"
)

// historicalOpenVikingImage is the memory image the OpenViking-era releases
// pinned. It is rendered only to recognize those generated deployments.
const historicalOpenVikingImage = "ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05"

type Options struct {
	HermesImage string
	UID, GID    int
	Development *development.Requirements
	DockerTests bool
	// HistoricalOpenViking reproduces the exact Compose that OpenViking-era
	// releases generated: an embedded-memory marker on the development
	// runtime, or a memory sidecar without it. It exists only so install can
	// recognize and upgrade those deployments; RepoKit never installs it.
	HistoricalOpenViking bool
	// SELinux selects private Docker bind relabeling for RepoKit-owned mounts.
	// The zero value disables relabeling, preserving historical preimages.
	SELinux selinux.State
}

var identity = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

func Render(id target.Identity, o Options) ([]byte, error) {
	if !identity.MatchString(id.Project) || !identity.MatchString(id.Container) || !qualification.ImmutableImage(o.HermesImage) || o.UID <= 0 || o.GID <= 0 {
		return nil, fmt.Errorf("invalid identity, host UID/GID or immutable Hermes image")
	}
	if o.DockerTests && o.Development == nil {
		return nil, fmt.Errorf("Docker acceptance requires the generated development runtime")
	}
	if o.Development != nil && o.HermesImage != qualification.FoundationImage {
		return nil, fmt.Errorf("development runtime requires the qualified Hermes base")
	}
	var s strings.Builder
	fmt.Fprintf(&s, `# Native state is authoritative. Ordinary Docker Compose owns this deployment.
name: %q
services:
  hermes:
`, id.Project)
	if o.Development != nil {
		fmt.Fprintf(&s, "    image: %q\n    build:\n      context: ./development-image\n    platform: linux/amd64\n    pull_policy: build\n", development.ImageName(id.Container, *o.Development))
	} else {
		fmt.Fprintf(&s, "    image: %q\n", o.HermesImage)
	}
	fmt.Fprintf(&s, `    container_name: %q
    restart: unless-stopped
    working_dir: /workspace
    # Keep the native exec endpoint available before interactive provider setup.
    command: ["sleep", "infinity"]
    environment:
      HERMES_HOME: /opt/data
      HERMES_WRITE_SAFE_ROOT: /opt/data:/workspace
      HERMES_UID: %q
      HERMES_GID: %q
`, id.Container, fmt.Sprint(o.UID), fmt.Sprint(o.GID))
	if o.Development != nil && o.HistoricalOpenViking {
		s.WriteString("      REPOKIT_OPENVIKING: \"1\"\n")
	}
	relabel := ""
	if mode := o.SELinux.RelabelMode(); mode != "" {
		relabel = "          selinux: " + mode + "\n"
	}
	s.WriteString("    volumes:\n")
	fmt.Fprintf(&s, `      - type: bind
        source: ".."
        target: /workspace
        bind:
          create_host_path: false
%s`, relabel)
	fmt.Fprintf(&s, `      - type: bind
        source: "."
        target: /opt/data
        bind:
          create_host_path: false
%s`, relabel)
	if o.DockerTests {
		s.WriteString(dockertest.Mounts())
	}
	if o.DockerTests {
		extra, err := dockertest.EmitServices(o.UID, o.GID)
		if err != nil {
			return nil, err
		}
		s.WriteString(extra)
		s.WriteString(dockertest.Resources())
	}
	if o.Development == nil && o.HistoricalOpenViking {
		fmt.Fprintf(&s, `  openviking:
    image: %q
    user: %q
    restart: unless-stopped
    environment:
      HOME: /app/.openviking
      OPENVIKING_CONFIG_FILE: /app/.openviking/ov.conf
      OPENVIKING_WITH_BOT: "0"
    healthcheck:
      test: ["CMD", "openviking-entrypoint", "--healthcheck"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s
    volumes:
      - type: bind
        source: "./openviking"
        target: /app/.openviking
        bind:
          create_host_path: false
%s`, historicalOpenVikingImage, fmt.Sprintf("%d:%d", o.UID, o.GID), relabel)
	}
	return []byte(s.String()), nil
}

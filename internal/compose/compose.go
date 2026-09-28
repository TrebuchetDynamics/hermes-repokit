// Package compose renders ordinary Compose files without runtime callbacks.
package compose

import (
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/dockertest"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"regexp"
	"strings"
)

type Options struct {
	HermesImage, OpenVikingImage string
	UID, GID                     int
	Development                  *development.Requirements
	DockerTests                  bool
}

var identity = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

func Render(id target.Identity, o Options) ([]byte, error) {
	if !identity.MatchString(id.Project) || !identity.MatchString(id.Container) || !qualification.ImmutableImage(o.HermesImage) || o.UID <= 0 || o.GID <= 0 {
		return nil, fmt.Errorf("invalid identity, host UID/GID or immutable Hermes image")
	}
	if o.OpenVikingImage != "" && !qualification.ImmutableImage(o.OpenVikingImage) {
		return nil, fmt.Errorf("OpenViking requires immutable image digest")
	}
	if o.DockerTests && o.Development == nil {
		return nil, fmt.Errorf("Docker acceptance requires the generated development runtime")
	}
	if o.Development != nil && o.HermesImage != qualification.FoundationImage {
		return nil, fmt.Errorf("development runtime requires the qualified Hermes base")
	}
	if o.Development != nil && o.OpenVikingImage != "" && o.OpenVikingImage != projectmemory.Image {
		return nil, fmt.Errorf("embedded OpenViking requires the qualified image")
	}
	var s strings.Builder
	fmt.Fprintf(&s, `# Native state is authoritative. Ordinary Docker Compose owns this deployment.
name: %q
services:
  hermes:
`, id.Project)
	if o.Development != nil {
		fmt.Fprintf(&s, "    image: %q\n    build:\n      context: ./development-image\n    platform: linux/amd64\n    pull_policy: build\n", development.ImageName(id.Project, *o.Development))
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
	if o.Development != nil && o.OpenVikingImage != "" {
		s.WriteString("      REPOKIT_OPENVIKING: \"1\"\n")
	}
	s.WriteString(`    volumes:
      - type: bind
        source: ".."
        target: /workspace
        bind:
          create_host_path: false
      - type: bind
        source: "."
        target: /opt/data
        bind:
          create_host_path: false
`)
	if o.DockerTests {
		s.WriteString(dockertest.Mounts())
	}
	// Retained solely to recognize historical generated deployments for upgrade.
	// Normal installation always selects the derived development runtime above.
	if o.Development == nil && o.OpenVikingImage != "" {
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
`, o.OpenVikingImage, fmt.Sprintf("%d:%d", o.UID, o.GID))
	}
	if o.DockerTests {
		extra, err := dockertest.EmitServices(o.UID, o.GID)
		if err != nil {
			return nil, err
		}
		s.WriteString(extra)
		s.WriteString(dockertest.Resources())
	}
	return []byte(s.String()), nil
}

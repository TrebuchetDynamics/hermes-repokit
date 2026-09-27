// Package compose renders ordinary Compose files without runtime callbacks.
package compose

import (
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"regexp"
	"strings"
)

type Options struct {
	HermesImage, OpenVikingImage string
	UID, GID                     int
}

var identity = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

func Render(id target.Identity, o Options) ([]byte, error) {
	if !identity.MatchString(id.Project) || !identity.MatchString(id.Container) || !qualification.ImmutableImage(o.HermesImage) || o.UID <= 0 || o.GID <= 0 {
		return nil, fmt.Errorf("invalid identity, host UID/GID or immutable Hermes image")
	}
	if o.OpenVikingImage != "" && !qualification.ImmutableImage(o.OpenVikingImage) {
		return nil, fmt.Errorf("OpenViking requires immutable image digest")
	}
	var s strings.Builder
	fmt.Fprintf(&s, `# Native state is authoritative. Ordinary Docker Compose owns this deployment.
name: %q
services:
  hermes:
    image: %q
    container_name: %q
    restart: unless-stopped
    working_dir: /workspace
    # Keep the native exec endpoint available before interactive provider setup.
    command: ["sleep", "infinity"]
    environment:
      HERMES_HOME: /opt/data
      HERMES_WRITE_SAFE_ROOT: /opt/data:/workspace
      HERMES_UID: %q
      HERMES_GID: %q
    volumes:
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
`, id.Project, o.HermesImage, id.Container, fmt.Sprint(o.UID), fmt.Sprint(o.GID))
	if o.OpenVikingImage != "" {
		fmt.Fprintf(&s, `  openviking:
    image: %q
    restart: unless-stopped
    environment:
      OPENVIKING_WITH_BOT: "0"
    volumes:
      - type: bind
        source: "./openviking"
        target: /app/.openviking
        bind:
          create_host_path: false
`, o.OpenVikingImage)
	}
	return []byte(s.String()), nil
}

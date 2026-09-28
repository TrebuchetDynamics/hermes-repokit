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
	HermesImage, OpenVikingImage, LayaImage string
	UID, GID                                int
	// LayaBuild selects the embedded local recipe instead of a content image ID.
	LayaBuild bool
}

var identity = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

func Render(id target.Identity, o Options) ([]byte, error) {
	if !identity.MatchString(id.Project) || !identity.MatchString(id.Container) || !qualification.ImmutableImage(o.HermesImage) || o.UID <= 0 || o.GID <= 0 {
		return nil, fmt.Errorf("invalid identity, host UID/GID or immutable Hermes image")
	}
	if o.OpenVikingImage != "" && !qualification.ImmutableImage(o.OpenVikingImage) {
		return nil, fmt.Errorf("OpenViking requires immutable image digest")
	}
	if o.LayaBuild && o.LayaImage != "" {
		return nil, fmt.Errorf("choose the local Laya build or a content image ID")
	}
	if o.LayaImage != "" && !LocalImageID(o.LayaImage) {
		return nil, fmt.Errorf("Laya requires a qualified local content image ID")
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
	if o.LayaImage != "" || o.LayaBuild {
		s.WriteString("  laya:\n")
		if o.LayaImage != "" {
			fmt.Fprintf(&s, "    image: %q\n    pull_policy: never\n", o.LayaImage)
		} else {
			s.WriteString("    build:\n      context: ./laya-image\n    platform: linux/amd64\n")
		}
		fmt.Fprintf(&s, `    network_mode: service:hermes
    depends_on:
      hermes:
        condition: service_started
        restart: true
    restart: unless-stopped
    user: %q
    read_only: true
    tmpfs:
      - /tmp:rw,nosuid,nodev,size=256m
    cpus: 4
    mem_limit: 6g
`, fmt.Sprintf("%d:%d", o.UID, o.GID))
		if o.LayaBuild {
			s.WriteString(`    environment:
      HF_HOME: /cache/huggingface
      TORCHINDUCTOR_CACHE_DIR: /cache/torchinductor
    volumes:
      - type: bind
        source: "./laya"
        target: /cache
        bind:
          create_host_path: false
`)
		}
	}
	return []byte(s.String()), nil
}

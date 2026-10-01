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

type Options struct {
	HermesImage string
	UID, GID    int
	Development *development.Requirements
	DockerTests bool
	// SELinux selects private Docker bind relabeling for RepoKit-owned mounts.
	// The zero value disables relabeling, preserving historical preimages.
	SELinux selinux.State
	// beforeToolchainCache reproduces v0.2.0's Go render, which had no
	// toolchain-cache volume. Only BeforeToolchainCache sets it, so install
	// can recognize and upgrade those deployments; RepoKit never installs it.
	beforeToolchainCache bool
	// beforeStateMask reproduces renders from before .hermes was hidden inside
	// /workspace (v0.2.3 and earlier). Only the previous-layout renders set it.
	beforeStateMask bool
}

var identity = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

// ToolchainCacheTarget and ToolchainCacheVolume name the toolchain cache
// mount; Compose scopes the volume as <project>_toolchain-cache.
const (
	ToolchainCacheTarget = "/var/cache/repokit"
	ToolchainCacheVolume = "toolchain-cache"
)

// ToolchainCacheMounted reports whether a render mounts the toolchain cache:
// Go's module and build caches and Cargo's registry stay out of the
// repository's .hermes.
func ToolchainCacheMounted(o Options) bool {
	return o.Development != nil && (o.Development.Go || o.Development.Rust || o.Development.Flutter) && !o.beforeToolchainCache
}

// StateMaskTarget is where the repository's own .hermes appears inside
// /workspace. An empty read-only tmpfs covers it, so private state is reached
// only through /opt/data: whole-tree commands in the repository never walk it,
// and Hermes never mistakes default's skills folder for repository skills.
const StateMaskTarget = "/workspace/.hermes"

// StateMasked reports whether a render hides .hermes inside /workspace.
func StateMasked(o Options) bool { return !o.beforeStateMask }

// WithoutStateMask returns o as rendered before the mask, for recognizing a
// container that awaits recreation.
func WithoutStateMask(o Options) Options {
	o.beforeStateMask = true
	return o
}

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
	if StateMasked(o) {
		fmt.Fprintf(&s, `      - type: tmpfs
        target: %s
        tmpfs:
          size: 4096
          mode: 0555
`, StateMaskTarget)
	}
	fmt.Fprintf(&s, `      - type: bind
        source: "."
        target: /opt/data
        bind:
          create_host_path: false
%s`, relabel)
	if ToolchainCacheMounted(o) {
		fmt.Fprintf(&s, `      - type: volume
        source: %s
        target: %s
`, ToolchainCacheVolume, ToolchainCacheTarget)
	}
	if o.DockerTests {
		s.WriteString(dockertest.Mounts())
	}
	if o.DockerTests {
		extra, err := dockertest.EmitServices(o.UID, o.GID)
		if err != nil {
			return nil, err
		}
		s.WriteString(extra)
		resources := dockertest.Resources()
		if ToolchainCacheMounted(o) {
			resources = strings.Replace(resources, "volumes:\n", "volumes:\n  "+ToolchainCacheVolume+":\n", 1)
		}
		s.WriteString(resources)
	} else if ToolchainCacheMounted(o) {
		s.WriteString("volumes:\n  " + ToolchainCacheVolume + ":\n")
	}
	return []byte(s.String()), nil
}

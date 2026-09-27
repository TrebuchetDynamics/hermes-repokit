package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func (a App) install(id target.Identity, report Plan, engineering bool, stdout, stderr io.Writer) int {
	if len(report.Unsupported) > 0 {
		fmt.Fprintln(stderr, "installation qualification incomplete:", strings.Join(report.Unsupported, "; "))
		return 1
	}
	if len(report.Collisions) > 0 {
		fmt.Fprintln(stderr, "target collisions:", strings.Join(report.Collisions, "; "))
		return 1
	}
	data, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	script, err := launcher.Render(id, report.DockerContext)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	artifacts := map[string]install.Artifact{
		"compose.yaml":          {Data: data, Mode: 0600},
		"config.yaml":           {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n  orchestrator_profile: default\n  max_in_progress: 1\ntoolsets: [kanban, memory]\nplatform_toolsets:\n  cli: [kanban, memory]\nterminal:\n  cwd: /workspace\n"), Mode: 0600},
		"bin/" + id.Container:   {Data: script, Mode: 0700},
		"openviking/.gitignore": {Data: []byte("*\n"), Mode: 0600},
	}
	previousCompose, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	created, err := install.PublishOpenVikingChecked(id, artifacts, previousCompose, func() error {
		current := a.plan(id, false)
		if len(current.Collisions) > 0 {
			return fmt.Errorf("target changed: %s", strings.Join(current.Collisions, "; "))
		}
		if current.DockerContext != report.DockerContext {
			return fmt.Errorf("Docker context changed during installation")
		}
		return nil
	})
	if err != nil {
		if created {
			fmt.Fprintln(stderr, "artifacts published, but durability confirmation failed; preserve .hermes and inspect before retrying")
		}
		fmt.Fprintln(stderr, "installation refused:", err)
		return 1
	}
	if created {
		fmt.Fprintln(stdout, "Created Hermes and OpenViking bootstrap artifacts; memory configuration is pending.")
	} else {
		fmt.Fprintln(stdout, "Preserved existing Hermes deployment and native configuration.")
	}
	if code := a.initialize(id, report.DockerContext, false, stdout, stderr); code != 0 {
		return code
	}
	fmt.Fprintln(stdout, "Start Hermes with ordinary Compose:")
	fmt.Fprintln(stdout, launcher.StartCommand(id.Compose, report.DockerContext))
	fmt.Fprintln(stdout, "Start OpenViking with ordinary Compose (unconfigured service remains pending):")
	fmt.Fprintln(stdout, launcher.StartMemoryCommand(id.Compose, report.DockerContext))
	fmt.Fprintln(stdout, "After private default setup, run hermes-repokit setup --memory in your terminal.")
	fmt.Fprintln(stdout, "After starting Hermes, run setup to configure default and provision the generic team. Rerun install to reconcile existing generated profiles.")
	return 0
}

func (a App) initialize(id target.Identity, dockerContext string, afterSetup bool, stdout, stderr io.Writer) int {
	const format = `{"status":{{json .State.Status}},"image":{{json .Config.Image}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"workspace":{{range .Mounts}}{{if eq .Destination "/workspace"}}{{json .Source}}{{end}}{{end}},"home":{{range .Mounts}}{{if eq .Destination "/opt/data"}}{{json .Source}}{{end}}{{end}}}`
	observed := a.Runner.Run(context.Background(), "docker", "--context", dockerContext, "container", "inspect", "--format", format, id.Container)
	var runtime struct {
		verify.Runtime
		Image string
	}
	if observed.Err != nil || observed.Truncated || json.Unmarshal([]byte(observed.Output), &runtime) != nil || runtime.Status != "running" {
		fmt.Fprintln(stdout, "Native initialization pending: start the existing Compose service, then rerun install.")
		return 0
	}
	if runtime.Image != qualification.FoundationImage || runtime.Project != id.Project || runtime.Workspace != id.Root || runtime.Home != filepath.Join(id.Root, ".hermes") {
		fmt.Fprintln(stderr, "native initialization refused: running container image or identity does not match the qualified deployment")
		return 1
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 2 * time.Minute}
	}
	if err := native.Initialize(context.Background(), id, dockerContext, afterSetup, runner); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Native shared Kanban checked; team requires successful default setup. Run verify for profile readiness.")
	return 0
}

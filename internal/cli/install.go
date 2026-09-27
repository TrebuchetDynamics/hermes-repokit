package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func (a App) install(id target.Identity, report Plan, stdout, stderr io.Writer) int {
	if len(report.Unsupported) > 0 {
		fmt.Fprintln(stderr, "installation qualification incomplete:", strings.Join(report.Unsupported, "; "))
		return 1
	}
	if len(report.Collisions) > 0 {
		fmt.Fprintln(stderr, "target collisions:", strings.Join(report.Collisions, "; "))
		return 1
	}
	data, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
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
		"compose.yaml":        {Data: data, Mode: 0600},
		"config.yaml":         {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n"), Mode: 0600},
		"bin/" + id.Container: {Data: script, Mode: 0700},
	}
	created, err := install.PublishChecked(id, artifacts, func() error {
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
		fmt.Fprintln(stdout, "Created Hermes-only bootstrap artifacts.")
	} else {
		fmt.Fprintln(stdout, "Preserved existing Hermes deployment and native configuration.")
	}
	fmt.Fprintln(stdout, "Start Hermes with ordinary Compose:")
	fmt.Fprintln(stdout, launcher.StartCommand(id.Compose, report.DockerContext))
	fmt.Fprintln(stdout, "Then run hermes-repokit setup, or the generated launcher with setup.")
	return 0
}

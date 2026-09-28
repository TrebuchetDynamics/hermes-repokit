package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"io"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
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
	// An old sidecar and embedded server must never share the same writable
	// database concurrently. Do not stop/delete owner runtime from install.
	oldMemory := a.Runner.Run(context.Background(), "docker", "--context", report.DockerContext, "container", "ls", "--filter", "label=com.docker.compose.project="+id.Project, "--filter", "label=com.docker.compose.service=openviking", "--format", "{{.ID}}")
	if oldMemory.Err != nil || oldMemory.Truncated || strings.TrimSpace(oldMemory.Output) != "" {
		fmt.Fprintln(stderr, "installation refused: verify that the previous OpenViking sidecar is stopped through its original Compose file before embedding memory; preserve .hermes/openviking data")
		return 1
	}
	data, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &report.Development, DockerTests: report.DockerTests, UID: os.Getuid(), GID: os.Getgid()})
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
		"config.yaml":           {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n  orchestrator_profile: default\n  max_in_progress: 1\ntoolsets: [kanban, memory]\nplatform_toolsets:\n  cli: [kanban, memory]\nterminal:\n  backend: local\n  cwd: /workspace\n"), Mode: 0600},
		"bin/" + id.Container:   {Data: script, Mode: 0700},
		"openviking/.gitignore": {Data: []byte("*\n"), Mode: 0600},
	}
	recipe, err := development.Recipe(report.Development)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for name, data := range recipe {
		artifacts["development-image/"+name] = install.Artifact{Data: data, Mode: 0600}
	}
	var previous []install.StackUpgrade
	for _, memory := range []string{"", projectmemory.Image} {
		old, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: memory, UID: os.Getuid(), GID: os.Getgid()})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, PrepareMemory: memory == ""})
	}
	if report.DockerTests {
		old, e := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, Development: &report.Development, UID: os.Getuid(), GID: os.Getgid()})
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		previous = append(previous, install.StackUpgrade{Compose: old, BackupName: "compose.before-docker-tests.yaml"})
	}
	created, err := install.PublishStackChecked(id, artifacts, previous, func() error {
		current := a.plan(id, false)
		if len(current.Unsupported) > 0 {
			return fmt.Errorf("selected integration became unqualified")
		}
		if len(current.Collisions) > 0 {
			return fmt.Errorf("target changed: %s", strings.Join(current.Collisions, "; "))
		}
		if current.Development.Go != report.Development.Go {
			return fmt.Errorf("repository toolchain requirements changed during installation")
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
		fmt.Fprintln(stdout, "Created Hermes with embedded OpenViking bootstrap artifacts; native setup is pending.")
	} else {
		fmt.Fprintln(stdout, "Preserved existing Hermes deployment and native configuration.")
	}
	if code := a.initialize(id, report.DockerContext, false, stdout, stderr); code != 0 {
		return code
	}
	// Only a complete scaffold can yield a generation. Initial publication and
	// pending native setup still report pending without starting any service.
	complete := true
	for _, probe := range verify.Profiles(id) {
		complete = complete && probe.Status == verify.Healthy
	}
	if ready, _ := a.nativeRuntimeReady(id, report.DockerContext); complete && ready {
		if code := a.finishSetup(id, report.DockerContext, 0, stdout, stderr); code != 0 {
			return code
		}
	}
	fmt.Fprintln(stdout, "Start Hermes with ordinary Compose:")
	fmt.Fprintln(stdout, strings.TrimSuffix(launcher.StartCommand(id.Compose, report.DockerContext), " up -d hermes)")+" up -d --build hermes)")
	fmt.Fprintln(stdout, "OpenViking runs inside Hermes; native memory setup remains pending until configured.")
	if report.DockerTests {
		fmt.Fprintln(stdout, "Docker acceptance is opt-in and privileged; its daemon owns only disposable test storage, not the host Docker socket.")
		fmt.Fprintln(stdout, strings.TrimSuffix(launcher.StartCommand(id.Compose, report.DockerContext), " up -d hermes)")+" --profile docker-tests up -d docker-test)")
		fmt.Fprintln(stdout, "After activation, authorized workers can run repokit-docker-test from /workspace; normal coding never needs this daemon.")
	}
	fmt.Fprintln(stdout, "Run hermes-repokit setup in your private terminal for native default, team and shared memory setup.")
	fmt.Fprintln(stdout, "Resume individual stages with setup --team or setup --memory. Rerun install to reconcile existing generated profiles.")
	return 0
}

func (a App) initialize(id target.Identity, dockerContext string, afterSetup bool, stdout, stderr io.Writer) int {
	ready, err := a.nativeRuntimeReady(id, dockerContext)
	if err != nil {
		fmt.Fprintln(stderr, "native initialization refused:", err)
		return 1
	}
	if !ready {
		fmt.Fprintln(stdout, "Native initialization pending: start the existing Compose service, then rerun install.")
		if afterSetup {
			return 1
		}
		return 0
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 2 * time.Minute}
	}
	if err := native.Initialize(context.Background(), id, dockerContext, afterSetup, runner); err != nil {
		if errors.Is(err, native.ErrTeamPending) && !afterSetup {
			fmt.Fprintln(stdout, "Native shared Kanban checked; team setup pending. After native default setup, run hermes-repokit setup --team without repeating login.")
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Native shared Kanban and six-profile team reconciled. Memory and model-driven acceptance are separate stages.")
	return 0
}

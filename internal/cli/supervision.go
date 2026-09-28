package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

func (a App) setupSupervision(id target.Identity, dockerContext string, stdout, stderr io.Writer) int {
	if dockerContext == "" || len(a.gitIssues(context.Background(), id)) > 0 {
		fmt.Fprintln(stderr, "supervision setup refused: deployment routing or private state protection is unverified")
		return 1
	}
	for _, p := range verify.Inspect(context.Background(), id, a.Runner) {
		if (p.Component == "compose" || p.Component == "hermes") && p.Status != verify.Healthy {
			fmt.Fprintln(stderr, "supervision setup requires generated Compose and the running pinned Hermes deployment")
			return 1
		}
	}
	for _, p := range verify.Profiles(id) {
		if p.Status != verify.Healthy {
			fmt.Fprintln(stderr, "supervision setup requires the six-profile scaffold; complete native default setup first")
			return 1
		}
	}
	if p := verify.Laya(context.Background(), id, a.Runner); p.Status != verify.Healthy {
		fmt.Fprintln(stderr, "supervision setup requires the qualified local Laya service; start Hermes and Laya together using Compose")
		return 1
	}
	r := a.Initializer
	if r == nil {
		r = process.Runner{Timeout: 20 * time.Minute}
	}
	fmt.Fprintln(stdout, "Checking real local Laya decisions and configuring pinned native Nerve for all six profiles.")
	if err := native.SetupSupervision(context.Background(), id, dockerContext, r); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// The shared network namespace requires coordinated recreation; restarting
	// Hermes alone can leave Laya attached to the old namespace. Native setup
	// first requires dispatch disabled and no running/review cards.
	fmt.Fprintln(stdout, "Recreating Hermes and Laya together to load the configured hooks.")
	restart := process.Runner{Timeout: 3 * time.Minute}.Run(context.Background(), "docker", "--context", dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose, "up", "-d", "--force-recreate", "hermes", "laya")
	if restart.Err != nil || restart.Truncated {
		fmt.Fprintln(stderr, "supervision configured; coordinated Compose recreation failed; inspect native service state and rerun setup --supervision")
		return 1
	}
	ready := false
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if verify.Laya(context.Background(), id, a.Runner).Status == verify.Healthy {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		fmt.Fprintln(stderr, "supervision configured; recreated local Laya did not become healthy")
		return 1
	}
	if err := native.SetupSupervision(context.Background(), id, dockerContext, r); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "All six profiles loaded native Nerve hooks and returned LOCAL_ONLY decisions after Compose recreation. Model-driven task acceptance remains separate.")
	return 0
}

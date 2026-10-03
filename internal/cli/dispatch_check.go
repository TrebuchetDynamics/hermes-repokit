package cli

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// dispatchCheck is the explicit mutating proof that the running gateway claims
// and completes work without a manual dispatch. Ordinary verify never runs it.
func (a App) dispatchCheck(out io.Writer) int {
	probe := verify.Probe{Component: "dispatch-check", Status: verify.Degraded}
	finish := func() int {
		if json.NewEncoder(out).Encode([]verify.Probe{probe}) != nil || probe.Status != verify.Healthy {
			return 1
		}
		return 0
	}
	id, err := target.Resolve(a.Directory)
	if err != nil {
		probe.Detail = "repository identity unavailable; no card created"
		return finish()
	}
	if len(target.Inspect(id, "")) != 0 || len(a.gitIssues(context.Background(), id)) != 0 {
		probe.Detail = "repository or private-state protection unverified; no card created"
		return finish()
	}
	dc, err := launcher.Context(id)
	if err != nil {
		probe.Detail = "recognized deployment launcher unavailable; no card created"
		return finish()
	}
	if ready, err := a.nativeRuntimeReady(id, dc); err != nil || !ready {
		probe.Detail = "pinned running Hermes container not verified; no card created"
		return finish()
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 2 * time.Minute}
	}
	task, err := native.DispatchCheck(context.Background(), id, dc, runner, 150*time.Second, 6*time.Minute, 3*time.Second)
	if err != nil {
		probe.Detail = err.Error()
		if task != "" {
			probe.Detail += " (" + task + ")"
		}
		return finish()
	}
	probe.Status = verify.Healthy
	probe.Detail = "gateway automatically ran canary card " + task + " with the expected no-write answer; card archived"
	return finish()
}

package native

import (
	"context"
	"errors"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// ErrRuntimeStarting means the container is running but the Hermes CLI does not
// answer yet. On a first boot Hermes remaps its user and fixes data ownership,
// and a command issued before that finishes cannot even read /opt/data.
var ErrRuntimeStarting = errors.New("Hermes runtime is still starting or unhealthy: the native CLI did not answer; wait a moment and retry")

// WaitForCLI waits a bounded time for a harmless public read to succeed before
// RepoKit issues native commands. An unset key still proves the CLI works.
func WaitForCLI(ctx context.Context, id target.Identity, dc string, r InputRunner, timeout, pause time.Duration) error {
	run := nativeTeamCLI(ctx, id, dc, r)
	deadline := time.Now().Add(timeout)
	for {
		if _, err := run("-p", "default", "config", "get", "kanban.dispatch_in_gateway", "--json"); err == nil {
			return nil
		}
		if !time.Now().Before(deadline) {
			return ErrRuntimeStarting
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pause):
		}
	}
}

// RunningWork reports whether any Kanban card is running, through the public
// `kanban stats --json`. Callers use it to avoid interrupting live work.
func RunningWork(ctx context.Context, id target.Identity, dc string, r InputRunner) (bool, error) {
	return runningWork(nativeTeamCLI(ctx, id, dc, r))
}

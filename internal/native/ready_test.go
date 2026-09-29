package native

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// bootingCLI fails the first `failures` CLI reads the way a first boot does.
type bootingCLI struct {
	failures, calls int
	output          string
}

func (b *bootingCLI) RunInput(_ context.Context, _ io.Reader, _ string, _ ...string) process.Result {
	b.calls++
	if b.calls <= b.failures {
		return process.Result{Err: errors.New("exit 1"), Output: "PermissionError: [Errno 13] Permission denied: '/opt/data/.env'"}
	}
	if b.output != "" {
		return process.Result{Err: errors.New("exit 1"), Output: b.output}
	}
	return process.Result{Output: "false"}
}

func TestWaitForCLIRidesOutFirstBoot(t *testing.T) {
	id := dispatchTestIdentity(t)
	cli := &bootingCLI{failures: 3}
	if err := WaitForCLI(context.Background(), id, "local", cli, time.Second, time.Millisecond); err != nil || cli.calls != 4 {
		t.Fatalf("boot not awaited: %v after %d calls", err, cli.calls)
	}
}

func TestWaitForCLIIsBoundedAndSpecific(t *testing.T) {
	id := dispatchTestIdentity(t)
	cli := &bootingCLI{failures: 1 << 30}
	start := time.Now()
	err := WaitForCLI(context.Background(), id, "local", cli, 50*time.Millisecond, 5*time.Millisecond)
	if !errors.Is(err, ErrRuntimeStarting) || time.Since(start) > 2*time.Second {
		t.Fatalf("unbounded or generic failure: %v", err)
	}
}

func TestWaitForCLIAcceptsUnsetKeyAsAnswer(t *testing.T) {
	id := dispatchTestIdentity(t)
	cli := &bootingCLI{output: "Config key not set: kanban.dispatch_in_gateway\n"}
	if err := WaitForCLI(context.Background(), id, "local", cli, time.Second, time.Millisecond); err != nil || cli.calls != 1 {
		t.Fatalf("working CLI with an unset key treated as booting: %v", err)
	}
}

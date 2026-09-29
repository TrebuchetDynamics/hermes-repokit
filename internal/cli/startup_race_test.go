package cli

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// firstBoot fails public CLI reads until Hermes finishes its first boot, then
// delegates; scripts count as native mutations.
type firstBoot struct {
	next            *foundationRunner
	failures, reads int
	scripts         int
}

func (f *firstBoot) RunInput(ctx context.Context, input io.Reader, program string, args ...string) process.Result {
	if input == nil {
		f.reads++
		if f.reads <= f.failures {
			return process.Result{Err: errors.New("exit 1"), Output: "PermissionError: [Errno 13] Permission denied: '/opt/data/.env'"}
		}
	} else {
		f.scripts++
	}
	return f.next.RunInput(ctx, input, program, args...)
}

func TestInstallRightAfterStartWaitsForHermes(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	boot := &firstBoot{next: r, failures: 3}
	a.Initializer = boot
	saved := nativeReadyTimeout
	defer func() { nativeReadyTimeout = saved }()
	nativeReadyTimeout = 30 * time.Second
	code, out, diag := invoke(t, a, "install")
	if code != 0 || !strings.Contains(out, "team setup pending") || strings.Contains(diag, "inspection unavailable") {
		t.Fatalf("install during first boot: code=%d out=%s diag=%s", code, out, diag)
	}
	if boot.reads <= boot.failures || boot.scripts == 0 {
		t.Fatalf("native initialization did not run after the wait: reads=%d scripts=%d", boot.reads, boot.scripts)
	}
}

func TestInstallReportsStartingRuntimeWithoutMutating(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	boot := &firstBoot{next: r, failures: 1 << 30}
	a.Initializer = boot
	saved := nativeReadyTimeout
	defer func() { nativeReadyTimeout = saved }()
	nativeReadyTimeout = 20 * time.Millisecond
	code, _, diag := invoke(t, a, "install")
	if code == 0 || !strings.Contains(diag, "still starting") || strings.Contains(diag, "inspection unavailable") || boot.scripts != 0 {
		t.Fatalf("stuck runtime: code=%d scripts=%d diag=%s", code, boot.scripts, diag)
	}
}

package native

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// idleGuard's exit 3 means work started after Go saw an idle board; that is a
// deferral, not a failed initialization.
func TestIdleGuardExitIsReportedAsWorkStarted(t *testing.T) {
	exit := func(code string) error { return exec.Command("sh", "-c", "exit "+code).Run() }
	if err := bootstrapError(process.Result{Err: exit("3")}); !errors.Is(err, ErrWorkStarted) {
		t.Fatalf("exit 3: %v", err)
	}
	for _, r := range []process.Result{{Err: exit("1")}, {Err: exit("3"), Truncated: true}, {Err: errors.New("killed")}} {
		if err := bootstrapError(r); err == nil || errors.Is(err, ErrWorkStarted) {
			t.Fatalf("%+v: %v", r, err)
		}
	}
	if bootstrapError(process.Result{Output: "ok"}) != nil {
		t.Fatal("success reported as an error")
	}
}

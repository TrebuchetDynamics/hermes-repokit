package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// diskRunner reports the deployment image as not built yet and serves a
// listing of the deployment's own images.
type diskRunner struct {
	*foundationRunner
	images, used string
	removed      []string
}

func (r *diskRunner) Run(ctx context.Context, program string, args ...string) process.Result {
	call := strings.Join(args, " ")
	switch {
	case strings.Contains(call, " image inspect ") && strings.Contains(call, "repokit/"):
		return process.Result{Err: fmt.Errorf("no such image")}
	case strings.Contains(call, " image ls "):
		return process.Result{Output: r.images}
	case strings.Contains(call, " container ls --all --format {{.Image}}"):
		return process.Result{Output: r.used}
	case strings.Contains(call, " image rm "):
		r.removed = append(r.removed, args[len(args)-1])
		return process.Result{}
	}
	return r.foundationRunner.Run(ctx, program, args...)
}

// A build that would fill the disk is refused before Docker starts it.
func TestStartRefusesToBuildWithoutDiskSpace(t *testing.T) {
	a, base := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	a.Runner = &diskRunner{foundationRunner: base}
	a.FreeSpace = func(string) (uint64, bool) { return 2 << 30, true }
	base.composeCalls = nil
	code, out, diag := invoke(t, a, "start")
	if code == 0 || len(base.composeCalls) != 0 || !strings.Contains(out+diag, "GB free where Docker keeps images") {
		t.Fatalf("low-disk build not refused: code=%d calls=%v\n%s%s", code, base.composeCalls, out, diag)
	}
	a.FreeSpace = func(string) (uint64, bool) { return 0, false } // remote daemon: unknown
	base.composeCalls = nil
	if code, out, diag := invoke(t, a, "start"); len(base.composeCalls) == 0 {
		t.Fatalf("unknown free space blocked the build: code=%d\n%s%s", code, out, diag)
	}
}

// After the current image runs, the deployment's earlier images go, but never
// another repository's image or one a container still uses.
func TestRemoveSupersededKeepsCurrentAndUsedImages(t *testing.T) {
	a, base := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	data, err := os.ReadFile(base.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	current := composeImage.FindStringSubmatch(string(data))[1]
	repo := "repokit/" + base.id.Container
	r := &diskRunner{foundationRunner: base,
		images: strings.Join([]string{current, repo + ":old1", repo + ":stopped", "repokit/hermes-other:x"}, "\n"),
		used:   repo + ":stopped\n"}
	a.Runner = r
	if n := a.removeSuperseded(base.id, "local-test"); n != 1 || len(r.removed) != 1 || r.removed[0] != repo+":old1" {
		t.Fatalf("removed %d: %v", n, r.removed)
	}
}

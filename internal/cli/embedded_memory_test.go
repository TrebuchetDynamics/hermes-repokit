package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

type sidecarRunner struct {
	*foundationRunner
	result process.Result
}

func (r sidecarRunner) Run(ctx context.Context, program string, args ...string) process.Result {
	if program == "docker" && strings.Contains(strings.Join(args, " "), "label=com.docker.compose.service=openviking") {
		return r.result
	}
	return r.foundationRunner.Run(ctx, program, args...)
}

func TestEmbeddedInstallRefusesRunningOrUnobservableLegacyWriter(t *testing.T) {
	for _, result := range []process.Result{{Output: "abcdef123456"}, {Err: fmt.Errorf("daemon unavailable")}, {Truncated: true}} {
		a, r := foundationApp(t)
		a.Runner = sidecarRunner{r, result}
		code, _, diag := invoke(t, a, "install")
		if code == 0 || !strings.Contains(diag, "previous OpenViking sidecar") {
			t.Fatalf("%d %s", code, diag)
		}
		if _, err := os.Stat(r.id.Compose); !os.IsNotExist(err) {
			t.Fatal("published new topology before stopping old writer")
		}
	}
}

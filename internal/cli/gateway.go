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

// finishSetup is the single activation boundary after a successful stage.
// Failed stages never enable dispatch.
func (a App) finishSetup(id target.Identity, dc string, code int, out, diag io.Writer) int {
	if code != 0 {
		return code
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 3 * time.Minute}
	}
	state, err := native.ConvergeGateway(context.Background(), id, dc, runner, a.startGateway)
	if err != nil {
		fmt.Fprintln(diag, "Native state saved; automatic dispatch not enabled:", err)
		return 1
	}
	switch state {
	case "current":
		fmt.Fprintln(out, "Automatic dispatch already configured on the running default gateway; nothing changed.")
	case "restarted":
		fmt.Fprintln(out, "Configured native automatic dispatch on default (review dispatch, seven-profile allowlist, max_in_progress=1, auto_decompose=false) and restarted the gateway.")
		fmt.Fprintln(out, "Start a fresh conversation (/new in Telegram) so sessions see current tools.")
	case "started":
		fmt.Fprintln(out, "Started the default gateway; Hermes keeps it running across container restarts.")
	case "not-running":
		fmt.Fprintf(out, "Automatic dispatch is configured, but the default gateway is not running; messaging and dispatch start with it: %s -p default gateway start\n", id.Container)
	}
	fmt.Fprintln(out, "No worker has been exercised by setup. Prove the loop with `hermes-repokit verify --dispatch-check` (one researcher card, model cost) or a real reviewed task.")
	for _, p := range verify.RuntimeIntegrations(context.Background(), id, a.Runner) {
		if p.Component == "memory" {
			fmt.Fprintf(diag, "Optional memory: %s. %s. Core dispatch does not depend on it; memory is user-managed.\n", p.Status, p.Detail)
		}
	}
	return 0
}

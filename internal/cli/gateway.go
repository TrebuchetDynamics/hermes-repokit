package cli

import (
	"context"
	"io"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
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
	u := newUI(out, diag)
	state, err := native.ConvergeGateway(context.Background(), id, dc, runner, a.startGateway)
	if err != nil {
		u.fail("native state saved; automatic dispatch not enabled: %v", err)
		return 1
	}
	switch state {
	case "current":
		u.ok("Dispatch", "automatic dispatch already configured; nothing changed")
	case "restarted":
		u.ok("Dispatch", "native automatic dispatch configured on default (review, seven-profile allowlist, one card at a time)")
		u.ok("Gateway", "restarted")
		u.note("start a fresh conversation (/new in Telegram) so sessions see current tools")
	case "started":
		u.ok("Gateway", "default gateway started; Hermes keeps it running across restarts")
	case "not-running":
		u.pending("Gateway", "not running; start it: "+id.Container+" -p default gateway start")
	}
	u.note("no worker has run yet; prove the loop with repokit verify --dispatch-check or a real task")
	return 0
}

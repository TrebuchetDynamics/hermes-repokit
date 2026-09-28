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

// finishSetup is the single convergence boundary after a successful stage or
// the entire full-setup sequence; failed stages never publish a fresh receipt.
func (a App) finishSetup(id target.Identity, dc string, code int, out, diag io.Writer) int {
	if code != 0 {
		return code
	}
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 12 * time.Minute}
	}
	if err := native.ProvisionMaintenance(context.Background(), id, dc, runner); err != nil {
		fmt.Fprintln(diag, "Native state saved; maintenance plugin admission incomplete. Scanner rejection or owner source drift must be resolved before activation; no gateway restart performed.")
		return 1
	}
	if !operationalIntegrationsReady(verify.RuntimeIntegrations(context.Background(), id, a.Runner)) {
		fmt.Fprintln(diag, "Operational dispatch pending: shared OpenViking memory is mandatory. Complete setup --memory, then rerun setup --team. Dispatch remains disabled.")
		return 1
	}
	state, err := native.ConvergeGateway(context.Background(), id, dc, runner)
	if err != nil {
		fmt.Fprintln(diag, "Native state saved; gateway convergence incomplete. Check active work, native health and owner drift, then rerun the same setup stage.")
		return 1
	}
	if state == "not-running" {
		fmt.Fprintln(out, "Gateway not running; messaging is not ready. Start the default gateway through the native launcher when ready.")
	} else {
		fmt.Fprintln(out, "Gateway generation current. Start a fresh Hermes conversation (/new in Telegram) to refresh session identity and tools.")
	}
	fmt.Fprintln(out, "Dispatch operational: default gateway owns automatic execution and review; six-profile allowlist, max_in_progress=1, auto_decompose=false. Researcher canary completed through the gateway.")
	return 0
}

// Memory is mandatory in the normal RepoKit operational policy. Configuration
// alone cannot admit a disconnected memory service.
func operationalIntegrationsReady(probes []verify.Probe) bool {
	wanted := map[string]verify.Status{"memory": verify.Active}
	for _, p := range probes {
		if status, ok := wanted[p.Component]; ok {
			if p.Status != status {
				return false
			}
			delete(wanted, p.Component)
		}
	}
	return len(wanted) == 0
}

func (a App) prepareDispatch(id target.Identity, dc string, diag io.Writer) int {
	runner := a.Initializer
	if runner == nil {
		runner = process.Runner{Timeout: 6 * time.Minute}
	}
	if err := native.PrepareDispatch(context.Background(), id, dc, runner); err != nil {
		fmt.Fprintln(diag, "Setup deferred: cannot safely suspend the native dispatcher. Finish active work and inspect gateway health; queued cards are preserved.")
		return 1
	}
	return 0
}

package native

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/gateway"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed dispatch.py
var dispatchScript string

func dispatchInitializationScript(id target.Identity, action string) string {
	payload, _ := json.Marshal(map[string]any{"roles": team.ForRepository(id), "repo_id": id.Project, "action": action, "integrations_ready": action == "activate"})
	return bootstrapScript + "\n/opt/hermes/.venv/bin/python -B - <<'REPOKIT_GATEWAY_PY'\n" + team.KanbanPolicy + "\n" + teamScript + "\n" + gateway.SupportScript() + "\n" + dispatchScript + "\nimport base64\nintegration_scope={'__name__':'repokit_setup_checks'}\nexec(base64.b64decode('" + base64.StdEncoding.EncodeToString([]byte(verify.IntegrationCheckSource())) + "'), integration_scope)\nimport base64\ntry:\n    dispatch_main(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\nexcept Exception:\n    print('Operational dispatch incomplete; inspect native gates, active work, gateway startup and canary evidence. Native state preserved.', file=sys.stderr)\n    sys.exit(1)\nREPOKIT_GATEWAY_PY\n"
}

// PrepareDispatch suspends an existing managed dispatcher under native board
// locks before any setup mutation. It never claims a card or interrupts a worker.
func PrepareDispatch(ctx context.Context, id target.Identity, dc string, r InputRunner) error {
	result, err := runBootstrap(ctx, id, dc, false, dispatchInitializationScript(id, "prepare"), r)
	if err != nil || !strings.Contains(result.Output, "REPOKIT_DISPATCH=prepared") {
		return fmt.Errorf("dispatch suspension unverified; active workers and owner policy preserved")
	}
	return nil
}

// ConvergeGateway activates only after the caller has checked required runtime
// integrations. Native gates are rechecked under the bootstrap lock. A matching
// live generation and successful canary are reused; otherwise the native gateway
// restarts and executes a real no-write researcher canary.
func ConvergeGateway(ctx context.Context, id target.Identity, dc string, r InputRunner) (string, error) {
	result, err := runBootstrap(ctx, id, dc, false, dispatchInitializationScript(id, "activate"), r)
	if err != nil {
		return "", err
	}
	if strings.Contains(result.Output, "REPOKIT_GATEWAY=current") && strings.Contains(result.Output, "REPOKIT_CANARY=researcher-done") {
		return "current", nil
	}
	return "", fmt.Errorf("gateway dispatch/canary did not establish operational readiness")
}

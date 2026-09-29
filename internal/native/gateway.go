package native

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/gateway"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed dispatch.py
var dispatchScript string

var dispatchFailurePattern = regexp.MustCompile(`^(RuntimeError|ValueError|OSError|TimeoutExpired|CalledProcessError|Exception)\|((dispatch_main|check_activation|switch_gateway|dispatcher_canary|dispatch_fence|startup_since|log_cursor|create_dispatch_canary|retry_dispatch_canary|suspend_failed_activation|release_dispatch):[1-9][0-9]{0,5}(>(dispatch_main|check_activation|switch_gateway|dispatcher_canary|dispatch_fence|startup_since|log_cursor|create_dispatch_canary|retry_dispatch_canary|suspend_failed_activation|release_dispatch):[1-9][0-9]{0,5}){0,7}|unknown:0)$`)

type dispatchFailure struct{ diagnostic string }

func (e *dispatchFailure) Error() string { return "native dispatch failed: " + e.diagnostic }

// DispatchDiagnostic exposes only validated stage names, exception categories,
// and source line numbers; native output and exception values remain private.
func DispatchDiagnostic(err error) string {
	var failure *dispatchFailure
	if errors.As(err, &failure) {
		return failure.diagnostic
	}
	return ""
}

func dispatchFailureFromOutput(output string) error {
	for _, line := range strings.Split(output, "\n") {
		value, ok := strings.CutPrefix(line, "REPOKIT_DISPATCH_FAILURE=")
		if ok && dispatchFailurePattern.MatchString(value) {
			return &dispatchFailure{diagnostic: value}
		}
	}
	return nil
}

func dispatchInitializationScript(id target.Identity, action string) string {
	payload, _ := json.Marshal(map[string]any{"roles": team.ForRepository(id), "repo_id": id.Project, "action": action})
	return bootstrapScript + "\n/opt/hermes/.venv/bin/python -B - <<'REPOKIT_GATEWAY_PY'\n" + team.KanbanPolicy + "\n" + teamScript + "\n" + gateway.SupportScript() + "\n" + dispatchScript + "\nimport base64\ntry:\n    dispatch_main(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\nexcept Exception as error:\n    print('REPOKIT_DISPATCH_FAILURE='+dispatch_failure_diagnostic(error), file=sys.stderr)\n    print('Operational dispatch incomplete; inspect native gates, active work, gateway startup and canary evidence. Native state preserved.', file=sys.stderr)\n    sys.exit(1)\nREPOKIT_GATEWAY_PY\n"
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

// ConvergeGateway checks core native gates under the bootstrap lock. Optional
// memory health is reported independently by the caller. A matching
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

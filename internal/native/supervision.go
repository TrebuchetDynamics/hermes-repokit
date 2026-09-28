package native

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/supervision"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

//go:embed supervision.py
var supervisionScript string

// SetupSupervision is mutating setup: native admission, pinned source checks,
// configure-before-enable and actual local inference in every profile scope.
// Callers verify the selected deployment/sidecar first. Output never contains
// native logs or decision contexts; those belong to private native state.
func SetupSupervision(ctx context.Context, id target.Identity, dockerContext string, r InputRunner) error {
	var roles []string
	for _, role := range team.Roster() {
		roles = append(roles, role.Name)
	}
	payload, _ := json.Marshal(map[string]any{"roles": roles, "settings": supervision.LocalLayaSettings(), "revision": supervision.NerveRevision})
	script := bootstrapScript + "\npython - <<'REPOKIT_SUPERVISION_PY'\n" + supervision.CheckoutScript + "\n" + supervisionScript + "\nimport base64\ntry:\n    main(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\nexcept Exception:\n    print('Native supervision setup incomplete: inspect local Laya, native plugin admission, revision and profile drift.',file=sys.stderr)\n    sys.exit(1)\nREPOKIT_SUPERVISION_PY\n"
	result, err := runBootstrap(ctx, id, dockerContext, false, script, r)
	if err != nil || !strings.Contains(result.Output, "REPOKIT_SUPERVISION all profiles loaded native hooks and LOCAL_ONLY decisions") {
		return fmt.Errorf("native supervision setup incomplete; local Laya, pinned plugin admission or profile drift requires inspection; no hosted fallback was configured")
	}
	return nil
}

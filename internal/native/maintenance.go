package native

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/packaging/maintenance"
)

//go:embed maintenance.py
var maintenanceScript string

func maintenanceInitializationScript() string {
	payload, _ := json.Marshal(struct {
		Name  string            `json:"name"`
		Files map[string]string `json:"files"`
	}{maintenance.Name, maintenance.Files()})
	return bootstrapScript + "\n/opt/hermes/.venv/bin/python -B - <<'REPOKIT_MAINTENANCE_PY'\n" +
		team.KanbanPolicy + "\n" + teamScript + "\n" + maintenanceScript +
		"\nimport base64\ntry:\n    maintenance_main(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\n" +
		"except Exception:\n    print('Native maintenance provisioning failed; preserve plugin state and inspect native scanner or drift diagnostics privately.', file=sys.stderr)\n    sys.exit(1)\nREPOKIT_MAINTENANCE_PY\n"
}

// ProvisionMaintenance installs the exact bundled native plugin using the native
// scanner and installer under the same container-held bootstrap writer lock.
// An absent receipt is failure; local file presence never establishes readiness.
func ProvisionMaintenance(ctx context.Context, id target.Identity, dockerContext string, r InputRunner) error {
	result, err := runBootstrap(ctx, id, dockerContext, false, maintenanceInitializationScript(), r)
	if err != nil {
		return fmt.Errorf("native maintenance provisioning failed; plugin scan admission and configuration are unconfirmed")
	}
	for _, line := range strings.Split(result.Output, "\n") {
		if line == "REPOKIT_MAINTENANCE=configured" {
			return nil
		}
	}
	return fmt.Errorf("native maintenance provisioning did not establish scanner admission and configuration")
}

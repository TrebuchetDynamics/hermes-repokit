package native

import (
	"context"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// DefaultModelConfigured reports whether Hermes's own private setup has
// already chosen a model for default, read through public `config get`. Only
// presence is observed; the value is never returned or printed.
func DefaultModelConfigured(ctx context.Context, id target.Identity, dockerContext string, r InputRunner) (bool, error) {
	model, err := configValue(nativeTeamCLI(ctx, id, dockerContext, r), "default", "model.default")
	if err != nil {
		return false, err
	}
	name, _ := model.(string)
	return name != "", nil
}

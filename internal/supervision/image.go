package supervision

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

type Runner interface {
	Run(context.Context, string, ...string) process.Result
}

const imageFormat = `{"id":{{json .Id}},"os":{{json .Os}},"arch":{{json .Architecture}},"recipe":{{json (index .Config.Labels "io.repokit.laya.recipe")}},"nerve":{{json (index .Config.Labels "io.repokit.nerve.revision")}},"model":{{json (index .Config.Labels "io.repokit.laya.model-revision")}}}`

func CheckImage(ctx context.Context, dockerContext, image string, r Runner) error {
	if !compose.LocalImageID(image) {
		return fmt.Errorf("Laya selection requires a full local content image ID")
	}
	result := r.Run(ctx, "docker", "--context", dockerContext, "image", "inspect", "--format", imageFormat, image)
	var observed struct{ ID, OS, Arch, Recipe, Nerve, Model string }
	if result.Err != nil || result.Truncated || json.Unmarshal([]byte(result.Output), &observed) != nil || observed.ID != image || observed.OS != "linux" || observed.Arch != "amd64" || observed.Recipe != "1" || observed.Nerve != NerveRevision || observed.Model != LayaModelRevision {
		return fmt.Errorf("local Laya image does not match the qualified Linux amd64 recipe; build packaging/laya first")
	}
	return nil
}

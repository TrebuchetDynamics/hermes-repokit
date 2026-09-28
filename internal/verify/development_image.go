package verify

import (
	"context"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// HermesImageMatches checks the selected derived image by content ID, exact
// generated recipe and immutable base layers. A mutable local tag alone proves
// nothing. It observes Docker metadata only and never builds/pulls images.
func HermesImageMatches(ctx context.Context, id target.Identity, dc, image, imageID string, r Runner) bool {
	o, selected := compose.DevelopmentSelected(id)
	if !selected {
		return image == qualification.FoundationImage
	}
	if image != development.ImageName(id.Project, *o.Development) || !compose.LocalImageID(imageID) || !compose.DevelopmentRecipeMatches(id, *o.Development) {
		return false
	}
	const format = `{"id":{{json .Id}},"os":{{json .Os}},"arch":{{json .Architecture}},"recipe":{{json (index .Config.Labels "org.repokit.development.recipe")}},"base":{{json (index .Config.Labels "org.repokit.hermes.base")}},"layers":{{json .RootFS.Layers}}}`
	var built struct {
		ID, OS, Arch, Recipe, Base string
		Layers                     []string
	}
	out := r.Run(ctx, "docker", "--context", dc, "image", "inspect", "--format", format, image)
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &built) != nil || built.ID != imageID || built.OS != "linux" || built.Arch != "amd64" || built.Recipe != development.Fingerprint(*o.Development) || built.Base != qualification.FoundationImage {
		return false
	}
	base := r.Run(ctx, "docker", "--context", dc, "image", "inspect", "--format", "{{json .RootFS.Layers}}", qualification.FoundationImage)
	var layers []string
	if base.Err != nil || base.Truncated || json.Unmarshal([]byte(base.Output), &layers) != nil || len(layers) == 0 || len(built.Layers) <= len(layers) {
		return false
	}
	for i, layer := range layers {
		if !compose.LocalImageID(layer) || built.Layers[i] != layer {
			return false
		}
	}
	return true
}

type RuntimeMount struct {
	Type, Source, Destination, Name string
	RW                              bool
}

func RuntimeMountsMatch(id target.Identity, unexpected string, mounts []RuntimeMount) bool {
	o, selected := compose.DevelopmentSelected(id)
	if !selected {
		return unexpected == ""
	}
	need := 2
	if o.DockerTests {
		need = 4
	}
	if len(mounts) != need {
		return false
	}
	seen := map[string]bool{}
	for _, m := range mounts {
		if seen[m.Destination] || (m.Destination != "/docker-test/run" && !m.RW) {
			return false
		}
		seen[m.Destination] = true
		switch m.Destination {
		case "/workspace":
			if m.Type != "bind" || m.Source != id.Root {
				return false
			}
		case "/opt/data":
			if m.Type != "bind" || m.Source != id.Root+"/.hermes" {
				return false
			}
		case "/docker-test/run":
			if !o.DockerTests || m.RW || m.Type != "volume" || m.Name != id.Project+"_docker-test-run" {
				return false
			}
		case "/docker-tests":
			if !o.DockerTests || m.Type != "volume" || m.Name != id.Project+"_docker-test-work" {
				return false
			}
		default:
			return false
		}
	}
	return seen["/workspace"] && seen["/opt/data"]
}

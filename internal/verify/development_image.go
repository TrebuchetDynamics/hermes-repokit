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
	if image != development.ImageName(id.Container, *o.Development) {
		return false
	}
	return developmentImageContentMatches(ctx, id, dc, image, imageID, *o.Development, r)
}

// PreviousNamedImageMatches is only an installation transition check. A proven
// image with the old tag can await Compose recreation; it is never reported as
// a ready current runtime and no native setup runs against it.
func PreviousNamedImageMatches(ctx context.Context, id target.Identity, dc, image, imageID string, r Runner) bool {
	o, selected := compose.DevelopmentSelected(id)
	if !selected || image != id.Project+"-hermes-dev:"+development.Fingerprint(*o.Development)[:24] {
		return false
	}
	return developmentImageContentMatches(ctx, id, dc, image, imageID, *o.Development, r)
}

func developmentImageContentMatches(ctx context.Context, id target.Identity, dc, image, imageID string, req development.Requirements, r Runner) bool {
	if !compose.LocalImageID(imageID) || !compose.DevelopmentRecipeMatches(id, req) {
		return false
	}
	const format = `{"id":{{json .Id}},"os":{{json .Os}},"arch":{{json .Architecture}},"recipe":{{json (index .Config.Labels "org.repokit.development.recipe")}},"base":{{json (index .Config.Labels "org.repokit.hermes.base")}},"layers":{{json .RootFS.Layers}}}`
	var built struct {
		ID, OS, Arch, Recipe, Base string
		Layers                     []string
	}
	out := r.Run(ctx, "docker", "--context", dc, "image", "inspect", "--format", format, image)
	if out.Err != nil || out.Truncated || json.Unmarshal([]byte(out.Output), &built) != nil || built.ID != imageID || built.OS != "linux" || built.Arch != "amd64" || built.Recipe != development.Fingerprint(req) || built.Base != qualification.FoundationImage {
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
	return mountsMatch(id, o, mounts)
}

// PriorRuntimeMountsMatch also accepts a container created before the
// toolchain cache volume existed. It qualifies only a pending recreation of
// an older generated image, never a current runtime.
func PriorRuntimeMountsMatch(id target.Identity, unexpected string, mounts []RuntimeMount) bool {
	if RuntimeMountsMatch(id, unexpected, mounts) {
		return true
	}
	o, selected := compose.DevelopmentSelected(id)
	if !selected || !compose.ToolchainCacheMounted(o) {
		return false
	}
	o.ToolchainCache = false
	return mountsMatch(id, o, mounts)
}

func mountsMatch(id target.Identity, o compose.Options, mounts []RuntimeMount) bool {
	need := 2
	if o.DockerTests {
		need += 2
	}
	if compose.ToolchainCacheMounted(o) {
		need++
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
		case compose.ToolchainCacheTarget:
			if !compose.ToolchainCacheMounted(o) || m.Type != "volume" || m.Name != id.Project+"_"+compose.ToolchainCacheVolume {
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

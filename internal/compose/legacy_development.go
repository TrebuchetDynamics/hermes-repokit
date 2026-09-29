package compose

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// LegacyDevelopment renders the pre-readable-names Compose for an older
// generated recipe identified by its self-certified fingerprint. Recognizing
// Compose is not adoption: the recipe bytes are validated before publication.
func LegacyDevelopment(id target.Identity, o Options, fingerprint string) ([]byte, error) {
	return olderRecipe(target.PreviousNames(id), o, id.Project+"-hermes-dev:", fingerprint)
}

// OlderRecipe renders current Compose for an older self-certified recipe.
func OlderRecipe(id target.Identity, o Options, fingerprint string) ([]byte, error) {
	return olderRecipe(id, o, "repokit/"+id.Container+":", fingerprint)
}

func olderRecipe(id target.Identity, o Options, prefix, fingerprint string) ([]byte, error) {
	if o.Development == nil || len(fingerprint) != 64 {
		return nil, fmt.Errorf("older development Compose requires a recipe fingerprint")
	}
	data, err := Render(id, o)
	if err != nil {
		return nil, err
	}
	current := []byte(fmt.Sprintf("    image: %q\n", development.ImageName(id.Container, *o.Development)))
	prior := []byte(fmt.Sprintf("    image: %q\n", prefix+fingerprint[:24]))
	return bytes.Replace(data, current, prior, 1), nil
}

// PreviousNames is the exact generated Compose immediately before readable
// public names. The recipe fingerprint and stable project ID are unchanged.
func PreviousNames(id target.Identity, o Options) ([]byte, error) {
	id = target.PreviousNames(id)
	data, err := Render(id, o)
	if err != nil || o.Development == nil {
		return data, err
	}
	current := []byte(fmt.Sprintf("    image: %q\n", development.ImageName(id.Container, *o.Development)))
	prior := []byte(fmt.Sprintf("    image: %q\n", id.Project+"-hermes-dev:"+development.Fingerprint(*o.Development)[:24]))
	return bytes.Replace(data, current, prior, 1), nil
}

// DevelopmentInstallSelected preserves installation options only. Runtime
// qualification continues to recognize the current generated recipe alone.
func DevelopmentInstallSelected(id target.Identity) (Options, bool) {
	data := generatedData(id)
	recipe, fingerprint, generated := development.ReadGeneratedRecipe(filepath.Join(id.Root, ".hermes", "development-image"))
	// Older renders share one on-disk image tag, so the toolchain selection
	// must come from the recipe itself rather than from the Compose bytes.
	recipeGo := bytes.Contains(recipe["Dockerfile"], []byte("https://go.dev/dl/go"))
	for _, goTool := range []bool{false, true} {
		for _, tests := range []bool{false, true} {
			for _, memory := range []string{"", projectmemory.Image} {
				for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
					req := development.Requirements{Go: goTool}
					o := Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: tests, OpenVikingImage: memory, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}
					renders := []func(target.Identity, Options) ([]byte, error){Render, PreviousNames}
					if generated && goTool == recipeGo {
						for _, older := range []func(target.Identity, Options, string) ([]byte, error){LegacyDevelopment, OlderRecipe} {
							renders = append(renders, func(id target.Identity, o Options) ([]byte, error) { return older(id, o, fingerprint) })
						}
					}
					for _, render := range renders {
						expected, err := render(id, o)
						if err == nil && bytes.Equal(data, expected) {
							return o, true
						}
					}
				}
			}
		}
	}
	return Options{}, false
}

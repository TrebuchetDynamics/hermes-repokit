package compose

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// OlderRecipe renders the current Compose for a different self-certified
// recipe, identified by its fingerprint. It covers reconfiguring a current
// deployment, such as a repository gaining a go.mod; the recipe bytes are
// validated before anything is replaced.
func OlderRecipe(id target.Identity, o Options, fingerprint string) ([]byte, error) {
	if o.Development == nil || len(fingerprint) != 64 {
		return nil, fmt.Errorf("older development Compose requires a recipe fingerprint")
	}
	data, err := Render(id, o)
	if err != nil {
		return nil, err
	}
	current := []byte(fmt.Sprintf("    image: %q\n", development.ImageName(id.Container, *o.Development)))
	prior := []byte(fmt.Sprintf("    image: %q\n", "repokit/"+id.Container+":"+fingerprint[:24]))
	return bytes.Replace(data, current, prior, 1), nil
}

// BeforeToolchainCache renders the Compose v0.2.0 generated for a Go
// repository: the older recipe's image tag and no toolchain-cache volume. It
// lets install upgrade the previous release's Go deployments in place; every
// earlier layout stays unrecognized.
func BeforeToolchainCache(id target.Identity, o Options, fingerprint string) ([]byte, error) {
	if o.Development == nil || !o.Development.Go {
		return nil, fmt.Errorf("the pre-toolchain-cache render exists only for Go deployments")
	}
	o.beforeToolchainCache = true
	o.beforeStateMask = true
	return OlderRecipe(id, o, fingerprint)
}

// BeforeStateMask renders the Compose the previous release generated: an
// older recipe's image tag and .hermes still visible inside /workspace. It lets
// install upgrade those deployments in place.
func BeforeStateMask(id target.Identity, o Options, fingerprint string) ([]byte, error) {
	o.beforeStateMask = true
	return OlderRecipe(id, o, fingerprint)
}

// DevelopmentInstallSelected recovers the installation options of a current
// generated Compose, including one awaiting a recipe change. Runtime
// qualification continues to recognize the current generated recipe alone.
func DevelopmentInstallSelected(id target.Identity) (Options, bool) {
	data := generatedData(id)
	recipe, fingerprint, generated := development.ReadGeneratedRecipe(filepath.Join(id.Root, ".hermes", "development-image"))
	// A recipe awaiting replacement determines the Go selection its Compose
	// was rendered with; the image tag alone cannot.
	recipeGo := development.RecipeGo(recipe)
	for _, goTool := range []bool{false, true} {
		for _, tests := range []bool{false, true} {
			for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
				req := development.Requirements{Go: goTool}
				o := Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: tests, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}
				if expected, err := Render(id, o); err == nil && bytes.Equal(data, expected) {
					return o, true
				}
				if generated && goTool == recipeGo {
					if expected, err := OlderRecipe(id, o, fingerprint); err == nil && bytes.Equal(data, expected) {
						return o, true
					}
					if expected, err := BeforeToolchainCache(id, o, fingerprint); err == nil && bytes.Equal(data, expected) {
						return o, true
					}
					if expected, err := BeforeStateMask(id, o, fingerprint); err == nil && bytes.Equal(data, expected) {
						return o, true
					}
				}
			}
		}
	}
	return Options{}, false
}

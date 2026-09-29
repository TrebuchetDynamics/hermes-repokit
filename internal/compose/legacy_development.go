package compose

import (
	"bytes"
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// LegacyDevelopment renders only the generated pre-PATH-fix image reference.
// The Compose topology did not change in that fix. Full recipe validation is
// separately required before publication; recognizing Compose is not adoption.
func LegacyDevelopment(id target.Identity, o Options) ([]byte, error) {
	if o.Development == nil {
		return nil, fmt.Errorf("legacy development requires a recipe")
	}
	id = target.PreviousNames(id)
	data, err := Render(id, o)
	if err != nil {
		return nil, err
	}
	current := []byte(fmt.Sprintf("    image: %q\n", development.ImageName(id.Container, *o.Development)))
	prior := []byte(fmt.Sprintf("    image: %q\n", id.Project+"-hermes-dev:"+development.LegacyFingerprint(*o.Development)[:24]))
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
	for _, goTool := range []bool{false, true} {
		for _, tests := range []bool{false, true} {
			for _, memory := range []string{"", projectmemory.Image} {
				for _, state := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
					req := development.Requirements{Go: goTool}
					o := Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: tests, OpenVikingImage: memory, UID: os.Getuid(), GID: os.Getgid(), SELinux: state}
					for _, render := range []func(target.Identity, Options) ([]byte, error){Render, PreviousNames, LegacyDevelopment} {
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

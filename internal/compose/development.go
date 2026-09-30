package compose

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func generatedData(id target.Identity) []byte {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return nil
	}
	defer root.Close()
	info, err := root.Lstat(".hermes/compose.yaml")
	if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil
	}
	f, err := root.Open(".hermes/compose.yaml")
	if err != nil {
		return nil
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return nil
	}
	return data
}

// DevelopmentSelected recognizes generated source, never a mutable receipt.
// Detecting current manifests separately lets verify report added requirements.
func DevelopmentSelected(id target.Identity) (Options, bool) {
	data := generatedData(id)
	if len(data) == 0 {
		return Options{}, false
	}
	for _, goTool := range []bool{false, true} {
		for _, tests := range []bool{false, true} {
			for _, sel := range []selinux.State{"", selinux.Detect()} {
				req := development.Requirements{Go: goTool}
				o := Options{HermesImage: qualification.FoundationImage, Development: &req, DockerTests: tests, UID: os.Getuid(), GID: os.Getgid(), SELinux: sel}
				expected, err := Render(id, o)
				if err == nil && bytes.Equal(data, expected) {
					return o, true
				}
			}
		}
	}
	return Options{}, false
}

func DevelopmentRecipeMatches(id target.Identity, req development.Requirements) bool {
	expected, err := development.Recipe(req)
	if err != nil {
		return false
	}
	root, err := os.OpenRoot(filepath.Join(id.Root, ".hermes"))
	if err != nil {
		return false
	}
	defer root.Close()
	info, err := root.Lstat("development-image")
	if err != nil || !info.IsDir() {
		return false
	}
	dir, err := root.Open("development-image")
	if err != nil {
		return false
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil || len(entries) != len(expected) {
		return false
	}
	for name, want := range expected {
		info, err := root.Lstat("development-image/" + name)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1048576 {
			return false
		}
		f, err := root.Open("development-image/" + name)
		if err != nil {
			return false
		}
		got, err := io.ReadAll(io.LimitReader(f, 1048577))
		f.Close()
		if err != nil || !bytes.Equal(got, want) {
			return false
		}
	}
	return true
}

var imageID = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func LocalImageID(value string) bool { return imageID.MatchString(value) }

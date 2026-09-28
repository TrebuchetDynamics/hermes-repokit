package compose

import (
	"bytes"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"io"
	"os"
	"regexp"
)

var imageID = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var layaSelection = regexp.MustCompile(`(?m)^  laya:\n    image: "(sha256:[a-f0-9]{64})"\n`)

func LocalImageID(value string) bool { return imageID.MatchString(value) }

// SelectedLaya recovers selection only from a bounded exact generated document.
// Owner-edited Compose never establishes installer authority.
func SelectedLaya(id target.Identity) string {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return ""
	}
	defer root.Close()
	f, err := root.Open(".hermes/compose.yaml")
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return ""
	}
	match := layaSelection.FindSubmatch(data)
	if len(match) != 2 {
		return ""
	}
	for _, memory := range []string{"", projectmemory.Image} {
		expected, err := Render(id, Options{HermesImage: qualification.FoundationImage, OpenVikingImage: memory, LayaImage: string(match[1]), UID: os.Getuid(), GID: os.Getgid()})
		if err == nil && bytes.Equal(expected, data) {
			return string(match[1])
		}
	}
	return ""
}

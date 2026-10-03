package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

// install --team single records the shape in .hermes; --team seven removes
// the record, so the deployment is back to the seven-profile team.
func TestRecordShape(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	id := target.Identity{Root: root}
	if err := recordShape(id, team.Single); err != nil || team.ShapeOf(root) != team.Single {
		t.Fatalf("single not recorded: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, ".hermes", team.ShapeFile))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("record mode: %v %v", info, err)
	}
	if err := recordShape(id, team.Seven); err != nil || team.ShapeOf(root) != team.Seven {
		t.Fatalf("seven not restored: %v", err)
	}
	if err := recordShape(id, team.Seven); err != nil {
		t.Fatalf("seven without a record: %v", err)
	}
	// A symlinked record is never written through.
	if err := os.Symlink("/etc/hostname", filepath.Join(root, ".hermes", team.ShapeFile)); err != nil {
		t.Fatal(err)
	}
	if err := recordShape(id, team.Single); err == nil {
		t.Fatal("wrote through a symlinked record")
	}
}

func TestTeamFlagTakesOnlyKnownShapes(t *testing.T) {
	var stderr bytes.Buffer
	a := App{Directory: t.TempDir()}
	if code := a.Run([]string{"install", "--team", "three"}, &bytes.Buffer{}, &stderr); code != 2 || !strings.Contains(stderr.String(), "--team takes single or seven") {
		t.Fatalf("code %d: %s", code, stderr.String())
	}
}

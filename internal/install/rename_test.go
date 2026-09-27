package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationNeverReplacesAppearingDirectory(t *testing.T) {
	p := t.TempDir()
	os.Mkdir(filepath.Join(p, "stage"), 0700)
	os.Mkdir(filepath.Join(p, ".hermes"), 0700)
	before, _ := os.Stat(filepath.Join(p, ".hermes"))
	root, e := os.OpenRoot(p)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	if e = publishDirectory(root, "stage"); e == nil {
		t.Fatal("replaced existing native directory")
	}
	after, _ := os.Stat(filepath.Join(p, ".hermes"))
	if !os.SameFile(before, after) {
		t.Fatal("native directory identity changed")
	}
}

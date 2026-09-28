package target

import (
	"os"
	"strings"
	"testing"
)

func TestBroadMountRootsAreRefused(t *testing.T) {
	for _, broad := range []string{"/", "/home", "/usr", "/etc", "/var", "/opt", "/boot", "/srv", "/root"} {
		if err := mountRootSafe(broad); err == nil {
			t.Fatalf("accepted broad host path %s", broad)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if err := mountRootSafe(home); err == nil {
			t.Fatalf("accepted user home %s", home)
		}
	}
	if err := mountRootSafe(t.TempDir()); err != nil {
		t.Fatalf("refused a narrow repository path: %v", err)
	}
}

func TestInspectFlagsBroadMountSource(t *testing.T) {
	issues := Inspect(Identity{Root: "/"}, "")
	for _, issue := range issues {
		if strings.Contains(issue, "broad host path") {
			return
		}
	}
	t.Fatalf("Inspect did not flag broad mount source: %v", issues)
}

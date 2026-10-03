package team

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func shapedRoot(t *testing.T, marker string) target.Identity {
	t.Helper()
	root := t.TempDir()
	if marker != "" {
		if err := os.MkdirAll(filepath.Join(root, ".hermes"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".hermes", ShapeFile), []byte(marker), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return target.Identity{Name: "atlas", Project: "repokit-123", Root: root}
}

// A deployment is seven profiles unless its .hermes records the single shape.
func TestShapeOf(t *testing.T) {
	for marker, want := range map[string]Shape{"": Seven, "single\n": Single, "single": Single, "seven\n": Seven, "other\n": Seven} {
		if got := ShapeOf(shapedRoot(t, marker).Root); got != want {
			t.Errorf("marker %q: %s, want %s", marker, got, want)
		}
	}
}

// In the single shape default does every card itself: it implements in one
// run and verifies in a fresh run, researches through subagents and keeps the
// owner's decisions in memory. The idle profiles keep their SOULs.
func TestSingleShapeDefault(t *testing.T) {
	seven := ForRepository(shapedRoot(t, ""))
	single := ForRepository(shapedRoot(t, "single\n"))
	def := single[0]
	if def.Name != "default" {
		t.Fatal("default is not first")
	}
	for _, want := range []string{
		"# Role: The whole team", `reviewer="default"`, "hermes kanban reassign <id> default",
		"delegate_task", "Memory is on for you", "## Verifying a card", "## Running a card",
		"## Talking with the owner", "workspace_path \"/workspace\"", "Never call kanban_complete in the run that implemented the change",
	} {
		if !strings.Contains(def.Soul, want) {
			t.Errorf("single default SOUL lacks %q", want)
		}
	}
	for _, unwanted := range []string{`reviewer="tester"`, "straight to executor", "these seven profiles", "Assign edits\nto AGENTS.md", "executor's card work"} {
		if strings.Contains(def.Soul, unwanted) {
			t.Errorf("single default SOUL still routes to the seven: %q", unwanted)
		}
	}
	for _, need := range []string{"kanban", "memory", "delegation"} {
		if !slices.Contains(def.Toolsets, need) {
			t.Errorf("single default does not require %s", need)
		}
	}
	for i := 1; i < len(single); i++ {
		if single[i].Soul != seven[i].Soul {
			t.Errorf("%s SOUL changed in the single shape", single[i].Name)
		}
	}
	if seven[0].Soul == def.Soul || !strings.Contains(seven[0].Soul, `reviewer="tester"`) {
		t.Fatal("seven-profile default changed")
	}
}

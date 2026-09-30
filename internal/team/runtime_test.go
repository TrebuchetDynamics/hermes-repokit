package team

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestSameCardChainContract(t *testing.T) {
	roles := map[string]Role{}
	for _, role := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		roles[role.Name] = role
		if strings.Contains(role.Soul, `reviewer="reviewer" on the SAME`) {
			t.Fatalf("%s still routes implementation straight to reviewer", role.Name)
		}
	}
	for name, texts := range map[string][]string{
		"executor": {`reviewer="tester"`, "Every revision goes back through tester"},
		"tester":   {"must not modify the repository", `reviewer="reviewer"`, "kanban_request_changes", "relay", "Never approve or complete"},
		"reviewer": {"tester pass", "relays it unchanged"},
		"default":  {"executor -> tester -> reviewer", `reviewer="tester"`, "seven-role"},
	} {
		for _, text := range texts {
			if !strings.Contains(roles[name].Soul, text) {
				t.Errorf("%s missing %q", name, text)
			}
		}
	}
}

func TestDefaultChannelsDoNotRequireMemory(t *testing.T) {
	if strings.Contains(ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"})[0].Soul, "Kanban and memory") {
		t.Fatal("default still requires memory on its channels")
	}
}

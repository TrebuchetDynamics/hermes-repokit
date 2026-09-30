package team

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestRepositoryIdentityContract(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123", Root: "/private/atlas"}
	roles := ForRepository(id)
	if len(roles) != 7 {
		t.Fatalf("permanent roster has %d roles", len(roles))
	}
	for _, role := range roles {
		for _, want := range []string{"Repository name: atlas", "Stable repository ID: repokit-123", "Native profile: " + role.Name, "Role description: " + role.Description, "default, researcher, planner, executor, tester, reviewer, steward", "seven permanent profiles", "Hermes is the runtime", "running workers", "Dispatch policy: bootstrap off; operational automatic", "primary human-facing"} {
			if !strings.Contains(role.Soul, want) {
				t.Errorf("%s identity missing %q", role.Name, want)
			}
		}
		if strings.Contains(role.Soul, id.Root) {
			t.Errorf("%s exposes host path", role.Name)
		}
	}
	other := ForRepository(target.Identity{Name: "other", Project: "repokit-456"})
	if roles[0].Soul == other[0].Soul {
		t.Fatal("distinct repositories share identity")
	}
	roles[0].Toolsets[0] = "changed"
	if ForRepository(id)[0].Toolsets[0] != "kanban" {
		t.Fatal("caller mutated subsequent roster")
	}
}

func TestDefaultChannelAndMaintenanceContract(t *testing.T) {
	roles := ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"})
	for _, want := range []string{"Profile is identity", "Platform is the conversation surface", "Session is conversation history", "Runtime is the serving process", "## Self-maintenance", "native Hermes configuration commands", "gateway_restart_after_turn", "Do not use a synchronous terminal gateway restart", "steward", "credentials", "fresh conversation"} {
		if !strings.Contains(roles[0].Soul, want) {
			t.Errorf("default contract missing %q", want)
		}
	}
	for _, role := range roles[1:] {
		if strings.Contains(role.Soul, "## Self-maintenance") {
			t.Errorf("%s received default maintenance authority", role.Name)
		}
	}
}

func TestDefaultTinyEditExceptionDoesNotWeakenIndependentReview(t *testing.T) {
	r := ForRepository(target.Identity{Name: "repo", Project: "repokit-test"})[0]
	if !strings.Contains(r.Soul, "tiny bounded owner-authorized edit") || strings.Contains(r.Soul, "Do not perform implementation work yourself.") {
		t.Fatal("tiny-edit policy contradicts remote development")
	}
	if !strings.Contains(r.Soul, "approving your own work") {
		t.Fatal("review contract lost")
	}
}

func TestCoordinatorUsesAutomaticReviewDispatch(t *testing.T) {
	roles := ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"})
	if strings.Contains(roles[0].Soul, "kanban dispatch --max") || !strings.Contains(roles[0].Soul, `reviewer="tester"`) {
		t.Fatal("coordinator bypasses automatic review dispatch")
	}
}

func TestCurrentPoliciesHaveNoSupervision(t *testing.T) {
	for _, role := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		for _, obsolete := range []string{"Nerve", "Laya", "supervision", "hosted Jev"} {
			if strings.Contains(role.Soul, obsolete) {
				t.Errorf("%s retains active obsolete policy %q", role.Name, obsolete)
			}
		}
	}
}

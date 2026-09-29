package team

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestRepositoryIdentityContract(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123", Root: "/private/atlas"}
	roles := ForRepository(id)
	legacy := map[string]Role{}
	for _, role := range sixRoleRoster() {
		legacy[role.Name] = role
	}
	if len(roles) != 7 {
		t.Fatalf("permanent roster has %d roles", len(roles))
	}
	for _, role := range roles {
		for _, want := range []string{"Repository name: atlas", "Stable repository ID: repokit-123", "Native profile: " + role.Name, "Role description: " + role.Description, "default, researcher, planner, executor, tester, reviewer, steward", "seven permanent profiles", "Hermes is the runtime", "running workers", "Dispatch policy: bootstrap off; operational automatic", "primary human-facing"} {
			if !strings.Contains(role.Soul, want) {
				t.Errorf("%s identity missing %q", role.Name, want)
			}
		}
		if prior, ok := legacy[role.Name]; ok && (role.LegacySoul != prior.Soul || (!strings.HasSuffix(role.PreviousSoul, prior.Soul) && !strings.HasSuffix(role.PreviousManagedSouls[3], prior.Soul))) {
			t.Errorf("%s lost exact historical contract", role.Name)
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
	if !strings.Contains(r.Soul, "approving your own work") || r.PreviousSoul == "" {
		t.Fatal("review or exact migration contract lost")
	}
}

func TestOperationalCoordinatorMigratesPriorRepositorySoul(t *testing.T) {
	roles := ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"})
	for _, r := range roles {
		if r.Name == "tester" {
			continue // added after the manual/off generation; no prior identity
		}
		if !strings.Contains(r.PreviousRepositorySoul, "Dispatch policy: manual/off") || r.PreviousRepositorySoul == r.Soul {
			t.Fatal("missing exact prior identity")
		}
	}
	if strings.Contains(roles[0].Soul, "kanban dispatch --max") || !strings.Contains(roles[0].Soul, `reviewer="tester"`) {
		t.Fatal("coordinator bypasses automatic review dispatch")
	}
}

func TestHistoricalPoliciesRemainExactAndCurrentPoliciesHaveNoSupervision(t *testing.T) {
	roles := ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"})
	for _, role := range roles {
		for _, obsolete := range []string{"Nerve", "Laya", "supervision", "hosted Jev"} {
			if strings.Contains(role.Soul, obsolete) {
				t.Errorf("%s retains active obsolete policy %q", role.Name, obsolete)
			}
		}
	}
	// Hashes captured from the exact pre-removal managed SOUL generations.
	for _, fixture := range []struct {
		index  int
		hashes []string
	}{
		{0, []string{"a317b87096c0b324d48d6848aad1d89c887778342f24df01104d8e2fd7c28d32", "b8964d60f4877b2414ff3527f3ff48fec00142860f59ae9c552e191733077ef0", "a8da2dc258c5acbe4079065b205fdc6ec99933a8fe482d02e78d1737f30adec3", "460293c75437932775b65e75695c87169ebdbd34d8d15b022b25d145134fbd7b", "d2c6d554eb3029f634ffe4cb6d2ab8b70fd34bb6a41e2a83e1938c3a67122ff9"}},
		{6, []string{"92e2fe45cd06872345c820b8def866308dbe557babcc7d7093a2f55b50b1c308", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "b2c310d45105e15342f749f148c5049a85e00f4d34950b97f25b989f8db9a4bc", "b1c6b46811085190fec0c6b7f4e2f3052a94caeaf158bf59edf9bc98b2e6803d", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}},
	} {
		role := roles[fixture.index]
		historical := append(append([]string{}, role.PreviousManagedSouls[:3]...), role.PreviousRepositorySoul, role.PreviousRepositoryOriginalSoul)
		for i, soul := range historical {
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(soul))); got != fixture.hashes[i] {
				t.Errorf("%s historical policy %d changed: %s", role.Name, i, got)
			}
		}
	}
}

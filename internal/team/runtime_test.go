package team

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestRuntimePolicyMigration(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123"}
	previous := sixRoleRepositoryRoles(id, sixRoleRoster(), sixRoleMaintenance)
	// Captured before the runtime policy was introduced. These bytes are the
	// installer's ownership proof, not a prompt wording assertion.
	hashes := []string{
		"81cc476f24e7d02d881db391e4bd10a46053dd3978f940066c5236d28419b6b3",
		"af7e04d450297e17b201ae76fcaf6cd4ee1c8d5ebe94affb7f0cbc7434cc43e9",
		"4b4ba28b280877626a0c73844643544ab803092e0b60728a56803e03548fd3ca",
		"54b8d0d698a924e7f49ef7e551a281b986f6c2122c822a09f279603677b4bf56",
		"5ed99108e18aa32f10f451baa4000c7da9a63f9a3e1030a02253294e81e97b06",
		"c0b3e5439cb47ff2fe8d041693bf975969902cb829911cd135397fecefbff0a8",
	}
	for i, role := range sixRoleGeneration(id) {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(previous[i].Soul))); got != hashes[i] {
			t.Fatalf("%s historical migration source changed: %s", role.Name, got)
		}
		if len(role.PreviousManagedSouls) != 4 || role.PreviousManagedSouls[3] != previous[i].Soul {
			t.Fatalf("%s must retain the exact preceding managed SOUL after the three historical generations", role.Name)
		}
		if !reflect.DeepEqual(role.Toolsets, previous[i].Toolsets) {
			t.Fatalf("%s runtime policy widened capabilities", role.Name)
		}
	}
}

// The six-profile release installed these exact bytes. They prove existing
// deployments are still recognized as managed after tester was added.
func TestSixRoleGenerationIsFrozen(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123"}
	hashes := map[string]string{
		"default":    "c4a0e9dbc7b7213353d8aac14b2f1e464c441544d47cc87f89245afe6dca643e",
		"researcher": "8296ddb3c96502972705c8b201a9b89da5d38599dcbf42411cc5802839228969",
		"planner":    "dfecc99acc681aba7f0af42ca510a1f44bf427bb0290728f4017e942684d1e62",
		"executor":   "666390c96987fdc5b66bf5edf7e3d7fb6a930be991bdcc2cbfc4e9998b0fa458",
		"reviewer":   "43f9ff7ef535f4a0732ea859b9e7b66cd646a63215f98fd9eb253c038a7a9d52",
		"steward":    "6dc57f3f49b5a1c5b657ede4e11dbbd74f96a613c20d02dd22ce1a80db0a9dd7",
	}
	six := sixRoleGeneration(id)
	if len(six) != len(hashes) {
		t.Fatalf("six-role generation has %d roles", len(six))
	}
	for _, role := range six {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(role.Soul))); got != hashes[role.Name] {
			t.Fatalf("%s six-role migration source changed: %s", role.Name, got)
		}
	}
}

func TestTesterMigrationIsByNameNotPosition(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123"}
	six := map[string]Role{}
	for _, role := range sixRoleGeneration(id) {
		six[role.Name] = role
	}
	for _, role := range ForRepository(id) {
		prior, existed := six[role.Name]
		if !existed {
			if role.Name != "tester" || role.LegacySoul != "" || role.PreviousSoul != "" || len(role.PreviousManagedSouls) != 0 {
				t.Fatalf("%s must be new with no managed history: %+v", role.Name, role.PreviousManagedSouls)
			}
			continue
		}
		n := len(role.PreviousManagedSouls)
		if n != 5 || role.PreviousManagedSouls[n-1] != prior.Soul || role.LegacySoul != prior.LegacySoul {
			t.Fatalf("%s lost its own six-role history", role.Name)
		}
		for _, other := range six {
			if other.Name != role.Name && role.PreviousManagedSouls[n-1] == other.Soul {
				t.Fatalf("%s inherited %s history", role.Name, other.Name)
			}
		}
		if role.Soul == prior.Soul || role.Description != prior.Description || !reflect.DeepEqual(role.Toolsets, prior.Toolsets) {
			t.Fatalf("%s must change SOUL only; description and tools identify managed profiles", role.Name)
		}
	}
}

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

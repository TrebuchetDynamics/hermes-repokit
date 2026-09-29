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
		"1c490e8ac7ee99c2affe05f1c44a8165bf1c37fb8492bf4c21c06b1f16a2e713",
		"af7e04d450297e17b201ae76fcaf6cd4ee1c8d5ebe94affb7f0cbc7434cc43e9",
		"4b4ba28b280877626a0c73844643544ab803092e0b60728a56803e03548fd3ca",
		"54b8d0d698a924e7f49ef7e551a281b986f6c2122c822a09f279603677b4bf56",
		"5ed99108e18aa32f10f451baa4000c7da9a63f9a3e1030a02253294e81e97b06",
		"7d026f6f39ec17fe4a2b334c074a75b11c64fcffd6224a95fa77c386be8d88fc",
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
		"default":    "051c52a0471fb3ff8c4770b3da4492f77b7baea7092baea22fc5775ab8a17996",
		"researcher": "5f79d1b78100d875de3b3b0f3727b30929b9de71c774c289e137395ce6e76c9f",
		"planner":    "5b5431cb95559e052031d1b5344b25a9c85a9d54d79453e2de6965fe5e328205",
		"executor":   "53628081ae27186b8f48beef365869f5719381b806aaf6c71ba56fa7f62c25a5",
		"reviewer":   "485c906874c8158451282635aabc6d5ef974054c0e578e71166c2a2ac1bcef4d",
		"steward":    "0cd7cc3e951b362051ebd37dc04bc4e22e8e0b9280f20d25026380c577f10bc5",
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

// RepoKit prepares the environment; Hermes owns its memory providers. The
// frozen six-role generation above still describes embedded memory.
func TestCurrentSoulsLeaveMemoryProvidersToHermes(t *testing.T) {
	for _, role := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		if strings.Contains(role.Soul, "OpenViking") || strings.Contains(role.Soul, "1933") {
			t.Fatalf("%s still describes a memory provider", role.Name)
		}
		if !strings.Contains(role.Soul, "Memory providers and other optional integrations are native Hermes\nfeatures") {
			t.Fatalf("%s lacks the Hermes-owned features contract", role.Name)
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

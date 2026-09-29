package team

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestRuntimePolicyMigration(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123"}
	current := ForRepository(id)
	previous := repositoryRoles(id, Roster(), defaultMaintenance)
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
	for i, role := range current {
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

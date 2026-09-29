package native

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

func TestManagedSoulsPreserveExactRepositoryMigrationSources(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123"}
	roles := team.ForRepository(id)
	if len(roles) != 7 {
		t.Fatalf("roles=%d", len(roles))
	}
	for _, role := range roles {
		// tester postdates every historical generation and has none to match.
		historical := role.Name == "tester" || matchingSoul(role.LegacySoul, role) && matchingSoul(role.PreviousRepositorySoul, role)
		if !strings.Contains(role.Soul, "repokit-123") || !historical {
			t.Fatalf("%s lost exact migration source", role.Name)
		}
		if matchingSoul(role.Soul+"\nowner change", role) {
			t.Fatalf("%s accepted altered SOUL", role.Name)
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(role.Soul))
		if !strings.Contains(soulWrite(role.Name, role.Soul), encoded) {
			t.Fatalf("%s SOUL bytes not preserved", role.Name)
		}
	}
}
func TestTeamCommandsQuoteUntrustedDescription(t *testing.T) {
	command := teamCommand("profile", "describe", "reviewer", "--text", "owner's $(touch /tmp/never)")
	if !strings.Contains(command, "'owner'\\''s $(touch /tmp/never)'") {
		t.Fatalf("unsafe shell argument: %s", command)
	}
}

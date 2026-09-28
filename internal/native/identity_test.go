package native

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

func TestInitializationCarriesRepositoryIdentityAndExactMigrationSource(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123"}
	script := initializationScript(id, false)
	_, payload, ok := strings.Cut(script, "main(json.loads(base64.b64decode('")
	if !ok {
		t.Fatal("team payload missing")
	}
	encoded, _, ok := strings.Cut(payload, "')")
	if !ok {
		t.Fatal("team payload incomplete")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Roles      []team.Role `json:"roles"`
		AfterSetup bool        `json:"after_setup"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := team.ForRepository(id)
	if got.AfterSetup || len(got.Roles) != len(want) {
		t.Fatalf("wrong bootstrap payload: after_setup=%t, roles=%d", got.AfterSetup, len(got.Roles))
	}
	for i, role := range got.Roles {
		if role.Name != want[i].Name || role.Soul != want[i].Soul || role.LegacySoul != want[i].LegacySoul {
			t.Fatalf("%s missing repository identity or migration source", want[i].Name)
		}
	}
}

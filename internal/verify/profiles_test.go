package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

func TestProfilesReportEarlierManagedGenerationAsUpgradePending(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123", Root: t.TempDir()}
	executor := team.ForRepository(id)[3]
	dir := filepath.Join(id.Root, ".hermes", "profiles", "executor")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for soul, want := range map[string]Status{
		executor.Soul: Healthy,
		executor.PreviousManagedSouls[len(executor.PreviousManagedSouls)-1]: PendingSetup,
		executor.Soul + "\nowner edit":                                      Degraded,
	} {
		for name, data := range map[string]string{"config.yaml": "x", "profile.yaml": "x", "SOUL.md": soul} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if got := status(Profiles(id), "profile:executor"); got != want {
			t.Errorf("got %s want %s", got, want)
		}
	}
}

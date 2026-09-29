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

func stewardProbe(t *testing.T, id target.Identity) Probe {
	t.Helper()
	for _, p := range Profiles(id) {
		if p.Component == "profile:steward" {
			return p
		}
	}
	t.Fatal("steward probe missing")
	return Probe{}
}

// An exact earlier RepoKit steward SOUL is upgradeable; an edited one is drift.
func TestPreviousManagedSoulIsUpgradeNotDrift(t *testing.T) {
	id, err := target.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roles := team.ForRepository(id)
	for _, role := range roles {
		dir := filepath.Join(id.Root, ".hermes")
		if role.Name != "default" {
			dir = filepath.Join(dir, "profiles", role.Name)
		}
		os.MkdirAll(dir, 0700)
		for _, f := range []string{"config.yaml", "profile.yaml"} {
			os.WriteFile(filepath.Join(dir, f), []byte("x: 1\n"), 0600)
		}
		os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte(role.Soul), 0600)
	}
	soul := filepath.Join(id.Root, ".hermes/profiles/steward/SOUL.md")
	if p := stewardProbe(t, id); p.Status != Healthy {
		t.Fatalf("current steward: %+v", p)
	}
	var steward team.Role
	for _, role := range roles {
		if role.Name == "steward" {
			steward = role
		}
	}
	previous := steward.PreviousManagedSouls[len(steward.PreviousManagedSouls)-1]
	os.WriteFile(soul, []byte(previous), 0600)
	if p := stewardProbe(t, id); p.Status != PendingSetup {
		t.Fatalf("previous managed steward reported as drift: %+v", p)
	}
	os.WriteFile(soul, []byte(previous+"\nOwner addition.\n"), 0600)
	if p := stewardProbe(t, id); p.Status != Degraded {
		t.Fatalf("owner-edited steward not reported as drift: %+v", p)
	}
}

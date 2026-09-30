package verify

import (
	"os"
	"path/filepath"
	"strings"
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
		executor.Soul + "\nowner edit":                                      Customized,
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

func TestCustomizedProfileIsNamedButNotACoreFailure(t *testing.T) {
	probes := []Probe{{"hermes", Healthy, ""}, {"profile:executor", Customized, ""}, {"profile:tester", Healthy, ""}, {"review:evidence", Healthy, ""}}
	readiness := Readiness(probes)
	if readiness[0].Status != Healthy || !CoreUsable(readiness) || !strings.Contains(readiness[0].Detail, "owner-customized profiles: executor") {
		t.Fatalf("customized profile mishandled: %+v", readiness[0])
	}
	probes[1].Status = Degraded
	if readiness = Readiness(probes); readiness[0].Status != Degraded || strings.Contains(readiness[0].Detail, "owner-customized") {
		t.Fatalf("degraded profile not a core failure: %+v", readiness[0])
	}
}

func TestAbsentProfilesPointAtRepoKitSetup(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123", Root: t.TempDir()}
	if err := os.MkdirAll(filepath.Join(id.Root, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, p := range Profiles(id) {
		if p.Status != PendingSetup || p.Detail != "profile files absent; run repokit setup" {
			t.Fatalf("%s: %s %q", p.Component, p.Status, p.Detail)
		}
	}
}

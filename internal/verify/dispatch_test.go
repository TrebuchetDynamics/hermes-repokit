package verify

import "testing"

func TestConfiguredDispatchIsNotProofOfLiveDispatch(t *testing.T) {
	yes, no := true, false
	good := dispatchObservation{Configured: "enabled", Live: "enabled", Owner: "default", Max: 1, Auto: &no, Review: &yes, Allowlist: []string{"default"}, Policy: true, Canary: true}
	for _, p := range dispatchProbes(good) {
		if p.Status != Healthy {
			t.Fatal(p)
		}
	}
	for _, live := range []string{"stale", "unknown", "paused", "disabled", ""} {
		observed := good
		observed.Live = live
		if dispatchProbes(observed)[1].Status == Healthy {
			t.Fatal("false live readiness", live)
		}
	}
	good.Owner = "unknown"
	if dispatchProbes(good)[1].Status == Healthy || dispatchProbes(good)[2].Status == Healthy {
		t.Fatal("unknown dispatcher owner accepted")
	}
}

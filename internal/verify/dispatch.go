package verify

import "strings"

type dispatchObservation struct {
	Configured string   `json:"configured"`
	Live       string   `json:"live"`
	Owner      string   `json:"owner"`
	Max        int      `json:"max_in_progress"`
	Auto       *bool    `json:"auto_decompose"`
	Review     *bool    `json:"review_dispatch"`
	Allowlist  []string `json:"allowlist"`
	Canary     bool     `json:"canary"`
	Policy     bool     `json:"policy"`
}

func dispatchProbes(d dispatchObservation) []Probe {
	configured := Probe{"kanban:dispatch-configured", Unknown, "configuration unavailable"}
	live := Probe{"kanban:dispatch-live", Degraded, "stale or unknown: active default gateway has not proven loaded dispatch settings and singleton lock ownership"}
	if d.Configured == "enabled" {
		configured = Probe{configured.Component, Healthy, "enabled"}
	}
	if d.Configured == "disabled" {
		configured = Probe{configured.Component, Inactive, "disabled: bootstrap or incomplete setup"}
	}
	if d.Live == "enabled" && d.Policy && d.Owner == "default" {
		live = Probe{live.Component, Healthy, "enabled: matching gateway generation, native startup evidence and live dispatcher lock"}
	}
	if d.Live == "disabled" {
		live.Detail = "disabled: assigned cards await operational setup"
		live.Status = Inactive
	}
	if d.Live == "paused" {
		live.Detail = "paused by native emergency stop; owner pause preserved"
		live.Status = Inactive
	}
	policy := Probe{"kanban:dispatch-policy", Degraded, "expected default owner, six-profile allowlist, review_dispatch=true, auto_decompose=false, max_in_progress=1"}
	expected := "default,researcher,planner,executor,reviewer,steward"
	if d.Policy && d.Owner == "default" && d.Max == 1 && d.Auto != nil && !*d.Auto && d.Review != nil && *d.Review && strings.Join(d.Allowlist, ",") == expected {
		policy = Probe{policy.Component, Healthy, "owner=default; max_in_progress=1; auto_decompose=disabled; review_dispatch=enabled; worker_allowlist=" + expected}
	}
	canary := Probe{"kanban:dispatcher-canary", Unqualified, "researcher gateway execution has not passed for this gateway generation"}
	if d.Canary && live.Status == Healthy {
		canary.Status = Healthy
		canary.Detail = "observed gateway-spawned researcher running and completing the no-write canary; native card archived"
	}
	return []Probe{configured, live, policy, canary}
}

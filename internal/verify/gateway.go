package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/gateway"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

// Gateway observes current inputs AND the native process. The receipt alone
// never certifies health or session freshness.
func Gateway(ctx context.Context, id target.Identity, r Runner) []Probe {
	result := []Probe{{"gateway-generation", Unknown, "native gateway generation unavailable"}}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return append(result, dispatchProbes(dispatchObservation{})...)
	}
	output := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", container, "/opt/hermes/.venv/bin/python", "-I", "-B", "-c", gateway.Script(id, false))
	var observed struct {
		Gateway                  string
		Dispatch                 dispatchObservation `json:"dispatch"`
		Identities, Descriptions map[string]bool
		PID                      int    `json:"pid"`
		ManagedGeneration        string `json:"managed_generation"`
		LiveGeneration           string `json:"live_generation"`
		RestartPending           string `json:"restart_pending"`
		Channels                 struct {
			Rows []channelObservation `json:"rows"`
		} `json:"channels"`
	}
	if output.Err != nil || output.Truncated || json.Unmarshal([]byte(output.Output), &observed) != nil {
		return append(result, dispatchProbes(dispatchObservation{})...)
	}
	switch observed.Gateway {
	case "current":
		result[0] = Probe{"gateway-generation", Healthy, "managed generation matches healthy gateway and observed adapters; configured-channel completeness and existing sessions unverified"}
	case "stale":
		result[0] = Probe{"gateway-generation", Degraded, "running gateway predates changed managed state; reconcile through setup"}
	case "not-running":
		result[0] = Probe{"gateway-generation", Inactive, "gateway not running; no messaging readiness claimed"}
	}
	roster := Probe{"team-roster", Healthy, "all six repository identities and descriptions match; native worker execution unqualified"}
	for _, role := range team.Roster() {
		p := Probe{"description:" + role.Name, Healthy, "native profile description matches repository role"}
		if !observed.Descriptions[role.Name] {
			p.Status = Degraded
			p.Detail = "profile description absent or drifted; owner state preserved"
		}
		result = append(result, p)
		if !observed.Identities[role.Name] || !observed.Descriptions[role.Name] {
			roster.Status = Degraded
			roster.Detail = "repository identity, permanent roster or role description missing/drifted"
		}
	}
	identity := Probe{"default-identity", Healthy, "repository-specific default orchestrator identity and permanent roster match"}
	if !observed.Identities["default"] {
		identity.Status = Degraded
		identity.Detail = "repository-specific orchestrator SOUL missing or drifted"
	}
	if observed.PID > 1 {
		result = append(result, Probe{"gateway-process", Healthy, fmt.Sprintf("observed PID=%d; this is process evidence, not a message round trip", observed.PID)})
	}
	if generationPattern.MatchString(observed.ManagedGeneration) {
		live := "unknown"
		if generationPattern.MatchString(observed.LiveGeneration) {
			live = observed.LiveGeneration
		}
		pending := "unknown"
		if observed.RestartPending == "yes" || observed.RestartPending == "no" {
			pending = observed.RestartPending
		}
		result = append(result, Probe{"gateway-inputs", Unknown, "managed=" + observed.ManagedGeneration + "; live=" + live + "; restart_pending=" + pending})
	}
	result = append(result, dispatchProbes(observed.Dispatch)...)
	result = append(result, channelProbes(observed.Channels.Rows)...)
	result = append(result, Probe{"maintenance:live", Unqualified, "native restart broker requires scanner admission and model-initiated successor acceptance; configuration does not prove self-restart"})
	return append(result, identity, roster)
}

var generationPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type channelObservation struct {
	Platform      string `json:"platform"`
	Preset        string `json:"preset"`
	Routing       string `json:"default_routing"`
	Core          string `json:"core_selection"`
	Owner         string `json:"owner"`
	Adapter       string `json:"recorded_adapter_state"`
	Authorization string `json:"authorization"`
}

func channelProbes(rows []channelObservation) []Probe {
	if len(rows) == 0 || len(rows) > 128 {
		return []Probe{{"channels", Unknown, "channel projection unavailable; no routing or interactive capability acceptance claimed"}}
	}
	var result []Probe
	for _, row := range rows {
		if !platformName.MatchString(row.Platform) || !platformName.MatchString(row.Preset) {
			continue
		}
		core := Probe{"channel:" + row.Platform + ":core", Unknown, "native preset=" + row.Preset + "; saved core selection unresolved"}
		if row.Core == "complete" {
			core.Status = Healthy
			core.Detail = "native preset=" + row.Preset + "; saved core selection includes required categories; provider availability and session schemas unverified"
		}
		if row.Owner == "other" {
			core.Status = Unknown
			core.Detail = "saved default selections do not establish capabilities of another profile's adapter"
		}
		if row.Core == "incomplete" {
			core.Status = Degraded
			core.Detail = "required core categories missing/disabled; reconcile with setup --team"
		}
		route := Probe{"channel:" + row.Platform + ":route", Unknown, "effective routing unresolved; recorded adapter state is not a message probe"}
		if row.Routing == "default" {
			route.Detail = "default in declared routing projection; effective overlays and live message routing unverified"
		}
		if row.Routing == "conditional_other" {
			route.Status = Unqualified
			route.Detail = "declared routes may select another served profile; this channel can represent another agent"
		}
		if row.Owner == "other" {
			route.Status = Unqualified
			route.Detail = "recorded adapter belongs to another profile; default parity is not established"
		}
		auth := Probe{"channel:" + row.Platform + ":authorization", Unknown, "effective authorization unresolved; no credential stores or pairing records inspected"}
		if row.Platform == "cli" {
			auth.Status = Unqualified
			auth.Detail = "local shell access; remote channel authorization does not apply"
		}
		if row.Authorization == "restricted" {
			auth.Detail = "restrictive sender declarations present; environment overrides, pairing and effective access unverified"
		}
		if row.Authorization == "open" {
			auth.Status = Degraded
			auth.Detail = "declared open/wildcard access; do not certify this remote development channel until authorization is qualified"
		}
		result = append(result, core, route, auth)
	}
	return result
}

package verify

import (
	"context"
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

//go:embed kanban_probe.py
var kanbanProbeBody string

var kanbanProbe = team.KanbanPolicy + "\n" + kanbanProbeBody
var platformName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// DefaultKanban observes required Kanban and memory selections and dispatch
// configuration, not live sessions or per-source overrides. No Hermes imports
// or inference. The name is retained for existing callers.
func DefaultKanban(ctx context.Context, id target.Identity, r Runner) []Probe {
	failure := []Probe{
		{"kanban:default", Unknown, "native tool selections unavailable; no session tools probed"},
		{"memory:default", Unknown, "native tool selections unavailable; no session tools probed"},
		{"kanban:dispatch", Unknown, "native dispatch configuration unavailable"},
	}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return failure
	}
	result := r.Run(ctx, "docker", "--context", dc, "exec", "--user", "hermes", "--workdir", "/", container, "/opt/hermes/.venv/bin/python", "-I", "-B", "-c", kanbanProbe, "/opt/data")
	var observed struct {
		Kanban        map[string]string `json:"kanban"`
		Memory        map[string]string `json:"memory"`
		Dispatch      string            `json:"dispatch"`
		Notifications string            `json:"notifications"`
	}
	if result.Err != nil || result.Truncated || json.Unmarshal([]byte(result.Output), &observed) != nil {
		return failure
	}
	var probes []Probe
	for _, selection := range []struct {
		name   string
		states map[string]string
	}{{"kanban", observed.Kanban}, {"memory", observed.Memory}} {
		if len(selection.states) == 0 || len(selection.states) > 128 || selection.states["fallback"] == "" {
			return failure
		}
		keys := make([]string, 0, len(selection.states))
		for key := range selection.states {
			if !platformName.MatchString(key) {
				return failure
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			p := Probe{selection.name + ":default:" + key, Degraded, "required " + selection.name + " opt-in missing or invalid; reconcile with install"}
			switch selection.states[key] {
			case "enabled":
				p.Status = Healthy
				p.Detail = selection.name + " enabled in native selection; existing sessions and source overrides not probed"
			case "fallback":
				p.Status = Unknown
				p.Detail = "native platform preset unresolved; global memory selection is not a channel fallback"
				if selection.name == "kanban" {
					p.Status = Healthy
					p.Detail = "no saved platform selection; native profile-wide Kanban opt-in applies; live session unverified"
				}
			case "not-configured":
				p.Status = Inactive
				p.Detail = "optional channel not configured in observed declarations"
				if key == "fallback" && selection.name == "memory" {
					p.Detail = "memory has no native profile-wide channel fallback; inspect each configured platform selection"
				}
			case "unknown":
				p.Status = Unknown
				p.Detail = "native platform default or preset unresolved; no effective tool availability claimed"
			}
			probes = append(probes, p)
		}
	}
	dispatch := Probe{"kanban:dispatch", Degraded, "kanban.dispatch_in_gateway missing or invalid"}
	switch observed.Dispatch {
	case "manual":
		dispatch.Status = Inactive
		dispatch.Detail = "kanban.dispatch_in_gateway=false; bootstrap/incomplete setup; ordinary assigned work cannot execute automatically"
	case "enabled":
		dispatch.Status = Healthy
		dispatch.Detail = "automatic dispatch configured; live dispatcher is checked separately"
	}
	notifications := Probe{"kanban:notifications", Unknown, "native completion subscription/delivery configuration unresolved"}
	if observed.Notifications == "enabled" {
		notifications.Status = Healthy
		notifications.Detail = "native create subscriptions and gateway notifications enabled; originating-channel delivery and wake unverified"
	} else if observed.Notifications == "disabled" {
		notifications.Status = Degraded
		notifications.Detail = "native completion subscription/delivery disabled; reconcile with setup --team"
	}
	return append(probes, notifications, dispatch)
}

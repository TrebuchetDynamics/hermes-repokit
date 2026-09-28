package verify

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

type gatewayRunner struct {
	*integrationRunner
	output string
}

func (r *gatewayRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	if strings.Contains(strings.Join(args, " "), "One-shot generation diagnostics") {
		return process.Result{Output: r.output}
	}
	return r.integrationRunner.Run(ctx, p, args...)
}

// RunInput exercises the stdin delivery used when the one-shot script exceeds
// the per-argument exec limit.
func (r *gatewayRunner) RunInput(ctx context.Context, input io.Reader, p string, args ...string) process.Result {
	data, _ := io.ReadAll(input)
	if strings.Contains(string(data), "One-shot generation diagnostics") {
		return process.Result{Output: r.output}
	}
	return r.integrationRunner.Run(ctx, p, args...)
}
func TestGatewayFreshnessAndIdentityReportedIndependently(t *testing.T) {
	id, base := integrationFixture(t)
	roles := map[string]bool{"default": true, "researcher": true, "planner": true, "executor": true, "reviewer": true, "steward": true}
	for _, tc := range []struct {
		state string
		want  Status
	}{{"current", Healthy}, {"stale", Degraded}, {"not-running", Inactive}, {"unknown", Unknown}} {
		encoded, _ := json.Marshal(map[string]any{"gateway": tc.state, "identities": roles, "descriptions": roles})
		probes := Gateway(context.Background(), id, &gatewayRunner{base, string(encoded)})
		if probes[0].Status != tc.want {
			t.Fatalf("%s: %+v", tc.state, probes)
		}
		if probes[1].Component != "description:default" || probes[1].Status != Healthy {
			t.Fatal("gateway should report native profile readiness directly")
		}
		if probes[len(probes)-1].Status != Healthy {
			t.Fatal("matching roster not observed")
		}
	}
	roles["default"] = false
	encoded, _ := json.Marshal(map[string]any{"gateway": "current", "identities": roles, "descriptions": roles})
	probes := Gateway(context.Background(), id, &gatewayRunner{base, string(encoded)})
	if probes[len(probes)-1].Status != Degraded || probes[len(probes)-2].Status != Degraded {
		t.Fatal("identity drift passed")
	}
}

func TestChannelProjectionDoesNotCertifyLiveRouting(t *testing.T) {
	rows := []channelObservation{{Platform: "telegram", Preset: "hermes-telegram", Core: "complete", Routing: "default", Owner: "default", Authorization: "restricted"}, {Platform: "discord", Preset: "hermes-discord", Core: "incomplete", Routing: "conditional_other", Authorization: "open"}}
	probes := channelProbes(rows)
	if len(probes) != 6 || probes[0].Status != Healthy || probes[1].Status != Unknown || probes[2].Status != Unknown || probes[3].Status != Degraded || probes[4].Status != Unqualified || probes[5].Status != Degraded {
		t.Fatalf("incorrect channel claims: %+v", probes)
	}
	if len(channelProbes([]channelObservation{{Platform: "token\nvalue", Preset: "unknown"}})) != 0 {
		t.Fatal("untrusted labels emitted")
	}
}

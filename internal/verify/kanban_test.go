package verify

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// hermesRunner answers public `hermes -p default ...` reads by suffix.
type hermesRunner struct {
	*integrationRunner
	replies map[string]string
}

func (r *hermesRunner) Run(ctx context.Context, command string, args ...string) process.Result {
	call := strings.Join(args, " ")
	if strings.Contains(call, "/usr/local/bin/hermes -p default ") {
		r.calls = append(r.calls, append([]string{command}, args...))
		for suffix, reply := range r.replies {
			if strings.HasSuffix(call, suffix) {
				return process.Result{Output: reply}
			}
		}
		return process.Result{Err: context.DeadlineExceeded}
	}
	return r.integrationRunner.Run(ctx, command, args...)
}

const operationalKanban = `{"dispatch_in_gateway":true,"review_dispatch":true,"max_in_progress":1,"auto_decompose":false,"orchestrator_profile":"default","dispatch_profiles":["default","researcher","planner","executor","tester","reviewer","steward"],"auto_subscribe_on_create":true,"notify_in_gateway":true}`

func status(probes []Probe, component string) Status {
	for _, p := range probes {
		if p.Component == component {
			return p.Status
		}
	}
	return ""
}

func TestKanbanObservesPolicyNotificationsAndChannelTools(t *testing.T) {
	id, base := integrationFixture(t)
	r := &hermesRunner{integrationRunner: base, replies: map[string]string{
		"config get kanban --json":            operationalKanban,
		"config get platform_toolsets --json": `{"cli":["kanban","memory","file"],"telegram":["file","terminal"],"api_server":["file"]}`,
	}}
	probes := DefaultKanban(context.Background(), id, r)
	if status(probes, "kanban:dispatch") != Healthy || status(probes, "kanban:notifications") != Healthy ||
		status(probes, "channel:cli") != Healthy || status(probes, "channel:telegram") != Degraded || status(probes, "channel:api_server") != "" {
		t.Fatalf("%+v", probes)
	}
	for _, call := range r.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, "python") || strings.Contains(joined, " set ") {
			t.Fatalf("verify used a private or mutating command: %s", joined)
		}
	}
}

func TestKanbanDistinguishesOffOwnerChangedAndUnreadable(t *testing.T) {
	id, base := integrationFixture(t)
	for reply, want := range map[string]Status{
		`{"dispatch_in_gateway":false}`: Inactive,
		`{}`:                            Inactive,
		strings.Replace(operationalKanban, `"max_in_progress":1`, `"max_in_progress":4`, 1): Degraded,
		`not json`: Unknown,
		strings.Replace(operationalKanban, `"tester",`, "", 1): PendingSetup,
	} {
		r := &hermesRunner{integrationRunner: base, replies: map[string]string{"config get kanban --json": reply}}
		if got := status(DefaultKanban(context.Background(), id, r), "kanban:dispatch"); got != want {
			t.Fatalf("%s: got %s want %s", reply, got, want)
		}
	}
}

func TestGatewayAndReviewEvidenceFromPublicCLI(t *testing.T) {
	id, base := integrationFixture(t)
	r := &hermesRunner{integrationRunner: base, replies: map[string]string{
		"gateway status": "✓ Gateway is running (PID: 9)",
		"kanban list --status done --sort completed-desc --json": `[{"id":"t_a"},{"id":"t_b"}]`,
		"kanban show t_a --json":                                 `{"runs":[{"profile":"executor","outcome":"completed"}]}`,
		"kanban show t_b --json":                                 `{"runs":[{"profile":"executor","outcome":"review_requested"},{"profile":"tester","outcome":"review_requested"},{"profile":"reviewer","outcome":"completed"}]}`,
	}}
	if got := status(Gateway(context.Background(), id, r), "gateway"); got != Healthy {
		t.Fatal(got)
	}
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Healthy || !strings.Contains(p.Detail, "t_b") {
		t.Fatal(p)
	}
	// A card completed by its implementer alone is not independent review.
	r.replies["kanban list --status done --sort completed-desc --json"] = `[{"id":"t_a"}]`
	r.replies["gateway status"] = "✗ Gateway is not running"
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Unqualified {
		t.Fatal(p)
	}
	if got := status(Gateway(context.Background(), id, r), "gateway"); got != Inactive {
		t.Fatal(got)
	}
}

func TestReadinessSeparatesConfiguredFromProved(t *testing.T) {
	healthy := []Probe{{"compose", Healthy, ""}, {"gateway", Healthy, ""}, {"kanban:dispatch", Healthy, ""}, {"development:go", Healthy, ""}, {"docker_acceptance", Inactive, ""}}
	got := Readiness(append(healthy, Probe{"review:evidence", Unqualified, ""}))
	if got[0].Component != "CORE_TEAM" || got[0].Status != Healthy || got[1].Component != "DISPATCH" || got[1].Status != Unqualified || len(got) != 4 || got[2].Component != "DEVELOPMENT_RUNTIME" || got[3].Component != "HOST_LAUNCHER" || !CoreUsable(got) {
		t.Fatalf("configured core must be usable but dispatch unproved: %+v", got)
	}
	got = Readiness(append(healthy, Probe{"review:evidence", Healthy, ""}))
	if got[0].Status != Healthy || got[1].Status != Healthy {
		t.Fatalf("observed review must prove dispatch: %+v", got)
	}
	got = Readiness(append(healthy, Probe{"channel:telegram", Degraded, ""}, Probe{"review:evidence", Healthy, ""}))
	if got[0].Status != Healthy || got[1].Status != Degraded || !strings.Contains(got[1].Detail, "channel:telegram") || CoreUsable(got) {
		t.Fatalf("broken channel must block dispatch, not the core team: %+v", got)
	}
}

// Memory providers are Hermes features: RepoKit readiness never reports or
// depends on them.
func TestReadinessIgnoresHermesMemoryProviders(t *testing.T) {
	got := Readiness([]Probe{{"compose", Healthy, ""}, {"gateway", Healthy, ""}, {"kanban:dispatch", Healthy, ""}, {"review:evidence", Healthy, ""}, {"memory", Degraded, "x"}})
	if len(got) != 4 || got[0].Status != Healthy || got[1].Status != Healthy || !CoreUsable(got) {
		t.Fatalf("memory affected RepoKit readiness: %+v", got)
	}
}

func TestAcceptanceChainRequiresTesterAfterLatestImplementation(t *testing.T) {
	type runs = []struct{ Profile, Outcome string }
	rr := func(pairs ...string) runs {
		out := runs{}
		for i := 0; i < len(pairs); i += 2 {
			out = append(out, struct{ Profile, Outcome string }{pairs[i], pairs[i+1]})
		}
		return out
	}
	for name, c := range map[string]struct {
		runs runs
		ok   bool
	}{
		"simple chain":                               {rr("executor", "review_requested", "tester", "review_requested", "reviewer", "completed"), true},
		"tester rejection then pass":                 {rr("executor", "review_requested", "tester", "changes_requested", "executor", "review_requested", "tester", "review_requested", "reviewer", "completed"), true},
		"reviewer rejection relayed and re-verified": {rr("executor", "review_requested", "tester", "review_requested", "reviewer", "changes_requested", "tester", "review_requested", "executor", "review_requested", "tester", "review_requested", "reviewer", "completed"), true},
		"no tester":                                  {rr("executor", "review_requested", "reviewer", "completed"), false},
		"relay without re-verify":                    {rr("executor", "review_requested", "tester", "review_requested", "reviewer", "changes_requested", "tester", "review_requested", "executor", "review_requested", "reviewer", "completed"), false},
		"tester completed the card":                  {rr("executor", "review_requested", "tester", "completed"), false},
		"tester is implementer":                      {rr("tester", "review_requested", "reviewer", "completed"), false},
		"reviewer not last":                          {rr("executor", "review_requested", "tester", "review_requested", "reviewer", "completed", "executor", "completed"), false},
	} {
		if got := acceptanceChain(c.runs) != ""; got != c.ok {
			t.Errorf("%s: accepted=%v", name, got)
		}
	}
}

func TestToolchainAndLauncherAreSeparateReadiness(t *testing.T) {
	base := []Probe{{"compose", Healthy, ""}, {"gateway", Healthy, ""}, {"review:evidence", Healthy, ""}}
	got := Readiness(append(base, Probe{"development:go", Degraded, ""}))
	if got[0].Status != Healthy || got[2].Status != Degraded || !strings.Contains(got[2].Detail, "development:go") || CoreUsable(got) {
		t.Fatalf("missing toolchain must degrade DEVELOPMENT_RUNTIME only: %+v", got)
	}
	got = Readiness(append(base, Probe{"launcher", Unknown, ""}))
	if got[0].Status != Healthy || got[3].Status != Degraded || CoreUsable(got) {
		t.Fatalf("unusable launcher must degrade HOST_LAUNCHER only: %+v", got)
	}
	got = Readiness(append(base, Probe{"docker_acceptance", Inactive, ""}, Probe{"development_environment", Unqualified, ""}))
	if !CoreUsable(got) {
		t.Fatalf("optional acceptance probes affected readiness: %+v", got)
	}
}

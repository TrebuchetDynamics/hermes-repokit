package verify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
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
		"config get platform_toolsets --json": `{"cli":["kanban","file"],"telegram":["file","terminal"],"api_server":["file"]}`,
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
		strings.Replace(operationalKanban, `"tester",`, "", 1): Degraded,
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
		"kanban list --status done --assignee reviewer --sort completed-desc": "✓ t_0000000a  done  reviewer  first\n✓ t_0000000b  done  reviewer  second\n",
		"kanban runs t_0000000a --json":                                       `[{"profile":"executor","outcome":"completed"}]`,
		"kanban runs t_0000000b --json":                                       `[{"profile":"executor","outcome":"review_requested"},{"profile":"tester","outcome":"review_requested"},{"profile":"reviewer","outcome":"completed"}]`,
	}}
	if got := status(Gateway(context.Background(), id, r), "gateway"); got != Healthy {
		t.Fatal(got)
	}
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Healthy || !strings.Contains(p.Detail, "t_0000000b") {
		t.Fatal(p)
	}
	// A card completed by its implementer alone is not independent review.
	r.replies["kanban list --status done --assignee reviewer --sort completed-desc"] = "✓ t_0000000a  done  reviewer  first\n"
	r.replies["gateway status"] = "✗ Gateway is not running"
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Unqualified {
		t.Fatal(p)
	}
	if got := status(Gateway(context.Background(), id, r), "gateway"); got != Inactive {
		t.Fatal(got)
	}
}

func TestReadinessSeparatesConfiguredFromProved(t *testing.T) {
	healthy := []Probe{{"compose", Healthy, ""}, {"gateway", Healthy, ""}, {"kanban:dispatch", Healthy, ""}, {"development:go", Healthy, ""}, {"docker_acceptance", Inactive, ""}, {"memory", Inactive, ""}}
	got := Readiness(append(healthy, Probe{"review:evidence", Unqualified, ""}))
	if len(got) != 1 || got[0].Status != Unqualified || !CoreUsable(got) {
		t.Fatalf("configured core must be usable but unproved: %+v", got)
	}
	got = Readiness(append(healthy, Probe{"review:evidence", Healthy, ""}))
	if got[0].Status != Healthy {
		t.Fatalf("proved core must be healthy: %+v", got)
	}
	got = Readiness(append(healthy, Probe{"channel:telegram", Degraded, ""}, Probe{"review:evidence", Healthy, ""}))
	if got[0].Status != Degraded || !strings.Contains(got[0].Detail, "channel:telegram") || CoreUsable(got) {
		t.Fatalf("broken channel must block core: %+v", got)
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
		"planner is not an implementer":              {rr("planner", "review_requested", "tester", "review_requested", "reviewer", "completed"), false},
		"steward is not an implementer":              {rr("steward", "review_requested", "tester", "review_requested", "reviewer", "completed"), false},
	} {
		if got := acceptanceChain(c.runs) != ""; got != c.ok {
			t.Errorf("%s: accepted=%v", name, got)
		}
	}
}

// The qualifying card may be older than recent research or admin work: only
// reviewer-completed cards are read, within a bounded window.
func TestReviewEvidenceSurvivesUnrelatedLaterWork(t *testing.T) {
	id, base := integrationFixture(t)
	reviewed := []string{}
	replies := map[string]string{}
	for i := 0; i < reviewEvidenceWindow; i++ {
		card := fmt.Sprintf("t_%08x", i)
		reviewed = append(reviewed, "✓ "+card+"  done  reviewer  card "+card)
		replies["kanban runs "+card+" --json"] = `[{"profile":"reviewer","outcome":"completed"}]`
	}
	// The only qualifying card is the oldest inside the window.
	last := fmt.Sprintf("t_%08x", reviewEvidenceWindow-1)
	replies["kanban runs "+last+" --json"] = `[{"profile":"executor","outcome":"review_requested"},{"profile":"tester","outcome":"review_requested"},{"profile":"reviewer","outcome":"completed"}]`
	replies["kanban list --status done --assignee reviewer --sort completed-desc"] = strings.Join(reviewed, "\n") + "\n✓ t_000000ff  done  reviewer  old\n"
	replies["kanban runs t_000000ff --json"] = replies["kanban runs "+last+" --json"]
	r := &hermesRunner{integrationRunner: base, replies: replies}
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Healthy || !strings.Contains(p.Detail, last) {
		t.Fatalf("evidence inside the window lost: %+v", p)
	}
	for _, call := range r.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, "t_000000ff") {
			t.Fatal("read past the bounded window")
		}
		if strings.Contains(joined, "kanban list") && strings.Contains(joined, "--json") || strings.Contains(joined, "kanban show") {
			t.Fatalf("review evidence read whole card bodies: %s", joined)
		}
	}
}

// A channel configured only through the environment has no saved selection;
// it is judged by Hermes's effective tools, and its targets are never shown.
func TestEnvOnlyChannelsUseEffectiveTools(t *testing.T) {
	id, base := integrationFixture(t)
	r := &hermesRunner{integrationRunner: base, replies: map[string]string{
		"config get kanban --json":            operationalKanban,
		"config get platform_toolsets --json": `{"cli":["kanban"]}`,
		"send --list --json":                  `{"platforms":{"telegram":[{"id":"6586915095","name":"Private Name","type":"dm"}],"discord":[{"id":"1","name":"x"}],"cron":[]}}`,
		"tools list --platform telegram":      "Built-in toolsets (telegram):\n  ✓ enabled  web  🔍 Web\n  ✓ enabled  kanban  📌 Kanban\n  ✓ enabled  memory  💾 Memory\n",
		"tools list --platform discord":       "Built-in toolsets (discord):\n  ✗ disabled  kanban  📌 Kanban\n",
	}}
	probes := DefaultKanban(context.Background(), id, r)
	if status(probes, "channel:telegram") != Healthy || status(probes, "channel:discord") != Degraded || status(probes, "channel:cli") != Healthy || status(probes, "channel:cron") != "" {
		t.Fatalf("%+v", probes)
	}
	for _, p := range probes {
		if strings.Contains(p.Detail, "6586915095") || strings.Contains(p.Detail, "Private Name") {
			t.Fatalf("private target leaked: %+v", p)
		}
	}
	for out, want := range map[string][2]bool{
		"  ✓ enabled  kanban  📌 Kanban": {true, true}, "  ✗ disabled  kanban  📌 Kanban": {false, true},
		"  ✓ enabled  kanban_extra  x": {false, false}, "unexpected output": {false, false},
	} {
		if enabled, known := kanbanEnabled(out); enabled != want[0] || known != want[1] {
			t.Errorf("%q: enabled=%v known=%v", out, enabled, known)
		}
	}
}

// When default is the whole team, a card it completed counts only after a
// separate default run requested review: implementation and verification are
// distinct runs. Seven-profile history still counts until the first one lands.
func TestReviewEvidenceInTheSingleShape(t *testing.T) {
	id, base := integrationFixture(t)
	if err := os.WriteFile(filepath.Join(id.Root, ".hermes", team.ShapeFile), []byte("single\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := &hermesRunner{integrationRunner: base, replies: map[string]string{
		"kanban list --status done --assignee default --sort completed-desc":  "✓ t_0000000c  done  default  self only\n✓ t_0000000d  done  default  verified\n",
		"kanban runs t_0000000c --json":                                       `[{"profile":"default","outcome":"completed"}]`,
		"kanban runs t_0000000d --json":                                       `[{"profile":"default","outcome":"review_requested"},{"profile":"default","outcome":"changes_requested"},{"profile":"default","outcome":"review_requested"},{"profile":"default","outcome":"completed"}]`,
		"kanban list --status done --assignee reviewer --sort completed-desc": "",
	}}
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Healthy || !strings.Contains(p.Detail, "t_0000000d") || !strings.Contains(p.Detail, "separate default run") {
		t.Fatal(p)
	}
	// Only a self-completed card: not independent.
	r.replies["kanban list --status done --assignee default --sort completed-desc"] = "✓ t_0000000c  done  default  self only\n"
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Unqualified {
		t.Fatal(p)
	}
	// The seven-profile history still proves review until a single-shape card lands.
	r.replies["kanban list --status done --assignee reviewer --sort completed-desc"] = "✓ t_0000000b  done  reviewer  second\n"
	r.replies["kanban runs t_0000000b --json"] = `[{"profile":"executor","outcome":"review_requested"},{"profile":"tester","outcome":"review_requested"},{"profile":"reviewer","outcome":"completed"}]`
	if p := ReviewEvidence(context.Background(), id, r); p.Status != Healthy || !strings.Contains(p.Detail, "t_0000000b") {
		t.Fatal(p)
	}
}

func TestSingleShapeChain(t *testing.T) {
	type runs = []struct{ Profile, Outcome string }
	for name, c := range map[string]struct {
		runs runs
		ok   bool
	}{
		"verified":                     {runs{{"default", "review_requested"}, {"default", "completed"}}, true},
		"self-completed":               {runs{{"default", "completed"}}, false},
		"other profile completed":      {runs{{"default", "review_requested"}, {"executor", "completed"}}, false},
		"revision not re-verified":     {runs{{"default", "review_requested"}, {"default", "changes_requested"}, {"default", "completed"}}, false},
		"crashed verification retried": {runs{{"default", "review_requested"}, {"default", "crashed"}, {"default", "completed"}}, true},
	} {
		if got := singleChain(c.runs) != ""; got != c.ok {
			t.Errorf("%s: accepted=%v", name, got)
		}
	}
}

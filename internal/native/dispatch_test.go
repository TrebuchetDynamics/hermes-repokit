package native

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func dispatchTestIdentity(t *testing.T) target.Identity {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hermes-repokit.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// nativeGateway simulates the public Hermes CLI surface of one deployment.
type nativeGateway struct {
	kanban   map[string]any
	pid      int
	restart  int // PID after a restart script runs
	stats    string
	scripts  []string
	statuses int
}

func (g *nativeGateway) RunInput(_ context.Context, input io.Reader, _ string, args ...string) process.Result {
	joined := strings.Join(args, " ")
	if input != nil {
		data, _ := io.ReadAll(input)
		g.scripts = append(g.scripts, string(data))
		if strings.Contains(string(data), "gateway' 'restart'") || strings.Contains(string(data), "gateway' 'start'") {
			g.pid = g.restart
		}
		return process.Result{Output: "ok"}
	}
	switch {
	case strings.HasSuffix(joined, "config get kanban --json"):
		b, _ := json.Marshal(g.kanban)
		return process.Result{Output: string(b)}
	case strings.HasSuffix(joined, "gateway status"):
		g.statuses++
		if g.pid == 0 {
			return process.Result{Output: "✗ Gateway is not running\n"}
		}
		return process.Result{Output: "✓ Gateway is running (PID: " + itoa(g.pid) + ")\n"}
	case strings.HasSuffix(joined, "kanban stats --json"):
		return process.Result{Output: g.stats}
	}
	return process.Result{Err: io.EOF}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func operational() map[string]any {
	m := map[string]any{"auto_subscribe_on_create": true, "failure_limit": 2}
	for _, f := range DispatchPolicy() {
		m[f.Key] = f.Value
	}
	return m
}

func TestOperationalDispatchIsObservedWithoutRestart(t *testing.T) {
	g := &nativeGateway{kanban: operational(), pid: 42}
	state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0)
	if err != nil || state != "current" || len(g.scripts) != 0 {
		t.Fatalf("operational deployment mutated: %q %v %d", state, err, len(g.scripts))
	}
}

func TestSixRolePolicyUpgradesAllowlistAndRestarts(t *testing.T) {
	k := operational()
	k["dispatch_profiles"] = []any{"default", "researcher", "planner", "executor", "reviewer", "steward"}
	g := &nativeGateway{kanban: k, pid: 42, restart: 77, stats: `{"by_status":{"done":3}}`}
	state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0)
	if err != nil || state != "restarted" || len(g.scripts) != 1 || !strings.Contains(g.scripts[0], `"tester"`) {
		t.Fatalf("six-role policy not upgraded: %q %v %d", state, err, len(g.scripts))
	}
	// An owner who edited the six-role allowlist is still preserved.
	k["dispatch_profiles"] = []any{"default", "executor", "reviewer"}
	g = &nativeGateway{kanban: k, pid: 42, restart: 77, stats: `{"by_status":{"done":3}}`}
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0); err == nil || len(g.scripts) != 0 {
		t.Fatal("owner allowlist overwritten")
	}
	// The upgrade never restarts over a running card.
	k["dispatch_profiles"] = []any{"default", "researcher", "planner", "executor", "reviewer", "steward"}
	g = &nativeGateway{kanban: k, pid: 42, restart: 77, stats: `{"by_status":{"running":1}}`}
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0); err == nil || len(g.scripts) != 0 {
		t.Fatal("upgrade restarted over a running card")
	}
}

func TestOwnerChangedPolicyIsPreserved(t *testing.T) {
	k := operational()
	k["max_in_progress"] = float64(3)
	g := &nativeGateway{kanban: k, pid: 42}
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0); err == nil || len(g.scripts) != 0 {
		t.Fatal("owner dispatch policy overwritten")
	}
	g.kanban = map[string]any{"dispatch_in_gateway": "yes"}
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0); err == nil || len(g.scripts) != 0 {
		t.Fatal("ambiguous dispatch setting overwritten")
	}
}

func TestActivationSetsPolicyLastAndRestartsIdleGateway(t *testing.T) {
	g := &nativeGateway{kanban: map[string]any{"dispatch_in_gateway": false}, pid: 42, restart: 77, stats: `{"by_status":{"done":3}}`}
	state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0)
	if err != nil || state != "restarted" || len(g.scripts) != 1 {
		t.Fatalf("activation: %q %v", state, err)
	}
	script := g.scripts[0]
	last := strings.Index(script, "'kanban.dispatch_in_gateway' 'true'")
	for _, want := range []string{"'kanban.review_dispatch' 'true'", "'kanban.max_in_progress' '1'", "'kanban.auto_decompose' 'false'", "'kanban.orchestrator_profile' 'default'", `'kanban.dispatch_profiles' '["default","researcher","planner","executor","tester","reviewer","steward"]'`} {
		if i := strings.Index(script, want); i < 0 || i > last {
			t.Fatalf("%s missing or written after enabling dispatch:\n%s", want, script)
		}
	}
	if strings.Index(script, "'gateway' 'restart'") < last || !strings.Contains(script, `grep -q '"running"'`) {
		t.Fatal("restart must follow the policy and re-check running work under the lock")
	}
	for _, forbidden := range []string{"python", "kanban create", "kanban dispatch", "/proc", "gateway.log"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("activation used %q", forbidden)
		}
	}
}

func TestActivationRefusesRunningWorkAndUnconfirmedRestart(t *testing.T) {
	g := &nativeGateway{kanban: map[string]any{"dispatch_in_gateway": false}, pid: 42, restart: 77, stats: `{"by_status":{"running":1}}`}
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0); err == nil || len(g.scripts) != 0 {
		t.Fatal("restarted over a running card")
	}
	g.stats = `{"by_status":{}}`
	g.restart = 42 // restart did not produce a new process
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, time.Millisecond); err == nil {
		t.Fatal("unobserved replacement gateway certified")
	}
}

func TestActivationWithStoppedGatewayDoesNotStartIt(t *testing.T) {
	g := &nativeGateway{kanban: map[string]any{}, stats: `{"by_status":{}}`}
	state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, false, 3, 0)
	if err != nil || state != "not-running" || len(g.scripts) != 1 || strings.Contains(g.scripts[0], "'gateway' 'restart'") || strings.Contains(g.scripts[0], "'gateway' 'start'") {
		t.Fatalf("stopped gateway: %q %v", state, err)
	}
}

func TestGatewayPIDParsesPublicStatusOnly(t *testing.T) {
	if pid, err := GatewayPID("✓ Gateway is running (PID: 14037)\n  (Running manually)"); pid != 14037 || err != nil {
		t.Fatal(pid, err)
	}
	if pid, err := GatewayPID("✗ Gateway is not running"); pid != 0 || err != nil {
		t.Fatal(pid, err)
	}
	if _, err := GatewayPID("Traceback: boom"); err == nil {
		t.Fatal("unrecognized status accepted")
	}
}

func TestDispatchCheckRequiresResearcherEvidence(t *testing.T) {
	var rec taskRecord
	body := `{"task":{"id":"t_1","status":"done"},"runs":[{"profile":"researcher","status":"done","outcome":"completed","summary":"read","metadata":{"first_line":"# Title","changed_files":[]}}]}`
	if json.Unmarshal([]byte(body), &rec) != nil || !DispatchCheckPassed(rec, "# Title") {
		t.Fatal("valid evidence rejected")
	}
	for _, bad := range []string{
		strings.Replace(body, `"researcher"`, `"executor"`, 1),
		strings.Replace(body, `"# Title"`, `"# Other"`, 1),
		strings.Replace(body, `"changed_files":[]`, `"changed_files":["README.md"]`, 1),
		strings.Replace(body, `,"changed_files":[]`, ``, 1),
		strings.Replace(body, `"status":"done"}`, `"status":"running"}`, 1),
	} {
		var r taskRecord
		if json.Unmarshal([]byte(bad), &r) != nil || DispatchCheckPassed(r, "# Title") {
			t.Fatalf("invalid evidence accepted: %s", bad)
		}
	}
	stringMeta := `{"task":{"id":"t_1","status":"done"},"runs":[{"profile":"researcher","status":"done","outcome":"completed","summary":"s","metadata":"{\"first_line\":\"\",\"changed_files\":[]}"}]}`
	if json.Unmarshal([]byte(stringMeta), &rec) != nil || !DispatchCheckPassed(rec, "") {
		t.Fatal("string-encoded metadata or empty first line rejected")
	}
}

func TestExplicitSetupStartsStoppedGatewayInstallDoesNot(t *testing.T) {
	for _, kanban := range []map[string]any{{"dispatch_in_gateway": false}, operational()} {
		leave := &nativeGateway{kanban: kanban, restart: 55, stats: `{"by_status":{}}`}
		if state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", leave, false, 3, 0); err != nil || state != "not-running" {
			t.Fatalf("install path changed a stopped gateway: %q %v", state, err)
		}
		for _, script := range leave.scripts {
			if strings.Contains(script, "'gateway' 'start'") {
				t.Fatal("install path started the gateway")
			}
		}
		start := &nativeGateway{kanban: kanban, restart: 55, stats: `{"by_status":{}}`}
		state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", start, true, 3, 0)
		if err != nil || state != "started" || start.pid != 55 || !strings.Contains(start.scripts[len(start.scripts)-1], "'-p' 'default' 'gateway' 'start'") {
			t.Fatalf("setup did not start the stopped gateway: %q %v", state, err)
		}
	}
}

func TestGatewayStartMustBeObserved(t *testing.T) {
	g := &nativeGateway{kanban: operational(), restart: 0, stats: `{"by_status":{}}`}
	if _, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, true, 3, time.Millisecond); err == nil {
		t.Fatal("unobserved gateway start reported as started")
	}
}

func TestRunningGatewayIsNotStartedAgain(t *testing.T) {
	g := &nativeGateway{kanban: operational(), pid: 42, restart: 99}
	if state, err := convergeGateway(context.Background(), dispatchTestIdentity(t), "local", g, true, 3, 0); err != nil || state != "current" || len(g.scripts) != 0 {
		t.Fatalf("running operational gateway touched: %q %v %d", state, err, len(g.scripts))
	}
}

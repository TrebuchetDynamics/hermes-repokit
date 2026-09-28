package verify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestDefaultKanbanRequiresExplicitDispatchConfiguration(t *testing.T) {
	for _, state := range []string{"missing", "invalid", ""} {
		t.Run(state, func(t *testing.T) {
			id, r := integrationFixture(t)
			r.config = fmt.Sprintf(`{"kanban":{"fallback":"enabled"},"memory":{"fallback":"enabled"},"dispatch":%q}`, state)
			probes := DefaultKanban(context.Background(), id, r)
			p := probes[len(probes)-1]
			if p.Component != "kanban:dispatch" || p.Status != Degraded {
				t.Fatalf("dispatch drift not reported: %+v", probes)
			}
		})
	}
}

func TestDefaultKanbanUnavailableReportsBothToolsetsAndDispatch(t *testing.T) {
	id, r := integrationFixture(t)
	r.config = `{"kanban":{"fallback":"enabled"}}`
	probes := DefaultKanban(context.Background(), id, r)
	if len(probes) != 3 {
		t.Fatalf("missing failure probes: %+v", probes)
	}
	for i, name := range []string{"kanban:default", "memory:default", "kanban:dispatch"} {
		if probes[i].Component != name || probes[i].Status != Unknown {
			t.Fatalf("incomplete observation accepted: %+v", probes)
		}
	}
}

func TestDefaultKanbanReportsPlatformDrift(t *testing.T) {
	id, r := integrationFixture(t)
	r.config = `{"kanban":{"fallback":"enabled","cli":"enabled","telegram":"missing","discord":"fallback"},"memory":{"fallback":"enabled","telegram":"missing","discord":"fallback"},"dispatch":"manual"}`
	probes := DefaultKanban(context.Background(), id, r)
	found := map[string]bool{}
	for _, p := range probes {
		if p.Component == "kanban:default:telegram" || p.Component == "memory:default:telegram" {
			found[p.Component] = true
			if p.Status != Degraded {
				t.Fatal(p)
			}
		}
		if p.Component == "kanban:dispatch" && p.Status == Inactive && strings.Contains(p.Detail, "dispatch_in_gateway=false") && strings.Contains(p.Detail, "bootstrap") {
			found[p.Component] = true
		}
	}
	if len(found) != 3 {
		t.Fatalf("missing Telegram drift: %+v", probes)
	}
}

func TestDefaultKanbanReadOnlyFixture(t *testing.T) {
	cmd := exec.Command("python3", "-B", "kanban_test.py")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	} else if !strings.Contains(string(out), "OK") {
		t.Fatal(string(out))
	}
}

func TestOptionalChannelsAndUnresolvedMemoryDoNotClaimFallbackHealth(t *testing.T) {
	id, r := integrationFixture(t)
	r.config = `{"kanban":{"fallback":"enabled","telegram":"not-configured","discord":"fallback"},"memory":{"fallback":"not-configured","telegram":"not-configured","discord":"unknown","slack":"fallback"},"dispatch":"manual"}`
	probes := DefaultKanban(context.Background(), id, r)
	want := map[string]Status{
		"kanban:default:telegram": Inactive,
		"kanban:default:discord":  Healthy,
		"memory:default:fallback": Inactive,
		"memory:default:telegram": Inactive,
		"memory:default:discord":  Unknown,
		"memory:default:slack":    Unknown,
	}
	for _, p := range probes {
		if state, ok := want[p.Component]; ok {
			if p.Status != state {
				t.Fatalf("unsupported channel status: %+v; want %v", p, state)
			}
			delete(want, p.Component)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing observations: %+v", want)
	}
}

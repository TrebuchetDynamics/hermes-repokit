package cli

import (
	"bytes"
	"strings"
	"testing"
)

// On a terminal the team step redraws one bar in place and clears it; piped,
// each step is its own line. Other output is ignored.
func TestProgressRendersTeamSteps(t *testing.T) {
	var live bytes.Buffer
	onLine, done := ui{out: &live, live: true}.progress("Team")
	onLine("Native shared Kanban initialized")
	onLine("REPOKIT_PROGRESS=1/4 researcher: create profile, settings and skills")
	onLine("REPOKIT_PROGRESS=4/4 steward: create profile, settings and skills")
	done()
	out := live.String()
	if !strings.Contains(out, "\r\033[K") || !strings.Contains(out, "█████░░░░░░░░░░░░░░░ 1/4 researcher") || !strings.Contains(out, "████████████████████ 4/4 steward") || strings.Contains(out, "Kanban") || strings.Contains(out, "\n") || !strings.HasSuffix(out, "\r\033[K") {
		t.Fatalf("live progress: %q", out)
	}
	var piped bytes.Buffer
	onLine, done = ui{out: &piped}.progress("Team")
	onLine("REPOKIT_PROGRESS=2/7 planner: new identity")
	onLine("REPOKIT_PROGRESS=bad")
	done()
	if got := piped.String(); strings.Count(got, "\n") != 1 || !strings.Contains(got, "Team") || !strings.Contains(got, "2/7 planner: new identity") || strings.Contains(got, "\r") {
		t.Fatalf("piped progress: %q", got)
	}
}

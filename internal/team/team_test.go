package team

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// RepoKit is one profile, default, bound to its repository.
func TestOneProfileIdentity(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-123", Root: "/private/atlas"}
	roles := ForRepository(id)
	if len(roles) != 1 || roles[0].Name != "default" {
		t.Fatalf("roster %+v", roles)
	}
	soul := roles[0].Soul
	for _, want := range []string{
		"Repository name: atlas", "Stable repository ID: repokit-123", "Native profile: default",
		"\n# RepoKit Agent Identity\n", "Hermes is the runtime", "Dispatch policy: bootstrap off; operational automatic",
		"## Self-maintenance", "## Talking with the owner", "reply\nexactly [SILENT]",
		`reviewer="default"`, "## Running a card", "## Verifying a card", "delegate_task", "Memory is on for you",
		`workspace_path "/workspace"`, "Never download, install or run another model",
		"create the Hermes cron job with a\nmonitor script", "part of the card's work: write it", "never blocks finished work",
		"No test was deleted, weakened, skipped or special-cased", "A round that finds a\nnew, different defect is progress",
		"fails on something an earlier round already asked to fix, or after four", `"work on repo" is a mandate`,
		"holds up only its own card, never other\nwork",
		"Never call kanban_complete in the run that implemented the change",
	} {
		if !strings.Contains(soul, want) {
			t.Errorf("SOUL lacks %q", want)
		}
	}
	for _, gone := range []string{"executor", "tester", "reviewer=\"tester\"", "steward", "seven", "Nerve", "supervision", id.Root} {
		if strings.Contains(soul, gone) {
			t.Errorf("SOUL still says %q", gone)
		}
	}
	if soul == ForRepository(target.Identity{Name: "other", Project: "repokit-456"})[0].Soul {
		t.Fatal("distinct repositories share identity")
	}
	roles[0].Toolsets[0] = "changed"
	if ForRepository(id)[0].Toolsets[0] != CoordinatorTools[0] {
		t.Fatal("caller mutated the roster")
	}
}

// default runs every card, so it is granted whole-card effort, checkpoints and
// room for a verification run's subagents, on top of the autonomy grants.
func TestDefaultGrants(t *testing.T) {
	r := Roster()[0]
	for key, want := range map[string]any{
		"approvals.mode": "off", "security.protected_instruction_files": false, "web.backend": "exa",
		"kanban.dispatch_interval_seconds": 10, "goals.max_turns": 100, "agent.max_turns": 0,
		"agent.reasoning_effort": "high", "checkpoints.enabled": true, "delegation.oneshot_max_children": 4,
	} {
		if r.Settings[key] != want {
			t.Errorf("%s = %v, want %v", key, r.Settings[key], want)
		}
	}
	for _, need := range []string{"kanban", "memory", "delegation"} {
		found := false
		for _, tool := range r.Toolsets {
			found = found || tool == need
		}
		if !found {
			t.Errorf("default lacks %s", need)
		}
	}
	for _, skill := range r.Skills {
		if parts := strings.Split(skill, "/"); len(parts) != 3 || parts[0] != "official" {
			t.Errorf("skill %q is not an official category/name identifier", skill)
		}
	}
	kept := WithApprovalPrompts(Roster())[0].Settings
	if _, ok := kept["approvals.mode"]; ok || kept["checkpoints.enabled"] != true {
		t.Fatalf("approval-prompt posture: %v", kept)
	}
}

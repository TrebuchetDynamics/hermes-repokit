package team

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestStableGenericRoster(t *testing.T) {
	roles := Roster()
	if len(roles) != 7 {
		t.Fatalf("role count: %d", len(roles))
	}
	for i, name := range []string{"default", "researcher", "planner", "executor", "tester", "reviewer", "steward"} {
		r := roles[i]
		if r.Name != name || r.Description == "" {
			t.Fatalf("role: %+v", r)
		}
		if !strings.HasPrefix(r.Soul, "# RepoKit Agent Identity\n") || strings.Count(r.Soul, "# Role:") != 1 {
			t.Fatalf("invalid soul %s", name)
		}
		for _, contract := range []string{"Hermes Kanban", "parent handoffs", "secrets", "irreversible", "memory"} {
			if !strings.Contains(r.Soul, contract) {
				t.Errorf("%s missing %s", name, contract)
			}
		}
	}
	if strings.Join(roles[0].Toolsets, ",") != strings.Join(CoordinatorTools, ",") || strings.Join(roles[0].Required, ",") != "kanban" {
		t.Fatal("coordinator has implementation authority")
	}
	if !strings.Contains(roles[3].Soul, "same-card review") || !strings.Contains(roles[5].Soul, "Do not modify") || !strings.Contains(roles[4].Soul, "must not modify") {
		t.Fatal("missing review boundary")
	}
}

func TestStewardOwnsLifecycleAndDefaultOwnsConversation(t *testing.T) {
	roles := Roster()
	for _, text := range []string{"Skills first", "Never attempt to delete", "explicit user authorization", "retire"} {
		if !strings.Contains(roles[6].Soul, text) {
			t.Errorf("steward missing %s", text)
		}
	}
	for _, text := range []string{"lightweight questions directly", "Do not force every", "steward"} {
		if !strings.Contains(roles[0].Soul, text) {
			t.Errorf("default missing %s", text)
		}
	}
}

func TestExecutorHasNativeCodingToolsWithoutOrchestratorKanban(t *testing.T) {
	role := Roster()[3]
	tools := "," + strings.Join(role.Toolsets, ",") + ","
	for _, want := range []string{"file", "terminal", "code_execution", "skills"} {
		if !strings.Contains(tools, ","+want+",") {
			t.Errorf("executor lacks %s", want)
		}
	}
	if strings.Contains(tools, ",kanban,") {
		t.Fatal("worker has persistent orchestrator tools")
	}
}

func TestTesterCannotEditRepositoryFiles(t *testing.T) {
	role := Roster()[4]
	if role.Name != "tester" || strings.Join(role.Required, ",") != "terminal" {
		t.Fatalf("tester required tools: %+v", role.Required)
	}
	// Hermes's file bundle cannot be read-only, so tester reads through terminal.
	for _, tool := range role.Toolsets {
		if tool == "file" {
			t.Fatal("tester granted the file-editing toolset")
		}
	}
}

// Every role's required toolsets are part of its granted baseline, and every
// granted skill is an official catalog identifier (category/name).
func TestRoleBaselinesAreConsistent(t *testing.T) {
	for _, role := range Roster() {
		granted := map[string]bool{}
		for _, tool := range role.Toolsets {
			granted[tool] = true
		}
		for _, tool := range role.Required {
			if !granted[tool] {
				t.Errorf("%s requires %s without granting it", role.Name, tool)
			}
		}
		for _, skill := range role.Skills {
			if parts := strings.Split(skill, "/"); len(parts) != 3 || parts[0] != "official" {
				t.Errorf("%s skill %q is not an official category/name identifier", role.Name, skill)
			}
		}
	}
	if len(Roster()[3].Skills) == 0 || Roster()[3].Name != "executor" {
		t.Fatal("executor lost its granted skills")
	}
}

// The owner runs the team without approval prompts: every profile is granted
// approvals off and the protected instruction-file gate lifted, default still
// routes instruction-file edits to executor, and every role can attach files.
func TestProtectedInstructionWritesAreOwnerSteps(t *testing.T) {
	for _, r := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		if !strings.Contains(r.Soul, "runs without approval prompts") || !strings.Contains(r.Soul, "hermes kanban attach <your card id> <path>") {
			t.Errorf("%s lacks the no-approval or attachment rule", r.Name)
		}
		if r.Name == "default" && (!strings.Contains(r.Soul, "Assign edits\nto AGENTS.md") || !strings.Contains(r.Soul, "tell them in one line")) {
			t.Error("default does not route protected writes to executor")
		}
		// RepoKit's default posture: no approval prompts on any profile.
		if r.Settings["approvals.mode"] != "off" || r.Settings["security.protected_instruction_files"] != false {
			t.Errorf("%s approval settings: %v", r.Name, r.Settings)
		}
	}
	// An owner who keeps prompts at setup gets no approval grants, and every
	// other granted setting stays.
	for _, r := range WithApprovalPrompts(ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"})) {
		if _, ok := r.Settings["approvals.mode"]; ok {
			t.Errorf("%s still granted approvals off", r.Name)
		}
		if _, ok := r.Settings["security.protected_instruction_files"]; ok {
			t.Errorf("%s still lifts the protected-file gate", r.Name)
		}
		if r.Name == "default" && r.Settings["kanban.dispatch_interval_seconds"] != 10 {
			t.Errorf("default lost its other grants: %v", r.Settings)
		}
	}
}

// Live boards showed duplicate follow-up cards, over-routing and handoffs that
// made the next role rediscover work; the contract guards each.
func TestTeamBehaviorContract(t *testing.T) {
	for _, r := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		if !strings.Contains(r.Soul, "list the board's open cards") {
			t.Errorf("%s may create duplicate cards", r.Name)
		}
	}
	want := map[string][]string{
		"default":    {"Profiles are capabilities, not stations", "goes\nstraight to executor"},
		"researcher": {`"affected_files"`, `"recommended_next"`},
		"planner":    {`"likely_files"`, `"non_goals"`},
		"executor":   {`"changed_artifacts"`, `"verification"`},
		"tester":     {`"commands"`, `"edge_cases"`, `"verdict"`},
		"reviewer":   {`"verdict"`, `"concerns"`, `"reason"`},
	}
	for _, r := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		if r.Name == "default" && !strings.Contains(r.Soul, "Pin a skill to a card only after confirming the assignee has") {
			t.Error("default may pin skills the assignee lacks")
		}
	}
	for _, r := range Roster() {
		for _, text := range want[r.Name] {
			if !strings.Contains(r.Soul, text) {
				t.Errorf("%s handoff/routing lacks %s", r.Name, text)
			}
		}
	}
}

// Workers get the whole card: no turn cap and high reasoning effort, granted
// at creation or reset (the owner's to change). Executor works test-first,
// researcher researches before concluding, and every worker keeps going
// until acceptance is met; default puts open-ended cards in goal mode.
func TestWorkersAreAutonomousTestFirstAndThorough(t *testing.T) {
	for _, r := range Roster() {
		worker := r.Name != "default" && r.Name != "steward"
		if worker && (r.Settings["agent.max_turns"] != 0 || r.Settings["agent.reasoning_effort"] != "high") {
			t.Errorf("%s lacks worker effort: %v", r.Name, r.Settings)
		}
		if !worker && r.Settings["agent.max_turns"] != nil {
			t.Errorf("%s should keep its own turn budget", r.Name)
		}
	}
	souls := map[string]string{}
	for _, r := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		souls[r.Name] = r.Soul
		if !strings.Contains(r.Soul, "Work autonomously until the card's acceptance is met") {
			t.Errorf("%s may stop at the first obstacle", r.Name)
		}
	}
	souls["default-roster"], souls["default-owner"] = souls["default"], souls["default"]
	for name, text := range map[string]string{"executor": "Work test-first", "researcher": "Research thoroughly before concluding", "default": "Never\nuse goal_mode on a card that needs same-card review", "default-roster": "never block because a documented agent does\nnot exist here", "default-owner": "An explicit instruction from the owner in this conversation is an owner\ndecision"} {
		if !strings.Contains(souls[name], text) {
			t.Errorf("%s lacks %q", name, text)
		}
	}
}

// The owner's coordinator is granted memory with its full toolset; no worker
// baseline includes memory (an owner who adds it keeps it).
func TestWorkerBaselinesGrantNoMemory(t *testing.T) {
	for _, role := range Roster()[1:] {
		for _, name := range append(append([]string{}, role.Toolsets...), role.Required...) {
			if name == "memory" {
				t.Errorf("%s is granted memory", role.Name)
			}
		}
	}
}

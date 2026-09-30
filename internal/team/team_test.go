package team

import (
	"strings"
	"testing"
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
	if strings.Join(roles[0].Toolsets, ",") != "kanban,memory" {
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
	for _, want := range []string{"file", "terminal", "code_execution", "skills", "memory"} {
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

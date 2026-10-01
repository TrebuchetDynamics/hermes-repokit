package team

import (
	"crypto/sha256"
	"fmt"
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

// The previous release's SOULs are pinned: these are the exact SOULs v0.2.3
// installed for this identity (hashes computed from the v0.2.3 source).
func TestPreviousReleaseSoulsAreFrozen(t *testing.T) {
	want := map[string]string{
		"default":    "a0e888b68ba09421634bfe220e8f278366e8c788a56d36498ac070b0bfb46f56",
		"researcher": "92af343d24bed01aa7099c64a0a724aa659766807c5b65c2339a0682da569521",
		"planner":    "7b5c202af506368a3f6412951b43e562b47ac51b7aa2b33572d69b7d86ae1269",
		"executor":   "a2a93014c3d31f6599929e542aff8e0aa0e5c320f08d26342a385e188378f706",
		"tester":     "05f2f5b73811dadff4463ac564a07c0efc55040f4d3c399d1f051d52f6637b17",
		"reviewer":   "799ff9d795d6f8a9b6e7b9f1f2118371fab642252c1ae2eb3a3dba7844ef82f2",
		"steward":    "47f5d4a6d78a9b391dccad4671c9d946ec88cb7598e2c8e6de9cf81cef0e7c54",
	}
	if PreviousRelease != "v0.2.3" {
		t.Fatalf("PreviousRelease is %s; regenerate souls/previous from that release and update these hashes", PreviousRelease)
	}
	for _, r := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(r.PreviousSoul))); got != want[r.Name] {
			t.Errorf("%s previous SOUL changed: %s", r.Name, got)
		}
		if strings.Contains(r.PreviousSoul, "{{") {
			t.Errorf("%s previous SOUL has an unfilled placeholder", r.Name)
		}
	}
}

// Hermes requires a per-write human approval for protected instruction files,
// which a Kanban worker can never obtain: only executor has the gate lifted,
// default routes such edits to it, and every role knows how to attach files.
func TestProtectedInstructionWritesAreOwnerSteps(t *testing.T) {
	for _, r := range ForRepository(target.Identity{Name: "atlas", Project: "repokit-123"}) {
		if !strings.Contains(r.Soul, "RepoKit lifts that gate for executor only") || !strings.Contains(r.Soul, "hermes kanban attach <your card id> <path>") {
			t.Errorf("%s lacks the protected instruction or attachment rule", r.Name)
		}
		if r.Name == "default" && (!strings.Contains(r.Soul, "Assign edits\nto AGENTS.md") || !strings.Contains(r.Soul, "tell them in one line")) {
			t.Error("default does not route protected writes to executor")
		}
		settings := r.Settings["security.protected_instruction_files"]
		if (r.Name == "executor") != (settings == false) {
			t.Errorf("%s protected-instruction setting: %v", r.Name, settings)
		}
	}
}

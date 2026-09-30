package team

import (
	"crypto/sha256"
	"fmt"
	"os"
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
	if role.Name != "tester" || strings.Join(role.Toolsets, ",") != "terminal,memory" {
		t.Fatalf("tester tools: %+v", role)
	}
}

func TestArchiveFilesAreNamedByTheirDigest(t *testing.T) {
	if bad := archiveDigestsMatch(); len(bad) > 0 {
		t.Fatalf("archived generations must never change; edited: %v", bad)
	}
	if len(archivedSouls(target.Identity{Name: "atlas", Project: "repokit-x"}, "executor")) == 0 {
		t.Fatal("archive not embedded")
	}
}

// Every SOUL a released RepoKit revision could write for this identity (current
// and history, dumped from each revision that changed internal/team) must stay
// recognized. Editing a historical generation turns deployed RepoKit output
// into apparent owner customization, which is then never upgraded.
func TestEveryShippedSoulStaysRecognized(t *testing.T) {
	id := target.Identity{Name: "atlas", Project: "repokit-0123456789abcdef01234567", Container: "hermes-atlas"}
	known := map[string]bool{}
	for _, role := range ForRepository(id) {
		for _, soul := range append([]string{role.Soul}, role.History()...) {
			known[fmt.Sprintf("%s %x", role.Name, sha256.Sum256([]byte(soul)))] = true
		}
	}
	data, err := os.ReadFile("testdata/shipped-souls.sha256")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if !known[line] {
			t.Errorf("shipped SOUL no longer recognized: %s", line)
		}
	}
}

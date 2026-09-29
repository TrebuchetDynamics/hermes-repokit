package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
)

// removeRunner records Docker mutations instead of performing them.
type removeRunner struct {
	*foundationRunner
	absent    bool
	mutations []string
}

func (r *removeRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	call := strings.Join(args, " ")
	switch {
	case strings.Contains(call, " compose ") && strings.Contains(call, " down "):
		r.mutations = append(r.mutations, "down "+call)
		return process.Result{}
	case strings.Contains(call, " image rm "):
		r.mutations = append(r.mutations, "rmi "+args[len(args)-1])
		return process.Result{}
	case strings.Contains(call, "container inspect") && r.absent:
		return process.Result{Err: errors.New("No such container")}
	}
	return r.foundationRunner.Run(ctx, p, args...)
}

func installedForRemoval(t *testing.T) (App, *removeRunner) {
	t.Helper()
	a, base := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	base.runtime = developmentRuntimeFixture(base.id)
	r := &removeRunner{foundationRunner: base}
	a.Runner = r
	return a, r
}

func TestRemoveDeletesOnlyTheRecognizedDeployment(t *testing.T) {
	a, r := installedForRemoval(t)
	exclude := excludePath(t, a.Directory)
	data, _ := os.ReadFile(exclude)
	if err := os.WriteFile(exclude, append([]byte("# owner rule\n*.swp\n"), data...), 0644); err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	command := filepath.Join(home, ".local", "bin", r.id.Container)
	if dest, err := os.Readlink(command); err != nil || dest != r.id.Launcher {
		t.Fatalf("install did not create the host command: %v", err)
	}
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	code, out, diag := invoke(t, a, "remove")
	if code != 0 {
		t.Fatalf("remove: %s %s", out, diag)
	}
	for _, gone := range []string{".hermes", ".hermes-repokit.lock"} {
		if _, err := os.Lstat(filepath.Join(a.Directory, gone)); !os.IsNotExist(err) {
			t.Fatalf("%s survived", gone)
		}
	}
	if _, err := os.Lstat(command); !os.IsNotExist(err) {
		t.Fatal("host command survived")
	}
	if got, _ := os.ReadFile(exclude); strings.Contains(string(got), lockExcludeEntry) || !strings.Contains(string(got), "# owner rule\n*.swp") {
		t.Fatalf("exclude file not cleaned exactly:\n%s", got)
	}
	joined := strings.Join(r.mutations, "\n")
	if !strings.Contains(joined, "-f "+r.id.Compose+" --profile docker-tests down --volumes --remove-orphans") || !strings.Contains(joined, "rmi repokit/"+r.id.Container+":") {
		t.Fatalf("unexpected Docker mutations:\n%s", joined)
	}
	if status := gitOut(t, a.Directory, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("remove left repository changes:\n%s", status)
	}
}

func TestRemoveChangesNothingWithoutExactConfirmation(t *testing.T) {
	for _, answer := range []string{"", "yes", "y", "WRONG"} {
		a, r := installedForRemoval(t)
		a.Confirm = func(string) (string, error) { return answer, nil }
		if code, _, _ := invoke(t, a, "remove"); code == 0 || len(r.mutations) != 0 {
			t.Fatalf("answer %q removed something: %v", answer, r.mutations)
		}
		if _, err := os.Stat(filepath.Join(a.Directory, ".hermes", "compose.yaml")); err != nil {
			t.Fatalf("answer %q deleted state", answer)
		}
	}
}

func TestRemoveRequiresInteractiveTerminal(t *testing.T) {
	a, r := installedForRemoval(t)
	a.Stdin = strings.NewReader(r.id.Name + "\n") // piped input is not a confirmation
	code, _, diag := invoke(t, a, "remove")
	if code == 0 || !strings.Contains(diag, "interactive terminal") || len(r.mutations) != 0 {
		t.Fatalf("non-interactive removal accepted: %s", diag)
	}
}

func TestRemoveRefusesStateItDidNotCreate(t *testing.T) {
	a, r := installedForRemoval(t)
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	data, _ := os.ReadFile(r.id.Compose)
	if err := os.WriteFile(r.id.Compose, append(data, []byte("# owner edit\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if code, _, diag := invoke(t, a, "remove"); code == 0 || !strings.Contains(diag, "not a RepoKit-generated Compose") || len(r.mutations) != 0 {
		t.Fatalf("edited deployment removed: %s", diag)
	}
	b, r2 := installedForRemoval(t)
	b.Confirm = func(string) (string, error) { return r2.id.Name, nil }
	r2.runtime = strings.Replace(r2.runtime, r2.id.Project, "someone-else", 1)
	if code, _, diag := invoke(t, b, "remove"); code == 0 || !strings.Contains(diag, "does not belong") || len(r2.mutations) != 0 {
		t.Fatalf("foreign container removed: %s", diag)
	}
	if _, err := os.Stat(filepath.Join(b.Directory, ".hermes")); err != nil {
		t.Fatal("state deleted after refusal")
	}
}

func TestRemovePreservesForeignHostCommandAndHandlesAbsentContainer(t *testing.T) {
	a, r := installedForRemoval(t)
	r.absent = true
	home, _ := os.UserHomeDir()
	command := filepath.Join(home, ".local", "bin", r.id.Container)
	os.Remove(command)
	if err := os.Symlink("/usr/bin/true", command); err != nil {
		t.Fatal(err)
	}
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	if code, _, diag := invoke(t, a, "remove"); code != 0 {
		t.Fatal(diag)
	}
	if dest, err := os.Readlink(command); err != nil || dest != "/usr/bin/true" {
		t.Fatal("removed a host command RepoKit did not create")
	}
	if !strings.Contains(strings.Join(r.mutations, "\n"), " down ") {
		t.Fatal("compose down must still run to clean a leftover network")
	}
}

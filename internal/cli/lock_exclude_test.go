package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return string(out)
}

func excludePath(t *testing.T, root string) string {
	t.Helper()
	p := strings.TrimSpace(gitOut(t, root, "rev-parse", "--git-path", "info/exclude"))
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p
}

func TestInstallKeepsGitStatusClean(t *testing.T) {
	a, _ := foundationApp(t)
	exclude := excludePath(t, a.Directory)
	if err := os.WriteFile(exclude, []byte("# owner rule\n*.swp"), 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if code, _, diag := invoke(t, a, "install"); code != 0 || strings.Contains(diag, "may appear in git status") {
			t.Fatalf("install %d: %s", i, diag)
		}
	}
	if status := gitOut(t, a.Directory, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("RepoKit files visible to git:\n%s", status)
	}
	data, _ := os.ReadFile(exclude)
	if !strings.HasPrefix(string(data), "# owner rule\n*.swp\n") || strings.Count(string(data), lockExcludeEntry) != 1 {
		t.Fatalf("owner exclude content not preserved or entry duplicated:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(a.Directory, ".gitignore")); !os.IsNotExist(err) {
		t.Fatal("tracked .gitignore was created or edited")
	}
}

func TestInstallSkipsExcludeWhenAlreadyIgnored(t *testing.T) {
	a, _ := foundationApp(t)
	if err := os.WriteFile(filepath.Join(a.Directory, ".gitignore"), []byte(".hermes-repokit.lock\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	if data, _ := os.ReadFile(excludePath(t, a.Directory)); strings.Contains(string(data), lockExcludeEntry) {
		t.Fatal("wrote an exclude entry although the owner already ignores the lock")
	}
}

func TestInstallWarnsInsteadOfFollowingExcludeSymlink(t *testing.T) {
	a, _ := foundationApp(t)
	exclude := excludePath(t, a.Directory)
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(target, []byte("outside\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Remove(exclude)
	if err := os.Symlink(target, exclude); err != nil {
		t.Fatal(err)
	}
	code, _, diag := invoke(t, a, "install")
	if code != 0 || !strings.Contains(diag, "may appear in git status") || !strings.Contains(diag, lockExcludeEntry) {
		t.Fatalf("expected a warning with the manual fix: code=%d diag=%s", code, diag)
	}
	if data, _ := os.ReadFile(target); string(data) != "outside\n" {
		t.Fatal("followed the exclude symlink")
	}
}

func TestInstallExcludesLockInLinkedWorktree(t *testing.T) {
	a, r := foundationApp(t)
	main := a.Directory
	if err := os.WriteFile(filepath.Join(main, "README.md"), []byte("# x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitOut(t, main, "add", "README.md")
	gitOut(t, main, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-qm", "init")
	linked := filepath.Join(filepath.Dir(main), "linked-tree")
	gitOut(t, main, "worktree", "add", "-q", linked)
	if err := os.Chmod(linked, 0700); err != nil {
		t.Fatal(err)
	}
	a.Directory = linked
	r.id.Root = linked
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	if status := gitOut(t, linked, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("lock visible in linked worktree:\n%s", status)
	}
}

func TestPlanDisclosesExcludeEntry(t *testing.T) {
	a, _ := foundationApp(t)
	_, out, _ := invoke(t, a, "plan")
	var p Plan
	if json.Unmarshal([]byte(out), &p) != nil || !strings.Contains(strings.Join(p.ProposedChanges, "\n"), ".git/info/exclude") {
		t.Fatalf("plan does not disclose the exclude entry: %s", out)
	}
}

package cli

import (
	"context"
	"errors"
	"io/fs"
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
	volumes   map[string]bool // existing Docker volumes
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
	case strings.Contains(call, " volume inspect "):
		if r.volumes[args[len(args)-1]] {
			return process.Result{Output: "[{}]"}
		}
		return process.Result{Err: errors.New("no such volume")}
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
	if code, _, diag := invoke(t, a, "remove"); code == 0 || !strings.Contains(diag, "not RepoKit's current generated Compose") || len(r.mutations) != 0 {
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

// Go marks its module cache read-only. Deployments installed before the
// toolchain cache volume kept that cache in .hermes/development, so remove
// must delete read-only directories inside .hermes, and nothing outside it.
func TestRemoveDeletesReadOnlyToolchainCache(t *testing.T) {
	a, r := installedForRemoval(t)
	module := filepath.Join(a.Directory, ".hermes", "development", "go-mod", "example.test", "mod@v1.0.0")
	if err := os.MkdirAll(module, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "mod.go"), []byte("package mod\n"), 0444); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{module, filepath.Dir(module)} {
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatal(err)
		}
	}
	outside := t.TempDir()
	if err := os.Chmod(outside, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(outside, 0700) })
	if err := os.Symlink(outside, filepath.Join(a.Directory, ".hermes", "development", "link")); err != nil {
		t.Fatal(err)
	}
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	code, out, diag := invoke(t, a, "remove")
	if code != 0 || strings.Contains(diag, "could not be deleted") {
		t.Fatalf("read-only cache blocked removal: code=%d out=%s diag=%s", code, out, diag)
	}
	if _, err := os.Lstat(filepath.Join(a.Directory, ".hermes")); !os.IsNotExist(err) {
		t.Fatal(".hermes survived")
	}
	if info, err := os.Stat(outside); err != nil || info.Mode().Perm() != 0555 {
		t.Fatalf("remove changed a directory outside .hermes: %v %v", info, err)
	}
}

func TestRemovePreviewNamesTheVolumesItDeletes(t *testing.T) {
	for _, exists := range []bool{true, false} {
		a, base := foundationApp(t)
		if err := os.WriteFile(filepath.Join(a.Directory, "go.mod"), []byte("module example.test/demo\n\ngo 1.26.0\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if code, _, diag := invoke(t, a, "install"); code != 0 {
			t.Fatal(diag)
		}
		base.runtime = developmentRuntimeFixture(base.id)
		volume := base.id.Project + "_toolchain-cache"
		r := &removeRunner{foundationRunner: base, volumes: map[string]bool{volume: exists}}
		a.Runner = r
		a.Confirm = func(string) (string, error) { return "no", nil }
		_, out, _ := invoke(t, a, "remove")
		if listed := strings.Contains(out, "volumes "+volume+" "); listed != exists {
			t.Fatalf("volume exists=%v but listed=%v:\n%s", exists, listed, out)
		}
	}
}

// leftoverRunner answers the label-scoped listing and removal calls that
// `remove` uses once .hermes is gone.
type leftoverRunner struct {
	*removeRunner
}

func (r *leftoverRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	call := strings.Join(args, " ")
	label := "label=com.docker.compose.project=" + r.id.Project
	switch {
	case strings.Contains(call, "{{.Config.Image}}"):
		return process.Result{Output: "repokit/" + r.id.Container + ":abc\n"}
	case strings.Contains(call, " volume ls ") && strings.HasSuffix(call, label):
		return process.Result{Output: r.id.Project + "_toolchain-cache\n"}
	case strings.Contains(call, " network ls ") && strings.HasSuffix(call, label):
		return process.Result{Output: "n1\n"}
	case strings.Contains(call, " container rm "), strings.Contains(call, " network rm "), strings.Contains(call, " volume rm "):
		r.mutations = append(r.mutations, call[strings.Index(call, " ")+1:])
		return process.Result{}
	}
	return r.removeRunner.Run(ctx, p, args...)
}

// An install whose .hermes is gone leaves its container, volumes, network,
// image and host link; remove clears exactly those after confirmation so the
// next install starts cleanly.
func TestRemoveClearsLeftoversWhenStateIsGone(t *testing.T) {
	a, base := installedForRemoval(t)
	r := &leftoverRunner{base}
	a.Runner = r
	state := filepath.Join(a.Directory, ".hermes")
	filepath.WalkDir(state, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			os.Chmod(p, 0700)
		}
		return nil
	})
	if err := os.RemoveAll(state); err != nil {
		t.Fatal(err)
	}
	base.names = r.id.Container
	_, out, diag := invoke(t, a, "plan")
	if report := out + diag; !strings.Contains(report, "left from an earlier install") || strings.Contains(report, "PATH collision") {
		t.Fatalf("plan did not explain the leftover: %s", report)
	}
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	code, out, diag := invoke(t, a, "remove")
	if code != 0 {
		t.Fatalf("remove: %s %s", out, diag)
	}
	got := strings.Join(r.mutations, "\n")
	for _, want := range []string{"container rm --force " + r.id.Container, "volume rm " + r.id.Project + "_toolchain-cache", "network rm n1", "rmi repokit/" + r.id.Container + ":abc"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", r.id.Container)); !os.IsNotExist(err) {
		t.Fatal("host link survived")
	}
	if _, err := os.Lstat(filepath.Join(a.Directory, ".hermes-repokit.lock")); !os.IsNotExist(err) {
		t.Fatal("lock survived")
	}
}

func TestRemoveLeftoversRefusesAForeignContainer(t *testing.T) {
	a, base := installedForRemoval(t)
	base.runtime = `{"status":"running","project":"someone-else","workspace":"/elsewhere","home":"/elsewhere/.hermes"}`
	r := &leftoverRunner{base}
	a.Runner = r
	os.RemoveAll(filepath.Join(a.Directory, ".hermes"))
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	if code, _, diag := invoke(t, a, "remove"); code == 0 || !strings.Contains(diag, "does not belong") || len(r.mutations) != 0 {
		t.Fatalf("foreign container touched: %d %s %v", code, diag, r.mutations)
	}
}

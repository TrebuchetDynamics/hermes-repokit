package native

import (
	"context"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func identityForTest(t *testing.T, p string) string {
	t.Helper()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	st := info.Sys().(*syscall.Stat_t)
	return fmt.Sprintf("%d:%d", st.Dev, st.Ino)
}
func TestNativeBootstrapPreservesSharedBoard(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, ".hermes")
	os.Mkdir(state, 0700)
	os.WriteFile(filepath.Join(state, "config.yaml"), []byte("native config"), 0600)
	os.WriteFile(filepath.Join(state, "kanban.db"), []byte("existing board"), 0600)
	lock := filepath.Join(root, ".hermes-repokit.lock")
	os.WriteFile(lock, nil, 0600)
	before, _ := os.Stat(filepath.Join(state, "kanban.db"))
	cmd := exec.Command("flock", "-n", lock, "sh", "-s", "--", root, state, identityForTest(t, root), identityForTest(t, state), identityForTest(t, lock), "false")
	cmd.Stdin = strings.NewReader(bootstrapScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	after, _ := os.Stat(filepath.Join(state, "kanban.db"))
	if !os.SameFile(before, after) {
		t.Fatal("shared board replaced")
	}
	if _, err := os.Stat(filepath.Join(state, "profiles")); !os.IsNotExist(err) {
		t.Fatal("bootstrap created profiles before setup")
	}
}

func TestTeamProvisioningFilesystem(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python required for installer-script development tests; end users use Hermes Python")
	}
	cmd := exec.Command(python, "team_test.py")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

type bootstrapInput struct {
	args   []string
	data   string
	result process.Result
}

func (r *bootstrapInput) RunInput(_ context.Context, input io.Reader, program string, args ...string) process.Result {
	r.args = append([]string{program}, args...)
	data, _ := io.ReadAll(input)
	r.data = string(data)
	return r.result
}
func TestInitializeUsesContainerLockAndSanitizesFailure(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	os.Mkdir(filepath.Join(root, ".hermes"), 0700)
	os.WriteFile(filepath.Join(root, ".hermes-repokit.lock"), nil, 0600)
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	r := &bootstrapInput{result: process.Result{Err: fmt.Errorf("secret native error"), Output: "secret native output"}}
	err = Initialize(context.Background(), id, "local", true, r)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("failure diagnostic: %v", err)
	}
	args := strings.Join(r.args, " ")
	if !strings.Contains(args, "--context local compose --env-file /dev/null -f ") || !strings.Contains(args, "exec -T --user hermes --env HOME=/opt/data --workdir /workspace hermes /usr/bin/flock -n /workspace/.hermes-repokit.lock /bin/sh -s -- /workspace /opt/data") || r.data == "" {
		t.Fatalf("unsafe invocation: %v", r.args)
	}
	r.result = process.Result{}
	if err := Initialize(context.Background(), id, "local", false, r); err != nil {
		t.Fatal(err)
	}
}

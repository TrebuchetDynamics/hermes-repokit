package launcher

import (
	"bytes"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func fixture(t *testing.T, body string) (string, string, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "repo ' $; thing")
	os.Mkdir(p, 0700)
	id, e := target.Resolve(p)
	if e != nil {
		t.Fatal(e)
	}
	data, e := Render(id, "default")
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(filepath.Dir(id.Launcher), 0700)
	os.WriteFile(id.Launcher, data, 0700)
	bin := t.TempDir()
	record := filepath.Join(t.TempDir(), "argv")
	os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n"+body), 0700)
	return id.Launcher, bin, record
}
func TestLauncherPreservesArgumentsStreamsExitAndUnrelatedCWD(t *testing.T) {
	file, bin, record := fixture(t, `printf '%s\000' "$@" > "$RECORD"
[ -z "${COMPOSE_FILE+x}${COMPOSE_PROJECT_NAME+x}${COMPOSE_PROFILES+x}${COMPOSE_ENV_FILES+x}" ] || exit 99
IFS= read -r line
printf 'out:%s' "$line"
printf 'diagnostic' >&2
exit 37
`)
	args := []string{"unknown", "a b", "'\"", "$HOME", "; touch nope", "line1\nline2", ""}
	cmd := exec.Command(file, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "PATH="+bin, "RECORD="+record, "COMPOSE_FILE=bad", "COMPOSE_PROJECT_NAME=bad", "COMPOSE_PROFILES=bad", "COMPOSE_ENV_FILES=bad")
	cmd.Stdin = strings.NewReader("input\n")
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 37 {
		t.Fatalf("exit: %v %s", err, errout.String())
	}
	if out.String() != "out:input" || errout.String() != "diagnostic" {
		t.Fatal("streams changed")
	}
	raw, _ := os.ReadFile(record)
	got := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	compose := filepath.Join(filepath.Dir(filepath.Dir(file)), "compose.yaml")
	want := append([]string{"--context", "default", "compose", "--env-file", "/dev/null", "-f", compose, "exec", "-T", "--workdir", "/workspace", "hermes", "hermes"}, args...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv\ngot %#v\nwant %#v", got, want)
	}
}
func TestBareLauncherSelectsPrimaryDefaultProfile(t *testing.T) {
	file, bin, record := fixture(t, `printf '%s\000' "$@" > "$RECORD"`)
	cmd := exec.Command(file)
	cmd.Env = append(os.Environ(), "PATH="+bin, "RECORD="+record)
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(record)
	if !bytes.HasSuffix(raw, []byte("hermes\x00hermes\x00-p\x00default\x00")) {
		t.Fatalf("native argv: %q", raw)
	}
}
func TestLauncherExecPreservesSignalExit(t *testing.T) {
	file, bin, _ := fixture(t, `kill -INT "$$"`)
	cmd := exec.Command(file)
	cmd.Env = append(os.Environ(), "PATH="+bin)
	e := cmd.Run()
	x, ok := e.(*exec.ExitError)
	if !ok {
		t.Fatalf("signal missing: %v", e)
	}
	if s := x.Sys().(syscall.WaitStatus); !s.Signaled() || s.Signal() != syscall.SIGINT {
		t.Fatalf("signal changed: %v", s)
	}
}

func TestContextInspectionRejectsOwnerEditedOrMissingLauncher(t *testing.T) {
	file, _, _ := fixture(t, "exit 0\n")
	id, _ := target.Resolve(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	ctx, e := Context(id)
	if e != nil || ctx != "default" {
		t.Fatalf("%q %v", ctx, e)
	}
	data, _ := os.ReadFile(file)
	os.WriteFile(file, append(data, []byte("# owner edit\n")...), 0700)
	if _, e = Context(id); e == nil {
		t.Fatal("claimed edited launcher context was verified")
	}
}

func TestRuntimeContextRequiresCurrentLauncherAfterNameChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hermes-repokit")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	id, err := target.Resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(id.Launcher), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := Render(id, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(id.Launcher), "hermes-hermes-repokit"), data, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Context(id); err == nil {
		t.Fatal("runtime reported current launcher ready using only the old command")
	}
}

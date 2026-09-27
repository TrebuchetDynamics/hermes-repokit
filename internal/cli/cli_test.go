package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"os"
	"strings"
	"testing"
)

func TestHelpListsOnlyInstallerCommands(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"plan", "--help"}, {"install", "-h"}, {"setup", "--help"}, {"verify", "-h"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostics.String())
			}
			for _, command := range []string{"plan", "install", "setup", "verify"} {
				if !strings.Contains(out.String(), command) {
					t.Fatalf("help omitted %q: %q", command, out.String())
				}
			}
			for _, extra := range []string{"chat", "status", "start", "stop", "uninstall"} {
				if strings.Contains(out.String(), extra) {
					t.Fatalf("help exposed %q: %q", extra, out.String())
				}
			}
		})
	}
}

func TestCommandsReturnsFourIndependentNames(t *testing.T) {
	first := Commands()
	if got, want := strings.Join(first, ","), "plan,install,setup,verify"; got != want {
		t.Fatalf("commands = %q, want %q", got, want)
	}
	first[0] = "changed"
	if got := Commands()[0]; got != "plan" {
		t.Fatalf("Commands exposed mutable internal list: %q", got)
	}
}

func TestInvalidInputIsUsageErrorWithoutEchoingInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no arguments", nil},
		{"unknown command", []string{"secret-command"}},
		{"unknown top-level flag", []string{"--secret-flag"}},
		{"unknown command flag", []string{"plan", "--secret-flag"}},
		{"unexpected positional", []string{"install", "secret-value"}},
		{"extra after help", []string{"verify", "--help", "secret-value"}},
		{"help with value", []string{"verify", "--help=secret-value"}},
		{"bare separator", []string{"setup", "--"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			if code := Run(tt.args, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostics.String())
			}
			for _, secret := range []string{"secret-command", "secret-flag", "secret-value"} {
				if strings.Contains(diagnostics.String(), secret) {
					t.Fatalf("diagnostic echoed input: %q", diagnostics.String())
				}
			}
		})
	}
}

type fakeRunner struct{ calls []string }

func (f *fakeRunner) Run(_ context.Context, p string, args ...string) process.Result {
	c := p + " " + strings.Join(args, " ")
	f.calls = append(f.calls, c)
	switch {
	case strings.Contains(c, "rev-parse"):
		return process.Result{Output: fRoot}
	case strings.Contains(c, "context show"):
		return process.Result{Output: "default\n"}
	case strings.Contains(c, "context inspect"):
		return process.Result{Output: "unix:///var/run/docker.sock\n"}
	case strings.Contains(c, "container ls"):
		return process.Result{}
	case strings.Contains(c, "container inspect"):
		return process.Result{Err: errors.New("absent")}
	case strings.Contains(c, "ls-files"):
		return process.Result{}
	default:
		return process.Result{Err: errors.New("unexpected command")}
	}
}

var fRoot string

func TestPlanAndVerifyInspectWithoutWritingOrNativeCalls(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	fRoot = root
	for _, command := range []string{"plan", "verify"} {
		r := &fakeRunner{}
		var out, errout bytes.Buffer
		app := App{Directory: root, Path: "", Runner: r, Stdin: strings.NewReader("")}
		code := app.Run([]string{command, "--engineering"}, &out, &errout)
		if command == "plan" && code != 0 {
			t.Fatalf("%d %s", code, errout.String())
		}
		if !json.Valid(out.Bytes()) {
			t.Fatalf("not structured output: %s", out.String())
		}
		entries, _ := os.ReadDir(root)
		if len(entries) != 0 {
			t.Fatal("read-only command wrote files")
		}
		for _, call := range r.calls {
			if strings.Contains(call, " exec ") || strings.Contains(call, " pull ") || strings.Contains(call, " up ") {
				t.Fatalf("mutating probe: %s", call)
			}
		}
	}
}
func TestInstallRefusesUnqualifiedPresetBeforeAnyWrites(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	fRoot = root
	var out, errout bytes.Buffer
	app := App{Directory: root, Runner: &fakeRunner{}}
	if code := app.Run([]string{"install", "--engineering"}, &out, &errout); code == 0 || !strings.Contains(errout.String(), "qualification") {
		t.Fatalf("%d %s", code, errout.String())
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("unqualified installation wrote files")
	}
}

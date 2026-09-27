package cli

import (
	"bytes"
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

func TestRecognizedCommandsAreExplicitlyNotImplemented(t *testing.T) {
	for _, command := range []string{"plan", "install", "setup", "verify"} {
		t.Run(command, func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			if code := Run([]string{command}, &out, &diagnostics); code != 1 || out.Len() != 0 || !strings.Contains(diagnostics.String(), "not implemented") {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostics.String())
			}
		})
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

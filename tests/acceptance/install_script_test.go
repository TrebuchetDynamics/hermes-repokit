package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func installScript(t *testing.T, home, path string, extraEnv ...string) (string, error) {
	t.Helper()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate install script test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "../.."))
	cmd := exec.Command("sh", filepath.Join(root, "install.sh"))
	cmd.Dir = home
	cmd.Env = append(append(os.Environ(), "HOME="+home, "PATH="+path), extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInstallScriptPublishesWorkingCLIAndRerunsSafely(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	path := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	out, err := installScript(t, home, path)
	if err != nil {
		t.Fatalf("first install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Building") || !strings.Contains(out, "repokit plan") || !strings.Contains(out, "repokit install") || !strings.Contains(out, "Compose") {
		t.Fatalf("installer omitted build progress or deployment handoff: %s", out)
	}
	command := filepath.Join(bin, "repokit")
	info, err := os.Lstat(command)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0100 == 0 {
		t.Fatalf("installed command is not an executable regular file: %v %v", info, err)
	}
	help, err := exec.Command(command, "--help").CombinedOutput()
	if err != nil || !strings.Contains(string(help), "plan|install|setup|verify") {
		t.Fatalf("installed command cannot run: %v\n%s", err, help)
	}
	before, err := os.ReadFile(command)
	if err != nil {
		t.Fatal(err)
	}
	out, err = installScript(t, home, path)
	if err != nil {
		t.Fatalf("idempotent install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "repokit plan") {
		t.Fatalf("rerun omitted next step: %s", out)
	}
	after, err := os.ReadFile(command)
	if err != nil || string(before) != string(after) {
		t.Fatalf("idempotent install changed the command: %v", err)
	}
}

func TestInstallScriptPreservesConflictingCommand(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(bin, "repokit")
	if err := os.WriteFile(command, []byte("owner command\n"), 0755); err != nil {
		t.Fatal(err)
	}
	out, err := installScript(t, home, os.Getenv("PATH"))
	if err == nil {
		t.Fatalf("conflicting command was accepted: %s", out)
	}
	if !strings.Contains(out, "relocate") {
		t.Fatalf("collision has no recovery action: %s", out)
	}
	got, err := os.ReadFile(command)
	if err != nil || string(got) != "owner command\n" {
		t.Fatalf("conflicting command changed: %v %q", err, got)
	}
}

func TestInstallScriptRejectsRedirectedBinDirectory(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".local"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, ".local", "bin")); err != nil {
		t.Fatal(err)
	}
	out, err := installScript(t, home, os.Getenv("PATH"))
	if err == nil {
		t.Fatalf("redirected bin was accepted: %s", out)
	}
	if _, err := os.Lstat(filepath.Join(outside, "repokit")); !os.IsNotExist(err) {
		t.Fatalf("redirected bin was modified: %v", err)
	}
}

func TestInstallScriptReportsMissingPathEntry(t *testing.T) {
	home := t.TempDir()
	out, err := installScript(t, home, os.Getenv("PATH"))
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "PATH") || !strings.Contains(out, filepath.Join(home, ".local", "bin", "repokit")) {
		t.Fatalf("missing PATH warning or absolute command: %s", out)
	}
}

func TestInstallScriptHandlesRelativeTempDirectory(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "tmp"), 0700); err != nil {
		t.Fatal(err)
	}
	out, err := installScript(t, home, os.Getenv("PATH"), "TMPDIR=tmp")
	if err != nil {
		t.Fatalf("relative TMPDIR broke install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "repokit")); err != nil {
		t.Fatalf("command missing after relative TMPDIR install: %v", err)
	}
}

func TestInstallScriptWarnsWhenEarlierPathCommandShadowsIt(t *testing.T) {
	home := t.TempDir()
	foreign := t.TempDir()
	if err := os.WriteFile(filepath.Join(foreign, "repokit"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	path := foreign + string(os.PathListSeparator) + bin + string(os.PathListSeparator) + os.Getenv("PATH")
	out, err := installScript(t, home, path)
	if err != nil {
		t.Fatalf("install with shadowing command: %v\n%s", err, out)
	}
	if !strings.Contains(out, "shadow") || !strings.Contains(out, filepath.Join(bin, "repokit")) {
		t.Fatalf("shadowing command was not reported: %s", out)
	}
}

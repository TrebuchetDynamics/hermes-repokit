package acceptance

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// quietGo switches Go telemetry off in a throwaway HOME. Otherwise the go
// command install.sh runs writes telemetry there from a background process
// that can outlive the script and race the test's directory cleanup.
func quietGo(t *testing.T, home string) {
	t.Helper()
	cmd := exec.Command("go", "telemetry", "off")
	cmd.Env = append(os.Environ(), "HOME="+home)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go telemetry off: %v\n%s", err, out)
	}
}

func installScript(t *testing.T, home, path string, extraEnv ...string) (string, error) {
	t.Helper()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	quietGo(t, home)
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

// pipedInstallScript simulates `curl ... | sh`: the shell reads the script from
// stdin, so $0 is not a readable file and the installer must fetch the source.
func pipedInstallScript(t *testing.T, home, path, sourceURL string) (string, error) {
	t.Helper()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	quietGo(t, home)
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate install script test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "../.."))
	script, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh")
	cmd.Dir = home
	cmd.Stdin = bytes.NewReader(script)
	cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+path, "REPOKIT_SOURCE_URL="+sourceURL)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func repoSourceArchive(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate install script test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "../.."))
	archive := filepath.Join(t.TempDir(), "source.tar.gz")
	cmd := exec.Command("tar", "-czf", archive, "-C", root, "go.mod", "cmd", "internal", "packaging")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tar: %v\n%s", err, out)
	}
	return "file://" + archive
}

func TestInstallScriptPublishesWorkingCLIAndRerunsSafely(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	path := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	out, err := installScript(t, home, path)
	if err != nil {
		t.Fatalf("first install: %v\n%s", err, out)
	}
	// The handoff is the one-command path (install continues into setup),
	// with no Compose choreography.
	if !strings.Contains(out, "Building") || !strings.Contains(out, "repokit install") || !strings.Contains(out, "then setup") || strings.Contains(out, "Compose") {
		t.Fatalf("installer omitted build progress or deployment handoff: %s", out)
	}
	command := filepath.Join(bin, "repokit")
	info, err := os.Lstat(command)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0100 == 0 {
		t.Fatalf("installed command is not an executable regular file: %v %v", info, err)
	}
	program := filepath.Join(bin, "hermes-repokit")
	pinfo, err := os.Lstat(program)
	if err != nil || !pinfo.Mode().IsRegular() || pinfo.Mode().Perm()&0100 == 0 {
		t.Fatalf("canonical command is not an executable regular file: %v %v", pinfo, err)
	}
	if info, err := os.Stat(program); err != nil || !os.SameFile(info, mustStat(t, command)) {
		t.Fatalf("both names must publish one binary: %v", err)
	}
	if version, err := exec.Command(command, "version").CombinedOutput(); err != nil || string(version) != "repokit checkout\n" {
		t.Fatalf("installed command does not report its stamped version: %v %q", err, version)
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
	if !strings.Contains(out, "repokit install") {
		t.Fatalf("rerun omitted next step: %s", out)
	}
	after, err := os.ReadFile(command)
	if err != nil || string(before) != string(after) {
		t.Fatalf("idempotent install changed the command: %v", err)
	}
}

func TestInstallScriptPreservesGeneratedLauncher(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	launcherDir := filepath.Join(home, ".hermes", "bin")
	if err := os.MkdirAll(launcherDir, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(launcherDir, "hermes-repokit")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, "hermes-repokit")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	out, err := installScript(t, home, os.Getenv("PATH"))
	if err != nil {
		t.Fatalf("generated launcher should be preserved without failing the install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "generated repository launcher") || !strings.Contains(out, "mv -- ") {
		t.Fatalf("missing launcher relocation guidance: %s", out)
	}
	if got, e := os.Readlink(link); e != nil || got != target {
		t.Fatalf("launcher symlink changed: %v %q", e, got)
	}
	if _, e := os.Lstat(filepath.Join(bin, "repokit")); e != nil {
		t.Fatalf("alias was not installed alongside the preserved name: %v", e)
	}
}

func TestInstallScriptUpdatesOwnedBootstrap(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	path := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	if out, err := installScript(t, home, path); err != nil {
		t.Fatalf("first install: %v\n%s", err, out)
	}
	alias := filepath.Join(bin, "repokit")
	file, err := os.OpenFile(alias, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("stale")); err != nil {
		t.Fatal(err)
	}
	file.Close()
	out, err := installScript(t, home, path)
	if err != nil {
		t.Fatalf("owned bootstrap was not updated: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Updated RepoKit bootstrap") {
		t.Fatalf("update was not reported: %s", out)
	}
	got, err := os.ReadFile(alias)
	if err != nil {
		t.Fatal(err)
	}
	program, err := os.ReadFile(filepath.Join(bin, "hermes-repokit"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(program) {
		t.Fatalf("owned alias was not replaced with the current build")
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
	if !strings.Contains(out, "PATH") || !strings.Contains(out, filepath.Join(home, ".local", "bin", "hermes-repokit")) {
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
	if err := os.WriteFile(filepath.Join(foreign, "hermes-repokit"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	path := foreign + string(os.PathListSeparator) + bin + string(os.PathListSeparator) + os.Getenv("PATH")
	out, err := installScript(t, home, path)
	if err != nil {
		t.Fatalf("install with shadowing command: %v\n%s", err, out)
	}
	if !strings.Contains(out, "shadow") || !strings.Contains(out, filepath.Join(bin, "hermes-repokit")) {
		t.Fatalf("shadowing command was not reported: %s", out)
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestInstallScriptBootstrapsFromRemoteSource(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	path := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	out, err := pipedInstallScript(t, home, path, repoSourceArchive(t))
	if err != nil {
		t.Fatalf("remote install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Downloading RepoKit source") || !strings.Contains(out, "Installed RepoKit bootstrap") {
		t.Fatalf("remote install output: %s", out)
	}
	// A custom source URL is reported as what was downloaded, not as "main".
	if !strings.Contains(out, "Downloaded and unpacked file://") || strings.Contains(out, "unpacked main") {
		t.Fatalf("custom source mislabeled: %s", out)
	}
	command := filepath.Join(bin, "hermes-repokit")
	info, err := os.Lstat(command)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0100 == 0 {
		t.Fatalf("remote command is not an executable regular file: %v %v", info, err)
	}
	if _, err := os.Lstat(filepath.Join(bin, "repokit")); err != nil {
		t.Fatalf("remote alias missing: %v", err)
	}
}

// REPOKIT_REF may name a branch, a release tag or a commit. GitHub serves all
// three at archive/<ref>.tar.gz; archive/refs/heads/<ref> serves branches only.
func TestInstallScriptFetchesTagsAndCommitsByRef(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate install script test")
	}
	script, err := os.ReadFile(filepath.Join(filepath.Dir(here), "../../install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"main", "v0.2.0", "5a6daf296677af55c6a462e795e29b614cd50309"} {
		home := t.TempDir()
		if err := os.Chmod(home, 0700); err != nil {
			t.Fatal(err)
		}
		quietGo(t, home)
		fake := t.TempDir()
		record := filepath.Join(fake, "url")
		curl := "#!/bin/sh\nfor a; do last=$a; done\nprintf '%s' \"$last\" > " + record + "\nexit 1\n"
		if err := os.WriteFile(filepath.Join(fake, "curl"), []byte(curl), 0700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("sh")
		cmd.Dir = home
		cmd.Stdin = bytes.NewReader(script)
		cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+fake+string(os.PathListSeparator)+os.Getenv("PATH"), "REPOKIT_REF="+ref)
		cmd.CombinedOutput()
		got, _ := os.ReadFile(record)
		want := "https://github.com/TrebuchetDynamics/hermes-repokit/archive/" + ref + ".tar.gz"
		if string(got) != want {
			t.Fatalf("REPOKIT_REF=%s fetched %q, want %q", ref, got, want)
		}
	}
}

// A failed download or unpack must not leave the source directory in TMPDIR.
func TestInstallScriptCleansUpAfterFailedDownload(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate install script test")
	}
	script, err := os.ReadFile(filepath.Join(filepath.Dir(here), "../../install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	home, temp, fake := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	quietGo(t, home)
	if err := os.WriteFile(filepath.Join(fake, "curl"), []byte("#!/bin/sh\nexit 22\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh")
	cmd.Dir = home
	cmd.Stdin = bytes.NewReader(script)
	cmd.Env = append(os.Environ(), "HOME="+home, "TMPDIR="+temp, "PATH="+fake+string(os.PathListSeparator)+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "cannot download") {
		t.Fatalf("failed download not reported: %v\n%s", err, out)
	}
	if left, _ := filepath.Glob(filepath.Join(temp, "repokit-*")); len(left) != 0 {
		t.Fatalf("temporary source left behind: %v", left)
	}
}

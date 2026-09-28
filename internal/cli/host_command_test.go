package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallExposesHostCommandAndPreservesItOnRerun(t *testing.T) {
	a, r := foundationApp(t)
	bin := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	a.Path = bin
	link := filepath.Join(bin, "hermes-test-project")
	for i := 0; i < 2; i++ {
		code, out, diag := invoke(t, a, "install")
		if code != 0 || !strings.Contains(out, link) {
			t.Fatalf("install %d: %d %s %s", i, code, out, diag)
		}
		if got, err := os.Readlink(link); err != nil || got != r.id.Launcher {
			t.Fatalf("host link = %q, %v; want %q", got, err, r.id.Launcher)
		}
	}
	// Exercise PATH lookup from outside the repository, across the real launcher.
	// Docker alone is replaced to capture the process boundary without a runtime.
	dockerBin := t.TempDir()
	if err := os.WriteFile(filepath.Join(dockerBin, "docker"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"kanban", "list"}, {"profile", "list"}, {"setup"}} {
		cmd := exec.Command("/bin/sh", append([]string{"-c", `exec hermes-test-project "$@"`, "host-test"}, args...)...)
		cmd.Dir = t.TempDir()
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+dockerBin)
		out, err := cmd.CombinedOutput()
		wantArgs := strings.Join(args, "\n")
		if len(args) == 0 {
			wantArgs = "-p\ndefault"
		}
		want := "--context\nlocal-test\ncompose\n--env-file\n/dev/null\n-f\n" + r.id.Compose + "\nexec\n-T\n--workdir\n/workspace\nhermes\nhermes\n" + wantArgs + "\n"
		if err != nil || string(out) != want {
			t.Fatalf("command %v: %v\ngot %q\nwant %q", args, err, out, want)
		}
	}
}

func TestInstallReportsHostCommandMissingFromPATH(t *testing.T) {
	a, _ := foundationApp(t)
	code, out, diag := invoke(t, a, "install")
	link := filepath.Join(os.Getenv("HOME"), ".local/bin/hermes-test-project")
	if code != 0 || !strings.Contains(diag, "not on PATH") || !strings.Contains(diag, link) {
		t.Fatalf("install: %d %s %s", code, out, diag)
	}
	if _, err := os.Readlink(link); err != nil {
		t.Fatal(err)
	}
}

func TestInstallPreservesConflictingHostEntry(t *testing.T) {
	for _, kind := range []string{"file", "directory", "dangling-link"} {
		t.Run(kind, func(t *testing.T) {
			a, _ := foundationApp(t)
			bin := filepath.Join(os.Getenv("HOME"), ".local/bin")
			if err := os.MkdirAll(bin, 0700); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(bin, "hermes-test-project")
			var err error
			switch kind {
			case "file":
				err = os.WriteFile(link, []byte("owner command"), 0700)
			case "directory":
				err = os.Mkdir(link, 0700)
			case "dangling-link":
				err = os.Symlink("/missing-owner-command", link)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, _ := os.Lstat(link)
			code, out, diag := invoke(t, a, "install")
			after, err := os.Lstat(link)
			if code != 0 || !strings.Contains(diag, "conflicts with this launcher") || err != nil || !os.SameFile(before, after) {
				t.Fatalf("collision not preserved/reported: %d %s %s %v", code, out, diag, err)
			}
		})
	}
}

func TestRejectedInstallDoesNotCreateHostCommand(t *testing.T) {
	a, r := foundationApp(t)
	r.names = r.id.Container
	if code, _, _ := invoke(t, a, "install"); code == 0 {
		t.Fatal("accepted foreign container")
	}
	if _, err := os.Lstat(filepath.Join(os.Getenv("HOME"), ".local")); !os.IsNotExist(err) {
		t.Fatalf("refused install modified host: %v", err)
	}
}

func TestInstallReportsUnusableHostDirectoryWithoutChangingIt(t *testing.T) {
	for _, kind := range []string{"writable", "symlink", "file"} {
		t.Run(kind, func(t *testing.T) {
			a, r := foundationApp(t)
			local := filepath.Join(os.Getenv("HOME"), ".local")
			destination := t.TempDir()
			var err error
			switch kind {
			case "writable":
				err = os.Mkdir(local, 0700)
				if err == nil {
					err = os.Chmod(local, 0777)
				}
			case "symlink":
				err = os.Symlink(destination, local)
			case "file":
				err = os.WriteFile(local, []byte("owner file"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, _ := os.Lstat(local)
			code, out, diag := invoke(t, a, "install")
			after, err := os.Lstat(local)
			if code != 0 || !strings.Contains(diag, "host command unavailable") || !strings.Contains(diag, r.id.Launcher) || err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Fatalf("unsafe host directory: %d %s %s %v", code, out, diag, err)
			}
			entries, err := os.ReadDir(destination)
			if err != nil || len(entries) != 0 {
				t.Fatalf("followed redirected home directory: %v %v", entries, err)
			}
		})
	}
}

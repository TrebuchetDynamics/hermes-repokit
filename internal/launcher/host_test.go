package launcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestExposeReusesMatchingRelativeLinkAndRejectsChangedLauncher(t *testing.T) {
	file, _, _ := fixture(t, "exit 0\n")
	id, err := target.Resolve(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err = os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local/bin")
	if err = os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, id.Container)
	rel, err := filepath.Rel(bin, file)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(rel, link); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Lstat(link)
	if got, err := Expose(id, home); err != nil || got != link {
		t.Fatalf("reuse: %q %v", got, err)
	}
	after, _ := os.Lstat(link)
	if !os.SameFile(before, after) {
		t.Fatal("replaced matching link")
	}
	if err = os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, []byte("#!/bin/sh\n# owner replacement\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = Expose(id, home); err == nil {
		t.Fatal("exposed unrecognized launcher")
	}
	if _, err = os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("published despite unrecognized launcher: %v", err)
	}
}

func TestHostCommandPATHRequiresStableDirectory(t *testing.T) {
	bin := t.TempDir()
	alias := filepath.Join(t.TempDir(), "bin-alias")
	if err := os.Symlink(bin, alias); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		path string
		want bool
	}{
		{bin, true}, {alias, true}, {bin + "-different", false}, {".local/bin", false}, {"", false},
	} {
		if got := OnPath(filepath.Join(bin, "hermes-test"), tt.path); got != tt.want {
			t.Fatalf("PATH %q: %v", tt.path, got)
		}
	}
}

func TestExposeDoesNotReuseLinkWithUnresolvableTraversal(t *testing.T) {
	file, _, _ := fixture(t, "exit 0\n")
	id, err := target.Resolve(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err = os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local/bin")
	if err = os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, id.Container)
	// Deliberately do not use filepath.Join: the missing component must survive.
	dangling := filepath.Dir(file) + "/missing/../" + id.Container
	if err = os.Symlink(dangling, link); err != nil {
		t.Fatal(err)
	}
	if _, err = Expose(id, home); err == nil {
		t.Fatal("reported dangling link as usable host command")
	}
	if got, err := os.Readlink(link); err != nil || got != dangling {
		t.Fatalf("changed dangling link: %q %v", got, err)
	}
}

func TestTrustedDirectoryAllowsGroupWriteOnlyForPrivateGroup(t *testing.T) {
	dir := t.TempDir()
	saved := target.PrivateGroup
	defer func() { target.PrivateGroup = saved }()
	for _, tc := range []struct {
		mode    os.FileMode
		private bool
		want    bool
	}{{0755, false, true}, {0775, true, true}, {0775, false, false}, {0777, true, false}} {
		target.PrivateGroup = func(uint32) bool { return tc.private }
		if err := os.Chmod(dir, tc.mode); err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got := trustedDirectory(info); got != tc.want {
			t.Fatalf("%o private=%v: trusted=%v want %v", tc.mode, tc.private, got, tc.want)
		}
	}
}

package install

import (
	"errors"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) (target.Identity, map[string]Artifact) {
	t.Helper()
	p := t.TempDir()
	os.Chmod(p, 0700)
	id, e := target.Resolve(p)
	if e != nil {
		t.Fatal(e)
	}
	return id, map[string]Artifact{"compose.yaml": {[]byte("compose"), 0600}, "config.yaml": {[]byte("dispatch: false"), 0600}, "bin/" + id.Container: {[]byte("#!/bin/sh\nexit 0\n"), 0700}}
}
func TestFreshPublicationAndRerunPreserveNativeFiles(t *testing.T) {
	id, files := fixture(t)
	if _, e := Publish(id, files, nil); e != nil {
		t.Fatal(e)
	}
	state := filepath.Join(id.Root, ".hermes")
	os.WriteFile(filepath.Join(state, "config.yaml"), []byte("owner edited"), 0600)
	os.WriteFile(filepath.Join(state, "unknown"), []byte("keep"), 0600)
	for _, receipt := range []string{"", "not json"} {
		if receipt != "" {
			os.WriteFile(filepath.Join(state, "repokit-install.json"), []byte(receipt), 0600)
		}
		created, e := Publish(id, files, nil)
		if e != nil || created {
			t.Fatalf("rerun: %t %v", created, e)
		}
	}
	for name, want := range map[string]string{"config.yaml": "owner edited", "unknown": "keep"} {
		got, _ := os.ReadFile(filepath.Join(state, name))
		if string(got) != want {
			t.Fatal("overwrote native state")
		}
	}
}
func TestFailedPreparationLeavesNoPublishedState(t *testing.T) {
	id, f := fixture(t)
	_, e := Publish(id, f, func() error { return errors.New("pull failed") })
	if e == nil {
		t.Fatal("ignored failure")
	}
	if _, e = os.Lstat(filepath.Join(id.Root, ".hermes")); !os.IsNotExist(e) {
		t.Fatal("published on failure")
	}
	if _, e = Publish(id, f, nil); e != nil {
		t.Fatal(e)
	}
}
func TestAmbiguousPartialAndOwnerComposeEditsRefuse(t *testing.T) {
	id, f := fixture(t)
	state := filepath.Join(id.Root, ".hermes")
	os.Mkdir(state, 0700)
	os.WriteFile(filepath.Join(state, "config.yaml"), []byte("mine"), 0600)
	if _, e := Publish(id, f, nil); e == nil {
		t.Fatal("adopted partial state")
	}
	os.WriteFile(filepath.Join(state, "compose.yaml"), []byte("owner compose"), 0600)
	if _, e := Publish(id, f, nil); e == nil {
		t.Fatal("adopted edited Compose")
	}
}
func TestArtifactTraversalRefuses(t *testing.T) {
	id, f := fixture(t)
	f["../outside"] = Artifact{[]byte("bad"), 0600}
	if _, e := Publish(id, f, nil); e == nil {
		t.Fatal("accepted traversal")
	}
	if _, e := os.Stat(filepath.Join(id.Root, "outside")); e == nil {
		t.Fatal("escaped state")
	}
}

func TestRerunRejectsMissingConfigAndNonExecutableLauncher(t *testing.T) {
	for _, broken := range []string{"config", "launcher"} {
		t.Run(broken, func(t *testing.T) {
			id, f := fixture(t)
			if _, e := Publish(id, f, nil); e != nil {
				t.Fatal(e)
			}
			if broken == "config" {
				os.Remove(filepath.Join(id.Root, ".hermes/config.yaml"))
			} else {
				os.Chmod(id.Launcher, 0600)
			}
			if _, e := Publish(id, f, nil); e == nil {
				t.Fatal("reported incomplete native state as safe rerun")
			}
		})
	}
}

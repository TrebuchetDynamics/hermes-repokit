package install

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func TestPreparationPathReplacementCannotRedirectStageWrites(t *testing.T) {
	for _, ancestor := range []bool{false, true} {
		name := "target"
		if ancestor {
			name = "ancestor"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			parent := base
			unrelated := filepath.Join(base, "unrelated")
			if ancestor {
				parent = filepath.Join(base, "parent")
				if err := os.Mkdir(parent, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(unrelated, 0700); err != nil {
					t.Fatal(err)
				}
				unrelated = filepath.Join(unrelated, "repo")
			}
			repo := filepath.Join(parent, "repo")
			for _, dir := range []string{repo, unrelated} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			id, err := target.Resolve(repo)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string]Artifact{
				"compose.yaml":        {[]byte("compose"), 0600},
				"config.yaml":         {[]byte("dispatch: false"), 0600},
				"bin/" + id.Container: {[]byte("#!/bin/sh\nexit 0\n"), 0700},
			}
			watch, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
			if err != nil {
				t.Fatal(err)
			}
			defer syscall.Close(watch)
			if _, err := syscall.InotifyAddWatch(watch, unrelated, syscall.IN_CREATE); err != nil {
				t.Fatal(err)
			}
			moved := filepath.Join(base, "original")
			created, err := Publish(id, files, func() error {
				if ancestor {
					if err := os.Rename(parent, moved); err != nil {
						return err
					}
					return os.Symlink(filepath.Dir(unrelated), parent)
				}
				if err := os.Rename(repo, moved); err != nil {
					return err
				}
				return os.Symlink(unrelated, repo)
			})
			if err == nil || created {
				t.Fatalf("accepted changed target: created=%v, err=%v", created, err)
			}
			events := make([]byte, 8192)
			n, readErr := syscall.Read(watch, events)
			if n > 0 {
				t.Fatalf("created files in unrelated directory before refusal (%d inotify bytes)", n)
			}
			if readErr != syscall.EAGAIN {
				t.Fatalf("unexpected inotify result: n=%d err=%v", n, readErr)
			}
			for _, dir := range []string{unrelated, moved} {
				if ancestor && dir == moved {
					dir = filepath.Join(dir, "repo")
				}
				entries, err := os.ReadDir(dir)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if entry.Name() == ".hermes" || len(entry.Name()) >= len(".hermes-stage-") && entry.Name()[:len(".hermes-stage-")] == ".hermes-stage-" {
						t.Fatalf("left installation state in %s: %s", dir, entry.Name())
					}
				}
			}
		})
	}
}

// Package acceptance contains offline contracts and explicitly gated runtime proofs.
package acceptance

import (
	"context"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedLauncherSurvivesInstallerArtifactsRemoval(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unrelated-repository")
	os.Mkdir(root, 0700)
	id, e := target.Resolve(root)
	if e != nil {
		t.Fatal(e)
	}
	composeData, e := compose.Render(id, compose.Options{HermesImage: "example/hermes@sha256:" + strings.Repeat("a", 64), UID: os.Getuid(), GID: os.Getgid()})
	if e != nil {
		t.Fatal(e)
	}
	script, e := launcher.Render(id, "default")
	if e != nil {
		t.Fatal(e)
	}
	artifacts := map[string]install.Artifact{"compose.yaml": {Data: composeData, Mode: 0600}, "config.yaml": {Data: []byte("kanban:\n  dispatch_in_gateway: false\n  auto_decompose: false\n"), Mode: 0600}, "bin/" + id.Container: {Data: script, Mode: 0700}, "repokit-install.json": {Data: []byte("{}"), Mode: 0600}}
	if _, e = install.Publish(id, artifacts, nil); e != nil {
		t.Fatal(e)
	}
	// This is offline launcher independence only, not the v1 removal-first gate.
	installer := filepath.Join(t.TempDir(), "installer-copy")
	os.WriteFile(installer, []byte("disposable installer artifact"), 0700)
	os.Remove(installer)
	os.Remove(filepath.Join(root, ".hermes/repokit-install.json"))
	bin := t.TempDir()
	os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n[ \"$1\" = --context ] || exit 2\nprintf 'native-command'\n"), 0700)
	cmd := exec.CommandContext(context.Background(), id.Launcher, "kanban", "list")
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"PATH=" + bin}
	out, e := cmd.Output()
	if e != nil || string(out) != "native-command" {
		t.Fatalf("standalone launcher: %v %q", e, out)
	}
	git := exec.Command("git", "-C", root, "init", "--quiet")
	if e = git.Run(); e != nil {
		t.Fatal(e)
	}
	git = exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=all", "--", ".hermes")
	out, e = git.Output()
	if e != nil || len(out) != 0 {
		t.Fatalf("private native state exposed to Git: %q %v", out, e)
	}
}

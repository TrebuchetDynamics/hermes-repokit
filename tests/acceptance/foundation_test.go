// Package acceptance contains offline contracts and explicitly gated runtime proofs.
package acceptance

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// Build from a disposable source copy so removal covers both source and binary.
func disposableCLI(t *testing.T) (string, func()) {
	t.Helper()
	original, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source")
	if err = os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"cmd", "internal"} {
		if err = os.CopyFS(filepath.Join(source, dir), os.DirFS(filepath.Join(original, dir))); err != nil {
			t.Fatal(err)
		}
	}
	mod, err := os.ReadFile(filepath.Join(original, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(source, "go.mod"), mod, 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(source, "hermes-repokit")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", binary, "./cmd/hermes-repokit")
	cmd.Dir = source
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	return binary, func() {
		t.Helper()
		if err := os.RemoveAll(source); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublishedLauncherSurvivesInstallerArtifactsRemoval(t *testing.T) {
	installer, removeInstaller := disposableCLI(t)
	root := filepath.Join(t.TempDir(), "unrelated-repository")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	id, err := target.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git: %v %s", err, out)
	}
	bin := t.TempDir()
	// Fake only Docker; install, Git inspection, lock and publication run for real.
	docker := `#!/bin/sh
case "$*" in
 'context show') printf 'default\n';;
 'context inspect default --format {{.Endpoints.docker.Host}}') printf 'unix:///var/run/docker.sock\n';;
 '--context default container ls --all --format {{.Names}}') :;;
 *)
  [ "$1" = --context ] && [ "$2" = default ] && [ "$3" = compose ] || exit 2
  shift 7
  [ "$1" = exec ] && [ "$2" = -T ] && [ "$3" = --workdir ] && [ "$4" = /workspace ] && [ "$5" = hermes ] && [ "$6" = hermes ] || exit 3
  shift 6
  printf '%s\n' "$@"
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(docker), 0700); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"plan", "install", "setup"} {
		cmd := exec.Command(installer, command)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if command == "setup" {
			if err == nil || !strings.Contains(string(out), "-p\ndefault\nsetup") || !strings.Contains(string(out), "noninteractive") {
				t.Fatalf("skipped setup provisioned: %v %s", err, out)
			}
		} else if err != nil {
			t.Fatalf("CLI %s: %v %s", command, err, out)
		}
	}
	receipt := filepath.Join(root, ".hermes/repokit-install.json")
	if err := os.WriteFile(receipt, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	removeInstaller()
	if err := os.Remove(receipt); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(id.Launcher, "unknown-native-command", "space ; $ literal")
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"PATH=" + bin}
	out, err := cmd.Output()
	if err != nil || string(out) != "unknown-native-command\nspace ; $ literal\n" {
		t.Fatalf("standalone: %v %q", err, out)
	}
	git := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=all", "--", ".hermes")
	if out, err := git.Output(); err != nil || len(out) != 0 {
		t.Fatalf("private state exposed to Git: %q %v", out, err)
	}
	if _, err := os.Stat(installer); !os.IsNotExist(err) {
		t.Fatal("installer survived removal")
	}
	data, err := os.ReadFile(id.Compose)
	if err != nil || strings.Contains(string(data), filepath.Dir(installer)) {
		t.Fatal("Compose depends on installer copy")
	}
}

// Package acceptance contains offline contracts and explicitly gated runtime proofs.
package acceptance

import (
	"context"
	"encoding/json"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
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
	for _, dir := range []string{"cmd", "internal", "packaging"} {
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
	// This intentionally source-only copy has no Git history to stamp. Do not
	// let ambient VCS discovery make the removal fixture depend on another repo.
	cmd := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-trimpath", "-o", binary, "./cmd/hermes-repokit")
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

// Keep installer-created host commands disposable while retaining the selected
// Docker context/credentials for opt-in Docker tests.
func installerEnvironment(t *testing.T) []string {
	t.Helper()
	home := t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	env := os.Environ()
	if os.Getenv("DOCKER_CONFIG") == "" {
		original, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		env = append(env, "DOCKER_CONFIG="+filepath.Join(original, ".docker"))
	}
	return append(env, "HOME="+home)
}

func TestPublishedLauncherSurvivesInstallerArtifactsRemoval(t *testing.T) {
	installer, removeInstaller := disposableCLI(t)
	installerEnv := installerEnvironment(t)
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
 '--context default image inspect '*)
  case "$*" in
   *'--format {{json .RootFS.Layers}}'*) printf '%s\n' "$REPOKIT_TEST_LAYERS";;
   *) printf '%s\n' "$REPOKIT_TEST_IMAGE";;
  esac
  ;;
 '--context default container inspect '*)
  [ -n "${REPOKIT_TEST_RUNTIME:-}" ] || exit 1
  printf '%s\n' "$REPOKIT_TEST_RUNTIME"
  ;;
 *)
  [ "$1" = --context ] && [ "$2" = default ] && [ "$3" = compose ] || exit 2
  shift 7
  # install builds and starts the deployment through ordinary Compose.
  if [ "$1" = up ] && [ "$2" = -d ] && [ "$3" = --build ] && [ "$4" = --timeout ] && [ "$5" = 60 ] && [ "$6" = hermes ]; then exit 0; fi
  # Setup first waits for the native CLI to answer a public read. This
  # fixture fakes Docker/native execution only; other native commands fail.
  if [ "$1" = exec ] && [ "$2" = -T ] && [ "$3" = --user ]; then
   case "$*" in
    *' config get '*) printf 'false\n'; exit 0;;
    *) exit 4;;
   esac
  fi
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
		cmd.Env = append(installerEnv, "PATH="+bin+":"+os.Getenv("PATH"))
		if command == "setup" {
			runtime, err := json.Marshal(map[string]any{"status": "running", "service": "hermes", "unexpectedMounts": "", "image": development.ImageName(id.Container, development.Requirements{}), "imageID": "sha256:" + strings.Repeat("d", 64), "project": id.Project, "workspace": id.Root, "home": filepath.Join(id.Root, ".hermes"), "mounts": []map[string]any{{"Type": "bind", "Source": id.Root, "Destination": "/workspace", "RW": true}, {"Type": "bind", "Source": filepath.Join(id.Root, ".hermes"), "Destination": "/opt/data", "RW": true}, {"Type": "tmpfs", "Destination": "/workspace/.hermes", "RW": true}}, "stopTimeout": compose.StopGraceSeconds})
			if err != nil {
				t.Fatal(err)
			}
			image, _ := json.Marshal(map[string]any{"id": "sha256:" + strings.Repeat("d", 64), "os": "linux", "arch": "amd64", "recipe": development.Fingerprint(development.Requirements{}), "base": qualification.FoundationImage, "layers": []string{"sha256:" + strings.Repeat("e", 64), "sha256:" + strings.Repeat("f", 64)}})
			cmd.Env = append(cmd.Env, "REPOKIT_TEST_RUNTIME="+string(runtime), "REPOKIT_TEST_IMAGE="+string(image), `REPOKIT_TEST_LAYERS=["sha256:`+strings.Repeat("e", 64)+`"]`)
		}
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

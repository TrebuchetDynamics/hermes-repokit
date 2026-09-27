package cli

import (
	"bytes"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemorySetupRequiresRunningHermesBeforeNativeWizards(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	var out, diag bytes.Buffer
	if code := a.setupMemory(r.id, r.context, &out, &diag); code == 0 || !strings.Contains(diag.String(), "running Hermes") {
		t.Fatalf("missing running-runtime preflight: %d %s", code, diag.String())
	}
}

func TestInstallScaffoldsPrivateOpenVikingWithoutModelOrCredentials(t *testing.T) {
	a, _ := foundationApp(t)
	code, out, diag := invoke(t, a, "install")
	if code != 0 {
		t.Fatalf("%d %s", code, diag)
	}
	data, err := os.ReadFile(filepath.Join(a.Directory, ".hermes/compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "  openviking:") {
		t.Fatal("normal installation omitted memory service")
	}
	dir := filepath.Join(a.Directory, ".hermes/openviking")
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("private sidecar directory: %v %v", info, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ov.conf")); !os.IsNotExist(err) {
		t.Fatal("installer invented model/credential configuration")
	}
	if !strings.Contains(out, "setup --memory") {
		t.Fatal("native memory setup handoff missing")
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
}

func legacyMemoryFixture(t *testing.T) (App, []byte) {
	t.Helper()
	a, r := foundationApp(t)
	old, err := compose.Render(r.id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	launch, err := launcher.Render(r.id, r.context)
	if err != nil {
		t.Fatal(err)
	}
	_, err = install.Publish(r.id, map[string]install.Artifact{
		"compose.yaml":          {Data: old, Mode: 0600},
		"config.yaml":           {Data: []byte("owner-config\n"), Mode: 0600},
		"bin/" + r.id.Container: {Data: launch, Mode: 0700},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return a, old
}

func TestInstallUpgradesOnlyRecognizedFoundationCompose(t *testing.T) {
	a, old := legacyMemoryFixture(t)
	state := filepath.Join(a.Directory, ".hermes")
	os.WriteFile(filepath.Join(state, ".env"), []byte("fixture credential must stay private"), 0600)
	os.MkdirAll(filepath.Join(state, "profiles/owner"), 0700)
	os.WriteFile(filepath.Join(state, "profiles/owner/SOUL.md"), []byte("owner identity"), 0600)
	code, _, diag := invoke(t, a, "install")
	if code != 0 {
		t.Fatalf("recognized foundation upgrade failed: %s", diag)
	}
	got, _ := os.ReadFile(filepath.Join(state, "compose.hermes-only.yaml"))
	if string(got) != string(old) {
		t.Fatal("original Compose backup missing")
	}
	for p, want := range map[string]string{"config.yaml": "owner-config\n", ".env": "fixture credential must stay private", "profiles/owner/SOUL.md": "owner identity"} {
		got, _ := os.ReadFile(filepath.Join(state, p))
		if string(got) != want {
			t.Fatalf("changed native %s", p)
		}
	}
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("upgrade rerun: %s", diag)
	}
}

func TestMemoryUpgradePreservesUnknownFiles(t *testing.T) {
	for _, which := range []string{"compose", "directory", "empty-directory", "marker-only", "backup"} {
		t.Run(which, func(t *testing.T) {
			a, old := legacyMemoryFixture(t)
			state := filepath.Join(a.Directory, ".hermes")
			switch which {
			case "empty-directory", "marker-only":
				os.Mkdir(filepath.Join(state, "openviking"), 0700)
				if which == "marker-only" {
					os.WriteFile(filepath.Join(state, "openviking/.gitignore"), []byte("*\n"), 0600)
				}
			case "compose":
				old = append(old, []byte("# owner change\n")...)
				os.WriteFile(filepath.Join(state, "compose.yaml"), old, 0600)
			case "directory":
				os.Mkdir(filepath.Join(state, "openviking"), 0700)
				os.WriteFile(filepath.Join(state, "openviking/ov.conf"), []byte("owner config"), 0600)
			case "backup":
				os.WriteFile(filepath.Join(state, "compose.hermes-only.yaml"), []byte("owner backup"), 0600)
			}
			if code, _, _ := invoke(t, a, "install"); code == 0 {
				t.Fatal("adopted ambiguous native state")
			}
			got, _ := os.ReadFile(filepath.Join(state, "compose.yaml"))
			if string(got) != string(old) {
				t.Fatal("changed original Compose on refusal")
			}
		})
	}
}

func TestMemorySetupRequiresPrivateTerminalBeforeNativeActions(t *testing.T) {
	a, _ := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	code, _, diag := invoke(t, a, "setup", "--memory")
	if code != 1 || !strings.Contains(diag, "terminal") {
		t.Fatalf("memory setup must request private terminal: %d %s", code, diag)
	}
	if _, err := os.Stat(filepath.Join(a.Directory, ".hermes/openviking/ov.conf")); !os.IsNotExist(err) {
		t.Fatal("noninteractive setup wrote native config")
	}
}

func TestMemoryUpgradeResumesRecognizedPreparation(t *testing.T) {
	for _, marker := range []bool{false, true} {
		t.Run(fmt.Sprint(marker), func(t *testing.T) {
			a, old := legacyMemoryFixture(t)
			state := filepath.Join(a.Directory, ".hermes")
			os.WriteFile(filepath.Join(state, "compose.hermes-only.yaml"), old, 0600)
			os.Mkdir(filepath.Join(state, "openviking"), 0700)
			if marker {
				os.WriteFile(filepath.Join(state, "openviking/.gitignore"), []byte("*\n"), 0600)
			}
			if code, _, diag := invoke(t, a, "install"); code != 0 {
				t.Fatal(diag)
			}
		})
	}
}

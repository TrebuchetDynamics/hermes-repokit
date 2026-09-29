package cli

import (
	"os"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/install"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
)

// hermesOnlyFixture publishes the former generated Hermes-only Compose, which
// install still recognizes and upgrades while preserving native state.
func hermesOnlyFixture(t *testing.T) (App, []byte) {
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
	a, old := hermesOnlyFixture(t)
	state := a.Directory + "/.hermes/"
	os.WriteFile(state+".env", []byte("fixture credential must stay private"), 0600)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatalf("recognized foundation upgrade failed: %s", diag)
	}
	if got, _ := os.ReadFile(state + "compose.hermes-only.yaml"); string(got) != string(old) {
		t.Fatal("original Compose backup missing")
	}
	for p, want := range map[string]string{"config.yaml": "owner-config\n", ".env": "fixture credential must stay private"} {
		if got, _ := os.ReadFile(state + p); string(got) != want {
			t.Fatalf("changed native %s", p)
		}
	}
}

func TestUpgradeRefusesEditedComposeOrConflictingBackup(t *testing.T) {
	for _, which := range []string{"compose", "backup"} {
		t.Run(which, func(t *testing.T) {
			a, old := hermesOnlyFixture(t)
			state := a.Directory + "/.hermes/"
			if which == "compose" {
				old = append(old, []byte("# owner change\n")...)
				os.WriteFile(state+"compose.yaml", old, 0600)
			} else {
				os.WriteFile(state+"compose.hermes-only.yaml", []byte("owner backup"), 0600)
			}
			if code, _, _ := invoke(t, a, "install"); code == 0 {
				t.Fatal("adopted ambiguous native state")
			}
			if got, _ := os.ReadFile(state + "compose.yaml"); string(got) != string(old) {
				t.Fatal("changed original Compose on refusal")
			}
		})
	}
}

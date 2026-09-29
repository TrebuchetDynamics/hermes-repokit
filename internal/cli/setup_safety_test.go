package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateSetupRefusesGitExposedStateBeforeWizard(t *testing.T) {
	for _, exposure := range []string{"unignored-state", "future-oauth"} {
		t.Run(exposure, func(t *testing.T) {
			a, r := foundationApp(t)
			if code, _, diag := invoke(t, a, "install"); code != 0 {
				t.Fatal(diag)
			}
			r.runtime = developmentRuntimeFixture(r.id)
			ignore := filepath.Join(r.id.Root, ".hermes/.gitignore")
			if exposure == "future-oauth" {
				if err := os.WriteFile(ignore, []byte("*\n!auth.json\n"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Remove(ignore); err != nil {
				t.Fatal(err)
			}
			bin := t.TempDir()
			marker := filepath.Join(bin, "wizard-started")
			t.Setenv("REPOKIT_TEST_WIZARD_MARKER", marker)
			t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n: > \"$REPOKIT_TEST_WIZARD_MARKER\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			if code, _, _ := invoke(t, a, "setup"); code == 0 {
				t.Fatal("accepted Git-exposed private state")
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("native credential wizard started before checking Git protection")
			}
		})
	}
}

func TestPrivateSetupChecksDeploymentBeforeWizard(t *testing.T) {
	for _, kind := range []string{"qualified", "edited-compose", "foreign-image", "wrong-mount", "extra-mount", "wrong-service", "stopped", "unavailable"} {
		t.Run(kind, func(t *testing.T) {
			a, r := foundationApp(t)
			if code, _, diag := invoke(t, a, "install"); code != 0 {
				t.Fatal(diag)
			}
			r.runtime = developmentRuntimeFixture(r.id)
			switch kind {
			case "edited-compose":
				data, err := os.ReadFile(r.id.Compose)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(r.id.Compose, []byte(strings.Replace(string(data), "context: ./development-image", "context: ./owner-image", 1)), 0600); err != nil {
					t.Fatal(err)
				}
			case "foreign-image":
				r.runtime = strings.Replace(r.runtime, "repokit/", "foreign/", 1)
			case "wrong-mount":
				r.runtime = strings.ReplaceAll(r.runtime, r.id.Root, "/unrelated")
			case "extra-mount":
				r.runtime = strings.Replace(r.runtime, `"Destination":"/opt/data"`, `"Destination":"/foreign"`, 1)
			case "wrong-service":
				r.runtime = strings.Replace(r.runtime, `"service":"hermes"`, `"service":"other"`, 1)
			case "stopped":
				r.runtime = strings.Replace(r.runtime, "running", "exited", 1)
			case "unavailable":
				r.runtime = ""
			}
			bin := t.TempDir()
			marker := filepath.Join(bin, "wizard-started")
			t.Setenv("REPOKIT_TEST_WIZARD_MARKER", marker)
			t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n: > \"$REPOKIT_TEST_WIZARD_MARKER\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			a.Initializer = &gatewayInput{state: "current"}
			invoke(t, a, "setup")
			_, err := os.Stat(marker)
			if got, want := err == nil, kind == "qualified"; got != want {
				t.Fatalf("wizard started=%v, want %v", got, want)
			}
		})
	}
}

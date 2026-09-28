package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	layapackage "github.com/TrebuchetDynamics/hermes-repokit/packaging/laya"
)

func addLayaFixture(t *testing.T, files map[string]Artifact) {
	t.Helper()
	files["compose.yaml"] = Artifact{[]byte("stack compose"), 0600}
	files["laya/.gitignore"] = Artifact{[]byte("*\n"), 0600}
	entries, err := layapackage.Assets.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := layapackage.Assets.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files["laya-image/"+entry.Name()] = Artifact{data, 0600}
	}
}

func TestLayaUpgradePublishesRecipeAndPreservesNativeCacheOnRerun(t *testing.T) {
	id, files := fixture(t)
	previous := append([]byte(nil), files["compose.yaml"].Data...)
	if _, err := Publish(id, files, nil); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(id.Root, ".hermes")
	if err := os.WriteFile(filepath.Join(state, "config.yaml"), []byte("owner config"), 0600); err != nil {
		t.Fatal(err)
	}
	addLayaFixture(t, files)
	upgrades := []StackUpgrade{{Compose: []byte("other historical document")}, {Compose: previous}}
	if changed, err := PublishStackChecked(id, files, upgrades, nil); err != nil || !changed {
		t.Fatalf("upgrade: %t %v", changed, err)
	}
	for name, want := range map[string][]byte{"compose.before-laya.yaml": previous, "config.yaml": []byte("owner config"), "laya-image/Dockerfile": files["laya-image/Dockerfile"].Data} {
		data, err := os.ReadFile(filepath.Join(state, name))
		if err != nil || !bytes.Equal(data, want) {
			t.Fatalf("changed/missing %s: %v", name, err)
		}
	}
	cache := filepath.Join(state, "laya", "native-cache")
	if err := os.WriteFile(cache, []byte("native cache"), 0600); err != nil {
		t.Fatal(err)
	}
	if changed, err := PublishStackChecked(id, files, upgrades, nil); err != nil || changed {
		t.Fatalf("rerun: %t %v", changed, err)
	}
	if data, err := os.ReadFile(cache); err != nil || string(data) != "native cache" {
		t.Fatal("native cache changed")
	}
}

func TestLayaUpgradeRefusesUnknownStateAndPreservesCompose(t *testing.T) {
	for _, which := range []string{"cache", "recipe", "compose", "backup"} {
		t.Run(which, func(t *testing.T) {
			id, files := fixture(t)
			previous := append([]byte(nil), files["compose.yaml"].Data...)
			if _, err := Publish(id, files, nil); err != nil {
				t.Fatal(err)
			}
			state := filepath.Join(id.Root, ".hermes")
			switch which {
			case "cache":
				if err := os.Mkdir(filepath.Join(state, "laya"), 0700); err != nil {
					t.Fatal(err)
				}
			case "recipe":
				if err := os.Mkdir(filepath.Join(state, "laya-image"), 0700); err != nil {
					t.Fatal(err)
				}
			case "compose":
				previous = []byte("owner Compose")
				if err := os.WriteFile(id.Compose, previous, 0600); err != nil {
					t.Fatal(err)
				}
			case "backup":
				if err := os.WriteFile(filepath.Join(state, "compose.before-laya.yaml"), []byte("owner backup"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			addLayaFixture(t, files)
			if _, err := PublishStackChecked(id, files, []StackUpgrade{{Compose: []byte("compose")}}, nil); err == nil {
				t.Fatal("unknown state adopted")
			}
			data, err := os.ReadFile(id.Compose)
			if err != nil || !bytes.Equal(data, previous) {
				t.Fatal("Compose changed on refusal")
			}
		})
	}
}

func TestLayaRerunPreservesEditedRecipeAndRefusesMissingCache(t *testing.T) {
	for _, which := range []string{"recipe", "extra-recipe", "cache"} {
		t.Run(which, func(t *testing.T) {
			id, files := fixture(t)
			addLayaFixture(t, files)
			if _, err := Publish(id, files, nil); err != nil {
				t.Fatal(err)
			}
			state := filepath.Join(id.Root, ".hermes")
			if which == "extra-recipe" {
				path := filepath.Join(state, "laya-image/.dockerignore")
				if err := os.WriteFile(path, []byte("Dockerfile\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := Publish(id, files, nil); err == nil {
					t.Fatal("unknown build recipe file adopted")
				}
				if data, err := os.ReadFile(path); err != nil || string(data) != "Dockerfile\n" {
					t.Fatal("owner file overwritten")
				}
			} else if which == "recipe" {
				path := filepath.Join(state, "laya-image/Dockerfile")
				if err := os.WriteFile(path, []byte("owner recipe"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := Publish(id, files, nil); err == nil {
					t.Fatal("edited recipe accepted")
				}
				if data, err := os.ReadFile(path); err != nil || string(data) != "owner recipe" {
					t.Fatal("edited recipe overwritten")
				}
			} else {
				if err := os.RemoveAll(filepath.Join(state, "laya")); err != nil {
					t.Fatal(err)
				}
				if _, err := Publish(id, files, nil); err == nil {
					t.Fatal("missing cache silently recreated")
				}
			}
		})
	}
}

func TestLayaUpgradeResumesOnlyExactPreparation(t *testing.T) {
	id, files := fixture(t)
	previous := append([]byte(nil), files["compose.yaml"].Data...)
	if _, err := Publish(id, files, nil); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(id.Root, ".hermes")
	addLayaFixture(t, files)
	if err := os.WriteFile(filepath.Join(state, "compose.before-laya.yaml"), previous, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(state, "laya-image"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "laya-image/Dockerfile"), files["laya-image/Dockerfile"].Data, 0600); err != nil {
		t.Fatal(err)
	}
	if changed, err := PublishStackChecked(id, files, []StackUpgrade{{Compose: previous}}, nil); err != nil || !changed {
		t.Fatalf("resume: %t %v", changed, err)
	}
	if changed, err := PublishStackChecked(id, files, nil, nil); err != nil || changed {
		t.Fatalf("resumed rerun: %t %v", changed, err)
	}
}

func TestExplicitLayaSelectionPreservesEarlierUpgradeBackup(t *testing.T) {
	id, files := fixture(t)
	if _, err := Publish(id, files, nil); err != nil {
		t.Fatal(err)
	}
	addLayaFixture(t, files)
	if _, err := PublishStackChecked(id, files, []StackUpgrade{{Compose: []byte("compose")}}, nil); err != nil {
		t.Fatal(err)
	}
	previous := append([]byte(nil), files["compose.yaml"].Data...)
	for name := range files {
		if strings.HasPrefix(name, "laya/") || strings.HasPrefix(name, "laya-image/") {
			delete(files, name)
		}
	}
	files["compose.yaml"] = Artifact{[]byte("explicit content image"), 0600}
	if changed, err := PublishStackChecked(id, files, []StackUpgrade{{Compose: previous, BackupName: "compose.before-laya-image.yaml"}}, nil); err != nil || !changed {
		t.Fatalf("explicit selection: %t %v", changed, err)
	}
	for name, want := range map[string]string{"compose.before-laya.yaml": "compose", "compose.before-laya-image.yaml": string(previous)} {
		data, err := os.ReadFile(filepath.Join(id.Root, ".hermes", name))
		if err != nil || string(data) != want {
			t.Fatalf("backup %s overwritten", name)
		}
	}
}

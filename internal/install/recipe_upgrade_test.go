package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
)

func TestRecipeUpgradeResumesOnlyKnownFileStates(t *testing.T) {
	for _, scenario := range []string{"old", "backup", "replaced", "replaced-without-backup", "wrong-backup", "backup-link", "recipe-drift", "unknown-file", "missing-file", "file-link", "directory-link", "compose-drift"} {
		t.Run(scenario, func(t *testing.T) {
			id, files := fixture(t)
			req := development.Requirements{Go: true}
			old, err := development.LegacyRecipe(req)
			if err != nil {
				t.Fatal(err)
			}
			current, err := development.Recipe(req)
			if err != nil {
				t.Fatal(err)
			}
			for name, data := range old {
				files["development-image/"+name] = Artifact{Data: data, Mode: 0600}
			}
			prior := []byte("services:\n  hermes:\n    build:\n      context: ./development-image\n")
			files["compose.yaml"] = Artifact{Data: prior, Mode: 0600}
			if _, err := Publish(id, files, nil); err != nil {
				t.Fatal(err)
			}
			for name, data := range current {
				files["development-image/"+name] = Artifact{Data: data, Mode: 0600}
			}
			files["compose.yaml"] = Artifact{Data: []byte("new compose"), Mode: 0600}
			state := filepath.Join(id.Root, ".hermes")
			write := func(name string, data []byte) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(state, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			write("config.yaml", []byte("owner native config"))
			write("native-data", []byte("owner database"))
			backup := "compose.before-path.yaml"
			switch scenario {
			case "backup":
				write(backup, prior)
			case "replaced":
				write(backup, prior)
				write("development-image/Dockerfile", current["Dockerfile"])
			case "replaced-without-backup":
				write("development-image/Dockerfile", current["Dockerfile"])
			case "wrong-backup":
				write(backup, []byte("owner backup"))
			case "backup-link":
				if err := os.Symlink("compose.yaml", filepath.Join(state, backup)); err != nil {
					t.Fatal(err)
				}
			case "recipe-drift":
				write("development-image/repokit-openviking", []byte("owner helper"))
			case "unknown-file":
				write("development-image/owner", []byte("owner file"))
			case "missing-file":
				if err := os.Remove(filepath.Join(state, "development-image/Dockerfile")); err != nil {
					t.Fatal(err)
				}
			case "file-link":
				if err := os.Rename(filepath.Join(state, "development-image/Dockerfile"), filepath.Join(state, "saved-dockerfile")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../saved-dockerfile", filepath.Join(state, "development-image/Dockerfile")); err != nil {
					t.Fatal(err)
				}
			case "directory-link":
				if err := os.Rename(filepath.Join(state, "development-image"), filepath.Join(state, "saved-recipe")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("saved-recipe", filepath.Join(state, "development-image")); err != nil {
					t.Fatal(err)
				}
			case "compose-drift":
				write("compose.yaml", append(append([]byte{}, prior...), []byte("# owner\n")...))
			}
			before := snapshotRecipeTree(t, state)
			upgrades := []StackUpgrade{{Compose: prior, BackupName: backup, PreviousRecipe: old}}
			created, err := PublishStackChecked(id, files, upgrades, nil)
			success := scenario == "old" || scenario == "backup" || scenario == "replaced"
			if success {
				if err != nil || !created {
					t.Fatalf("upgrade: %v %v", created, err)
				}
				for name, want := range current {
					got, err := os.ReadFile(filepath.Join(state, "development-image", name))
					if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("wrong upgraded %s: %v", name, err)
					}
				}
				for name, want := range map[string][]byte{backup: prior, "config.yaml": []byte("owner native config"), "native-data": []byte("owner database")} {
					got, _ := os.ReadFile(filepath.Join(state, name))
					if !bytes.Equal(got, want) {
						t.Fatalf("lost %s", name)
					}
				}
				if created, err := PublishStackChecked(id, files, upgrades, nil); err != nil || created {
					t.Fatalf("rerun: %v %v", created, err)
				}
			} else {
				if err == nil || created {
					t.Fatal("adopted owner drift")
				}
				after := snapshotRecipeTree(t, state)
				if len(before) != len(after) {
					t.Fatal("changed file set on refused upgrade")
				}
				for name, want := range before {
					if !bytes.Equal(after[name], want) {
						t.Fatalf("changed %s on refused upgrade", name)
					}
				}
			}
		})
	}
}

func snapshotRecipeTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			out[path] = []byte("symlink:" + link)
			return err
		}
		data, err := os.ReadFile(path)
		out[path] = data
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

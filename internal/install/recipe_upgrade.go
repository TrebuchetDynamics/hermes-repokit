package install

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// upgradeDevelopmentRecipe accepts the complete exact old recipe, or an
// interrupted old/new mix backed by the exact saved Compose. Files are replaced
// atomically and Compose is published last, so retries never accept partial
// file contents or arbitrary owner changes.
func upgradeDevelopmentRecipe(state *os.Root, files map[string]Artifact, previous []byte, backup string, old map[string][]byte, write bool) error {
	resumed := false
	if _, err := state.Lstat(backup); err == nil {
		saved, err := readRecipeFile(state, backup)
		if err != nil || !bytes.Equal(saved, previous) {
			return fmt.Errorf("conflicting recipe upgrade backup; owner state preserved")
		}
		resumed = true
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := fs.ReadDir(state.FS(), "development-image")
	if err != nil {
		return err
	}
	// Every present file must hold its exact old bytes; an interrupted retry
	// may also find new bytes, a new file already created, or an old-only file
	// already removed. Anything else is owner state and is preserved.
	present := map[string]bool{}
	for _, entry := range entries {
		before, known := old[entry.Name()]
		after, wanted := files["development-image/"+entry.Name()]
		if !known && !(resumed && wanted) {
			return fmt.Errorf("unknown development recipe file; owner state preserved")
		}
		got, err := readRecipeFile(state, "development-image/"+entry.Name())
		if err != nil {
			return err
		}
		if !(known && bytes.Equal(got, before)) && !(resumed && wanted && bytes.Equal(got, after.Data)) {
			return fmt.Errorf("development recipe differs; owner state preserved")
		}
		present[entry.Name()] = true
	}
	if !resumed && len(present) != len(old) {
		return fmt.Errorf("previous development recipe incomplete; owner state preserved")
	}
	if !write {
		return nil
	}
	if !resumed {
		return fmt.Errorf("recipe upgrade requires durable Compose backup")
	}
	// Persist the backup's directory entry and an exact copy of the old recipe
	// before changing any recipe bytes, so an interrupted retry can recognize it.
	if err := syncRecipeDirectory(state, "."); err != nil {
		return err
	}
	if err := backupRecipe(state, RecipeBackupName(backup), old); err != nil {
		return err
	}
	for name, after := range files {
		base, ok := strings.CutPrefix(name, "development-image/")
		if !ok {
			continue
		}
		if !present[base] {
			if err := replaceRecipeFile(state, name, nil, after); err != nil {
				return err
			}
			continue
		}
		got, err := readRecipeFile(state, name)
		if err != nil {
			return err
		}
		if bytes.Equal(got, after.Data) {
			continue
		}
		if !bytes.Equal(got, old[base]) {
			return fmt.Errorf("development recipe changed during upgrade; owner state preserved")
		}
		if err := replaceRecipeFile(state, name, got, after); err != nil {
			return err
		}
	}
	for base := range present {
		if _, wanted := files["development-image/"+base]; wanted {
			continue
		}
		got, err := readRecipeFile(state, "development-image/"+base)
		if err != nil || !bytes.Equal(got, old[base]) {
			return fmt.Errorf("development recipe changed during upgrade; owner state preserved")
		}
		if err := state.Remove("development-image/" + base); err != nil {
			return err
		}
	}
	if err := syncRecipeDirectory(state, "development-image"); err != nil {
		return err
	}
	return checkDevelopmentArtifacts(state, "", files)
}

func readRecipeFile(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1048576 {
		return nil, fmt.Errorf("development recipe missing or unsafe")
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("development recipe changed during read")
	}
	data, err := io.ReadAll(io.LimitReader(f, 1048577))
	if err != nil || len(data) > 1048576 {
		return nil, fmt.Errorf("development recipe unreadable or oversized")
	}
	return data, nil
}

func replaceRecipeFile(root *os.Root, name string, previous []byte, next Artifact) error {
	// Staging outside the recipe directory keeps interrupted temporary files from
	// looking like supported build inputs on a later attempt.
	temp := ".development-stage-" + rand.Text()
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, next.Mode)
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	_, err = f.Write(next.Data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if previous == nil {
		if _, err := root.Lstat(name); !os.IsNotExist(err) {
			return fmt.Errorf("development recipe changed during replacement; owner state preserved")
		}
	} else if got, err := readRecipeFile(root, name); err != nil || !bytes.Equal(got, previous) {
		return fmt.Errorf("development recipe changed during replacement; owner state preserved")
	}
	if err := root.Rename(temp, name); err != nil {
		return err
	}
	return syncRecipeDirectory(root, "development-image")
}

func syncRecipeDirectory(root *os.Root, name string) error {
	dir, err := root.Open(name)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// RecipeBackupName is the directory holding the exact old recipe of an upgrade.
func RecipeBackupName(composeBackup string) string {
	return strings.TrimSuffix(composeBackup, ".yaml") + ".development-image"
}

func backupRecipe(state *os.Root, name string, old map[string][]byte) error {
	if info, err := state.Lstat(name); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("unsafe recipe backup; owner state preserved")
		}
		entries, err := fs.ReadDir(state.FS(), name)
		if err != nil || len(entries) != len(old) {
			return fmt.Errorf("conflicting recipe backup; owner state preserved")
		}
		for _, entry := range entries {
			got, err := readRecipeFile(state, name+"/"+entry.Name())
			if err != nil || !bytes.Equal(got, old[entry.Name()]) {
				return fmt.Errorf("conflicting recipe backup; owner state preserved")
			}
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	stage := ".recipe-backup-stage-" + rand.Text()
	if err := state.Mkdir(stage, 0700); err != nil {
		return err
	}
	defer state.RemoveAll(stage)
	for file, data := range old {
		f, err := state.OpenFile(stage+"/"+file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err == nil {
			err = f.Sync()
		}
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	if err := syncRecipeDirectory(state, stage); err != nil {
		return err
	}
	if err := state.Rename(stage, name); err != nil {
		return err
	}
	return syncRecipeDirectory(state, ".")
}

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
	if len(entries) != len(old) {
		return fmt.Errorf("previous development recipe incomplete or unknown files; owner state preserved")
	}
	for _, entry := range entries {
		before, known := old[entry.Name()]
		after, wanted := files["development-image/"+entry.Name()]
		if !known || !wanted {
			return fmt.Errorf("unknown development recipe file; owner state preserved")
		}
		got, err := readRecipeFile(state, "development-image/"+entry.Name())
		if err != nil {
			return err
		}
		if !bytes.Equal(got, before) && !(resumed && bytes.Equal(got, after.Data)) {
			return fmt.Errorf("development recipe differs; owner state preserved")
		}
	}
	count := 0
	for name := range files {
		if strings.HasPrefix(name, "development-image/") {
			count++
		}
	}
	if count != len(old) {
		return fmt.Errorf("unsupported development recipe topology change")
	}
	if !write {
		return nil
	}
	if !resumed {
		return fmt.Errorf("recipe upgrade requires durable Compose backup")
	}
	// Persist the backup's directory entry before changing any recipe bytes.
	if err := syncRecipeDirectory(state, "."); err != nil {
		return err
	}
	for _, entry := range entries {
		name := "development-image/" + entry.Name()
		got, err := readRecipeFile(state, name)
		if err != nil {
			return err
		}
		after := files[name]
		if bytes.Equal(got, after.Data) {
			continue
		}
		if !bytes.Equal(got, old[entry.Name()]) {
			return fmt.Errorf("development recipe changed during upgrade; owner state preserved")
		}
		if err := replaceRecipeFile(state, name, got, after); err != nil {
			return err
		}
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
	got, err := readRecipeFile(root, name)
	if err != nil || !bytes.Equal(got, previous) {
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

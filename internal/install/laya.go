package install

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// A build recipe is installer-owned; the writable cache is native-owned once
// Compose is published. Neither is reconstructed silently on a no-op rerun.
func checkLayaArtifacts(root *os.Root, prefix string, files map[string]Artifact) error {
	if _, selected := files["laya/.gitignore"]; !selected {
		return nil
	}
	info, err := root.Lstat(prefix + "laya")
	if err != nil || !info.IsDir() {
		return fmt.Errorf("Laya cache directory absent or unsafe; preserve native state before repair")
	}
	entries, err := fs.ReadDir(root.FS(), prefix+"laya-image")
	if err != nil {
		return fmt.Errorf("Laya build directory absent or unsafe")
	}
	for _, entry := range entries {
		if _, ok := files["laya-image/"+entry.Name()]; !ok {
			return fmt.Errorf("unrecognized Laya build artifact; owner file preserved: %s", entry.Name())
		}
	}
	for name, artifact := range files {
		if !strings.HasPrefix(name, "laya-image/") {
			continue
		}
		info, err := root.Lstat(prefix + name)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("Laya build artifact absent or unsafe: %s", name)
		}
		data, err := root.ReadFile(prefix + name)
		if err != nil || !bytes.Equal(data, artifact.Data) {
			return fmt.Errorf("Laya build recipe differs; owner file preserved: %s", name)
		}
	}
	return nil
}

func validateLayaPreparation(state *os.Root, files map[string]Artifact, previous []byte, backupName string) error {
	if _, selected := files["laya/.gitignore"]; !selected {
		return nil
	}
	for _, dir := range []string{"laya", "laya-image"} {
		info, err := state.Lstat(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.IsDir() {
			return fmt.Errorf("unsafe existing Laya directory: %s", dir)
		}
		backup, err := state.ReadFile(backupName)
		if err != nil || !bytes.Equal(backup, previous) {
			return fmt.Errorf("unexplained Laya directory exists; owner state preserved")
		}
		entries, err := fs.ReadDir(state.FS(), dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := dir + "/" + entry.Name()
			artifact, ok := files[name]
			if !ok || !entry.Type().IsRegular() {
				return fmt.Errorf("unrecognized Laya preparation artifact; owner state preserved: %s", name)
			}
			data, err := state.ReadFile(name)
			if err != nil || !bytes.Equal(data, artifact.Data) {
				return fmt.Errorf("conflicting Laya preparation artifact; owner state preserved: %s", name)
			}
		}
	}
	return nil
}

func prepareLayaArtifacts(state *os.Root, files map[string]Artifact, previous []byte, backupName string) error {
	if _, selected := files["laya/.gitignore"]; !selected {
		return nil
	}
	if err := createOrMatch(state, backupName, previous); err != nil {
		return err
	}
	for _, dir := range []string{"laya", "laya-image"} {
		if err := state.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
			return err
		}
	}
	for name, artifact := range files {
		if strings.HasPrefix(name, "laya/") || strings.HasPrefix(name, "laya-image/") {
			if err := createOrMatch(state, name, artifact.Data); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"laya", "laya-image"} {
		dir, err := state.Open(name)
		if err != nil {
			return err
		}
		err = dir.Sync()
		closeErr := dir.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

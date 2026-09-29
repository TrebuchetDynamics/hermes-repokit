package install

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

func checkDevelopmentArtifacts(root *os.Root, prefix string, files map[string]Artifact) error {
	if _, selected := files["development-image/Dockerfile"]; !selected {
		return nil
	}
	info, err := root.Lstat(prefix + "development-image")
	if err != nil || !info.IsDir() {
		return fmt.Errorf("development build directory missing or unsafe")
	}
	entries, err := fs.ReadDir(root.FS(), prefix+"development-image")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, ok := files["development-image/"+e.Name()]; !ok {
			return fmt.Errorf("unknown development recipe file; owner state preserved")
		}
	}
	for name, want := range files {
		if !strings.HasPrefix(name, "development-image/") {
			continue
		}
		info, err := root.Lstat(prefix + name)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1048576 {
			return fmt.Errorf("development recipe missing or unsafe")
		}
		f, err := root.Open(prefix + name)
		if err != nil {
			return err
		}
		got, err := io.ReadAll(io.LimitReader(f, 1048577))
		f.Close()
		if err != nil || !bytes.Equal(got, want.Data) {
			return fmt.Errorf("development recipe differs; owner state preserved")
		}
	}
	return nil
}
func prepareDevelopmentArtifacts(state *os.Root, files map[string]Artifact, previous []byte, backup string, write bool, previousRecipe map[string][]byte) error {
	if _, selected := files["development-image/Dockerfile"]; !selected {
		return nil
	}
	info, err := state.Lstat("development-image")
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("unsafe development build directory")
		}
		if len(previousRecipe) > 0 {
			return upgradeDevelopmentRecipe(state, files, previous, backup, previousRecipe, write)
		}
		if strings.Contains(string(previous), "      context: ./development-image\n") {
			return checkDevelopmentArtifacts(state, "", files)
		}
		saved, e := state.ReadFile(backup)
		if e != nil || !bytes.Equal(saved, previous) {
			return fmt.Errorf("unexplained development directory; owner state preserved")
		}
		entries, e := fs.ReadDir(state.FS(), "development-image")
		if e != nil {
			return e
		}
		for _, entry := range entries {
			name := "development-image/" + entry.Name()
			a, ok := files[name]
			if !ok || !entry.Type().IsRegular() {
				return fmt.Errorf("unknown development preparation file")
			}
			b, e := state.ReadFile(name)
			if e != nil || !bytes.Equal(b, a.Data) {
				return fmt.Errorf("development preparation drift preserved")
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	} else if len(previousRecipe) > 0 {
		return fmt.Errorf("previous development recipe missing; owner state preserved")
	}
	if !write {
		return nil
	}
	if err := state.Mkdir("development-image", 0700); err != nil && !os.IsExist(err) {
		return err
	}
	for name, a := range files {
		if strings.HasPrefix(name, "development-image/") {
			if err := createOrMatch(state, name, a.Data); err != nil {
				return err
			}
		}
	}
	d, err := state.Open("development-image")
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

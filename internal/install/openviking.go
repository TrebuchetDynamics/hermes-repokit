package install

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"io/fs"
	"os"
)

// addOpenViking executes under the publication lock, with an exact public
// Compose preimage. It never opens or rewrites native configuration/credentials.
func addOpenViking(root *os.Root, id target.Identity, rootInfo, lockInfo fs.FileInfo, previous, next []byte) (bool, error) {
	state, err := root.OpenRoot(".hermes")
	if err != nil {
		return false, err
	}
	defer state.Close()
	stateInfo, err := state.Stat(".")
	if err != nil {
		return false, err
	}
	composeInfo, err := state.Lstat("compose.yaml")
	if err != nil {
		return false, err
	}
	// A partial preparation is resumable only while the directory contains no
	// native service data. Unknown existing data must never be adopted.
	entries, err := fs.ReadDir(state.FS(), "openviking")
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err == nil {
		backup, e := state.ReadFile("compose.hermes-only.yaml")
		if e != nil || !bytes.Equal(backup, previous) {
			return false, fmt.Errorf("unexplained OpenViking directory exists; owner state preserved")
		}
	}
	for _, entry := range entries {
		if entry.Name() != ".gitignore" {
			return false, fmt.Errorf("unrecognized OpenViking data exists; preserve it before selecting the service")
		}
		b, e := state.ReadFile("openviking/.gitignore")
		if e != nil || !bytes.Equal(b, []byte("*\n")) {
			return false, fmt.Errorf("OpenViking directory ownership is ambiguous")
		}
	}
	// Preserve the old public Compose before any replacement. Interrupted attempts
	// may leave this exact backup; conflicting backup content is owner state.
	if err := createOrMatch(state, "compose.hermes-only.yaml", previous); err != nil {
		return false, err
	}
	if err := state.Mkdir("openviking", 0700); err != nil && !os.IsExist(err) {
		return false, err
	}
	if err := createOrMatch(state, "openviking/.gitignore", []byte("*\n")); err != nil {
		return false, err
	}
	temp := ".compose-stage-" + rand.Text()
	f, err := state.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, err
	}
	defer state.Remove(temp)
	_, err = f.Write(next)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		return false, closeErr
	}
	current, err := root.Lstat(".hermes")
	if err != nil || !os.SameFile(current, stateInfo) || !sameRootPath(id.Root, rootInfo) || !sameLock(root, lockInfo) {
		return false, fmt.Errorf("deployment changed during memory preparation")
	}
	now, err := state.Lstat("compose.yaml")
	if err != nil || !os.SameFile(now, composeInfo) {
		return false, fmt.Errorf("Compose changed during memory preparation")
	}
	b, err := state.ReadFile("compose.yaml")
	if err != nil || !bytes.Equal(b, previous) {
		return false, fmt.Errorf("Compose changed during memory preparation")
	}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return false, fmt.Errorf("unsafe state appeared during memory preparation")
	}
	if err := state.Rename(temp, "compose.yaml"); err != nil {
		return false, err
	}
	dir, err := state.Open(".")
	if err != nil {
		return true, err
	}
	defer dir.Close()
	return true, dir.Sync()
}

func createOrMatch(root *os.Root, path string, data []byte) error {
	f, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		info, e := root.Lstat(path)
		if e != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("unsafe existing memory preparation artifact")
		}
		b, e := root.ReadFile(path)
		if e != nil || !bytes.Equal(b, data) {
			return fmt.Errorf("conflicting memory preparation artifact; owner file preserved")
		}
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

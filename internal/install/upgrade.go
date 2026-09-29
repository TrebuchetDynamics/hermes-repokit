package install

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/fs"
	"os"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// upgradeCompose executes under the publication lock, with an exact public
// Compose preimage. It never opens or rewrites native configuration/credentials.
func upgradeCompose(root *os.Root, id target.Identity, rootInfo, lockInfo fs.FileInfo, previous []byte, files map[string]Artifact, customBackupName string, previousRecipe map[string][]byte, previousLauncher string) (bool, error) {
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
	backupName := "compose.before-upgrade.yaml"
	if customBackupName != "" {
		backupName = customBackupName
	}
	if err := prepareDevelopmentArtifacts(state, files, previous, backupName, false, previousRecipe); err != nil {
		return false, err
	}
	if err := createOrMatch(state, backupName, previous); err != nil {
		return false, err
	}
	if err := prepareDevelopmentArtifacts(state, files, previous, backupName, true, previousRecipe); err != nil {
		return false, err
	}
	if previousLauncher != "" && previousLauncher != id.Container {
		if err := syncRecipeDirectory(state, "."); err != nil {
			return false, err
		}
		if err := checkLauncher(state, "bin/"+previousLauncher, files["bin/"+id.Container].Data, false); err != nil {
			return false, err
		}
		if err := publishLauncher(state, "bin/"+id.Container, files["bin/"+id.Container]); err != nil {
			return false, err
		}
	}
	temp := ".compose-stage-" + rand.Text()
	f, err := state.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, err
	}
	defer state.Remove(temp)
	_, err = f.Write(files["compose.yaml"].Data)
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
		return false, fmt.Errorf("deployment changed during Compose upgrade")
	}
	now, err := state.Lstat("compose.yaml")
	if err != nil || !os.SameFile(now, composeInfo) {
		return false, fmt.Errorf("Compose changed during upgrade")
	}
	b, err := state.ReadFile("compose.yaml")
	if err != nil || !bytes.Equal(b, previous) {
		return false, fmt.Errorf("Compose changed during upgrade")
	}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return false, fmt.Errorf("unsafe state appeared during Compose upgrade")
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
			return fmt.Errorf("unsafe existing upgrade artifact")
		}
		b, e := root.ReadFile(path)
		if e != nil || !bytes.Equal(b, data) {
			return fmt.Errorf("conflicting upgrade artifact; owner file preserved")
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

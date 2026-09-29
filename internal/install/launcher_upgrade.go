package install

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
)

func checkLauncher(root *os.Root, name string, expected []byte, allowMissing bool) error {
	info, err := root.Lstat(name)
	if os.IsNotExist(err) && allowMissing {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0100 == 0 {
		return fmt.Errorf("generated launcher missing or unsafe; owner state preserved")
	}
	data, err := readRecipeFile(root, name)
	if err != nil || !bytes.Equal(data, expected) {
		return fmt.Errorf("generated launcher differs; owner state preserved")
	}
	return nil
}

func publishLauncher(root *os.Root, name string, artifact Artifact) error {
	if _, err := root.Lstat(name); !os.IsNotExist(err) {
		return checkLauncher(root, name, artifact.Data, false)
	}
	// An exclusive hard link publishes a complete executable without replacing
	// a concurrently created owner command. Keep the old executable untouched.
	temp := ".launcher-stage-" + rand.Text()
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, artifact.Mode)
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	_, err = f.Write(artifact.Data)
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
	if err := root.Link(temp, name); err != nil {
		return err
	}
	return syncRecipeDirectory(root, "bin")
}

package install

import (
	"bytes"
	"fmt"
	"os"
)

// checkLauncher requires the exact generated launcher before a reconfiguration.
func checkLauncher(root *os.Root, name string, expected []byte) error {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0100 == 0 {
		return fmt.Errorf("generated launcher missing or unsafe; owner state preserved")
	}
	data, err := readRecipeFile(root, name)
	if err != nil || !bytes.Equal(data, expected) {
		return fmt.Errorf("generated launcher differs; owner state preserved")
	}
	return nil
}

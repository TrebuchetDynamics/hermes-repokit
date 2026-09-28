package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// Expose publishes an absolute host symlink only after the generated launcher
// exists. It never replaces an entry, including dangling links or old targets.
func Expose(id target.Identity, home string) (string, error) {
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("an absolute user home directory is required")
	}
	command := filepath.Join(home, ".local", "bin", id.Container)
	if _, err := Context(id); err != nil {
		return command, fmt.Errorf("generated launcher is not recognized: %w", err)
	}
	info, err := os.Lstat(id.Launcher)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0100 == 0 {
		return command, fmt.Errorf("generated launcher is not executable")
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return command, err
	}
	defer root.Close()
	info, err = root.Stat(".")
	if err != nil || !trustedDirectory(info) {
		return command, fmt.Errorf("home has unsafe ownership or permissions")
	}
	local, err := hostDirectory(root, ".local")
	if err != nil {
		return command, err
	}
	defer local.Close()
	bin, err := hostDirectory(local, "bin")
	if err != nil {
		return command, err
	}
	defer bin.Close()
	// Symlink is exclusive, so a concurrent installer or owner command wins
	// without ever being overwritten. Only the same destination is reusable.
	if err = bin.Symlink(id.Launcher, id.Container); err == nil {
		return command, nil
	} else if !os.IsExist(err) {
		return command, err
	}
	if _, err := bin.Readlink(id.Container); err == nil {
		// Cleaning a path alone would incorrectly accept missing/../launcher
		// and paths whose intermediate symlinks redirect traversal elsewhere.
		if resolved, err := filepath.EvalSymlinks(command); err == nil && resolved == id.Launcher {
			return command, nil
		}
	}
	return command, fmt.Errorf("existing entry at %s conflicts with this launcher; preserved unchanged", command)
}

func trustedDirectory(info os.FileInfo) bool {
	if info == nil || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Geteuid())
}

func hostDirectory(parent *os.Root, name string) (*os.Root, error) {
	if err := parent.Mkdir(name, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	info, err := parent.Lstat(name)
	if err != nil || !trustedDirectory(info) {
		return nil, fmt.Errorf("host directory %s must be an owned directory without group/other write access or symlink redirection", name)
	}
	dir, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := dir.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		dir.Close()
		return nil, fmt.Errorf("host directory %s changed during installation", name)
	}
	return dir, nil
}

// OnPath requires an absolute PATH entry: relative entries do not expose a
// command reliably from other working directories. Directory aliases are valid.
func OnPath(command, pathEnv string) bool {
	bin, err := os.Stat(filepath.Dir(command))
	if err != nil {
		return false
	}
	for _, entry := range filepath.SplitList(pathEnv) {
		if !filepath.IsAbs(entry) {
			continue
		}
		info, err := os.Stat(entry)
		if err == nil && os.SameFile(bin, info) {
			return true
		}
	}
	return false
}

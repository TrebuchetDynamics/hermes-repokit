// Package target derives public identity and inspects filesystem collisions.
package target

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

type Identity struct{ Root, Name, Container, Project, Compose, Launcher string }

var separators = regexp.MustCompile(`[^a-z0-9]+`)

func Resolve(path string) (Identity, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Identity{}, err
	}
	root, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return Identity{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return Identity{}, err
	}
	if !info.IsDir() {
		return Identity{}, fmt.Errorf("target is not a directory")
	}
	name := strings.Trim(separators.ReplaceAllString(strings.ToLower(filepath.Base(root)), "-"), "-")
	// Bound the complete public name to 128 bytes. Refusal never changes identity.
	if name == "" || len("hermes-"+name) > 128 {
		return Identity{}, fmt.Errorf("unsupported repository basename")
	}
	for _, r := range root {
		if r < 32 || r == 127 {
			return Identity{}, fmt.Errorf("control characters in repository path are unsupported")
		}
	}
	sum := sha256.Sum256([]byte(root))
	container := "hermes-" + name
	return Identity{root, name, container, fmt.Sprintf("repokit-%x", sum[:12]), filepath.Join(root, ".hermes", "compose.yaml"), filepath.Join(root, ".hermes", "bin", container)}, nil
}

// Inspect reads metadata only. Docker and Git checks belong to the caller.
func Inspect(id Identity, pathEnv string) []string {
	var issues []string
	info, err := os.Lstat(id.Root)
	if err != nil {
		return []string{"cannot inspect target"}
	}
	if !owned(info) || info.Mode().Perm()&0022 != 0 {
		issues = append(issues, "target has unsafe ownership or permissions")
	}
	for _, name := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml", "compose.override.yaml", "compose.override.yml", "docker-compose.override.yaml", "docker-compose.override.yml"} {
		if _, err := os.Lstat(filepath.Join(id.Root, name)); err == nil {
			issues = append(issues, "root Compose collision: "+name)
		} else if !os.IsNotExist(err) {
			issues = append(issues, "cannot inspect root Compose: "+name)
		}
	}
	state := filepath.Join(id.Root, ".hermes")
	if _, err := os.Lstat(state); err == nil {
		count := 0
		err = filepath.WalkDir(state, func(p string, d fs.DirEntry, e error) error {
			count++
			if count > 100000 {
				return fmt.Errorf("native state inspection limit exceeded")
			}
			if e != nil {
				return e
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) || !owned(info) || info.Mode().Perm()&0022 != 0 || (p == state && info.Mode().Perm()&0077 != 0) {
				issues = append(issues, "unsafe native path: "+strings.TrimPrefix(p, id.Root+"/"))
			}
			return nil
		})
		if err != nil {
			issues = append(issues, "cannot fully inspect native state")
		}
	} else if !os.IsNotExist(err) {
		issues = append(issues, "cannot inspect native state")
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		p := filepath.Join(dir, id.Container)
		if _, err := os.Lstat(p); err == nil {
			resolved, e := filepath.EvalSymlinks(p)
			if e != nil || resolved != id.Launcher {
				issues = append(issues, "PATH collision: "+p)
			}
		} else if !os.IsNotExist(err) {
			issues = append(issues, "cannot inspect PATH entry")
		}
	}
	return issues
}
func owned(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Geteuid())
}

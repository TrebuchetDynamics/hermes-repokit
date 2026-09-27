// Package install publishes new private artifacts and preserves native authority.
package install

import (
	"bytes"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/locking"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Artifact struct {
	Data []byte
	Mode fs.FileMode
}

// Publish never merges an ambiguous partial tree or repairs from a receipt.
// prepare is limited to immutable image retrieval and must never write native
// state or spawn a native initializer. Future native subprocesses must hold
// an explicitly inherited installer lock across their whole lifetime.
func Publish(id target.Identity, files map[string]Artifact, prepare func() error) (bool, error) {
	for name, a := range files {
		if !fs.ValidPath(name) || name == "." || strings.Contains(name, "\\") || a.Mode.Perm()&0077 != 0 || !a.Mode.IsRegular() {
			return false, fmt.Errorf("unsafe generated artifact")
		}
	}
	for _, name := range []string{"compose.yaml", "config.yaml", "bin/" + id.Container} {
		if _, ok := files[name]; !ok {
			return false, fmt.Errorf("missing required generated artifact")
		}
	}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return false, fmt.Errorf("target preflight: %s", strings.Join(issues, "; "))
	}
	lock, e := locking.Acquire(filepath.Join(id.Root, ".hermes-repokit.lock"))
	if e != nil {
		return false, e
	}
	defer lock.Close()
	root, e := os.OpenRoot(id.Root)
	if e != nil {
		return false, e
	}
	defer root.Close()
	if _, e = root.Lstat(".hermes"); e == nil {
		for _, name := range []string{"compose.yaml", "config.yaml", "bin/" + id.Container} {
			info, err := root.Lstat(".hermes/" + name)
			if err != nil || !info.Mode().IsRegular() || (strings.HasPrefix(name, "bin/") && info.Mode().Perm()&0100 == 0) {
				return false, fmt.Errorf("required native file missing or unusable: %s", name)
			}
		}

		for _, name := range []string{"compose.yaml", "bin/" + id.Container} {
			b, e := root.ReadFile(".hermes/" + name)
			if e != nil || !bytes.Equal(b, files[name].Data) {
				return false, fmt.Errorf("existing native deployment differs or is incomplete; refusing automatic adoption")
			}
		}
		return false, nil
	} else if !os.IsNotExist(e) {
		return false, e
	}
	if prepare != nil {
		if e = prepare(); e != nil {
			return false, e
		}
	}
	stage, e := os.MkdirTemp(id.Root, ".hermes-stage-")
	if e != nil {
		return false, e
	}
	defer os.RemoveAll(stage)
	if e = os.Chmod(stage, 0700); e != nil {
		return false, e
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		a := files[name]
		p := filepath.Join(stage, filepath.FromSlash(name))
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return false, e
		}
		f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, a.Mode)
		if e != nil {
			return false, e
		}
		_, e = f.Write(a.Data)
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e != nil {
			return false, e
		}
		if closeErr != nil {
			return false, closeErr
		}
	}
	// Internal ignore rules also cover credentials created later by native setup.
	if _, ok := files[".gitignore"]; !ok {
		if e = os.WriteFile(filepath.Join(stage, ".gitignore"), []byte("*\n"), 0600); e != nil {
			return false, e
		}
	}
	// Flush generated files and directories before the atomic publication point.
	e = filepath.WalkDir(stage, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		return f.Sync()
	})
	if e != nil {
		return false, e
	}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return false, fmt.Errorf("target changed during preparation")
	}
	if _, e = root.Lstat(".hermes"); !os.IsNotExist(e) {
		return false, fmt.Errorf("native state appeared during preparation")
	}
	if e = publishDirectory(root, filepath.Base(stage)); e != nil {
		return false, e
	}
	dir, e := os.Open(id.Root)
	if e != nil {
		return true, e
	}
	defer dir.Close()
	if e = dir.Sync(); e != nil {
		return true, e
	}
	return true, nil
}

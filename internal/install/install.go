// Package install publishes new private artifacts and preserves native authority.
package install

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
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
	return publish(id, files, prepare, nil, nil)
}

// PublishChecked rechecks external read-only inventory while holding the same
// publication lock, including on no-op reruns. The check must not mutate state.
func PublishChecked(id target.Identity, files map[string]Artifact, check func() error) (bool, error) {
	return publish(id, files, nil, check, nil)
}

// StackUpgrade identifies an exact prior generated Compose document.
type StackUpgrade struct {
	Compose    []byte
	BackupName string
	// PreviousRecipe contains exact development-image filenames and bytes.
	PreviousRecipe map[string][]byte
}

// PublishStackChecked upgrades any recognized preimage while preserving native state.
func PublishStackChecked(id target.Identity, files map[string]Artifact, previous []StackUpgrade, check func() error) (bool, error) {
	return publish(id, files, nil, check, previous)
}

func publish(id target.Identity, files map[string]Artifact, prepare, check func() error, previous []StackUpgrade) (bool, error) {
	for _, prior := range previous {
		if prior.BackupName != "" && (!fs.ValidPath(prior.BackupName) || strings.ContainsAny(prior.BackupName, "/\\") || prior.BackupName == ".") {
			return false, fmt.Errorf("unsafe Compose backup name")
		}
	}
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
	initialRoot, e := os.Lstat(id.Root)
	if e != nil || !initialRoot.IsDir() {
		return false, fmt.Errorf("target is not an accessible directory")
	}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return false, fmt.Errorf("target preflight: %s", strings.Join(issues, "; "))
	}
	root, e := os.OpenRoot(id.Root)
	if e != nil {
		return false, e
	}
	defer root.Close()
	rootInfo, e := root.Stat(".")
	if e != nil || !os.SameFile(initialRoot, rootInfo) || !sameRootPath(id.Root, rootInfo) {
		return false, fmt.Errorf("target changed before installation")
	}
	lock, e := acquireRootLock(root)
	if e != nil {
		return false, e
	}
	defer lock.Close()
	lockInfo, e := lock.Stat()
	if e != nil || !sameLock(root, lockInfo) {
		return false, fmt.Errorf("installer lock changed")
	}
	if check != nil {
		if err := check(); err != nil {
			return false, err
		}
	}
	if _, e = root.Lstat(".hermes"); e == nil {
		// A reconfiguration is admitted only from an exact prior Compose with
		// the exact generated launcher, checked before any writes.
		b, readErr := root.ReadFile(".hermes/compose.yaml")
		if readErr == nil && !bytes.Equal(b, files["compose.yaml"].Data) {
			for _, prior := range previous {
				if len(prior.Compose) == 0 || !bytes.Equal(b, prior.Compose) {
					continue
				}
				if err := checkLauncher(root, ".hermes/bin/"+id.Container, files["bin/"+id.Container].Data); err != nil {
					return false, err
				}
				info, err := root.Lstat(".hermes/config.yaml")
				if err != nil || !info.Mode().IsRegular() {
					return false, fmt.Errorf("required native configuration missing or unusable")
				}
				return upgradeCompose(root, id, rootInfo, lockInfo, prior.Compose, files, prior.BackupName, prior.PreviousRecipe)
			}
		}
		for _, name := range []string{"compose.yaml", "config.yaml", "bin/" + id.Container} {
			info, err := root.Lstat(".hermes/" + name)
			if err != nil || !info.Mode().IsRegular() || (strings.HasPrefix(name, "bin/") && info.Mode().Perm()&0100 == 0) {
				return false, fmt.Errorf("required native file missing or unusable: %s", name)
			}
		}

		for _, name := range []string{"bin/" + id.Container, "compose.yaml"} {
			b, e := root.ReadFile(".hermes/" + name)
			if e != nil || !bytes.Equal(b, files[name].Data) {
				return false, fmt.Errorf("existing native deployment differs or is incomplete; refusing automatic adoption")
			}
		}
		if err := checkDevelopmentArtifacts(root, ".hermes/", files); err != nil {
			return false, err
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
	if !sameRootPath(id.Root, rootInfo) || !sameLock(root, lockInfo) {
		return false, fmt.Errorf("target changed during preparation")
	}
	stage := ".hermes-stage-" + rand.Text()
	e = root.Mkdir(stage, 0700)
	if e != nil {
		return false, e
	}
	createdStage, e := root.Lstat(stage)
	if e != nil || !createdStage.IsDir() {
		return false, fmt.Errorf("installer stage changed")
	}
	stageRoot, e := root.OpenRoot(stage)
	if e != nil {
		if sameStage(root, stage, createdStage) {
			root.RemoveAll(stage)
		}
		return false, e
	}
	stageInfo, e := stageRoot.Stat(".")
	if e != nil || !os.SameFile(createdStage, stageInfo) || !sameStage(root, stage, stageInfo) {
		stageRoot.Close()
		return false, fmt.Errorf("installer stage changed")
	}
	defer func() {
		stageRoot.Close()
		if sameStage(root, stage, stageInfo) {
			root.RemoveAll(stage)
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		a := files[name]
		p := name
		if e = stageRoot.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return false, e
		}
		f, e := stageRoot.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, a.Mode)
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
		ignore, err := stageRoot.OpenFile(".gitignore", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return false, err
		}
		_, e = ignore.Write([]byte("*\n"))
		closeErr := ignore.Close()
		if e != nil {
			return false, e
		}
		if closeErr != nil {
			return false, closeErr
		}
	}
	// Flush generated files and directories before the atomic publication point.
	e = fs.WalkDir(stageRoot.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		f, err := stageRoot.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		return f.Sync()
	})
	if e != nil {
		return false, e
	}
	if !sameRootPath(id.Root, rootInfo) || !sameLock(root, lockInfo) || !sameStage(root, stage, stageInfo) {
		return false, fmt.Errorf("target changed during preparation")
	}
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return false, fmt.Errorf("target changed during preparation")
	}
	if !sameRootPath(id.Root, rootInfo) || !sameLock(root, lockInfo) || !sameStage(root, stage, stageInfo) {
		return false, fmt.Errorf("target changed during preparation")
	}
	if _, e = root.Lstat(".hermes"); !os.IsNotExist(e) {
		return false, fmt.Errorf("native state appeared during preparation")
	}
	if e = publishDirectory(root, stage); e != nil {
		return false, e
	}
	dir, e := root.Open(".")
	if e != nil {
		return true, e
	}
	defer dir.Close()
	if e = dir.Sync(); e != nil {
		return true, e
	}
	return true, nil
}

func sameRootPath(path string, opened fs.FileInfo) bool {
	current, err := os.Lstat(path)
	return err == nil && current.IsDir() && os.SameFile(current, opened)
}

func sameLock(root *os.Root, held fs.FileInfo) bool {
	current, err := root.Lstat(".hermes-repokit.lock")
	return err == nil && os.SameFile(current, held)
}

func sameStage(root *os.Root, name string, opened fs.FileInfo) bool {
	current, err := root.Lstat(name)
	return err == nil && current.IsDir() && os.SameFile(current, opened)
}

// The installer lock must be opened through the same directory descriptor as
// staging and publication. Opening its absolute path can create it in a
// replacement directory before a later identity check catches the swap.
func acquireRootLock(root *os.Root) (*os.File, error) {
	lock, err := root.OpenFile(".hermes-repokit.lock", os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	info, err := lock.Stat()
	if err != nil {
		lock.Close()
		return nil, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || st.Nlink != 1 {
		lock.Close()
		return nil, fmt.Errorf("unsafe installer lock")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("installer is already running: %w", err)
	}
	return lock, nil
}

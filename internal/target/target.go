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
var gatewayTickSocket = regexp.MustCompile(`^state/gateway\.loop-tick\.[1-9][0-9]*\.sock$`)
var codeKernelSocket = regexp.MustCompile(`^(profiles/[^/]+/)?cache/scratch/hermes_rpc_[0-9a-f]{32}\.sock$`)
var huggingFaceModelLink = regexp.MustCompile(`^\.cache/huggingface/hub/models--[A-Za-z0-9][A-Za-z0-9._-]*/(blobs/[0-9a-f]{40}([0-9a-f]{24})?|snapshots/[0-9a-f]{40}/[^/]+(/[^/]+)*)$`)
var huggingFaceCacheMetadata = regexp.MustCompile(`^\.cache/huggingface/hub/(\.locks/models--[A-Za-z0-9][A-Za-z0-9._-]*/[0-9a-f]{40}([0-9a-f]{24})?\.lock|blobs/[0-9a-f]{2}/[0-9a-f]{64}\.(lock|refs))$`)

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
	container := name
	if !strings.HasPrefix(container, "hermes-") {
		container = "hermes-" + container
	}
	// Bound the complete public name to 128 bytes. Refusal never changes identity.
	if name == "" || len(container) > 128 {
		return Identity{}, fmt.Errorf("unsupported repository basename")
	}
	for _, r := range root {
		if r < 32 || r == 127 {
			return Identity{}, fmt.Errorf("control characters in repository path are unsupported")
		}
	}
	sum := sha256.Sum256([]byte(root))
	return Identity{root, name, container, fmt.Sprintf("repokit-%x", sum[:12]), filepath.Join(root, ".hermes", "compose.yaml"), filepath.Join(root, ".hermes", "bin", container)}, nil
}

// PreviousNames retains the stable repository identity while reconstructing the
// original public names for exact generated-deployment migration only.
func PreviousNames(id Identity) Identity {
	id.Container = "hermes-" + id.Name
	id.Launcher = filepath.Join(id.Root, ".hermes", "bin", id.Container)
	return id
}

// broadMountRoots are host paths whose recursive mount or SELinux relabel
// would touch unrelated system or user data rather than one repository.
var broadMountRoots = map[string]bool{"/": true, "/home": true, "/root": true, "/usr": true, "/etc": true, "/var": true, "/opt": true, "/boot": true, "/srv": true}

// mountRootSafe refuses to publish or relabel a bind source that is not a
// narrowly scoped repository path.
func mountRootSafe(root string) error {
	clean := filepath.Clean(root)
	if broadMountRoots[clean] {
		return fmt.Errorf("refusing to mount or relabel broad host path %s", clean)
	}
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(home) == clean {
		return fmt.Errorf("refusing to mount or relabel the user home directory")
	}
	return nil
}

// Inspect reads metadata only. Docker and Git checks belong to the caller.
func Inspect(id Identity, pathEnv string) []string {
	var issues []string
	if err := mountRootSafe(id.Root); err != nil {
		issues = append(issues, err.Error())
	}
	info, err := os.Lstat(id.Root)
	if err != nil {
		return []string{"cannot inspect target"}
	}
	if !owned(info) || info.Mode().Perm()&0022 != 0 {
		issues = append(issues, "target has unsafe ownership or permissions")
	}
	// Owner Compose files belong to the repository. RepoKit always selects its
	// private .hermes/compose.yaml and explicit project; never inspect or adopt
	// root Compose/override files, including malformed files and symlinks.
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
			rel, e := filepath.Rel(state, p)
			if e != nil {
				return e
			}
			if !safeNativeEntry(filepath.ToSlash(rel), id.Container, info) {
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

// Native tool managers use symlinks with container-only targets and writable
// cache locks. Inspect metadata without following these links. Their roots and
// all ancestor directories remain subject to the ordinary ownership/mode rules.
func safeNativeEntry(rel, launcher string, info fs.FileInfo) bool {
	if !owned(info) {
		return false
	}
	if rel == "." {
		return info.IsDir() && info.Mode().Perm()&0077 == 0
	}
	under := func(root string) bool { return strings.HasPrefix(rel, root+"/") }
	uv := under(".cache/uv") || under(".local/share/uv/tools") || under("home/.cache/uv")
	toolLink := uv || under(".local/bin") || under(".cua-driver/packages") ||
		(filepath.Dir(rel) == "bin" && filepath.Base(rel) != launcher)
	if info.Mode()&os.ModeSymlink != 0 {
		// Native model snapshots and per-repository blob entries are pointers;
		// inspect their metadata without following container-only cache targets.
		return toolLink || huggingFaceModelLink.MatchString(rel)
	}
	if info.IsDir() {
		return info.Mode().Perm()&0022 == 0
	}
	if info.Mode()&os.ModeSocket != 0 {
		// Native execute_code kernels bind private RPC sockets in profile scratch.
		// Preserve them during live inspection; never connect, chmod or delete.
		if codeKernelSocket.MatchString(rel) {
			return info.Mode().Perm() == 0600
		}
		return info.Mode().Perm()&0022 == 0 && (rel == "gateway.sock" || gatewayTickSocket.MatchString(rel))
	}
	if !info.Mode().IsRegular() {
		return false
	}
	// Hugging Face creates shared blob locks/manifests with mode 0666 and
	// repository download locks with mode 0664. Model/code files keep ordinary
	// mode checks, as do every cache directory and ancestor above these entries.
	hfMetadata := huggingFaceCacheMetadata.MatchString(rel) && info.Mode().Perm()&0111 == 0
	return info.Mode().Perm()&0022 == 0 || (uv && strings.HasSuffix(rel, ".lock")) || hfMetadata || rel == "lazy-packages/.lock"
}

func owned(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Geteuid())
}

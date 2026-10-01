// Package target derives public identity and inspects filesystem collisions.
package target

import (
	"bytes"
	"crypto/sha256"
	"errors"
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

// browserHarnessSocket is the browser tool's private control socket in a
// profile home; granted browser tools create it during ordinary work.
// profileHome matches a path inside default's home/ or a specialist's
// profiles/<p>/home/: tools write the same caches there as in the state root.
var profileHome = regexp.MustCompile(`^(?:profiles/[^/]+/)?home/(.+)$`)

// workerScratch is a worker's disposable scratch space. Workers copy whole
// repositories there, repository symlinks included; RepoKit never reads or
// writes it, so a link inside cannot redirect anything RepoKit manages. Each
// entry is checked itself, but its contents are not walked: repository copies
// would otherwise exhaust the inspection limit.
var workerScratch = regexp.MustCompile(`^(profiles/[^/]+/)?cache/scratch/.+`)
var workerScratchEntry = regexp.MustCompile(`^(profiles/[^/]+/)?cache/scratch/[^/]+$`)

// dartPerfSocket is the Dart analysis server's private performance socket in
// a home's state directory (named by process ID).
var dartPerfSocket = regexp.MustCompile(`^\.local/state/Dart/perf/[0-9]+$`)
var browserHarnessSocket = regexp.MustCompile(`^(profiles/[^/]+/)?home/\.config/browser-harness/runtime/[^/]+\.sock$`)
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
	if !owned(info) {
		issues = append(issues, "target is not owned by the current user")
	} else if reason := WritableByOthers(info); reason != "" {
		issues = append(issues, "target is "+reason+" "+id.Root)
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
			info, e := walkedInfo(p, state, d, e)
			if e != nil || info == nil {
				return e
			}
			rel, e := filepath.Rel(state, p)
			if e != nil {
				return e
			}
			if !safeNativeEntry(filepath.ToSlash(rel), id.Container, info) && !containedToolLink(filepath.ToSlash(rel), p, id.Container, info) {
				issues = append(issues, "unsafe native path: "+strings.TrimPrefix(p, id.Root+"/"))
			}
			if d.IsDir() && workerScratchEntry.MatchString(filepath.ToSlash(rel)) {
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			issues = append(issues, "cannot fully inspect native state")
		}
	} else if !os.IsNotExist(err) {
		issues = append(issues, "cannot inspect native state")
	}
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		// PATH often repeats a directory; one collision is one finding.
		if seen[filepath.Clean(dir)] {
			continue
		}
		seen[filepath.Clean(dir)] = true
		p := filepath.Join(dir, id.Container)
		if _, err := os.Lstat(p); err == nil {
			resolved, e := filepath.EvalSymlinks(p)
			// A repository named repokit shares its host command name with
			// the RepoKit bootstrap. That entry is preserved and reported by
			// install, like install.sh preserves a launcher; it never blocks.
			// This repository's own host link is dangling while .hermes is
			// absent (an interrupted or removed deployment); install recreates
			// the launcher it names and then reuses the link.
			own, _ := os.Readlink(p)
			if (e != nil || resolved != id.Launcher && !RepoKitBootstrap(resolved)) && own != id.Launcher {
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
// walkedInfo returns an entry's metadata during the native-state walk. A
// running Hermes creates and deletes short-lived files (temporary config
// writes, sockets, locks), so an entry listed by its directory can be gone
// before it is examined; nothing that no longer exists can be unsafe, so it is
// skipped (nil info, nil error). The state root itself must still exist, and
// every other error still fails the inspection.
func walkedInfo(p, state string, d fs.DirEntry, walkErr error) (fs.FileInfo, error) {
	if walkErr == nil {
		var info fs.FileInfo
		info, walkErr = d.Info()
		if walkErr == nil {
			return info, nil
		}
	}
	if p != state && errors.Is(walkErr, fs.ErrNotExist) {
		if d != nil && d.IsDir() {
			return nil, filepath.SkipDir
		}
		return nil, nil
	}
	return nil, walkErr
}

func safeNativeEntry(rel, launcher string, info fs.FileInfo) bool {
	if !owned(info) {
		return false
	}
	if rel == "." {
		return info.IsDir() && info.Mode().Perm()&0077 == 0
	}
	under := func(root string) bool { return strings.HasPrefix(rel, root+"/") }
	// Tool caches sit in the state root (default's HERMES_HOME) and in each
	// profile's home; only entries beneath a cache directory qualify, never
	// the directory itself.
	homeRel := ""
	if m := profileHome.FindStringSubmatch(rel); m != nil {
		homeRel = m[1]
	}
	cache := func(root string) bool { return under(root) || strings.HasPrefix(homeRel, root+"/") }
	uv := cache(".cache/uv") || cache(".local/share/uv/tools")
	// npm and npx link package binaries inside their own cache (node_modules/.bin).
	npm := cache(".npm")
	toolLink := uv || npm || cache(".local/bin") || under(".cua-driver/packages") ||
		(filepath.Dir(rel) == "bin" && filepath.Base(rel) != launcher)
	if info.Mode()&os.ModeSymlink != 0 {
		// Native model snapshots and per-repository blob entries are pointers;
		// inspect their metadata without following container-only cache targets.
		return toolLink || huggingFaceModelLink.MatchString(rel) || workerScratch.MatchString(rel)
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
		if browserHarnessSocket.MatchString(rel) {
			return info.Mode().Perm()&0077 == 0
		}
		if dartPerfSocket.MatchString(rel) || dartPerfSocket.MatchString(homeRel) {
			return info.Mode().Perm()&0022 == 0
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

// bootstrapUsage is RepoKit's self-identification: install.sh and
// RepoKitBootstrap recognize a RepoKit-built binary by finding this text in it.
// This constant is what embeds it in every build (usage output names the
// invoked command instead), so it must keep this exact text; the install-script
// acceptance tests fail if it is lost.
const bootstrapUsage = "usage: hermes-repokit <plan|install|setup|verify"

// RepoKitBootstrap reports whether path is a regular file holding a
// RepoKit-built bootstrap binary.
func RepoKitBootstrap(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 256<<20 {
		return false
	}
	data, err := os.ReadFile(path)
	return err == nil && bytes.Contains(data, []byte(bootstrapUsage))
}

// managedNames are files RepoKit itself writes or reads inside .hermes; a
// symlink there (or at the launcher or recipe) could redirect RepoKit, so it
// is never accepted.
var managedNames = map[string]bool{"SOUL.md": true, "config.yaml": true, "profile.yaml": true, "compose.yaml": true, "kanban.db": true, ".env": true, "auth.json": true}

// containedToolLink accepts a symlink that tools create during ordinary work
// (language servers, package managers) when it cannot redirect RepoKit: it is
// owned by the user, is not a managed file, a top-level entry or a profile
// directory itself, and points back inside the state, either relatively or
// through the container's /opt/data view of it.
func containedToolLink(rel, path, launcher string, info fs.FileInfo) bool {
	if info.Mode()&os.ModeSymlink == 0 || !owned(info) {
		return false
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 2 || parts[0] == "profiles" && len(parts) == 2 || managedNames[parts[len(parts)-1]] {
		return false
	}
	// RepoKit writes the generated launcher and the development recipe.
	if rel == "bin/"+launcher || parts[0] == "development-image" {
		return false
	}
	target, err := os.Readlink(path)
	if err != nil || target == "" {
		return false
	}
	if strings.HasPrefix(target, "/") {
		clean := filepath.Clean(target)
		return clean != "/opt/data" && strings.HasPrefix(clean, "/opt/data/")
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(rel), target))
	return resolved != ".." && !strings.HasPrefix(resolved, "../")
}

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

const lockExcludeEntry = "/.hermes-repokit.lock"

// excludeInstallLock keeps RepoKit's root lock out of `git status` through the
// repository's local, never-committed info/exclude. The lock must stay at the
// root: it guards creation of .hermes and is shared with the container flock.
// The owner's tracked .gitignore is never edited.
func (a App) excludeInstallLock(ctx context.Context, id target.Identity) error {
	if a.Runner.Run(ctx, "git", "-C", id.Root, "check-ignore", "--quiet", "--", ".hermes-repokit.lock").Err == nil {
		return nil // already ignored by some rule
	}
	out := a.Runner.Run(ctx, "git", "-C", id.Root, "rev-parse", "--git-path", "info/exclude")
	path := strings.TrimSpace(out.Output)
	if out.Err != nil || out.Truncated || path == "" || strings.ContainsAny(path, "\n\x00") {
		return fmt.Errorf("git exclude file unavailable")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(id.Root, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|syscall.O_NOFOLLOW, 0644)
	if err != nil {
		return fmt.Errorf("git exclude file unsafe or unwritable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || st.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("git exclude file is not a regular file owned by you")
	}
	prefix := ""
	if info.Size() > 0 {
		prefix = "\n"
	}
	if _, err := fmt.Fprintf(f, "%s# Hermes RepoKit installer lock (local only)\n%s\n", prefix, lockExcludeEntry); err != nil {
		return err
	}
	if a.Runner.Run(ctx, "git", "-C", id.Root, "check-ignore", "--quiet", "--", ".hermes-repokit.lock").Err != nil {
		return fmt.Errorf("exclude entry written but Git does not ignore the lock")
	}
	return nil
}

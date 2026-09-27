package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// gitIssues inspects index and effective ignore rules without printing filenames
// from private state. Both install admission and verify enforce these checks.
func (a App) gitIssues(ctx context.Context, id target.Identity) []string {
	var issues []string
	git := a.Runner.Run(ctx, "git", "-C", id.Root, "rev-parse", "--show-toplevel")
	if git.Err != nil || git.Truncated || strings.TrimSpace(git.Output) != id.Root {
		issues = append(issues, "target must be the canonical Git repository root")
	}
	tracked := a.Runner.Run(ctx, "git", "-C", id.Root, "ls-files", "-z", "--", ".hermes")
	if tracked.Err != nil || tracked.Truncated {
		issues = append(issues, "cannot inspect tracked private state")
	} else if tracked.Output != "" {
		issues = append(issues, "private .hermes state is tracked by Git")
	}

	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err == nil {
		visible := a.Runner.Run(ctx, "git", "-C", id.Root, "ls-files", "--others", "--exclude-standard", "-z", "--", ".hermes")
		if visible.Err != nil || visible.Truncated || visible.Output != "" {
			issues = append(issues, "private .hermes state is not fully ignored by Git")
		}
		// Include a future credential path; a current empty directory alone does not
		// establish that later native setup files are excluded.
		for _, path := range []string{".hermes/.repokit-ignore-probe", ".hermes/.env"} {
			ignored := a.Runner.Run(ctx, "git", "-C", id.Root, "check-ignore", "--quiet", "--no-index", "--", path)
			if ignored.Err != nil || ignored.Truncated {
				issues = append(issues, "private .hermes ignore protection cannot be verified")
				break
			}
		}
	}
	return issues
}

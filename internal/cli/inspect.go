package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// gitIssues inspects index and effective ignore rules. It names only paths Git
// already tracks, never untracked private state. Both install admission and
// verify enforce these checks.
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
		issues = append(issues, trackedStateIssue(strings.Split(strings.TrimSuffix(tracked.Output, "\x00"), "\x00")))
	}

	if _, err := os.Lstat(filepath.Join(id.Root, ".hermes")); err == nil {
		visible := a.Runner.Run(ctx, "git", "-C", id.Root, "ls-files", "--others", "--exclude-standard", "-z", "--", ".hermes")
		if visible.Err != nil || visible.Truncated || visible.Output != "" {
			issues = append(issues, "private .hermes state is not fully ignored by Git")
		}
		// Include future native API-key and OAuth stores; a current empty directory
		// alone does not establish that later setup files are excluded.
		for _, path := range []string{".hermes/.repokit-ignore-probe", ".hermes/.env", ".hermes/auth.json"} {
			ignored := a.Runner.Run(ctx, "git", "-C", id.Root, "check-ignore", "--quiet", "--no-index", "--", path)
			if ignored.Err != nil || ignored.Truncated {
				issues = append(issues, "private .hermes ignore protection cannot be verified")
				break
			}
		}
	}
	return issues
}

// nativeRuntimeReady checks the exact runtime before any native mutation or
// private wizard. Missing/stopped metadata is pending; a foreign runtime refuses.
func (a App) nativeRuntimeReady(id target.Identity, dockerContext string) (bool, error) {
	const format = `{"status":{{json .State.Status}},"image":{{json .Config.Image}},"imageID":{{json .Image}},"mounts":{{json .Mounts}},"project":{{json (index .Config.Labels "com.docker.compose.project")}},"service":{{json (index .Config.Labels "com.docker.compose.service")}},"workspace":{{range .Mounts}}{{if eq .Destination "/workspace"}}{{json .Source}}{{end}}{{end}},"home":{{range .Mounts}}{{if eq .Destination "/opt/data"}}{{json .Source}}{{end}}{{end}},"unexpectedMounts":"{{range .Mounts}}{{if or (ne .Type "bind") (and (ne .Destination "/workspace") (ne .Destination "/opt/data"))}}x{{end}}{{end}}"}`
	observed := a.Runner.Run(context.Background(), "docker", "--context", dockerContext, "container", "inspect", "--format", format, id.Container)
	var runtime struct {
		verify.Runtime
		Image, ImageID, Service, UnexpectedMounts string
		Mounts                                    []verify.RuntimeMount
	}
	if observed.Err != nil || observed.Truncated || json.Unmarshal([]byte(observed.Output), &runtime) != nil || runtime.Status != "running" {
		return false, nil
	}
	same := runtime.Service == "hermes" && runtime.Project == id.Project && runtime.Workspace == id.Root && runtime.Home == filepath.Join(id.Root, ".hermes")
	// The deployment was reconfigured on disk (a changed recipe, or newly
	// selected Go or Docker tests); install recreates the container.
	if same && olderGeneratedImage(id, runtime.Image) && verify.PriorRuntimeMountsMatch(id, runtime.UnexpectedMounts, runtime.Mounts, true) {
		return false, nil
	}
	if same && !verify.RuntimeMountsMatch(id, runtime.UnexpectedMounts, runtime.Mounts) && verify.PriorRuntimeMountsMatch(id, runtime.UnexpectedMounts, runtime.Mounts, false) && verify.HermesImageMatches(context.Background(), id, dockerContext, runtime.Image, runtime.ImageID, a.Runner) {
		return false, nil
	}
	if !verify.HermesImageMatches(context.Background(), id, dockerContext, runtime.Image, runtime.ImageID, a.Runner) || runtime.Service != "hermes" || !verify.RuntimeMountsMatch(id, runtime.UnexpectedMounts, runtime.Mounts) || runtime.Project != id.Project || runtime.Workspace != id.Root || runtime.Home != filepath.Join(id.Root, ".hermes") {
		return false, fmt.Errorf("running container image or identity does not match the qualified deployment")
	}
	return true, nil
}

var generatedTag = regexp.MustCompile(`^[0-9a-f]{24}$`)

// olderGeneratedImage recognizes RepoKit's own image names for this deployment
// whose recipe tag differs from the current recipe.
func olderGeneratedImage(id target.Identity, image string) bool {
	selected, ok := compose.DevelopmentSelected(id)
	if !ok {
		return false
	}
	tag, found := strings.CutPrefix(image, "repokit/"+id.Container+":")
	// The current recipe tag keeps its separate image-ID checks.
	return found && generatedTag.MatchString(tag) && tag != development.Fingerprint(*selected.Development)[:24]
}

// trackedStateIssue names committed .hermes paths, which Git already exposes,
// and the remedy. Control characters are dropped so a path cannot rewrite the
// terminal.
func trackedStateIssue(paths []string) string {
	shown := []string{}
	for _, p := range paths {
		if len(shown) == 3 {
			break
		}
		shown = append(shown, strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return -1
			}
			return r
		}, p))
	}
	more := ""
	if len(paths) > len(shown) {
		more = fmt.Sprintf(" and %d more", len(paths)-len(shown))
	}
	return fmt.Sprintf("private .hermes state is tracked by Git (%s%s); move anything you need outside .hermes, then untrack it with `git rm -r --cached .hermes` (files stay on disk) and commit", strings.Join(shown, ", "), more)
}

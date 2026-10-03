package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
)

// ui prints aligned status lines in the same style as install.sh: color only
// on an interactive terminal, never when piped or with NO_COLOR / TERM=dumb.
type ui struct {
	out, err                                         io.Writer
	reset, bold, dim, red, green, yellow, blue, cyan string
	// live: out is an interactive terminal, so progress can redraw in place.
	live bool
}

const uiLabel = 14

func newUI(out, err io.Writer) ui {
	u := ui{out: out, err: err}
	if f, tty := out.(*os.File); tty && native.InteractiveInput(f) && os.Getenv("TERM") != "dumb" {
		u.live = true
	}
	if u.live && os.Getenv("NO_COLOR") == "" {
		u.reset, u.bold, u.dim = "\033[0m", "\033[1m", "\033[2m"
		u.red, u.green, u.yellow, u.blue, u.cyan = "\033[31m", "\033[32m", "\033[33m", "\033[34m", "\033[36m"
	}
	return u
}

func (u ui) title(command, name, root string) {
	fmt.Fprintf(u.out, "%sRepoKit %s%s · %s  %s%s%s\n\n", u.bold+u.cyan, command, u.reset, name, u.dim, tildePath(root), u.reset)
}

func (u ui) line(mark, color, label, detail string) {
	fmt.Fprintf(u.out, "  %s%s%s %-*s %s\n", color, mark, u.reset, uiLabel, label, detail)
}

func (u ui) ok(label, detail string)      { u.line("✔", u.green, label, detail) }
func (u ui) working(label, detail string) { u.line("▸", u.blue, label, detail) }
func (u ui) pending(label, detail string) { u.line("•", u.yellow, label, detail) }
func (u ui) note(detail string) {
	fmt.Fprintf(u.out, "  %s  %-*s %s%s\n", u.dim, uiLabel, "", detail, u.reset)
}

// warn goes to stderr, like install.sh.
func (u ui) warn(format string, args ...any) {
	fmt.Fprintf(u.err, "  %s!%s %s\n", u.yellow, u.reset, fmt.Sprintf(format, args...))
}

func (u ui) fail(format string, args ...any) {
	fmt.Fprintf(u.err, "  %s✗%s %s\n", u.red, u.reset, fmt.Sprintf(format, args...))
}

// next prints the follow-up commands, command column aligned.
func (u ui) next(steps ...[2]string) {
	if len(steps) == 0 {
		return
	}
	fmt.Fprintf(u.out, "\n%sNext%s\n", u.bold, u.reset)
	width := 0
	for _, s := range steps {
		width = max(width, len(s[0]))
	}
	for _, s := range steps {
		fmt.Fprintf(u.out, "  %s%-*s%s  %s\n", u.bold, width, s[0], u.reset, s[1])
	}
}

// headline prints a standalone sentence, with optional plain lines under it,
// between status lines and next steps.
func (u ui) headline(text string, lines ...string) {
	fmt.Fprintf(u.out, "\n%s%s%s\n", u.bold, text, u.reset)
	for _, line := range lines {
		fmt.Fprintf(u.out, "%s\n", line)
	}
}

// roleLines maps a native convergence state to its status line.
var roleLines = map[string]struct {
	pending bool
	detail  string
}{
	"current":      {false, "current"},
	"missing":      {false, "created"},
	"adopt":        {false, "created from the stock default profile"},
	"customized":   {false, "owner-customized; preserved as is"},
	"reset":        {false, "reset to RepoKit's baseline"},
	"upgrade":      {false, "upgraded to the current RepoKit SOUL"},
	"deferred":     {true, "SOUL upgrade waits for the running card; rerun install when the board is idle"},
	"retired":      {false, "retired: exported to .hermes/backups and removed; its open cards now belong to default"},
	"retire-later": {true, "retirement waits for an idle board; rerun install when no card is running"},
}

// team prints one line per profile from the applied plan.
func (u ui) team(report native.TeamReport) {
	defer u.missingSkills(report.MissingSkills)
	if len(report.Roles) == 0 {
		u.ok("Team", "default reconciled")
		if len(report.Customized) > 0 {
			u.note("owner-customized, preserved as is: " + strings.Join(report.Customized, ", "))
		}
		return
	}
	for _, role := range report.Roles {
		line, known := roleLines[role.State]
		switch {
		case !known:
			u.pending(role.Profile, role.State)
		case line.pending:
			u.pending(role.Profile, line.detail)
		default:
			u.ok(role.Profile, line.detail)
		}
	}
}

// ready closes a successful run: the team is set up and the owner's next
// step is simply to talk to it.
func (u ui) ready(command string) {
	fmt.Fprintf(u.out, "\n%sRepoKit ready.%s\n", u.bold+u.green, u.reset)
	u.next([2]string{command, "open the project team"})
}

// tildePath shortens paths under the user's home for display only.
func tildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || home == "/" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		if rel == "." {
			return "~"
		}
		return "~/" + rel
	}
	return path
}

// missingSkills reports granted skills that did not install. The team works
// without them; they are retried by resetting the profile or installed natively.
func (u ui) missingSkills(missing []string) {
	if len(missing) == 0 {
		return
	}
	u.warn("granted skills not installed (offline, or blocked by Hermes's security scan): %s", strings.Join(missing, ", "))
	u.note("the team works without them; retry with " + self() + " install --reset-profile <profile>, or install one natively with -p <profile> skills install <identifier> --yes")
}

// progress renders the team script's REPOKIT_PROGRESS=<step>/<total> <what>
// lines: one bar redrawn in place on a terminal, one line per step otherwise.
// done clears a live bar before the step's result is printed.
func (u ui) progress(label string) (onLine func(string), done func()) {
	var mu sync.Mutex
	drawn := false
	onLine = func(line string) {
		rest, ok := strings.CutPrefix(line, native.ProgressLine)
		if !ok {
			return
		}
		count, what, _ := strings.Cut(rest, " ")
		var step, total int
		if _, err := fmt.Sscanf(count, "%d/%d", &step, &total); err != nil || total <= 0 || step < 0 || step > total {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if !u.live {
			u.working(label, fmt.Sprintf("%d/%d %s", step, total, what))
			return
		}
		const width = 20
		bar := strings.Repeat("█", step*width/total) + strings.Repeat("░", width-step*width/total)
		fmt.Fprintf(u.out, "\r\033[K  %s▸%s %-*s %s %d/%d %s", u.blue, u.reset, uiLabel, label, bar, step, total, what)
		drawn = true
	}
	done = func() {
		mu.Lock()
		defer mu.Unlock()
		if drawn {
			fmt.Fprint(u.out, "\r\033[K")
		}
	}
	return onLine, done
}

package cli

import (
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// githubLogin hands the owner's terminal to gh's own sign-in inside the
// deployment, so agents can push to GitHub over HTTPS. Like Hermes's private
// setup, RepoKit never sees the credential; gh keeps it in .hermes.
func (a App) githubLogin(id target.Identity, stdout, stderr io.Writer) int {
	u := newUI(stdout, stderr)
	u.title("github-login", id.Name, id.Root)
	dc, err := launcher.Context(id)
	if err != nil {
		u.fail("github-login refused: this deployment's launcher cannot be verified; run %s install", self())
		return 1
	}
	if !a.interactive() {
		u.fail("github-login needs your terminal: run %s github-login there", self())
		return 1
	}
	if ready, err := a.nativeRuntimeReady(id, dc); err != nil || !ready {
		u.fail("github-login needs the current deployment running; run %s install first", self())
		return 1
	}
	u.working("GitHub", "gh's own sign-in, in your terminal; RepoKit never sees the token")
	u.note("for least privilege, choose \"Paste an authentication token\" and use a fine-grained token limited to this repository")
	login := a.GitHubLogin
	if login == nil {
		login = native.GitHubLogin
	}
	if code := login(id.Compose, dc, a.Stdin, stdout, stderr); code != 0 {
		u.fail("GitHub sign-in did not finish; nothing changed. Run %s github-login again", self())
		return code
	}
	u.ok("GitHub", "signed in; agents in this deployment can now push over HTTPS")
	fmt.Fprintln(stdout)
	return 0
}

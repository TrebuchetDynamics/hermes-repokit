package cli

import (
	"io"
	"strings"
	"testing"
)

// github-login hands the owner's terminal to gh inside the running deployment
// and refuses without a terminal, so RepoKit never prompts for or sees a token.
func TestGitHubLoginRunsGhInTheOwnersTerminal(t *testing.T) {
	a, r := foundationApp(t)
	if code, _, diag := invoke(t, a, "install"); code != 0 {
		t.Fatal(diag)
	}
	r.runtime = developmentRuntimeFixture(r.id)
	var calls []string
	a.GitHubLogin = func(compose, dc string, _ io.Reader, _, _ io.Writer) int {
		calls = append(calls, compose+" "+dc)
		return 0
	}

	a.Interactive = func() bool { return false }
	if code, out, diag := invoke(t, a, "github-login"); code == 0 || len(calls) != 0 || !strings.Contains(out+diag, "needs your terminal") {
		t.Fatalf("github-login without a terminal: code=%d calls=%v\n%s%s", code, calls, out, diag)
	}

	a.Interactive = func() bool { return true }
	code, out, diag := invoke(t, a, "github-login")
	if code != 0 || len(calls) != 1 || !strings.HasPrefix(calls[0], r.id.Compose+" ") || !strings.Contains(out, "agents in this deployment can now push") {
		t.Fatalf("github-login: code=%d calls=%v\n%s%s", code, calls, out, diag)
	}

	a.GitHubLogin = func(string, string, io.Reader, io.Writer, io.Writer) int { return 1 }
	if code, out, diag := invoke(t, a, "github-login"); code == 0 || !strings.Contains(out+diag, "did not finish") {
		t.Fatalf("failed sign-in reported success: code=%d\n%s%s", code, out, diag)
	}
}

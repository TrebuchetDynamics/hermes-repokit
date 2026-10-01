package native

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// sendCLI answers `hermes send --list` and records each send.
type sendCLI struct{ sends []string }

func (s *sendCLI) RunInput(_ context.Context, _ io.Reader, _ string, args ...string) process.Result {
	call := strings.Join(args, " ")
	switch {
	case strings.HasSuffix(call, "send --list"):
		return process.Result{Output: "Available messaging targets:\n\nTelegram:\n  telegram:Owner (dm)\n\nDiscord:\n  discord:#ops\n\nUse these as the \"target\" parameter when sending.\n"}
	case strings.Contains(call, " send --to discord "):
		return process.Result{Err: errors.New("no home channel")}
	case strings.Contains(call, " send --to "):
		s.sends = append(s.sends, call[strings.Index(call, " send --to ")+11:])
		return process.Result{}
	}
	return process.Result{Err: errors.New("unexpected " + call)}
}

func TestNotifyChatsPostsToEachHomeChannel(t *testing.T) {
	r := &sendCLI{}
	sent := NotifyChats(context.Background(), target.Identity{Compose: "/x/.hermes/compose.yaml"}, "default", r, IdentityChangedNotice)
	if strings.Join(sent, ",") != "telegram" || len(r.sends) != 1 || !strings.HasPrefix(r.sends[0], "telegram --quiet RepoKit updated") {
		t.Fatalf("sent=%v sends=%v", sent, r.sends)
	}
	if !strings.Contains(IdentityChangedNotice, "/new") {
		t.Fatal("notice does not tell the owner what to do")
	}
}

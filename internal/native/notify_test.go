package native

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

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

// freshenCLI answers send --list, records the in-container session script's
// arguments and reports telegram ended (discord was in use).
type freshenCLI struct {
	sendCLI
	script []string
}

func (f *freshenCLI) RunInput(ctx context.Context, in io.Reader, p string, args ...string) process.Result {
	for i, a := range args {
		if a == "-c" && i+1 < len(args) && strings.Contains(args[i+1], "end_session") {
			f.script = args[i+2:]
			return process.Result{Output: "noise\nREPOKIT_FRESH={\"ended\": [\"telegram\"], \"busy\": []}\n"}
		}
	}
	return f.sendCLI.RunInput(ctx, in, p, args...)
}

func TestFreshenChatsEndsQuietEarlierConversations(t *testing.T) {
	r := &freshenCLI{}
	cutoff := time.Unix(1790880000, 0)
	ended, busy := FreshenChats(context.Background(), target.Identity{Compose: "/x/.hermes/compose.yaml"}, "default", r, cutoff)
	if strings.Join(ended, ",") != "telegram" || len(busy) != 0 {
		t.Fatalf("ended %v busy %v", ended, busy)
	}
	if strings.Join(r.script, " ") != "1790880000 300 discord telegram" {
		t.Fatalf("session script args: %v", r.script)
	}
	// Chats that were not freshened get the /new reminder instead.
	r.sends = nil
	if sent := NotifyChatsExcept(context.Background(), target.Identity{Compose: "/x/.hermes/compose.yaml"}, "default", r, ended, IdentityChangedNotice); len(sent) != 0 {
		t.Fatalf("reminder posted to a freshened or unreachable chat: %v", sent)
	}
	if !strings.Contains(freshenChats, "\"repokit_identity_change\"") || !strings.Contains(freshenChats, "> idle") || !strings.Contains(freshenChats, "starts fresh anyway") {
		t.Fatal("session script lost its reason or its in-use guard")
	}
}

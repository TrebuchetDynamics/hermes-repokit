package native

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// sendTarget matches one `hermes send --list` target line ("  telegram:Juan (dm)").
var sendTarget = regexp.MustCompile(`^\s+([a-z][a-z0-9_]*):\S`)

// IdentityChangedNotice is posted to the owner's chats when default's identity
// changes: a chat keeps the identity it started with until the owner sends /new.
const IdentityChangedNotice = "RepoKit updated this repository team's coordinator identity. Send /new to start a conversation that uses it; this chat keeps the earlier identity until then."

// FreshChatNotice is posted where RepoKit ended the earlier conversation.
const FreshChatNotice = "RepoKit updated this repository team's coordinator identity and closed the earlier conversation (its history is kept). Your next message here starts a fresh conversation with the new identity."

// chatIdleSeconds keeps a conversation the owner is in the middle of: only a
// chat quiet for this long is ended.
const chatIdleSeconds = 300

// freshenChats ends, through Hermes's own session store, each platform's
// newest conversation that began before cutoff and has been quiet for
// chatIdleSeconds. The gateway then opens a fresh conversation, with the
// current identity, on that chat's next message, exactly as /new would; the
// ended conversation keeps its history. It prints the platforms it ended.
const freshenChats = `import json, sys, time
from hermes_state import SessionDB
cutoff, idle = float(sys.argv[1]), float(sys.argv[2])
db = SessionDB()
ended, busy = [], []
for source in sys.argv[3:]:
    for s in db.list_sessions_rich(source=source, limit=1):
        if s.get("ended_at") is not None or float(s.get("started_at") or 0) >= cutoff:
            continue  # the next message starts fresh anyway
        if time.time() - float(s.get("last_active") or 0) > idle:
            db.end_session(s["id"], "repokit_identity_change")
            ended.append(source)
        else:
            busy.append(source)
print("REPOKIT_FRESH=" + json.dumps({"ended": ended, "busy": busy}))
`

var freshResult = regexp.MustCompile(`(?m)^REPOKIT_FRESH=(\{.*\})$`)

// FreshenChats starts a fresh conversation, on the next message, in every
// messaging chat whose conversation predates cutoff (default's identity
// change) and is quiet. It returns the platforms whose conversation it ended
// and those whose earlier conversation is in active use, left alone; a chat
// already starting fresh is in neither.
func FreshenChats(ctx context.Context, id target.Identity, dc string, r InputRunner, cutoff time.Time) (ended, busy []string) {
	platforms := chatPlatforms(nativeTeamCLI(ctx, id, dc, r))
	if len(platforms) == 0 {
		return nil, nil
	}
	args := []string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--workdir", "/opt/data", "hermes", "/opt/hermes/.venv/bin/python", "-c", freshenChats, strconv.FormatInt(cutoff.Unix(), 10), strconv.Itoa(chatIdleSeconds)}
	result := r.RunInput(ctx, nil, "docker", append(args, platforms...)...)
	m := freshResult.FindStringSubmatch(result.Output)
	var out struct{ Ended, Busy []string }
	if result.Err != nil || m == nil || json.Unmarshal([]byte(m[1]), &out) != nil {
		// Unobserved: remind every chat rather than stay silent.
		return nil, platforms
	}
	sort.Strings(out.Ended)
	sort.Strings(out.Busy)
	return out.Ended, out.Busy
}

// chatPlatforms lists the messaging platforms Hermes can reach, from public
// `hermes send --list`.
func chatPlatforms(run teamCLI) []string {
	out, err := run("-p", "default", "send", "--list")
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for line := range strings.SplitSeq(string(out), "\n") {
		if m := sendTarget.FindStringSubmatch(line); m != nil {
			seen[m[1]] = true
		}
	}
	platforms := make([]string, 0, len(seen))
	for p := range seen {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)
	return platforms
}

// NotifyChats posts message to the home channel of every messaging platform
// Hermes already has, through public `hermes send` (no model, no agent loop).
// A platform without a home channel is skipped. It returns the platforms that
// accepted the message.
func NotifyChats(ctx context.Context, id target.Identity, dc string, r InputRunner, message string) []string {
	return NotifyChatsExcept(ctx, id, dc, r, nil, message)
}

// NotifyChatsExcept posts message to every platform's home channel except the
// excluded platforms.
func NotifyChatsExcept(ctx context.Context, id target.Identity, dc string, r InputRunner, exclude []string, message string) []string {
	run := nativeTeamCLI(ctx, id, dc, r)
	var platforms []string
	for _, p := range chatPlatforms(run) {
		if !slices.Contains(exclude, p) {
			platforms = append(platforms, p)
		}
	}
	return sendTo(run, platforms, message)
}

// NotifyChatsOn posts message to the named platforms' home channels.
func NotifyChatsOn(ctx context.Context, id target.Identity, dc string, r InputRunner, platforms []string, message string) []string {
	return sendTo(nativeTeamCLI(ctx, id, dc, r), platforms, message)
}

func sendTo(run teamCLI, platforms []string, message string) []string {
	var sent []string
	for _, platform := range platforms {
		if _, err := run("-p", "default", "send", "--to", platform, "--quiet", message); err == nil {
			sent = append(sent, platform)
		}
	}
	sort.Strings(sent)
	return sent
}

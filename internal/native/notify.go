package native

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// sendTarget matches one `hermes send --list` target line ("  telegram:Juan (dm)").
var sendTarget = regexp.MustCompile(`^\s+([a-z][a-z0-9_]*):\S`)

// IdentityChangedNotice is posted to the owner's chats when default's identity
// changes: a chat keeps the identity it started with until the owner sends /new.
const IdentityChangedNotice = "RepoKit updated this repository team's coordinator identity. Send /new to start a conversation that uses it; this chat keeps the earlier identity until then."

// NotifyChats posts message to the home channel of every messaging platform
// Hermes already has, through public `hermes send` (no model, no agent loop).
// A platform without a home channel is skipped. It returns the platforms that
// accepted the message.
func NotifyChats(ctx context.Context, id target.Identity, dc string, r InputRunner, message string) []string {
	run := nativeTeamCLI(ctx, id, dc, r)
	out, err := run("-p", "default", "send", "--list")
	if err != nil {
		return nil
	}
	platforms := map[string]bool{}
	for line := range strings.SplitSeq(string(out), "\n") {
		if m := sendTarget.FindStringSubmatch(line); m != nil {
			platforms[m[1]] = true
		}
	}
	var sent []string
	for platform := range platforms {
		if _, err := run("-p", "default", "send", "--to", platform, "--quiet", message); err == nil {
			sent = append(sent, platform)
		}
	}
	sort.Strings(sent)
	return sent
}

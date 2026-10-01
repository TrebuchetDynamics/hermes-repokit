package verify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// sessionStart matches a native session ID, whose prefix is its UTC start time
// (20260929_192739_30656db1).
var sessionStart = regexp.MustCompile(`\b(\d{8}_\d{6})_[0-9a-f]{8}\b`)

// ChannelSessions reports messaging conversations that began before default's
// current identity. Hermes keeps a conversation's system prompt from when it
// started, so such a chat still acts on the earlier SOUL (for example an older
// roster) until the owner starts a fresh one with /new. It reads only each
// platform's newest session ID from public `sessions list`; titles, previews
// and chat targets are never read or reported.
func ChannelSessions(ctx context.Context, id target.Identity, r Runner) []Probe {
	info, err := os.Lstat(filepath.Join(id.Root, ".hermes", "SOUL.md"))
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	identity := info.ModTime().UTC()
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		// Report it like every other runtime probe instead of vanishing
		// (for example while a recipe change awaits recreation).
		return []Probe{{"sessions", Unknown, "runtime unavailable; conversation ages not observed"}}
	}
	platforms := configuredPlatforms(ctx, r, dc, container)
	sort.Strings(platforms)
	ended := endedConversations(ctx, r, dc, container, platforms)
	var stale []string
	checked := 0
	for _, platform := range platforms {
		if nonInteractive[platform] || !platformName.MatchString(platform) {
			continue
		}
		out, ok := hermesCLI(ctx, r, dc, container, "sessions", "list", "--source", platform, "--limit", "1")
		if !ok {
			continue
		}
		m := sessionStart.FindStringSubmatch(out)
		if m == nil {
			continue // no conversation on this platform yet
		}
		start, err := time.ParseInLocation("20060102_150405", m[1], time.UTC)
		if err != nil {
			continue
		}
		checked++
		// An ended conversation (RepoKit or /new closed it) is not the one
		// the next message continues: that message starts fresh.
		if start.Before(identity) && !ended[platform] {
			stale = append(stale, platform)
		}
	}
	switch {
	case len(stale) > 0:
		return []Probe{{"sessions", Degraded, "the current conversation on " + strings.Join(stale, ", ") + " began before default's current identity and still acts on the earlier SOUL; send /new there to start a fresh one"}}
	case checked > 0:
		return []Probe{{"sessions", Healthy, "current messaging conversations began with default's current identity"}}
	}
	return nil
}

// endedNewest reads, read-only through Hermes's own session store, whether
// each platform's newest conversation has ended; nothing else is read.
const endedNewest = `import json, sys
from hermes_state import SessionDB
db = SessionDB(read_only=True)
out = {}
for source in sys.argv[1:]:
    rows = db.list_sessions_rich(source=source, limit=1)
    out[source] = bool(rows) and rows[0].get("ended_at") is not None
print("REPOKIT_ENDED=" + json.dumps(out))
`

var endedResult = regexp.MustCompile(`(?m)^REPOKIT_ENDED=(\{.*\})$`)

func endedConversations(ctx context.Context, r Runner, dc, container string, platforms []string) map[string]bool {
	ended := map[string]bool{}
	if len(platforms) == 0 {
		return ended
	}
	args := append([]string{"--context", dc, "exec", "--user", "hermes", "--env", "HOME=/opt/data", "--workdir", "/", container, "/opt/hermes/.venv/bin/python", "-c", endedNewest}, platforms...)
	out := r.Run(ctx, "docker", args...)
	if m := endedResult.FindStringSubmatch(out.Output); out.Err == nil && !out.Truncated && m != nil {
		json.Unmarshal([]byte(m[1]), &ended)
	}
	return ended
}

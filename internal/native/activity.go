package native

import (
	"context"
	"regexp"
	"strconv"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// ChatQuietSeconds is how long a human conversation must have been quiet
// before RepoKit recreates the container or restarts the gateway: a restart
// mid-reply drops the owner's turn ("Operation interrupted ... Your request
// was not processed").
const ChatQuietSeconds = 600

// quietChat prints whether any human conversation (every source but Kanban
// workers, cron, subagents and tools) was active within argv[1] seconds, and
// exits 1 when one was.
const quietChat = `import sys, time
from hermes_state import SessionDB
window = float(sys.argv[1])
rows = SessionDB().list_sessions_rich(exclude_sources=["kanban", "cron", "subagent", "tool"], limit=5, order_by_last_active=True, include_hidden=True)
active = any(time.time() - float(r.get("last_active") or 0) < window for r in rows)
print("REPOKIT_CHAT=" + ("active" if active else "quiet"))
sys.exit(1 if active else 0)
`

var chatResult = regexp.MustCompile(`(?m)^REPOKIT_CHAT=(active|quiet)$`)

// ChatQuiet reports whether no human conversation was active in the last
// ChatQuietSeconds. ok is false when it could not be observed.
func ChatQuiet(ctx context.Context, id target.Identity, dc string, r InputRunner) (quiet, ok bool) {
	args := []string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--workdir", "/opt/data", "hermes", "/opt/hermes/.venv/bin/python", "-c", quietChat, strconv.Itoa(ChatQuietSeconds)}
	m := chatResult.FindStringSubmatch(r.RunInput(ctx, nil, "docker", args...).Output)
	if m == nil {
		return false, false
	}
	return m[1] == "quiet", true
}

// quietChatShell is a shell condition, run inside the container, that holds
// when no human conversation was active in the last ChatQuietSeconds.
func quietChatShell() string {
	return "/opt/hermes/.venv/bin/python -c " + shellQuote(quietChat) + " " + strconv.Itoa(ChatQuietSeconds) + " >/dev/null 2>&1"
}

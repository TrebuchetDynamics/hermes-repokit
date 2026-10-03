package verify

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/native"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

// hermesCLI runs one read-only public Hermes command in the verified runtime.
func hermesCLI(ctx context.Context, r Runner, dc, container string, args ...string) (string, bool) {
	argv := append([]string{"--context", dc, "exec", "--user", "hermes", "--workdir", "/", container, "/usr/local/bin/hermes", "-p", "default"}, args...)
	out := r.Run(ctx, "docker", argv...)
	if out.Err != nil || out.Truncated {
		return "", false
	}
	return out.Output, true
}

// Gateway reports the default gateway process from public `gateway status`.
// A running process is not proof that a message was delivered.
func Gateway(ctx context.Context, id target.Identity, r Runner) []Probe {
	probe := Probe{"gateway", Unknown, "gateway status unavailable"}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return []Probe{probe}
	}
	if out, ok := hermesCLI(ctx, r, dc, container, "gateway", "status"); ok {
		if pid, err := native.GatewayPID(out); err == nil && pid > 0 {
			probe = Probe{"gateway", Healthy, "default gateway running; message delivery is not observed by verify"}
		} else if err == nil {
			probe = Probe{"gateway", Inactive, "default gateway not running; no messaging or automatic dispatch"}
		}
	}
	return []Probe{probe}
}

var nonInteractive = map[string]bool{"acp": true, "api_server": true, "cron": true, "webhook": true}

// DefaultKanban observes the native dispatch policy, completion notifications
// and the default profile's per-channel Kanban tool through `config get`.
func DefaultKanban(ctx context.Context, id target.Identity, r Runner) []Probe {
	dispatch := Probe{"kanban:dispatch", Unknown, "native dispatch configuration unavailable"}
	notify := Probe{"kanban:notifications", Unknown, "completion notification configuration unavailable"}
	channels := []Probe{{"channels", Unknown, "default channel tool selections unavailable"}}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return append([]Probe{dispatch, notify}, channels...)
	}
	var kanban map[string]any
	if out, ok := hermesCLI(ctx, r, dc, container, "config", "get", "kanban", "--json"); ok && json.Unmarshal([]byte(out), &kanban) == nil && kanban != nil {
		switch {
		case native.OperationalPolicy(kanban):
			dispatch = Probe{"kanban:dispatch", Healthy, "automatic dispatch configured on default: review dispatch, seven-profile allowlist, max_in_progress=1, auto_decompose=false"}
		case kanban["dispatch_in_gateway"] == true:
			dispatch = Probe{"kanban:dispatch", Degraded, "dispatch enabled with an owner-changed policy; RepoKit preserves it"}
		case kanban["dispatch_in_gateway"] == false || kanban["dispatch_in_gateway"] == nil:
			dispatch = Probe{"kanban:dispatch", Inactive, "automatic dispatch off; run setup to enable it"}
		}
		if kanban["auto_subscribe_on_create"] == true && kanban["notify_in_gateway"] == true {
			notify = Probe{"kanban:notifications", Healthy, "completion subscriptions configured; originating-channel delivery not observed"}
		} else {
			notify = Probe{"kanban:notifications", Degraded, "completion subscription or gateway notification disabled"}
		}
	}
	var selections map[string]any
	if out, ok := hermesCLI(ctx, r, dc, container, "config", "get", "platform_toolsets", "--json"); ok && json.Unmarshal([]byte(out), &selections) == nil {
		effective := func(platform string) (bool, bool) {
			out, ok := hermesCLI(ctx, r, dc, container, "tools", "list", "--platform", platform)
			if !ok {
				return false, false
			}
			return kanbanEnabled(out)
		}
		channels = channelTools(selections, configuredPlatforms(ctx, r, dc, container), effective)
	}
	return append([]Probe{dispatch, notify}, channels...)
}

// configuredPlatforms names the messaging platforms Hermes can deliver to,
// from public `send --list --json` (no message is sent). This includes
// channels configured only through the environment, which have no saved tool
// selection. Only platform names are used; targets and chat names are private.
func configuredPlatforms(ctx context.Context, r Runner, dc, container string) []string {
	out, ok := hermesCLI(ctx, r, dc, container, "send", "--list", "--json")
	var listed struct {
		Platforms map[string]json.RawMessage `json:"platforms"`
	}
	if !ok || json.Unmarshal([]byte(out), &listed) != nil {
		return nil
	}
	names := make([]string, 0, len(listed.Platforms))
	for name := range listed.Platforms {
		names = append(names, name)
	}
	return names
}

var kanbanToolLine = regexp.MustCompile(`(?m)^\s*(✓ enabled|✗ disabled)\s+kanban\s`)

// kanbanEnabled reads the kanban row of public `tools list --platform` output:
// (enabled, known).
func kanbanEnabled(out string) (bool, bool) {
	m := kanbanToolLine.FindStringSubmatch(out)
	if m == nil {
		return false, false
	}
	return m[1] == "✓ enabled", true
}

// channelTools requires Kanban on every human-facing channel of default so
// Telegram and CLI can create and follow the same work. Saved per-channel
// selections are read directly; a configured channel without one (set up only
// through the environment) is judged by Hermes's effective tools for it.
// Memory and other optional tools are owner-managed and not checked.
func channelTools(selections map[string]any, configured []string, effective func(string) (bool, bool)) []Probe {
	var result []Probe
	for _, platform := range configured {
		if _, saved := selections[platform]; saved || nonInteractive[platform] || !platformName.MatchString(platform) {
			continue
		}
		p := Probe{"channel:" + platform, Unknown, "configured without a saved tool selection; effective tools unavailable"}
		if enabled, known := effective(platform); known && enabled {
			p = Probe{"channel:" + platform, Healthy, "default has the kanban tool on this channel (effective native tools; no saved selection)"}
		} else if known {
			p = Probe{"channel:" + platform, Degraded, "default lacks the kanban tool here; run: hermes-<repo> -p default tools enable kanban --platform " + platform}
		}
		result = append(result, p)
	}
	for platform, raw := range selections {
		if nonInteractive[platform] || !platformName.MatchString(platform) {
			continue
		}
		tools, _ := raw.([]any)
		have := map[string]bool{}
		for _, t := range tools {
			if s, ok := t.(string); ok {
				have[s] = true
			}
		}
		p := Probe{"channel:" + platform, Healthy, "default has the kanban tool on this channel"}
		if !have["kanban"] {
			p = Probe{"channel:" + platform, Degraded, "default lacks the kanban tool here; run: hermes-<repo> -p default tools enable kanban --platform " + platform}
		}
		result = append(result, p)
	}
	if len(result) == 0 {
		return []Probe{{"channels", Unknown, "no saved per-channel selections and no configured messaging platform observed for default"}}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Component < result[j].Component })
	return result
}

// ReviewEvidence searches recent reviewer-completed cards for the native
// same-card acceptance chain: an executor run requested review, a later tester
// run forwarded it, and reviewer completed the card.
// Read-only public Kanban output.
func ReviewEvidence(ctx context.Context, id target.Identity, r Runner) Probe {
	probe := Probe{"review:evidence", Unknown, "Kanban history unavailable"}
	dc, container, err := integrationRuntime(ctx, id, r)
	if err != nil {
		return probe
	}
	// Reviewer completes every accepted implementation card, so only cards it
	// finished are candidates; research, planning and admin work cannot push
	// the evidence out of view. The newest reviewEvidenceWindow are read.
	// The plain listing carries only IDs and titles; --json includes every
	// card body and outgrows the bounded output on a busy board (sdrhf: 120 KB).
	// A single-shape default completes its own verified cards; until the
	// first one lands, the seven-profile history still proves review.
	if team.ShapeOf(id.Root) == team.Single {
		if p, ok := evidenceFrom(ctx, r, dc, container, "default", singleChain); ok && p.Status == Healthy {
			return p
		}
	}
	p, ok := evidenceFrom(ctx, r, dc, container, "reviewer", acceptanceChain)
	if !ok {
		return probe
	}
	return p
}

// evidenceFrom reads the newest cards completed by assignee and reports the
// first whose run history satisfies chain.
func evidenceFrom(ctx context.Context, r Runner, dc, container, assignee string, chain func([]struct{ Profile, Outcome string }) string) (Probe, bool) {
	out, ok := hermesCLI(ctx, r, dc, container, "kanban", "list", "--status", "done", "--assignee", assignee, "--sort", "completed-desc")
	if !ok {
		return Probe{}, false
	}
	// One card per line; its ID comes first, and a title may name others.
	var ids []string
	seen := map[string]bool{}
	for line := range strings.SplitSeq(out, "\n") {
		if id := taskID.FindString(line); id != "" && !seen[id] && len(ids) < reviewEvidenceWindow {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, task := range ids {
		// Run history alone, not the card's body and comments.
		out, ok := hermesCLI(ctx, r, dc, container, "kanban", "runs", task, "--json")
		var runs []struct{ Profile, Outcome string }
		if !ok || json.Unmarshal([]byte(out), &runs) != nil {
			continue
		}
		if described := chain(runs); described != "" {
			return Probe{"review:evidence", Healthy, "card " + task + ": " + described + " on the same card"}, true
		}
	}
	return Probe{"review:evidence", Unqualified, "no same-card independent review among recent completed cards; run real reviewed work"}, true
}

// singleChain describes a single-shape default card: default completed it in
// a run that followed, apart from crashed or reclaimed attempts, a separate
// default run requesting review. The implementing run never completes it.
func singleChain(runs []struct{ Profile, Outcome string }) string {
	n := len(runs)
	if n < 2 || runs[n-1].Profile != "default" || runs[n-1].Outcome != "completed" {
		return ""
	}
	for i := n - 2; i >= 0; i-- {
		switch runs[i].Outcome {
		case "crashed", "timed_out", "reclaimed", "gave_up", "spawn_failed":
			continue
		case "review_requested":
			if runs[i].Profile == "default" {
				return "default implemented it, and a separate default run verified and completed it"
			}
		}
		return ""
	}
	return ""
}

// taskID matches a native Kanban task ID in listing output.
var taskID = regexp.MustCompile(`\bt_[0-9a-f]{8}\b`)

// reviewEvidenceWindow bounds how many reviewer-completed cards verify reads
// (one native `kanban show` each).
const reviewEvidenceWindow = 20

// acceptanceChain describes a run history whose final run is reviewer's
// completion and whose latest executor run requesting review is followed by a
// tester hand-off. Only executor implements: another profile requesting review
// never qualifies. A tester relay of reviewer-requested changes precedes the
// fix, so it never counts as verification of that fix.
func acceptanceChain(runs []struct{ Profile, Outcome string }) string {
	if len(runs) == 0 || runs[len(runs)-1].Profile != "reviewer" || runs[len(runs)-1].Outcome != "completed" {
		return ""
	}
	implementer, verified := "", false
	for _, run := range runs[:len(runs)-1] {
		switch {
		case run.Profile == "tester" || run.Profile == "reviewer":
			if run.Profile == "tester" && run.Outcome == "review_requested" && implementer != "" {
				verified = true
			}
		case run.Profile == "executor" && run.Outcome == "review_requested":
			implementer, verified = run.Profile, false
		}
	}
	if !verified {
		return ""
	}
	return implementer + " requested review, tester verified and reviewer completed it"
}

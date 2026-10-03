// Package team defines default, RepoKit's one profile, and its repository-scoped identity.
package team

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed souls/*.md
var souls embed.FS

// SoulRecord is the file beside a profile's SOUL.md in which RepoKit records
// the SHA-256 of the SOUL it last wrote. A SOUL that still matches its record
// is untouched, so install replaces it whichever RepoKit build wrote it; any
// other SOUL is the owner's. Git keeps every earlier SOUL; the binary keeps
// none.
const SoulRecord = ".repokit-soul"

// SoulDigest is the record RepoKit keeps for a SOUL it wrote.
func SoulDigest(soul string) string {
	sum := sha256.Sum256([]byte(soul))
	return hex.EncodeToString(sum[:])
}

// Role is the profile's current managed identity. A SOUL that is
// neither Soul nor recorded as written by RepoKit is owner state.
type Role struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Soul        string `json:"soul"`
	// Toolsets is the baseline RepoKit grants when it creates or resets the
	// profile. Required is the subset the role needs to do its work; only
	// Required is checked afterwards, so the owner may remove the rest.
	Toolsets []string `json:"toolsets"`
	Required []string `json:"required"`
	// Skills are official Hermes skill identifiers granted at creation or
	// reset. Like extra toolsets, the owner may remove them.
	Skills []string `json:"skills,omitempty"`
	// Settings are native configuration values granted at creation or reset
	// and never checked afterwards: the owner may change them.
	Settings map[string]any `json:"settings,omitempty"`
}

// ForRepository binds default to the selected repository.
func ForRepository(id target.Identity) []Role {
	roles := Roster()
	for i := range roles {
		roles[i].Soul = runtimeContract(id) + repositoryHeader(id, roles[i]) + maintenance + roles[i].Soul
	}
	return roles
}

func repositoryHeader(id target.Identity, role Role) string {
	return fmt.Sprintf(`# Repository team identity

Repository name: %s
Stable repository ID: %s
Native profile: %s
Role description: %s

Hermes is the runtime, not your repository identity. Present yourself through
this repository rather than as a generic assistant.

This deployment runs as one profile, default: the owner's coordinator, which
also does every card itself. When asked about the team, say so; report running
workers separately and only from inspected runtime state.

Dispatch policy: bootstrap off; operational automatic. After successful setup,
the default gateway dispatcher launches default's implementation and
verification runs automatically. Automatic decomposition stays disabled and
max_in_progress stays one. Inspect configured AND live dispatch state before
promising progress. If dispatch is disabled, stale, paused or degraded, report
that blocker rather than claiming queued work started. Do not use one-shot
dispatch to bypass incomplete setup or an inactive gateway.

`, id.Name, id.Project, role.Name, role.Description)
}

const maintenance = `## Channels and identity

Profile is identity. Platform is the conversation surface. Session is conversation history.
CLI, Telegram and other human-facing channels are surfaces for this same
default profile, not separate agents. They share this SOUL, the memory and the
repository Kanban board. Check routing before treating another profile's
channel as yours.

Messaging channels are full remote development consoles. Create work with the
native kanban_create tool in the originating conversation, assigned to
default with workspace_kind "dir" and workspace_path "/workspace", and verify
its subscribed result: native notify+wake subscriptions bring completion,
review requests, changes requested and blocked work back to this same
conversation. Do not invent destination IDs or send results to another chat.
The gateway dispatches cards automatically: do not invoke one-shot dispatch or
ask the owner to SSH to start ordinary work. If subscription, dispatch or
delivery fails, state the blocker in the originating conversation. Preserve
channel authorization; terminal or file capability never authorizes weakening
allowlists or authentication.

## Self-maintenance

You may inspect and update your own non-secret preferences, optional tools and
qualified behavior settings through native Hermes configuration commands. Do
not dump entire configuration or environment files. Creating, retiring or
changing profiles is the owner's decision: ask first.

You must not remove repository identity, Kanban availability, verification
runs or repository isolation. Never read raw credentials, private bot tokens
or OAuth state into a transcript; never weaken authentication, delete
profiles, or erase memory or board history. Preserve operational automatic
dispatch, review_dispatch=true, the managed allowlist, auto_decompose=false
and max_in_progress=1. Use gateway_restart_after_turn only when that native
maintenance tool is available, and confirm the replacement process is healthy;
never kill processes or restart Docker to apply preferences, and never invoke
a RepoKit executable.

`

// noApprovals is the owner's chosen default: agents never stop for an
// approval prompt. Hermes's approvals are off and its protected
// instruction-file gate is lifted; Hermes's hard floor (root wipe, raw device
// writes, shutdown) and any approvals.deny rule still block. Granted at
// creation, reset or migration; the owner may turn either back on.
var noApprovals = map[string]any{"approvals.mode": "off", "security.protected_instruction_files": false}

// webSearch makes Exa's keyless free tier the web search and extract provider:
// semantic search with page content, no key or account. Granted at creation or
// reset like every setting; the owner may pick another provider.
var webSearch = map[string]any{"web.backend": "exa", "web.provider_tier.exa": "free"}

// granted is what every profile starts with.
var granted = settings(noApprovals, webSearch)

// coordinator adds a 10s dispatch tick on default, whose config the gateway's
// dispatcher reads at boot. Hermes's 60s default left the board idle about
// half a minute between stages, a fifth of a typical run on a live board. A
// chat /goal gets 100 continuations instead of Hermes's 20, so a long program
// does not silently expire.
var coordinator = settings(granted, map[string]any{"kanban.dispatch_interval_seconds": 10, "goals.max_turns": 100})

// defaultSettings is everything default is granted: it talks with the owner
// and runs every card, so it also gets whole-card effort (no turn cap, high
// reasoning), checkpoints for /rollback, and room for a verification run's
// subagents (a one-shot card run spawns at most this many).
var defaultSettings = settings(coordinator, map[string]any{
	"agent.max_turns": 0, "agent.reasoning_effort": "high",
	"checkpoints.enabled": true, "delegation.oneshot_max_children": 4,
})

// defaultSkills are what the retired specialist profiles were granted that
// still serve one agent; the owner may remove any.
var defaultSkills = []string{
	"official/productivity/decision-questionnaire", "official/autonomous-ai-agents/dynamic-workflow",
	"official/research/domain-intel", "official/software-development/code-wiki", "official/research/duckduckgo-search",
	"official/software-development/grill-me", "official/software-development/ast-grep",
	"official/software-development/rest-graphql-debug", "official/software-development/subagent-driven-development",
	"official/autonomous-ai-agents/agent-merge-conflict-arbiter", "official/dogfood/adversarial-ux-test",
	"official/security/oss-forensics",
}

func settings(maps ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

// approvalGrants are the granted settings that run a profile without approval
// prompts.
var approvalGrants = []string{"approvals.mode", "security.protected_instruction_files"}

// WithApprovalPrompts returns roles whose granted settings keep Hermes's
// approval prompts and protected instruction-file gate: the posture an owner
// chooses at setup instead of RepoKit's autonomous default.
func WithApprovalPrompts(roles []Role) []Role {
	out := make([]Role, len(roles))
	for i, role := range roles {
		kept := map[string]any{}
		for k, v := range role.Settings {
			if !slices.Contains(approvalGrants, k) {
				kept[k] = v
			}
		}
		role.Settings = kept
		out[i] = role
	}
	return out
}

// CoordinatorTools is the toolset default is granted on the CLI and every chat
// channel when RepoKit adopts or resets it: everything the owner talks to can
// do everything. Only kanban is required; the owner may switch the rest off.
// Hermes's video generation, Home Assistant, Spotify, Discord and Yuanbao stay
// the owner's to enable.
var CoordinatorTools = []string{
	"web", "browser", "terminal", "file", "code_execution", "vision", "video",
	"image_gen", "x_search", "tts", "skills", "todo", "kanban", "memory",
	"context_engine", "session_search", "connections", "clarify", "delegation",
	"cronjob", "computer_use", "a2a",
}

// Roster returns RepoKit's one profile, default.
func Roster() []Role {
	common, _ := souls.ReadFile("souls/common.md")
	single, _ := souls.ReadFile("souls/single.md")
	owner, _ := souls.ReadFile("souls/owner.md")
	return []Role{{
		Name:        "default",
		Description: "The repository's whole team: the owner's coordinator, which researches through subagents, implements its own Kanban cards and verifies each in a separate fresh run, and keeps the owner's decisions in memory.",
		Toolsets:    slices.Clone(CoordinatorTools),
		Required:    []string{"kanban"},
		Skills:      slices.Clone(defaultSkills),
		Settings:    defaultSettings,
		Soul:        string(common) + "\n" + string(single) + string(owner),
	}}
}

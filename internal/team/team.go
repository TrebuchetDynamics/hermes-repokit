// Package team defines RepoKit's permanent roles and repository-scoped identities.
package team

import (
	"embed"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed souls/*.md souls/previous/*.md
var souls embed.FS

// PreviousRelease is the one earlier release whose managed SOULs RepoKit still
// recognizes, so an untouched profile from it upgrades in place instead of
// reading as owner-customized. Nothing older is recognized.
const PreviousRelease = "v0.2.3"

// Placeholders in souls/previous: each template is the previous release's exact
// rendered SOUL with the repository name and stable ID left open.
const (
	previousNameToken = "{{repository.name}}"
	previousIDToken   = "{{repository.id}}"
)

// Role is one permanent profile's current managed identity. A SOUL that is
// neither Soul nor PreviousSoul is owner state.
type Role struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Soul        string `json:"soul"`
	// PreviousSoul is the exact SOUL PreviousRelease installed for this role
	// and repository; install upgrades an untouched one to Soul.
	PreviousSoul string `json:"previous_soul,omitempty"`
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

// ForRepository binds each permanent role to the selected repository.
func ForRepository(id target.Identity) []Role {
	roles := Roster()
	for i := range roles {
		role := &roles[i]
		soul := runtimeContract(id, role.Name) + repositoryHeader(id, *role)
		if role.Name == "default" {
			soul += defaultMaintenance
		}
		role.Soul = soul + role.Soul
		role.PreviousSoul = previousSoul(id, role.Name)
	}
	return roles
}

// previousSoul renders PreviousRelease's SOUL for this repository, or "" when
// the role did not exist then.
func previousSoul(id target.Identity, name string) string {
	tpl, err := souls.ReadFile("souls/previous/" + name + ".md")
	if err != nil {
		return ""
	}
	return strings.NewReplacer(previousNameToken, id.Name, previousIDToken, id.Project).Replace(string(tpl))
}

func repositoryHeader(id target.Identity, role Role) string {
	return fmt.Sprintf(`# Repository team identity

Repository name: %s
Stable repository ID: %s
Native profile: %s
Role description: %s

Hermes is the runtime, not your repository identity. Present yourself through
this repository and your native profile's role rather than as a generic assistant.
The default profile is this repository's primary human-facing coordinator and
orchestrator. Users normally speak with default; the other profiles perform
bounded specialist work and return durable handoffs to default.

Permanent roster: default, researcher, planner, executor, tester, reviewer, steward.
These are seven permanent profiles, including default. Persistent profiles are
identities and capabilities; running workers are transient executions of those
profiles. The roster exists even when no workers are running. When asked how
many team members there are, answer seven permanent profiles and list this roster.
Report currently running workers separately and only from inspected runtime state.
Verify native inventory before claiming any additional owner-created profiles.

Dispatch policy: bootstrap off; operational automatic. After successful setup,
the default gateway dispatcher launches eligible assigned profiles and distinct
reviewers automatically. Specialists never own another dispatcher loop.
Automatic decomposition stays disabled and max_in_progress stays one. A profile
exists persistently; a worker runs only while executing a card. Inspect configured
AND live dispatch state before promising progress. If dispatch is disabled, stale,
paused or degraded, report that blocker rather than claiming queued work started.
Do not use one-shot dispatch to bypass incomplete setup or an inactive gateway.

`, id.Name, id.Project, role.Name, role.Description)
}

const defaultMaintenance = `## Channels and identity

Profile is identity. Platform is the conversation surface. Session is conversation history.
Runtime is the serving process. CLI, Telegram and other primary human-facing
channels are surfaces for this same default repository orchestrator, not separate
agents. They share this SOUL, the permanent roster and the repository Kanban
board. Histories, native platform extras and owner-configured Hermes features
such as memory can differ. Check routing before treating another profile's
channel as yours. Programmatic reduced-capability surfaces need not mirror chat.

Every configured human-facing default channel requires its native platform core
capabilities plus Kanban. Memory and other optional Hermes features are the
owner's configuration: preserve them, but do not require or enable them. Use
native preset-derived selections; never freeze a copied CLI tool list or remove
platform extras. Existing conversations can retain old tool schemas: request a
fresh conversation after changing tools.

Messaging channels are full remote development consoles. Handle lightweight file
reads, repository searches, Git inspection and diagnostic commands directly;
do not create ceremonial Kanban work for a small interactive question. Route
substantive artifact changes through executor, then same-card tester and reviewer.
Create work with the native kanban_create tool in the originating conversation
and verify its subscribed result. Native notify+wake subscriptions bring completion,
review requests, changes requested and blocked work back to this same conversation.
Do not invent destination IDs or send results to another chat.

Create and route bounded cards; the gateway dispatches them automatically.
Do not invoke one-shot dispatch or ask the user to SSH to start ordinary work.
On native review/completion wake, inspect the card and durable evidence before
reporting results in the originating conversation. Required review starts with
reviewer="tester" on the implementation card; tester forwards passing work to
reviewer. Never inherit the implementer's identity for verification or approval.
If subscription, dispatch or delivery fails, state the blocker in the originating
conversation. Do not claim a remote coding workflow passed without real execution,
distinct tester and reviewer identities and a returned result. Preserve channel authorization;
terminal/file capability never authorizes weakening allowlists or authentication.

## Self-maintenance

You may inspect and update your own non-secret preferences, optional tools and
qualified behavior settings. Prefer native Hermes configuration commands:
targeted config get/set, tools enable/disable with an explicit platform, profile
show and gateway status. Do not dump entire configuration or environment files.
File and terminal capability permits diagnosis and self-maintenance; substantive
repository artifact work still belongs to executor, tester and reviewer.
Route specialist creation, retirement, SOUL and capability changes to steward.

You may repair, but must not remove, repository identity, the permanent roster,
Kanban availability, independent review, repository isolation.
Never replace or
read raw credentials, private bot tokens or OAuth state into a transcript; never
weaken authentication, delete profiles, or erase memory or board history. Secret changes require owner-performed private native setup.
Preserve operational automatic dispatch, review_dispatch=true, the managed seven-role
allowlist, auto_decompose=false and max_in_progress=1. Incomplete setup must remain
dispatch-off; use setup reconciliation to repair it, never bypass its activation gates.

Use gateway_restart_after_turn only when that qualified native maintenance tool
is installed and available. It requests native drain/restart; an acknowledgement
is not proof of a healthy successor. Inspect its status on the replacement process,
including adapter recovery and configuration generation. If restart is pending,
uncertain or deferred for active work, do not replay it on session recovery.
Do not use a synchronous terminal gateway restart from your own active turn.
Never kill processes or restart Docker to apply preferences. If the native tool
is unavailable, report self-restart unqualified rather than pretending it succeeded.
Tool/session changes may need only a fresh conversation; restart only for a
qualified runtime change. This capability must never invoke a RepoKit executable.

`

// Roster returns independent role values in workflow order. Coarse file and
// terminal bundles mean specialist read-only boundaries are advisory in the
// qualified Hermes release; tester has no file-editing toolset at all.
func Roster() []Role {
	roles := []Role{
		{Name: "default", Description: "Primary human-facing repository coordinator and orchestrator. Understands user goals, answers lightweight questions directly, designs bounded Kanban workflows, assigns the appropriate team roles, establishes shared decisions, follows progress, and verifies that completed work has passed required review.", Toolsets: []string{"kanban", "memory"}, Required: []string{"kanban"}, Skills: []string{"official/productivity/decision-questionnaire", "official/autonomous-ai-agents/dynamic-workflow"}},
		{Name: "researcher", Description: "Investigates repository context, external sources and prior project knowledge. Resolves unknowns, compares alternatives and produces source-backed findings without changing the target artifact.", Toolsets: []string{"file", "web", "browser", "terminal", "skills", "memory"}, Required: []string{"file", "web"}, Skills: []string{"official/research/domain-intel", "official/software-development/code-wiki", "official/research/duckduckgo-search"}},
		{Name: "planner", Description: "Turns goals, constraints and research into a bounded execution contract with scope, decisions, dependencies, acceptance criteria, verification requirements and known risks. Does not perform the planned work.", Toolsets: []string{"file", "web", "skills", "memory"}, Required: []string{"file"}, Skills: []string{"official/software-development/grill-me", "official/productivity/decision-questionnaire"}},
		{Name: "executor", Description: "Produces one bounded repository artifact or change from an approved task contract, preserves unrelated state, performs appropriate verification and hands work to independent review when required.", Toolsets: []string{"file", "terminal", "code_execution", "web", "browser", "skills", "delegation", "memory"}, Required: []string{"file", "terminal", "code_execution", "skills"}, Skills: []string{"official/software-development/ast-grep", "official/software-development/rest-graphql-debug", "official/software-development/subagent-driven-development", "official/autonomous-ai-agents/agent-merge-conflict-arbiter"},
			// Hermes asks a human to approve every write to AGENTS.md, CLAUDE.md,
			// SOUL.md or .cursorrules, which an unattended worker can never get.
			// Executor writes them as reviewed card work instead; tester and
			// reviewer check the change on the same card. Every other role
			// keeps Hermes's gate.
			Settings: map[string]any{"security.protected_instruction_files": false}},
		{Name: "tester", Description: "Independently verifies an implementation's behavior on the same Kanban card before final review. Runs tests, builds and checks, reproduces failures and probes edge cases. Requests changes or forwards passing work to reviewer; never modifies the repository.", Toolsets: []string{"terminal", "code_execution", "web", "browser", "skills", "memory"}, Required: []string{"terminal"}, Skills: []string{"official/dogfood/adversarial-ux-test", "official/software-development/rest-graphql-debug"}},
		{Name: "reviewer", Description: "Independently evaluates completed work against its task contract, underlying artifacts and verification evidence. Approves or requests changes on the same Kanban card and does not implement the requested work.", Toolsets: []string{"file", "terminal", "code_execution", "web", "skills", "memory"}, Required: []string{"file", "terminal"}, Skills: []string{"official/security/oss-forensics", "official/software-development/grill-me"}},
		{Name: "steward", Description: "Maintains the repository's Hermes profile roster and agent capabilities. Creates, updates, configures, retires, backs up and, with explicit authorization, deletes profiles. Manages profile descriptions, SOUL contracts, skills, toolsets and profile distributions. Does not coordinate project work or modify project artifacts.", Toolsets: []string{"terminal", "file", "skills", "memory"}, Required: []string{"terminal", "file"}},
	}
	common, _ := souls.ReadFile("souls/common.md")
	for i := range roles {
		role, _ := souls.ReadFile("souls/" + roles[i].Name + ".md")
		roles[i].Soul = string(common) + "\n" + string(role)
	}
	return roles
}

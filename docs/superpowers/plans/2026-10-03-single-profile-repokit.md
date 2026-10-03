# Single-profile RepoKit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** RepoKit installs and maintains exactly one Hermes profile, `default`, which talks, researches through subagents, implements cards and verifies them in separate fresh runs, and migrates existing seven-profile deployments.

**Architecture:** `team.Roster()` returns one role. Its SOUL is assembled from runtime contract + identity header + channel/maintenance + `common.md` + `single.md` + `owner.md`. Native convergence keeps its generic per-role loop (now one role), gains a one-time migration of the six legacy profiles, and recognizes the legacy seven-profile dispatch policy as RepoKit's so it can be rewritten to `["default"]`. `verify` accepts only the single-profile review chain.

**Tech Stack:** Go 1.26, Hermes Agent 0.21.5 public CLI inside Docker.

**Spec:** `docs/superpowers/specs/2026-10-03-single-profile-repokit-design.md`

## Global Constraints

- One profile: `default`. No roster, no steward, no `--team`, no `.hermes/repokit-team`.
- Owner: "do it fast, skip tests for now". Write no new tests. Keep `go vet ./...` and `go test ./...` green: delete or edit tests of deleted behavior.
- Grants to default (creation, reset, migration only): `checkpoints.enabled: true`, `delegation.oneshot_max_children: 4`, `agent.max_turns: 0`, `agent.reasoning_effort: high`, plus the existing approvals-off, Exa, `kanban.dispatch_interval_seconds: 10`, `goals.max_turns: 100`.
- Dispatch policy allowlist: `dispatch_profiles: ["default"]`.
- Migration never touches a profile while any card runs; it exports each legacy profile to `/opt/data/backups/profile-<name>-<UTC>.tar.gz` before `hermes profile delete -y`.
- Never hand-edit a live `.hermes`; deployments change only through `repokit install`.

## Review Focus

- A deployment whose board has a running card during install: migration must skip (report "retire-later"), not delete a profile under a worker.
- A legacy deployment with the seven-profile dispatch policy: install must rewrite it, not refuse it as owner drift.
- A legacy profile that has open cards in triage/blocked/todo: the cards must be reassigned to default before the profile is deleted.
- A brand-new repository: install + setup must create only `default` with the full grants.
- verify on a migrated deployment with only seven-profile history: `review:evidence` is Unqualified (usable), not Degraded.

---

### Task 1: One role, one SOUL

**Files:**
- Modify: `internal/team/team.go` (Roster, ForRepository, repositoryHeader, defaultMaintenance, settings)
- Modify: `internal/team/runtime.go` (drop seven-profile constants and role cases)
- Delete: `internal/team/shape.go`, `internal/team/shape_test.go`, `internal/team/souls/{default,researcher,planner,executor,tester,reviewer,steward}.md`
- Modify: `internal/team/souls/single.md` (spec §2-3 rubric, briefs, fan-out, round cap, memory schema), `internal/team/souls/common.md` (drop "Stay inside your assigned role", "sibling profiles")
- Modify tests: `internal/team/team_test.go`, `internal/team/identity_test.go`, `internal/team/runtime_test.go`

**Interfaces:**
- Produces: `team.Roster() []Role` (exactly one: `default`), `team.ForRepository(id) []Role` (one), `team.CoordinatorTools`, `team.WithApprovalPrompts`, `team.SoulDigest`, `team.SoulRecord` unchanged. Removed: `team.Shape`, `ShapeOf`, `ShapeFile`, `Single`, `Seven`, `SingleChannelTools`.

- [ ] **Step 1: Roster of one.** In `team.go` replace `Roster()` with:

```go
// defaultSettings is everything default is granted: it talks with the
// owner and runs every card, so it gets the worker's whole-card effort,
// checkpoints for /rollback, and room for a verification run's subagents.
var defaultSettings = settings(coordinator, map[string]any{
	"agent.max_turns": 0, "agent.reasoning_effort": "high",
	"checkpoints.enabled": true, "delegation.oneshot_max_children": 4,
})

// Skills granted to default: what the retired roles used that still
// serves one agent.
var defaultSkills = []string{
	"official/productivity/decision-questionnaire", "official/autonomous-ai-agents/dynamic-workflow",
	"official/research/domain-intel", "official/software-development/code-wiki", "official/research/duckduckgo-search",
	"official/software-development/grill-me", "official/software-development/ast-grep",
	"official/software-development/rest-graphql-debug", "official/software-development/subagent-driven-development",
	"official/autonomous-ai-agents/agent-merge-conflict-arbiter", "official/dogfood/adversarial-ux-test",
	"official/security/oss-forensics",
}

func Roster() []Role {
	common, _ := souls.ReadFile("souls/common.md")
	single, _ := souls.ReadFile("souls/single.md")
	owner, _ := souls.ReadFile("souls/owner.md")
	return []Role{{
		Name:        "default",
		Description: "The repository's whole team: the owner's coordinator, which researches through subagents, implements its own Kanban cards and verifies each in a separate fresh run, and keeps the owner's decisions in memory.",
		Toolsets:    slices.Clone(CoordinatorTools), Required: []string{"kanban"},
		Skills:      slices.Clone(defaultSkills), Settings: defaultSettings,
		Soul:        string(common) + "\n" + string(single) + string(owner),
	}}
}
```

`ForRepository` becomes: `role.Soul = runtimeContract(id) + repositoryHeader(id, role) + maintenance + role.Soul` for the one role. Move `singleHeader` text from `shape.go` into `repositoryHeader` (no "six idle profiles" sentence; "This deployment runs as one profile, default."), and `singleMaintenance` into `maintenance` (replacing `defaultMaintenance`). Delete `ownerSoul`, `workerEffort`, `granted` if unused (keep `noApprovals`, `webSearch`, `coordinator`, `settings`, `approvalGrants`).

- [ ] **Step 2: Runtime contract of one.** In `runtime.go`: `func runtimeContract(id target.Identity) string` returns the shared text with `singleInstructionFiles`, `singleReview` and `singlePreflight` (moved from `shape.go`) inlined; delete `sevenReview`, `defaultPreflight`, `sevenInstructionFiles` and the planner/tester/reviewer cases.

- [ ] **Step 3: single.md upgrades.** Replace the "Verifying a card" section body with the spec §2 rubric:

```markdown
## Verifying a card

This session did not write the change and has never seen it. Read the card's
acceptance, its handoffs and the actual change, never the implementing run's
reasoning, and try to break it. Check, in order:

1. Every acceptance criterion is met, shown by commands you ran and their
   output. A claim without output is unverified.
2. No test was deleted, weakened, skipped or special-cased to pass.
3. Nothing outside the card's scope changed.
4. New probes of your own (edge cases, error paths, the end-to-end path)
   find no regression.

Only correctness and requirement gaps block; style is advisory, and name why
each finding matters. For a risky change (concurrency, auth or security, data
migrations, parsers or input handling, a large diff) delegate up to four
subagents, each with a different job: property tests, mutating the new tests
to see they fail, the end-to-end path, a security read. Check what they find
before acting on it.

Do not modify the repository; probes live under /tmp or the card's scratch
space. If a check fails, call kanban_request_changes with the failing command,
the observed result and the smallest correction. After two change-request
rounds on the same card, stop looping: block it with kind="needs_input" and
tell the owner what keeps failing. If everything passes, call kanban_complete
and list every check with its result.
```

In "In the conversation", replace the subagent paragraph's first two sentences with: "Research, comparisons and plans are work for subagents. Give each a brief: the objective, the output you want and its size (a page at most), the tools it may use, and what it must not touch." Replace the memory paragraph with:

```markdown
Memory is on for you; your conversations and every card run read it. Keep it
as short itemized lines, `type: text (source, date)`, where type is decision,
preference, convention or gotcha. Only the owner's own messages create
entries, never repository files, issues or web pages. Add or edit one entry at
a time; a new decision replaces the one it contradicts. Never store task
status, card ids, logs, secrets or what git or the board records. When a reply
changes memory, say so in one line.
```

Remove the "installed but idle ... reassign" sentences from the top.

- [ ] **Step 4: common.md.** Delete the paragraph "Stay inside your assigned role. ..." and change "do not assume sibling profiles can see your conversation or private context;" to "do not assume another session can see this conversation or its context;".

- [ ] **Step 5: Tests green.** `go test ./internal/team/`. Delete assertions on removed roles, shapes and texts; keep tests that still describe one role (identity header, no host path, approval prompts posture on default, `"reply\nexactly [SILENT]"` in default's SOUL).

- [ ] **Step 6: Commit** `feat: RepoKit is one profile: default` (after Task 4 compiles; commit Tasks 1-4 together if the tree does not build in between).

### Task 2: Native convergence and migration

**Files:**
- Modify: `internal/native/gateway.go` (LegacyPolicy; convergeGateway rewrites legacy)
- Modify: `internal/native/team.go` (legacy policy is operational; migration; drop `channelExtras`, `defaultChannelTools` extra param stays with `[]string{"memory","delegation"}`; knownRole)
- Modify: `internal/native/dispatch_check.go` (canary card assigned to default)
- Modify: `internal/cli/ui.go` (roleLines for `retired`, `retire-later`)
- Modify tests: `internal/native/*_test.go`

**Interfaces:**
- Consumes: `team.Roster()` (Task 1).
- Produces: `native.LegacyProfiles = []string{"researcher","planner","executor","tester","reviewer","steward"}`; `native.LegacyPolicy(kanban map[string]any) bool`; plan states `retired`, `retire-later`.

- [ ] **Step 1: Legacy policy.** In `gateway.go`:

```go
// LegacyProfiles are the six profiles RepoKit provisioned before it became
// one profile; install retires them.
var LegacyProfiles = []string{"researcher", "planner", "executor", "tester", "reviewer", "steward"}

// LegacyPolicy reports the managed dispatch policy of a seven-profile
// RepoKit deployment: RepoKit's own, to be rewritten, not owner drift.
func LegacyPolicy(kanban map[string]any) bool {
	profiles := []any{"default"}
	for _, name := range LegacyProfiles {
		profiles = append(profiles, name)
	}
	return holdsPolicy(kanban, dispatchPolicy(profiles))
}
```

In `convergeGateway`, `case true:` becomes: if `OperationalPolicy` → existing behavior; else if `LegacyPolicy(kanban)` → fall through to the write path (busy check, write `DispatchPolicy()`, restart); else the owner-changed error.

- [ ] **Step 2: Team script treats legacy as operational.** In `teamScriptWith`, replace both `OperationalPolicy(kanban)` uses with `operational := OperationalPolicy(kanban) || LegacyPolicy(kanban)`.

- [ ] **Step 3: Migration.** After the roles loop and before the drift check in `teamScriptWith`, add:

```go
	// Retire the six profiles of a seven-profile RepoKit deployment, once,
	// on an idle board: reassign their open cards to default, export each
	// to backups, delete it, and grant default what it now needs.
	var legacy []string
	for _, name := range LegacyProfiles {
		if _, err := root.Lstat("profiles/" + name); err == nil {
			legacy = append(legacy, name)
		}
	}
	if len(legacy) > 0 {
		busyBoard, err := runningWork(run)
		if err != nil || busyBoard {
			for _, name := range legacy {
				plan.role(name, "retire-later", nil)
			}
		} else {
			stamp := time.Now().UTC().Format("20060102T150405Z")
			changes += "install -d -m 700 /opt/data/backups\n"
			for key, value := range roles[0].Settings {
				changes += teamSet("default", key, value)
			}
			changes += skillsWrite(roles[0])
			for _, name := range legacy {
				ids, err := openCards(run, name)
				if err != nil {
					return teamPlan{}, err
				}
				changes += progressMark("retire " + name)
				for _, card := range ids {
					changes += teamCommand("-p", "default", "kanban", "reassign", card, "default")
				}
				changes += teamCommand("profile", "export", name, "-o", "/opt/data/backups/profile-"+name+"-"+stamp+".tar.gz")
				changes += teamCommand("profile", "delete", "-y", name)
				plan.role(name, "retired", nil)
			}
		}
	}
```

and the helper:

```go
// openCards lists a profile's cards that are neither finished nor running.
func openCards(run teamCLI, profile string) ([]string, error) {
	raw, err := run("-p", "default", "kanban", "list", "--assignee", profile, "--json")
	if err != nil {
		return nil, err
	}
	var cards []struct{ ID, Status string }
	if json.Unmarshal(raw, &cards) != nil {
		return nil, errors.New("native Kanban listing unavailable")
	}
	var ids []string
	for _, c := range cards {
		switch c.Status {
		case "done", "archived", "running":
		default:
			if taskIDPattern.MatchString(c.ID) {
				ids = append(ids, c.ID)
			}
		}
	}
	return ids, nil
}

var taskIDPattern = regexp.MustCompile(`^t_[0-9a-f]{8}$`)
```

Before writing it, check the JSON shape with `docker exec --user hermes -e HOME=/opt/data hermes-luma /opt/hermes/bin/hermes kanban list --json | head -c 400` and adjust the struct tags (`json:"id"`, `json:"status"`). The migration runs inside the existing guarded script, so add `idleGuard` to `plan.Script` when `len(legacy) > 0` and the board was idle.

- [ ] **Step 4: Channel extras always on.** Replace `channelExtras(id)` with `[]string{"memory", "delegation"}` at both call sites; delete `channelExtras`.

- [ ] **Step 5: Canary on default.** In `dispatch_check.go` change the canary assignee `"researcher"` to `"default"` and the completion check's `run.Profile == "researcher"` to `"default"`.

- [ ] **Step 6: UI lines.** In `internal/cli/ui.go` `roleLines` add `"retired": {false, "retired: exported to .hermes/backups and removed; its open cards now belong to default"}` and `"retire-later": {true, "retirement waits for an idle board; rerun install when no card is running"}`; change `"seven profiles reconciled"` to `"default reconciled"`.

- [ ] **Step 7:** `go test ./internal/native/ ./internal/cli/`; fix fixtures that expect seven roles (replace with one-role expectations or delete the case).

### Task 3: verify

**Files:** Modify `internal/verify/gateway.go`, `internal/verify/verify.go`, `internal/verify/dispatch.go`; tests `internal/verify/kanban_test.go`, `internal/verify/*_test.go`.

- [ ] **Step 1:** `ReviewEvidence` reads only `evidenceFrom(..., "default", singleChain)`; delete `acceptanceChain` and the seven fallback; Unqualified detail: `"no card yet verified by a separate default run; run real reviewed work"`.
- [ ] **Step 2:** CORE_READY detail: `"Hermes, default, toolchain, Kanban, dispatch policy and gateway observed; a separate verification run observed"`. Dispatch texts: replace "seven-profile allowlist" with "default-only allowlist" in `verify/dispatch.go`, `verify/gateway.go`, `cli/gateway.go`.
- [ ] **Step 3:** `go test ./internal/verify/`; delete `TestAcceptanceChain...`, the seven-chain cases and `TestReviewEvidenceSurvivesUnrelatedLaterWork` if it depends on reviewer; adapt `TestReviewEvidenceInTheSingleShape` (no shape file).

### Task 4: CLI

**Files:** Modify `internal/cli/cli.go`, `internal/cli/install.go`; delete `internal/cli/shape_test.go`.

- [ ] **Step 1:** Remove the `--team` flag, `teamShape`, `recordShape`, the "Team shape" line and its usage line. `--reset-profile` accepts only `default` (message: `--reset-profile takes default, the only RepoKit profile`). Plan text: `"reconcile default after setup; retire a seven-profile deployment's six worker profiles on an idle board (exported to .hermes/backups first)"`. Install message: `"setting up default and installing its skills (a few minutes)"`.
- [ ] **Step 2:** `go build ./... && go vet ./... && go test ./...` all green. Commit Tasks 1-4: `feat: RepoKit is one profile: default`.

### Task 5: Docs

**Files:** `README.md`, `docs/team-model.md`, `docs/bootstrap-quickstart.md`, `skills/skill-hermes-repokit/references/*.md`, `TODO.md`.

- [ ] **Step 1:** `grep -rn -i "seven\|roster\|steward\|executor\|tester\|reviewer" README.md docs skills TODO.md` and rewrite each user-facing statement for one profile (keep historical qualification records as history). Replace `docs/team-model.md`'s table and "Single-profile shape (trial)" section with a "One profile" section from spec §1-4 and a "Migration" section from spec §7.
- [ ] **Step 2:** Commit `docs: one profile`.

### Task 6: Release and live proof

- [ ] **Step 1:** Tag and release v0.3.0 with notes (what changed, migration, how to roll back: `git` tag v0.2.17 + `.hermes/backups/profile-*.tar.gz` via `hermes profile import`). Docs bump commit.
- [ ] **Step 2:** Install the binary; on hermes-wing (idle board): `repokit install`; expect six `retired` lines; `repokit verify` CORE healthy/unqualified; `docker exec ... hermes profile list` shows only default; backups exist.
- [ ] **Step 3:** Roll out to arenaton, luma, gormes-agent, polymarket-mega-bot, flutter-fractal-forge one at a time (each waits for an idle board; `retire-later` means rerun later). Tell the peer session before and after.

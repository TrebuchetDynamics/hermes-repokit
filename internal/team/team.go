// Package team defines RepoKit's stable, repository-agnostic identities.
package team

import "embed"

//go:embed souls/*.md
var souls embed.FS

type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Soul        string   `json:"soul"`
	Toolsets    []string `json:"toolsets"`
}

// Roster returns independent role values. Coarse file/terminal bundles mean
// specialist read-only boundaries are advisory in the qualified Hermes release.
func Roster() []Role {
	roles := []Role{
		{Name: "default", Description: "Primary human-facing repository coordinator and orchestrator. Understands user goals, answers lightweight questions directly, designs bounded Kanban workflows, assigns the appropriate team roles, establishes shared decisions, follows progress, and verifies that completed work has passed required review.", Toolsets: []string{"kanban", "memory"}},
		{Name: "researcher", Description: "Investigates repository context, external sources and prior project knowledge. Resolves unknowns, compares alternatives and produces source-backed findings without changing the target artifact.", Toolsets: []string{"file", "web", "memory"}},
		{Name: "planner", Description: "Turns goals, constraints and research into a bounded execution contract with scope, decisions, dependencies, acceptance criteria, verification requirements and known risks. Does not perform the planned work.", Toolsets: []string{"file", "memory"}},
		{Name: "executor", Description: "Produces one bounded repository artifact or change from an approved task contract, preserves unrelated state, performs appropriate verification and hands work to independent review when required.", Toolsets: []string{"file", "terminal", "memory"}},
		{Name: "reviewer", Description: "Independently evaluates completed work against its task contract, underlying artifacts and verification evidence. Approves or requests changes on the same Kanban card and does not implement the requested work.", Toolsets: []string{"file", "terminal", "memory"}},
		{Name: "steward", Description: "Maintains the repository's Hermes profile roster and agent capabilities. Creates, updates, configures, retires, backs up and, with explicit authorization, deletes profiles. Manages profile descriptions, SOUL contracts, skills, toolsets and profile distributions. Does not coordinate project work or modify project artifacts.", Toolsets: []string{"terminal", "file", "memory"}},
	}
	common, _ := souls.ReadFile("souls/common.md")
	for i := range roles {
		role, _ := souls.ReadFile("souls/" + roles[i].Name + ".md")
		roles[i].Soul = string(common) + "\n" + string(role)
	}
	return roles
}

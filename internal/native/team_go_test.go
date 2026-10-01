package native

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

func fakeTeamConfig(args ...string) ([]byte, error) {
	if len(args) == 3 && args[0] == "profile" && args[1] == "describe" {
		return []byte("(no description set for '" + args[2] + "')\n"), nil
	}
	key := args[len(args)-2]
	var value any
	switch key {
	case "kanban.dispatch_in_gateway":
		value = false
	case "model.default":
		value = "provider/model"
	case "toolsets":
		value = []string{"file"}
	case "skills.project_discovery":
		value = true
	}
	return json.Marshal(map[string]any{"value": value})
}
func TestTeamPreservesUnknownDefaultSoulBeforeAnyNativeWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hermes", "SOUL.md"), []byte("owner identity"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	plan, err := teamScript(target.Identity{Project: "repo-123", Name: "atlas"}, true, "", sectioned(fakeTeamConfig), root)
	if err != nil || plan.Script != "" || plan.Status != "drift" || strings.Join(plan.Drift, ",") != "default" {
		t.Fatalf("unknown owner SOUL not preserved: script=%q status=%q drift=%v err=%v", plan.Script, plan.Status, plan.Drift, err)
	}
}
func TestTeamNeedsExplicitSetupBeforeProvisioning(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	plan, err := teamScript(target.Identity{Project: "repo-123", Name: "atlas"}, false, "", sectioned(fakeTeamConfig), root)
	if err != nil || plan.Script != "" || plan.Status != "pending-setup" || len(plan.Drift) != 0 {
		t.Fatalf("unexpected early provisioning: %q %q %v %v", plan.Script, plan.Status, plan.Drift, err)
	}
}
func TestTeamConfigGetJSONWrapperAndOwnerSelection(t *testing.T) {
	got, err := configValue(fakeTeamConfig, "default", "toolsets")
	if err != nil || !holdsTeamValue(got, []string{"file"}) {
		t.Fatalf("native JSON config not decoded: %v %v", got, err)
	}
	if holdsTeamValue(got, []string{"memory"}) {
		t.Fatal("owner selection mistaken for managed selection")
	}
}

func TestTeamPlansSevenPublicNativeProfilesFromExactManagedDefault(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hermes", "SOUL.md"), []byte(roles[0].Soul), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hermes", "profile.yaml"), []byte("description: "+roles[0].Description+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	run := func(args ...string) ([]byte, error) {
		if args[0] == "profile" && args[1] == "describe" && args[2] == "default" {
			return []byte(roles[0].Description + "\n"), nil
		}
		key := args[len(args)-2]
		name := args[1]
		var value any
		switch key {
		case "model.default":
			value = "provider/model"
		case "toolsets":
			value = []string{"file", "memory"}
		case "skills.project_discovery":
			value = true
		default:
			for _, role := range roles {
				if role.Name == name {
					value = expectedTeamFields(role)[key]
					break
				}
			}
		}
		return json.Marshal(map[string]any{"value": value})
	}
	plan, err := teamScript(id, true, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || len(plan.Drift) != 0 {
		t.Fatalf("managed default not qualified: %q %v %v", plan.Status, plan.Drift, err)
	}
	if strings.Count(plan.Script, "'profile' 'create'") != 6 {
		t.Fatalf("expected six native worker creates: %s", plan.Script)
	}
	// RepoKit adds only Kanban to default's tools; the owner's memory choice is
	// preserved and never enabled by RepoKit.
	if !strings.Contains(plan.Script, `'-p' 'default' 'config' 'set' 'toolsets' '["file","memory","kanban"]'`) || strings.Contains(plan.Script, "'enable' 'kanban' 'memory'") {
		t.Fatalf("default tool reconciliation changed owner tools:\n%s", plan.Script)
	}
	for _, role := range roles[1:] {
		if !strings.Contains(plan.Script, "'profile' 'create' '"+role.Name+"'") {
			t.Fatalf("missing role %s", role.Name)
		}
	}
	if strings.Contains(plan.Script, "python") || strings.Contains(plan.Script, "repokit_maintenance") {
		t.Fatal("production plan contains private runtime code")
	}
}

func TestTeamResultDoesNotEchoUntrustedNativeOutput(t *testing.T) {
	_, err := teamResult("REPOKIT_TEAM={\"status\":\"drift\",\"drift\":[\"secret owner value\"]}\n")
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("untrusted native output escaped: %v", err)
	}
}

func TestDefaultChannelToolsCompletesOnlyMissingHumanChannels(t *testing.T) {
	run := func(args ...string) ([]byte, error) {
		return []byte(`{"cli":["file"],"telegram":["file","terminal"],"discord":["kanban"],"api_server":["file"]}`), nil
	}
	script, err := defaultChannelTools(run)
	if err != nil {
		t.Fatal(err)
	}
	// Kanban is RepoKit's; memory is the owner's and is neither required nor enabled.
	if !strings.Contains(script, "'tools' 'enable' 'kanban' '--platform' 'telegram'") || strings.Contains(script, "memory") || strings.Contains(script, "discord") || strings.Contains(script, "api_server") || strings.Contains(script, "'cli'") {
		t.Fatalf("unexpected channel reconciliation:\n%s", script)
	}
	bad := func(args ...string) ([]byte, error) { return []byte(`{"tele gram":["file"]}`), nil }
	if _, err := defaultChannelTools(bad); err == nil {
		t.Fatal("unsafe channel name accepted")
	}
}

func TestUnsetNativeConfigIsNullNotFailure(t *testing.T) {
	get := []string{"-p", "default", "config", "get", "model.default", "--json"}
	if !configUnset(get, "Config key not set: model.default\n") {
		t.Fatal("native unset report rejected")
	}
	for _, output := range []string{"Config key not set: other", "Traceback: boom", ""} {
		if configUnset(get, output) {
			t.Fatalf("failure treated as unset: %q", output)
		}
	}
	if configUnset([]string{"-p", "default", "kanban", "list"}, "Config key not set: model.default") {
		t.Fatal("non-config command treated as unset")
	}
}

func TestFreshDefaultWithoutSoulIsAdoptedNotDrift(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	plan, err := teamScript(target.Identity{Project: "repo-123", Name: "atlas"}, true, "", sectioned(fakeTeamConfig), root)
	if err != nil || plan.Status != "configured" || len(plan.Drift) != 0 {
		t.Fatalf("fresh default not adopted: %q %v %v", plan.Status, plan.Drift, err)
	}
	for _, want := range []string{"'profile' 'describe' 'default'", "/opt/data/SOUL.md", "'profile' 'create' 'executor'"} {
		if !strings.Contains(plan.Script, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if stockSoul("owner identity") {
		t.Fatal("owner SOUL treated as stock")
	}
}

func TestActivatedDefaultIsManagedOnlyWithCompletePolicy(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	role := team.ForRepository(id)[0]
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hermes", "SOUL.md"), []byte(role.Soul), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for maxInProgress, want := range map[float64]bool{1: true, 3: false} {
		kanban := map[string]any{"auto_subscribe_on_create": true, "notify_in_gateway": true}
		for _, f := range DispatchPolicy() {
			kanban[f.Key] = f.Value
		}
		kanban["max_in_progress"] = maxInProgress
		run := func(args ...string) ([]byte, error) {
			if args[0] == "profile" {
				return []byte(role.Description + "\n"), nil
			}
			key := args[len(args)-2]
			if key == "kanban" {
				return json.Marshal(kanban)
			}
			if value, ok := strings.CutPrefix(key, "kanban."); ok {
				return json.Marshal(kanban[value])
			}
			return json.Marshal(map[string]any{"terminal.cwd": "/workspace", "terminal.backend": "local"}[key])
		}
		if _, ok, err := inspectRole(sectioned(run), root, role); err != nil || ok != want {
			t.Fatalf("max_in_progress=%v: managed=%v err=%v", maxInProgress, ok, err)
		}
	}
}

func TestActivatedTeamWithOwnerAllowlistIsOnlyObserved(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	kanban := operationalKanban()
	kanban["dispatch_profiles"] = []any{"default", "executor", "reviewer"}
	souls := currentSouls(roles)
	delete(souls, "tester")
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: kanban, stats: `{"by_status":{"done":3}}`})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || strings.Contains(writes(plan.Script), "SOUL.md") || strings.Contains(plan.Script, "'profile' 'create'") {
		t.Fatalf("owner-changed team reprovisioned: %q %v %v\n%s", plan.Status, plan.Drift, err, plan.Script)
	}
	// An owner-changed policy is default drift; tester is missing.
	if strings.Join(plan.Drift, ",") != "default,tester" {
		t.Fatalf("drift must name the owner policy and missing tester: %v", plan.Drift)
	}
}

// teamFixture describes one existing native home. A role absent from souls has
// no profile; config overrides one role's native values.
type teamFixture struct {
	souls  map[string]string
	desc   map[string]string
	config map[string]map[string]any
	kanban map[string]any // default's kanban section; nil leaves dispatch off
	stats  string
}

func deployTeam(t *testing.T, roles []team.Role, f teamFixture) (*os.Root, teamCLI) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, soul := range f.souls {
		base := filepath.Join(dir, ".hermes")
		if name != "default" {
			base = filepath.Join(base, "profiles", name)
		}
		if err := os.MkdirAll(base, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "SOUL.md"), []byte(soul), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	run := func(args ...string) ([]byte, error) {
		if args[0] == "profile" && args[1] == "describe" {
			if d, ok := f.desc[args[2]]; ok {
				return []byte(d + "\n"), nil
			}
			for _, role := range roles {
				if role.Name == args[2] {
					return []byte(role.Description + "\n"), nil
				}
			}
		}
		name, key := args[1], args[len(args)-2]
		if key == "stats" {
			return []byte(f.stats), nil
		}
		if v, ok := f.config[name][key]; ok {
			return json.Marshal(map[string]any{"value": v})
		}
		var value any
		switch {
		case key == "kanban" && name == "default" && f.kanban != nil:
			value = f.kanban
		case strings.HasPrefix(key, "kanban.") && name == "default" && f.kanban != nil:
			value = f.kanban[strings.TrimPrefix(key, "kanban.")]
		case key == "model.default":
			value = "provider/model"
		case key == "toolsets" && name == "default":
			value = []string{"kanban", "memory"}
		case key == "platform_toolsets" && name == "default":
			value = map[string]any{"cli": []string{"kanban", "memory"}}
		case key == "skills.project_discovery":
			value = true
		default:
			for _, role := range roles {
				if role.Name == name {
					value = expectedTeamFields(role)[key]
				}
			}
		}
		return json.Marshal(map[string]any{"value": value})
	}
	return root, run
}

// currentSouls maps every role to its current managed SOUL.
func currentSouls(roles []team.Role) map[string]string {
	souls := map[string]string{}
	for _, role := range roles {
		souls[role.Name] = role.Soul
	}
	return souls
}

func operationalKanban() map[string]any {
	kanban := map[string]any{"auto_subscribe_on_create": true, "notify_in_gateway": true}
	for _, f := range DispatchPolicy() {
		kanban[f.Key] = f.Value
	}
	return kanban
}

func roleNamed(roles []team.Role, name string) team.Role {
	for _, role := range roles {
		if role.Name == name {
			return role
		}
	}
	panic(name)
}

// writes drops the lock-time state guard, which names every roster file it
// rechecks, leaving only the planned native writes.
func writes(script string) string {
	kept := []string{}
	for _, line := range strings.Split(script, "\n") {
		if !strings.HasPrefix(line, "[ ") && !strings.Contains(line, "sha256sum -c") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func TestCustomizedRoleDoesNotBlockRosterConvergence(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	souls["executor"] = "Prefer minimal patches. Never change public APIs without asking."
	delete(souls, "tester")
	// A changed description alone makes steward owner-controlled.
	root, run := deployTeam(t, roles, teamFixture{souls: souls, desc: map[string]string{"steward": "Owner steward"}})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || len(plan.Drift) != 0 {
		t.Fatalf("customization poisoned the roster: %q drift=%v err=%v", plan.Status, plan.Drift, err)
	}
	if strings.Join(plan.Customized, ",") != "executor,steward" {
		t.Fatalf("customized roles not reported: %v", plan.Customized)
	}
	if strings.Count(plan.Script, "'profile' 'create'") != 1 || !strings.Contains(plan.Script, "'profile' 'create' 'tester'") {
		t.Fatalf("missing tester not created alone:\n%s", plan.Script)
	}
	for _, name := range []string{"executor", "steward"} {
		if strings.Contains(writes(plan.Script), "/profiles/"+name+"/") || strings.Contains(plan.Script, "'-p' '"+name+"'") || strings.Contains(plan.Script, "'describe' '"+name+"'") {
			t.Fatalf("owner-customized %s written:\n%s", name, plan.Script)
		}
	}
}

func TestCustomizedDefaultIsPreservedWhenTeamIsManaged(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	souls["default"] = "Owner coordinator identity"
	root, run := deployTeam(t, roles, teamFixture{souls: souls})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || strings.Join(plan.Customized, ",") != "default" {
		t.Fatalf("customized default blocked a managed team: %q %v %v %v", plan.Status, plan.Drift, plan.Customized, err)
	}
	if strings.Contains(writes(plan.Script), "/opt/data/SOUL.md") {
		t.Fatalf("default overwritten:\n%s", plan.Script)
	}
}

func TestRequiredToolsetsAreASubset(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	executor := roleNamed(roles, "executor")
	added := append(append([]string{}, executor.Toolsets...), "owner-tool")
	root, run := deployTeam(t, roles, teamFixture{souls: currentSouls(roles), config: map[string]map[string]any{"executor": {"toolsets": added, "platform_toolsets.cli": added}}})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || len(plan.Drift)+len(plan.Customized) != 0 {
		t.Fatalf("owner tool addition treated as drift: %q %v %v %v", plan.Status, plan.Drift, plan.Customized, err)
	}
	if strings.Contains(plan.Script, "'-p' 'executor' 'config' 'set'") {
		t.Fatal("owner tool addition overwritten")
	}
	removed := executor.Toolsets[1:]
	root, run = deployTeam(t, roles, teamFixture{souls: currentSouls(roles), config: map[string]map[string]any{"executor": {"toolsets": removed}}})
	plan, err = teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "drift" || strings.Join(plan.Drift, ",") != "executor" || plan.Script != "" {
		t.Fatalf("missing required tool not reported before any write: %q %v %v", plan.Status, plan.Drift, err)
	}
}

func TestConvergedTeamPlansNoIdentityOrConfigurationWrites(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	for _, kanban := range []map[string]any{nil, operationalKanban()} {
		souls := currentSouls(roles)
		souls["planner"] = "Owner planner"
		root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: kanban, stats: `{"by_status":{"done":2}}`})
		plan, err := teamScript(id, false, "", sectioned(run), root)
		if err != nil || plan.Status != "configured" || len(plan.Drift) != 0 || strings.Join(plan.Customized, ",") != "planner" {
			t.Fatalf("converged team not stable: %q %v %v %v", plan.Status, plan.Drift, plan.Customized, err)
		}
		for _, write := range []string{"SOUL.md", "'profile' 'create'", "'config' 'set'", "'profile' 'describe'", "'tools' 'enable'"} {
			if strings.Contains(writes(plan.Script), write) {
				t.Fatalf("second convergence rewrites %s:\n%s", write, plan.Script)
			}
		}
	}
}

func TestTeamResultReportsPreservedAndResetRoles(t *testing.T) {
	report, err := teamResult(`REPOKIT_TEAM={"status":"configured","drift":[],"customized":["executor"],"reset":["tester"]}` + "\n")
	if err != nil || strings.Join(report.Customized, ",") != "executor" || strings.Join(report.Reset, ",") != "tester" {
		t.Fatalf("team report lost: %+v %v", report, err)
	}
	for _, bad := range []string{`{"status":"configured","customized":["secret owner value"]}`, `{"status":"configured","reset":["secret owner value"]}`} {
		if _, err := teamResult("REPOKIT_TEAM=" + bad + "\n"); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("untrusted role name accepted: %v", err)
		}
	}
}

func TestIdleGuardRefusesOnlyRunningWork(t *testing.T) {
	check := strings.SplitN(strings.Split(idleGuard, "\n")[1], "|", 2)[1]
	check = strings.TrimSuffix(strings.TrimSpace(check), "; then exit 3; fi")
	for stats, running := range map[string]bool{
		`{"by_status": {"running": 0, "done": 3}}`: false,
		`{"by_status":{"done":3}}`:                 false,
		`{"by_status": {"running": 2}}`:            true,
		`{"by_status":{"running":1}}`:              true,
	} {
		cmd := exec.Command("sh", "-c", check)
		cmd.Stdin = strings.NewReader(stats)
		err := cmd.Run()
		if (err == nil) != running {
			t.Fatalf("idle guard on %s: refused=%v", stats, err == nil)
		}
	}
}

func rowStates(plan teamPlan) string {
	rows := []string{}
	for _, row := range plan.Roles {
		rows = append(rows, row.Profile+"="+row.State+"["+strings.Join(row.Differs, ",")+"]")
	}
	return strings.Join(rows, " ")
}

func TestPlanRowsNameEveryProfileStateWithoutValues(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	souls["executor"] = "Owner executor"
	delete(souls, "tester")
	root, run := deployTeam(t, roles, teamFixture{souls: souls, desc: map[string]string{"steward": "Owner steward"}, config: map[string]map[string]any{"planner": {"terminal.cwd": "/private/owner/path"}}})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil {
		t.Fatal(err)
	}
	want := "default=current[] researcher=current[] planner=drift[terminal.cwd] executor=customized[SOUL] tester=missing[] reviewer=current[] steward=customized[description]"
	if got := rowStates(plan); got != want || plan.Status != "drift" {
		t.Fatalf("plan rows:\n got %s\nwant %s (status %s)", got, want, plan.Status)
	}
	encoded, _ := json.Marshal(plan.Roles)
	if strings.Contains(string(encoded), "/private/owner/path") || strings.Contains(string(encoded), "Owner") {
		t.Fatalf("plan rows expose owner values: %s", encoded)
	}
	for _, row := range plan.Roles {
		if row.Action == "" {
			t.Fatalf("%s has no action", row.Profile)
		}
	}
}

func TestResetProfileRestoresBaselineWithBackup(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	executor := roleNamed(roles, "executor")
	souls := currentSouls(roles)
	souls["executor"] = "Owner executor"
	souls["steward"] = "Owner steward"
	root, run := deployTeam(t, roles, teamFixture{souls: souls, config: map[string]map[string]any{"executor": {"toolsets": executor.Toolsets[1:]}}})
	// Without reset, executor's missing required tool blocks the run.
	if plan, err := teamScript(id, false, "", sectioned(run), root); err != nil || plan.Status != "drift" {
		t.Fatalf("drift not reported before reset: %q %v", plan.Status, err)
	}
	plan, err := teamScript(id, false, "executor", sectioned(run), root)
	if err != nil || plan.Status != "configured" {
		t.Fatalf("reset not planned: %q %v %v", plan.Status, plan.Drift, err)
	}
	if !strings.Contains(rowStates(plan), "executor=reset[SOUL,toolsets]") || !strings.Contains(rowStates(plan), "steward=customized[SOUL]") {
		t.Fatalf("reset rows: %s", rowStates(plan))
	}
	script := writes(plan.Script)
	for _, want := range []string{
		"cp -p '/opt/data/profiles/executor/SOUL.md' '/opt/data/profiles/executor/SOUL.md'.before-reset-",
		"cp -p '/opt/data/profiles/executor/config.yaml'",
		soulWrite("executor", executor.Soul),
		"'profile' 'describe' 'executor' '--text'",
		"'-p' 'executor' 'config' 'set' 'toolsets'",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("reset missing %q:\n%s", want, script)
		}
	}
	if strings.Index(script, "before-reset-") > strings.Index(script, soulWrite("executor", executor.Soul)) {
		t.Fatal("backup must precede the SOUL rewrite")
	}
	if strings.Contains(script, "/profiles/steward/") || strings.Contains(script, "'model") || strings.Contains(script, "provider") {
		t.Fatalf("reset touched another profile or owner model state:\n%s", script)
	}
}

func TestResetDefaultRestoresIdentityOnly(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	souls["default"] = "Owner coordinator"
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: `{"by_status":{}}`})
	plan, err := teamScript(id, false, "default", sectioned(run), root)
	if err != nil || !strings.Contains(rowStates(plan), "default=reset[SOUL]") {
		t.Fatalf("default reset not planned: %s %v", rowStates(plan), err)
	}
	script := writes(plan.Script)
	if !strings.Contains(script, soulWrite("default", roles[0].Soul)) || strings.Contains(script, "'-p' 'default' 'config' 'set'") {
		t.Fatalf("default reset must restore identity only:\n%s", script)
	}
}

func TestResetRefusesUnsafeOrUnknownTargets(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	root, run := deployTeam(t, roles, teamFixture{souls: currentSouls(roles), kanban: operationalKanban(), stats: `{"by_status":{"running":1}}`})
	if _, err := teamScript(id, false, "executor", sectioned(run), root); err == nil || !strings.Contains(err.Error(), "running") {
		t.Fatalf("reset under a running worker: %v", err)
	}
	owner := operationalKanban()
	owner["max_in_progress"] = float64(3)
	root, run = deployTeam(t, roles, teamFixture{souls: currentSouls(roles), kanban: owner, stats: `{"by_status":{}}`})
	if _, err := teamScript(id, false, "executor", sectioned(run), root); err == nil {
		t.Fatal("reset under an owner-changed dispatch policy")
	}
	root, run = deployTeam(t, roles, teamFixture{souls: currentSouls(roles)})
	if _, err := teamScript(id, false, "flutter-specialist", sectioned(run), root); err == nil {
		t.Fatal("reset accepted an owner-created profile")
	}
	souls := currentSouls(roles)
	delete(souls, "tester")
	root, run = deployTeam(t, roles, teamFixture{souls: souls})
	if plan, err := teamScript(id, false, "tester", sectioned(run), root); err != nil || !strings.Contains(rowStates(plan), "tester=missing[]") {
		t.Fatalf("reset of a missing role must create it: %s %v", rowStates(plan), err)
	}
}

// An operational team gets missing roles only while no card runs; a busy
// board is observed and nothing is written.
func TestOperationalTeamCreatesMissingRolesOnlyWhenIdle(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	delete(souls, "tester")
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: `{"by_status":{"done":3}}`})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || !strings.Contains(plan.Script, "'profile' 'create' 'tester'") || !strings.Contains(plan.Script, idleGuard) {
		t.Fatalf("idle operational team not reconciled or unguarded: %q %v %v\n%s", plan.Status, plan.Drift, err, plan.Script)
	}
	root, run = deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: `{"by_status":{"running":1}}`})
	plan, err = teamScript(id, false, "", sectioned(run), root)
	if err != nil || strings.Contains(plan.Script, "'profile' 'create'") || strings.Contains(writes(plan.Script), "SOUL.md") {
		t.Fatalf("profiles written under a running worker: %v\n%s", err, plan.Script)
	}
}

// Memory is granted to a new specialist but never required: removing it from
// an existing profile is the owner's choice, not drift, and is not re-added.
func TestMemoryToolIsGrantedNotRequired(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	config := map[string]map[string]any{}
	for _, role := range roles[1:] {
		without := []string{}
		for _, name := range role.Toolsets {
			if name != "memory" {
				without = append(without, name)
			}
		}
		if len(without) == len(role.Toolsets) {
			t.Fatalf("%s is not granted memory", role.Name)
		}
		config[role.Name] = map[string]any{"toolsets": without, "platform_toolsets.cli": without}
	}
	root, run := deployTeam(t, roles, teamFixture{souls: currentSouls(roles), config: config})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || len(plan.Drift)+len(plan.Customized) != 0 {
		t.Fatalf("owner-removed memory treated as drift: %q %v %v %v", plan.Status, plan.Drift, plan.Customized, err)
	}
	if strings.Contains(writes(plan.Script), "'config' 'set'") {
		t.Fatalf("memory re-added to an existing profile:\n%s", plan.Script)
	}
	tester := roleNamed(roles, "tester")
	souls := currentSouls(roles)
	delete(souls, "tester")
	root, run = deployTeam(t, roles, teamFixture{souls: souls})
	plan, err = teamScript(id, false, "", sectioned(run), root)
	granted, _ := json.Marshal(tester.Toolsets)
	if err != nil || !strings.Contains(plan.Script, "'-p' 'tester' 'config' 'set' 'toolsets' '"+string(granted)+"'") || !strings.Contains(string(granted), "memory") {
		t.Fatalf("new tester not granted its role toolsets with memory: %v\n%s", err, plan.Script)
	}
}

// A profile keeping only its Required toolsets is compatible: the rest of the
// granted baseline (memory, broader tools) is the owner's to remove.
func TestOnlyRequiredToolsetsAreChecked(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	researcher := roleNamed(roles, "researcher")
	root, run := deployTeam(t, roles, teamFixture{souls: currentSouls(roles), config: map[string]map[string]any{"researcher": {"toolsets": researcher.Required, "platform_toolsets.cli": researcher.Required}}})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || len(plan.Drift) != 0 {
		t.Fatalf("owner-trimmed granted tools treated as drift: %q %v %v", plan.Status, plan.Drift, err)
	}
	if strings.Contains(plan.Script, "'-p' 'researcher' 'config' 'set' 'toolsets'") {
		t.Fatal("granted tools re-added to an existing profile")
	}
}

func TestCreateAndResetInstallAndCheckGrantedSkills(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	delete(souls, "tester")
	root, run := deployTeam(t, roles, teamFixture{souls: souls})
	plan, err := teamScript(id, false, "executor", sectioned(run), root)
	if err != nil {
		t.Fatal(err)
	}
	script := writes(plan.Script)
	for _, want := range []string{
		"hermes -p 'tester' skills install 'official/dogfood/adversarial-ux-test' --yes",
		"hermes -p 'executor' skills install 'official/software-development/ast-grep' --yes",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("missing %q:\n%s", want, script)
		}
	}
	// The check line starts with "[", which writes() drops with the guard.
	if !strings.Contains(plan.Script, "[ -f '/opt/data/profiles/tester/skills/dogfood/adversarial-ux-test/SKILL.md' ] || printf") || !strings.Contains(plan.Script, "REPOKIT_SKILL_MISSING=executor official/software-development/ast-grep") {
		t.Fatal("installed skill not checked")
	}
	if strings.Contains(script, "hermes -p 'reviewer' skills install") {
		t.Fatal("skills installed into an existing, unreset profile")
	}
}

func TestMissingSkillReportsAreValidated(t *testing.T) {
	out := "REPOKIT_SKILL_MISSING=executor official/software-development/ast-grep\n" +
		"REPOKIT_SKILL_MISSING=executor official/finance/dcf-model\n" +
		"REPOKIT_SKILL_MISSING=owner secret\n" +
		`REPOKIT_TEAM={"status":"configured","drift":[]}` + "\n"
	report, err := teamResult(out)
	if err != nil || strings.Join(report.MissingSkills, ",") != "executor official/software-development/ast-grep" {
		t.Fatalf("missing skills: %v %v", report.MissingSkills, err)
	}
}

// An untouched SOUL from the previous release is RepoKit's: it upgrades in
// place while no card runs and waits while one does. Any other SOUL stays the
// owner's.
func TestPreviousReleaseSoulUpgradesOnlyWhenIdle(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	steward := roleNamed(roles, "steward")
	if steward.PreviousSoul == "" || steward.PreviousSoul == steward.Soul {
		t.Fatal("fixture needs a steward SOUL that changed since the previous release")
	}
	souls := currentSouls(roles)
	souls["steward"] = steward.PreviousSoul
	for _, c := range []struct {
		stats, want string
		writes      bool
	}{
		{`{"by_status":{}}`, "steward=upgrade[SOUL]", true},
		{`{"by_status":{"running":1}}`, "steward=deferred[SOUL]", false},
	} {
		root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: c.stats})
		plan, err := teamScript(id, false, "", sectioned(run), root)
		if err != nil || !strings.Contains(rowStates(plan), c.want) || len(plan.Customized)+len(plan.Drift) != 0 {
			t.Fatalf("%s: %s %v", c.stats, rowStates(plan), err)
		}
		if got := strings.Contains(writes(plan.Script), soulWrite("steward", steward.Soul)); got != c.writes {
			t.Fatalf("%s: current steward SOUL written=%v:\n%s", c.stats, got, plan.Script)
		}
	}
	souls["steward"] = steward.PreviousSoul + "\nowner edit"
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: `{"by_status":{}}`})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || strings.Join(plan.Customized, ",") != "steward" || strings.Contains(writes(plan.Script), "profiles/steward/SOUL.md") {
		t.Fatalf("edited previous SOUL must stay the owner's: %s %v", rowStates(plan), err)
	}
}

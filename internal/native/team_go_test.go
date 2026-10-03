package native

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
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
	script, err := defaultChannelTools(run, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Kanban is RepoKit's; memory is the owner's and is neither required nor enabled.
	if !strings.Contains(script, "'tools' 'enable' 'kanban' '--platform' 'telegram'") || strings.Contains(script, "memory") || strings.Contains(script, "discord") || strings.Contains(script, "api_server") || strings.Contains(script, "'cli'") {
		t.Fatalf("unexpected channel reconciliation:\n%s", script)
	}
	bad := func(args ...string) ([]byte, error) { return []byte(`{"tele gram":["file"]}`), nil }
	if _, err := defaultChannelTools(bad, nil); err == nil {
		t.Fatal("unsafe channel name accepted")
	}
	// When default is the whole team it also needs memory and delegation on
	// every human channel that lists its tools without them.
	single, err := defaultChannelTools(run, []string{"memory", "delegation"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"'enable' 'memory' '--platform' 'telegram'", "'enable' 'delegation' '--platform' 'telegram'", "'enable' 'memory' '--platform' 'discord'", "'enable' 'kanban' '--platform' 'telegram'"} {
		if !strings.Contains(single, want) {
			t.Errorf("single-shape channels missing %s:\n%s", want, single)
		}
	}
	if strings.Contains(single, "'enable' 'kanban' '--platform' 'discord'") || strings.Contains(single, "api_server") {
		t.Fatalf("single-shape reconciliation touched a channel it should not:\n%s", single)
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
	for _, want := range []string{"'profile' 'describe' 'default'", "/opt/data/SOUL.md"} {
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
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: kanban, stats: `{"by_status":{"done":3}}`})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || plan.Status != "configured" || strings.Contains(writes(plan.Script), "SOUL.md") || strings.Contains(plan.Script, "'profile' 'create'") {
		t.Fatalf("owner-changed team reprovisioned: %q %v %v\n%s", plan.Status, plan.Drift, err, plan.Script)
	}
	// An owner-changed policy is default drift.
	if strings.Join(plan.Drift, ",") != "default" {
		t.Fatalf("drift must name the owner policy: %v", plan.Drift)
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

func TestConvergedTeamPlansNoIdentityOrConfigurationWrites(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	for _, kanban := range []map[string]any{nil, operationalKanban()} {
		souls := currentSouls(roles)
		root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: kanban, stats: `{"by_status":{"done":2}}`})
		plan, err := teamScript(id, false, "", sectioned(run), root)
		if err != nil || plan.Status != "configured" || len(plan.Drift) != 0 || len(plan.Customized) != 0 {
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
	report, err := teamResult(`REPOKIT_TEAM={"status":"configured","drift":[],"customized":["default"],"reset":["default"]}` + "\n")
	if err != nil || strings.Join(report.Customized, ",") != "default" || strings.Join(report.Reset, ",") != "default" {
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

func TestResetDefaultRestoresIdentityAndGrantedSettingsOnly(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	souls := currentSouls(roles)
	souls["default"] = "Owner coordinator"
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: `{"by_status":{}}`})
	plan, err := teamScript(id, false, "default", sectioned(run), root)
	if err != nil || !strings.Contains(rowStates(plan), "default=reset[SOUL]") {
		t.Fatalf("default reset not planned: %s %v", rowStates(plan), err)
	}
	// Identity plus default's granted settings (approvals off, 10s dispatch
	// tick); never the owner's model, provider, toolsets or channels.
	script := writes(plan.Script)
	sets := regexp.MustCompile(`'-p' 'default' 'config' 'set' '([^']+)'`).FindAllStringSubmatch(script, -1)
	keys := []string{}
	for _, m := range sets {
		keys = append(keys, m[1])
	}
	sort.Strings(keys)
	if !strings.Contains(script, soulWrite("default", roles[0].Soul)) || strings.Join(keys, ",") != "agent.max_turns,agent.reasoning_effort,approvals.mode,checkpoints.enabled,delegation.oneshot_max_children,goals.max_turns,kanban.dispatch_interval_seconds,security.protected_instruction_files,web.backend,web.provider_tier.exa" {
		t.Fatalf("default reset must restore identity and granted settings only (%v):\n%s", keys, script)
	}
}

func TestMissingSkillReportsAreValidated(t *testing.T) {
	out := "REPOKIT_SKILL_MISSING=default official/software-development/ast-grep\n" +
		"REPOKIT_SKILL_MISSING=default official/finance/dcf-model\n" +
		"REPOKIT_SKILL_MISSING=owner secret\n" +
		`REPOKIT_TEAM={"status":"configured","drift":[]}` + "\n"
	report, err := teamResult(out)
	if err != nil || strings.Join(report.MissingSkills, ",") != "default official/software-development/ast-grep" {
		t.Fatalf("missing skills: %v %v", report.MissingSkills, err)
	}
}

// Every SOUL RepoKit writes is recorded, and a current SOUL that predates
// records is claimed by recording it, so the next SOUL change upgrades it.
func TestSoulWritesAreRecordedAndCurrentSoulsClaimed(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	def := roleNamed(roles, "default")
	if w := soulWrite("default", def.Soul); !strings.Contains(w, team.SoulDigest(def.Soul)) || !strings.Contains(w, "/opt/data/"+team.SoulRecord) {
		t.Fatalf("SOUL write not recorded:\n%s", w)
	}
	root, run := deployTeam(t, roles, teamFixture{souls: currentSouls(roles), kanban: operationalKanban(), stats: `{"by_status":{}}`})
	plan, err := teamScript(id, false, "", sectioned(run), root)
	if err != nil || !strings.Contains(plan.Script, recordWrite("default", def.Soul)) || strings.Contains(writes(plan.Script), soulWrite("default", def.Soul)) {
		t.Fatalf("current unrecorded SOUL not claimed (or rewritten): %v\n%s", err, plan.Script)
	}
}

// A team installed by an earlier RepoKit build carries different SOUL text;
// its repository ID line still proves the team exists, so the next install
// never reports it as unset; another repository's SOULs are still not adopted.
func TestEarlierBuildTeamIsRecognizedNotPendingSetup(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	for _, foreign := range []bool{false, true} {
		dir := t.TempDir()
		for _, role := range team.ForRepository(id) {
			path := filepath.Join(dir, ".hermes", "profiles", role.Name, "SOUL.md")
			if role.Name == "default" {
				path = filepath.Join(dir, ".hermes", "SOUL.md")
			}
			os.MkdirAll(filepath.Dir(path), 0700)
			soul := strings.Replace(role.Soul, "\n# RepoKit Agent Identity\n", "\n# RepoKit Agent Identity\n\nAn earlier build's wording.\n", 1)
			if foreign {
				soul = strings.Replace(soul, "Stable repository ID: repo-123", "Stable repository ID: other-456", 1)
			}
			if err := os.WriteFile(path, []byte(soul), 0600); err != nil {
				t.Fatal(err)
			}
		}
		root, err := os.OpenRoot(dir)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := teamScript(id, false, "", sectioned(fakeTeamConfig), root)
		root.Close()
		if foreign {
			if err != nil || plan.Status != "pending-setup" {
				t.Fatalf("another repository's SOULs adopted: %q %v", plan.Status, err)
			}
			continue
		}
		// The fake answers no team settings, so rows read drift here; what
		// matters is that the team is recognized rather than reported unset.
		if err != nil || plan.Status == "pending-setup" || !strings.Contains(rowStates(plan), "default=") {
			t.Fatalf("earlier-build team not recognized: %q %s %v", plan.Status, rowStates(plan), err)
		}
	}
}

func ptr(b bool) *bool { return &b }

// Default is granted its full toolset on every channel when it is adopted or
// reset, never on an ordinary run, so tools the owner switches off stay off.
func TestCoordinatorToolsAreGrantedOnResetOnly(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	grant := "'-p' 'default' 'tools' 'enable' '" + strings.Join(team.CoordinatorTools, "' '") + "' '--platform' 'cli'"
	souls := currentSouls(roles)
	souls["default"] = "Owner coordinator"
	root, run := deployTeam(t, roles, teamFixture{souls: souls, kanban: operationalKanban(), stats: `{"by_status":{}}`})
	plan, err := teamScript(id, false, "default", sectioned(run), root)
	if err != nil || !strings.Contains(plan.Script, grant) {
		t.Fatalf("default reset did not grant its toolset: %v\n%s", err, plan.Script)
	}
	root, run = deployTeam(t, roles, teamFixture{souls: currentSouls(roles), kanban: operationalKanban(), stats: `{"by_status":{}}`})
	plan, err = teamScript(id, false, "", sectioned(run), root)
	if err != nil || strings.Contains(plan.Script, "'tools' 'enable' 'web'") {
		t.Fatalf("an ordinary run re-granted default's toolset: %v\n%s", err, plan.Script)
	}
}

// oh-my-hermes setup runs again only when RepoKit's OMH setup changes: an
// install after the current record exists writes nothing, while a deployment
// set up by an earlier record (3.0.0, which left OMH memory on) runs it once.
func TestTeamRunsOMHSetupOncePerRecord(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hermes"), 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	for _, record := range []string{"", development.OMHVersion + "\n"} {
		if record != "" {
			if err := os.WriteFile(filepath.Join(dir, ".hermes", omhRecord), []byte(record), 0600); err != nil {
				t.Fatal(err)
			}
		}
		plan, err := teamScript(id, true, "", sectioned(fakeTeamConfig), root)
		if err != nil || !strings.Contains(plan.Script, "--core --default-executor hermes --memory-mode off") {
			t.Fatalf("record %q did not run OMH setup: err=%v", record, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".hermes", omhRecord), []byte(omhSetup+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := teamScript(id, true, "", sectioned(fakeTeamConfig), root)
	if err != nil || strings.Contains(plan.Script, "omh setup") {
		t.Fatalf("current record set up again: err=%v", err)
	}
}

// The OMH step clears default's memory provider only while it is still the
// omh an earlier setup chose; a provider the owner picked since is kept. The
// record is written only when OMH's setup succeeds.
func TestOMHStepKeepsOwnerMemoryProvider(t *testing.T) {
	for _, tc := range []struct {
		provider string
		setupOK  bool
		unset    bool
		recorded bool
	}{
		{"omh", true, true, true},
		{"holographic", true, false, true},
		{"", true, false, true},
		{"omh", false, false, false},
	} {
		data, bin := t.TempDir(), t.TempDir()
		log := filepath.Join(data, "calls")
		setupExit := "0"
		if !tc.setupOK {
			setupExit = "1"
		}
		stubs := map[string]string{
			"omh": "#!/bin/sh\nexit " + setupExit + "\n",
			"hermes": "#!/bin/sh\necho \"$*\" >> " + log + "\n" +
				"case \"$*\" in *'config get memory.provider'*) printf '%s\\n' " + shellQuote(tc.provider) + ";; *'gateway status'*) echo 'Gateway is not running';; esac\n",
		}
		for name, body := range stubs {
			if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0700); err != nil {
				t.Fatal(err)
			}
		}
		script := strings.ReplaceAll(omhWrite(), "/opt/data", data)
		cmd := exec.Command("/bin/sh", "-c", script)
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%+v: %v %s", tc, err, out)
		}
		calls, _ := os.ReadFile(log)
		record, err := os.ReadFile(filepath.Join(data, omhRecord))
		if strings.Contains(string(calls), "config unset memory.provider") != tc.unset || (err == nil) != tc.recorded || (tc.recorded && strings.TrimSpace(string(record)) != omhSetup) {
			t.Fatalf("%+v: calls=%q record=%q", tc, calls, record)
		}
	}
}

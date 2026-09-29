package native

import (
	"encoding/json"
	"os"
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
	script, status, drift, err := teamScript(target.Identity{Project: "repo-123", Name: "atlas"}, true, fakeTeamConfig, root)
	if err != nil || script != "" || status != "drift" || strings.Join(drift, ",") != "default" {
		t.Fatalf("unknown owner SOUL not preserved: script=%q status=%q drift=%v err=%v", script, status, drift, err)
	}
}
func TestTeamNeedsExplicitSetupBeforeProvisioning(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	script, status, drift, err := teamScript(target.Identity{Project: "repo-123", Name: "atlas"}, false, fakeTeamConfig, root)
	if err != nil || script != "" || status != "pending-setup" || len(drift) != 0 {
		t.Fatalf("unexpected early provisioning: %q %q %v %v", script, status, drift, err)
	}
}
func TestTeamConfigGetJSONWrapperAndOwnerSelection(t *testing.T) {
	got, err := configValue(fakeTeamConfig, "default", "toolsets")
	if err != nil || !equalTeamValue(got, []string{"file"}) {
		t.Fatalf("native JSON config not decoded: %v %v", got, err)
	}
	if equalTeamValue(got, []string{"memory"}) {
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
			value = []string{"kanban", "memory"}
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
	script, status, drift, err := teamScript(id, true, run, root)
	if err != nil || status != "configured" || len(drift) != 0 {
		t.Fatalf("managed default not qualified: %q %v %v", status, drift, err)
	}
	if strings.Count(script, "'profile' 'create'") != 6 {
		t.Fatalf("expected six native worker creates: %s", script)
	}
	for _, role := range roles[1:] {
		if !strings.Contains(script, "'profile' 'create' '"+role.Name+"'") {
			t.Fatalf("missing role %s", role.Name)
		}
	}
	if strings.Contains(script, "python") || strings.Contains(script, "repokit_maintenance") {
		t.Fatal("production plan contains private runtime code")
	}
}

func TestTeamResultDoesNotEchoUntrustedNativeOutput(t *testing.T) {
	err := teamResult("REPOKIT_TEAM={\"status\":\"drift\",\"drift\":[\"secret owner value\"]}\n")
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("untrusted native output escaped: %v", err)
	}
}

func TestDefaultChannelToolsCompletesOnlyMissingHumanChannels(t *testing.T) {
	run := func(args ...string) ([]byte, error) {
		return []byte(`{"cli":["file"],"telegram":["file","terminal"],"discord":["kanban","memory"],"api_server":["file"]}`), nil
	}
	script, err := defaultChannelTools(run)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "'tools' 'enable' 'kanban' 'memory' '--platform' 'telegram'") || strings.Contains(script, "discord") || strings.Contains(script, "api_server") || strings.Contains(script, "'cli'") {
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
	script, status, drift, err := teamScript(target.Identity{Project: "repo-123", Name: "atlas"}, true, fakeTeamConfig, root)
	if err != nil || status != "configured" || len(drift) != 0 {
		t.Fatalf("fresh default not adopted: %q %v %v", status, drift, err)
	}
	for _, want := range []string{"'profile' 'describe' 'default'", "/opt/data/SOUL.md", "'profile' 'create' 'executor'"} {
		if !strings.Contains(script, want) {
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
		if _, ok, err := inspectRole(run, root, role); err != nil || ok != want {
			t.Fatalf("max_in_progress=%v: managed=%v err=%v", maxInProgress, ok, err)
		}
	}
}

// sixRoleDeployment writes an activated six-profile release: each existing
// role holds its exact six-role SOUL and default holds the six-role policy.
func sixRoleDeployment(t *testing.T, roles []team.Role, allowlist []any, stats string) (*os.Root, teamCLI) {
	t.Helper()
	dir := t.TempDir()
	for _, role := range roles {
		if role.Name == "tester" {
			continue
		}
		base := filepath.Join(dir, ".hermes")
		if role.Name != "default" {
			base = filepath.Join(base, "profiles", role.Name)
		}
		if err := os.MkdirAll(base, 0700); err != nil {
			t.Fatal(err)
		}
		soul := role.PreviousManagedSouls[len(role.PreviousManagedSouls)-1]
		if err := os.WriteFile(filepath.Join(base, "SOUL.md"), []byte(soul), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	kanban := map[string]any{"auto_subscribe_on_create": true, "notify_in_gateway": true}
	for _, f := range DispatchPolicy() {
		kanban[f.Key] = f.Value
	}
	kanban["dispatch_profiles"] = allowlist
	run := func(args ...string) ([]byte, error) {
		if args[0] == "profile" && args[1] == "describe" {
			for _, role := range roles {
				if role.Name == args[2] {
					return []byte(role.Description + "\n"), nil
				}
			}
		}
		name, key := args[1], args[len(args)-2]
		if key == "stats" {
			return []byte(stats), nil
		}
		var value any
		switch {
		case key == "kanban" && name == "default":
			value = kanban
		case strings.HasPrefix(key, "kanban.") && name == "default":
			value = kanban[strings.TrimPrefix(key, "kanban.")]
		case key == "model.default":
			value = "provider/model"
		case key == "toolsets" && name == "default":
			value = []string{"kanban", "memory"}
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

func TestActivatedSixRoleTeamUpgradesInPlace(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	six := []any{"default", "researcher", "planner", "executor", "reviewer", "steward"}
	root, run := sixRoleDeployment(t, roles, six, `{"by_status":{"running":1}}`)
	if script, _, _, err := teamScript(id, false, run, root); err == nil || script != "" {
		t.Fatal("upgrade rewrote SOULs under a running worker")
	}
	root, run = sixRoleDeployment(t, roles, six, `{"by_status":{"done":3}}`)
	script, status, drift, err := teamScript(id, false, run, root)
	if err != nil || status != "configured" || len(drift) != 0 {
		t.Fatalf("six-role team not upgradable: %q %v %v", status, drift, err)
	}
	if !strings.Contains(script, `grep -q '"running"'`) {
		t.Fatal("upgrade must re-check running work under the lock")
	}
	if strings.Count(script, "'profile' 'create'") != 1 || !strings.Contains(script, "'profile' 'create' 'tester'") {
		t.Fatalf("expected only tester to be created:\n%s", script)
	}
	for _, role := range roles {
		if role.Name != "tester" && !strings.Contains(script, soulWrite(role.Name, role.Soul)) {
			t.Fatalf("%s kept its six-role SOUL", role.Name)
		}
	}
}

func TestActivatedTeamWithOwnerAllowlistIsOnlyObserved(t *testing.T) {
	id := target.Identity{Project: "repo-123", Name: "atlas"}
	roles := team.ForRepository(id)
	root, run := sixRoleDeployment(t, roles, []any{"default", "executor", "reviewer"}, `{"by_status":{"done":3}}`)
	script, status, drift, err := teamScript(id, false, run, root)
	if err != nil || status != "configured" || strings.Contains(script, "SOUL.md") || strings.Contains(script, "'profile' 'create'") {
		t.Fatalf("owner-changed team reprovisioned: %q %v %v\n%s", status, drift, err, script)
	}
	// An owner-changed policy is default drift, as before; tester is missing.
	if strings.Join(drift, ",") != "default,tester" {
		t.Fatalf("drift must name the owner policy and missing tester: %v", drift)
	}
}

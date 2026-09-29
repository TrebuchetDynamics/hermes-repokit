package native

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

var ErrTeamPending = errors.New("team setup pending: configure the default model privately, then run setup --team")

// teamCLI is the public Hermes surface. Output never enters an error message:
// config values and native diagnostics may contain private owner state.
type teamCLI func(...string) ([]byte, error)

func nativeTeamCLI(ctx context.Context, id target.Identity, dockerContext string, r InputRunner) teamCLI {
	return func(args ...string) ([]byte, error) {
		argv := []string{"--context", dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--workdir", "/workspace", "hermes", "hermes"}
		result := r.RunInput(ctx, nil, "docker", append(argv, args...)...)
		if result.Err != nil && !result.Truncated && configUnset(args, result.Output) {
			return []byte("null"), nil
		}
		if result.Err != nil || result.Truncated {
			return nil, errors.New("native team inspection unavailable")
		}
		return []byte(result.Output), nil
	}
}

// configUnset recognizes the pinned CLI's exit-1 report for an unset key:
// `config get <key>` prints exactly "Config key not set: <key>".
func configUnset(args []string, output string) bool {
	for i := 0; i+2 < len(args); i++ {
		if args[i] == "config" && args[i+1] == "get" {
			return strings.TrimSpace(output) == "Config key not set: "+args[i+2]
		}
	}
	return false
}

func configValue(run teamCLI, name, key string) (any, error) {
	raw, err := run("-p", name, "config", "get", key, "--json")
	if err != nil {
		return nil, err
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil, errors.New("native configuration JSON unavailable")
	}
	if object, ok := value.(map[string]any); ok {
		if v, found := object["value"]; found {
			return v, nil
		}
	}
	return value, nil
}
func shellQuote(v string) string { return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'" }
func teamCommand(args ...string) string {
	parts := []string{"hermes"}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ") + " >/dev/null 2>&1\n"
}
func teamSet(name, key string, value any) string {
	var encoded string
	if s, ok := value.(string); ok {
		encoded = s
	} else {
		b, _ := json.Marshal(value)
		encoded = string(b)
	}
	return teamCommand("-p", name, "config", "set", key, encoded)
}
func managedSouls(role team.Role) []string {
	return append([]string{role.Soul, role.LegacySoul, role.PreviousSoul, role.PreviousRepositorySoul, role.PreviousRepositoryOriginalSoul}, role.PreviousManagedSouls...)
}
func readSoul(root *os.Root, name string) (string, error) {
	path := ".hermes/SOUL.md"
	if name != "default" {
		path = filepath.Join(".hermes/profiles", name, "SOUL.md")
	}
	b, err := root.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func matchingSoul(soul string, role team.Role) bool {
	for _, s := range managedSouls(role) {
		if s != "" && s == soul {
			return true
		}
	}
	return false
}

// profileDescription reads the exact native description through the public
// CLI; native YAML folds long descriptions across lines.
func profileDescription(run teamCLI, name string) (string, error) {
	out, err := run("profile", "describe", name)
	if err != nil {
		return "", err
	}
	text := strings.TrimSuffix(string(out), "\n")
	if text == "(no description set for '"+name+"')" {
		return "", nil
	}
	return text, nil
}
func expectedTeamFields(role team.Role) map[string]any {
	fields := map[string]any{"kanban.dispatch_in_gateway": false, "kanban.auto_decompose": false, "kanban.orchestrator_profile": "default", "kanban.max_in_progress": float64(1), "terminal.cwd": "/workspace", "terminal.backend": "local"}
	if role.Name == "default" {
		fields["kanban.auto_subscribe_on_create"] = true
		fields["kanban.notify_in_gateway"] = true
	} else {
		fields["toolsets"] = role.Toolsets
		fields["platform_toolsets.cli"] = role.Toolsets
	}
	return fields
}

// holdsTeamValue reports whether an observed native value satisfies a managed
// one. Role toolsets are required as a subset, so owner additions survive.
func holdsTeamValue(got, want any) bool {
	if list, ok := want.([]string); ok {
		observed, ok := got.([]any)
		if !ok {
			return false
		}
		have := map[string]bool{}
		for _, v := range observed {
			s, ok := v.(string)
			if !ok {
				return false
			}
			have[s] = true
		}
		for _, s := range list {
			if !have[s] {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(got, want)
}

// classifyRole compares one existing roster profile with its managed role.
// customized means the owner changed its identity: a SOUL outside RepoKit's
// compiled history, or a different description. compatible means every
// managed configuration value the team depends on still holds.
func classifyRole(run teamCLI, root *os.Root, role team.Role) (soul string, customized, compatible bool, err error) {
	soul, err = readSoul(root, role.Name)
	if err != nil {
		return "", false, false, err
	}
	desc, err := profileDescription(run, role.Name)
	if err != nil {
		return "", false, false, err
	}
	customized = !matchingSoul(soul, role) || desc != role.Description
	fields := expectedTeamFields(role)
	if role.Name == "default" {
		// After activation default carries the complete managed dispatch
		// policy; anything between off and that policy is owner drift.
		value, err := configValue(run, role.Name, "kanban")
		if err != nil {
			return "", false, false, err
		}
		if kanban, ok := value.(map[string]any); ok && OperationalPolicy(kanban) {
			for _, field := range DispatchPolicy() {
				fields["kanban."+field.Key] = field.Value
			}
		} else if ok && UpgradablePolicy(kanban) {
			for _, field := range sixRoleDispatchPolicy() {
				fields["kanban."+field.Key] = field.Value
			}
		}
	}
	for key, want := range fields {
		got, err := configValue(run, role.Name, key)
		if err != nil {
			return "", false, false, err
		}
		if key == "terminal.backend" && got == nil {
			got = "local"
		}
		if !holdsTeamValue(got, want) {
			return soul, customized, false, nil
		}
	}
	return soul, customized, true, nil
}

// inspectRole reports whether a profile is RepoKit-managed and compatible.
func inspectRole(run teamCLI, root *os.Root, role team.Role) (string, bool, error) {
	soul, customized, compatible, err := classifyRole(run, root, role)
	return soul, err == nil && !customized && compatible, err
}
func soulWrite(name, soul string) string {
	path := "/opt/data/SOUL.md"
	if name != "default" {
		path = "/opt/data/profiles/" + name + "/SOUL.md"
	}
	return "printf '%s' " + shellQuote(base64.StdEncoding.EncodeToString([]byte(soul))) + " | base64 -d > " + shellQuote(path) + "\nchmod 600 " + shellQuote(path) + "\n"
}

// teamPlan is one convergence decision. Each roster profile is classified on
// its own: current, an untouched earlier RepoKit generation (upgraded), missing
// (created) or owner-customized (preserved and reported, never a failure).
// Drift names profiles lacking managed configuration the team depends on; it
// blocks every write because the roster can no longer be proved. Deferred names
// untouched earlier generations that wait for idle workers.
type teamPlan struct {
	Script     string
	Status     string
	Drift      []string
	Customized []string
	Deferred   []string
}

func teamScript(id target.Identity, afterSetup bool, run teamCLI, root *os.Root) (teamPlan, error) {
	roles := team.ForRepository(id)
	guard, err := teamStateGuard(root, roles)
	if err != nil {
		return teamPlan{}, err
	}
	dispatch, err := configValue(run, "default", "kanban.dispatch_in_gateway")
	if err != nil {
		return teamPlan{}, err
	}
	observe, busy := false, false
	if dispatch == true {
		value, err := configValue(run, "default", "kanban")
		if err != nil {
			return teamPlan{}, err
		}
		kanban, _ := value.(map[string]any)
		// An operational team, or the six-profile release's exact policy, is
		// reconciled while no card runs: untouched SOULs upgrade and the
		// six-profile team gains tester; activation then widens dispatch.
		// An owner-changed policy is only observed.
		sixRole := UpgradablePolicy(kanban)
		if sixRole || OperationalPolicy(kanban) {
			// Rewriting SOULs never happens under a live worker.
			running, err := runningWork(run)
			busy = err != nil || running
			if busy && sixRole {
				return teamPlan{}, errors.New("a card is running or Kanban is unreadable; the six-profile team upgrade waits for idle workers")
			}
			guard += idleGuard
		}
		observe = busy || !sixRole && !OperationalPolicy(kanban)
	}
	if observe {
		// Only default's own channel tools are completed so every channel can
		// reach Kanban; untouched SOUL upgrades wait for idle workers.
		plan := teamPlan{Status: "configured"}
		for _, role := range roles {
			soul, customized, compatible, e := classifyRole(run, root, role)
			switch {
			case e != nil || !compatible:
				plan.Drift = append(plan.Drift, role.Name)
			case customized:
				plan.Customized = append(plan.Customized, role.Name)
			case busy && soul != role.Soul:
				plan.Deferred = append(plan.Deferred, role.Name)
			}
		}
		channels, err := defaultChannelTools(run)
		if err != nil {
			return teamPlan{}, err
		}
		if channels != "" {
			plan.Script = bootstrapScript + "\n" + channels
		}
		return plan, nil
	}
	model, err := configValue(run, "default", "model.default")
	if err != nil {
		return teamPlan{}, err
	}
	managed := false
	for _, role := range roles {
		s, e := readSoul(root, role.Name)
		if e == nil && matchingSoul(s, role) {
			managed = true
			break
		}
	}
	if model == nil || model == "" || (!afterSetup && !managed) {
		return teamPlan{Status: "pending-setup"}, nil
	}
	// default is ours when it already holds a managed SOUL, or when it is still
	// the unclaimed native stock profile (absent or stock SOUL, no description).
	// An owner identity on default is preserved inside a team RepoKit already
	// manages; without one, this native home is not RepoKit's to claim.
	defaultSoul, soulErr := readSoul(root, "default")
	defaultDesc, descErr := profileDescription(run, "default")
	adoptDefault := descErr == nil && (defaultDesc == "" || defaultDesc == roles[0].Description) &&
		(os.IsNotExist(soulErr) || soulErr == nil && stockSoul(defaultSoul))
	if !adoptDefault && (soulErr != nil || !matchingSoul(defaultSoul, roles[0]) && !managed) {
		return teamPlan{Status: "drift", Drift: []string{"default"}}, nil
	}
	// Native CLI owns configuration semantics. The global fallback is the only
	// publicly inspectable default tool boundary; channel discovery is separate.
	// Required tools are added, never replaced, and only when missing.
	changes := ""
	tools, err := configValue(run, "default", "toolsets")
	if err != nil {
		return teamPlan{}, err
	}
	if tools == nil {
		tools = []any{}
	}
	selected, ok := tools.([]any)
	if !ok || !holdsTeamValue(tools, []string{}) {
		return teamPlan{}, errors.New("native default tool selection differs")
	}
	if !holdsTeamValue(tools, []string{"kanban", "memory"}) {
		for _, v := range []string{"kanban", "memory"} {
			if !holdsTeamValue(selected, []string{v}) {
				selected = append(selected, v)
			}
		}
		changes += teamSet("default", "toolsets", selected)
	}
	channelsValue, err := configValue(run, "default", "platform_toolsets")
	if err != nil {
		return teamPlan{}, err
	}
	configured, ok := channelsValue.(map[string]any)
	if channelsValue != nil && !ok {
		return teamPlan{}, errors.New("native channel tool selection differs")
	}
	if !holdsTeamValue(configured["cli"], []string{"kanban", "memory"}) {
		changes += teamCommand("-p", "default", "tools", "enable", "kanban", "memory", "--platform", "cli")
	}
	channelTools, err := defaultChannelTools(run)
	if err != nil {
		return teamPlan{}, err
	}
	changes += channelTools
	channels := []string{"cli"}
	defaultSkillDiscovery, err := configValue(run, "default", "skills.project_discovery")
	if err != nil {
		return teamPlan{}, err
	}
	for channel := range configured {
		if channel == "cli" || channel == "acp" || channel == "api_server" || channel == "cron" || channel == "webhook" {
			continue
		}
		if len(channel) == 0 || len(channel) > 64 || strings.IndexFunc(channel, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-')
		}) >= 0 {
			return teamPlan{}, errors.New("native channel name differs")
		}
		channels = append(channels, channel)
	}
	plan := teamPlan{Status: "configured"}
	for _, role := range roles {
		_, e := readSoul(root, role.Name)
		exists := e == nil
		if e != nil && !os.IsNotExist(e) {
			return teamPlan{}, errors.New("native profile identity unavailable")
		}
		if role.Name == "default" && adoptDefault {
			for key, value := range expectedTeamFields(role) {
				changes += teamSet(role.Name, key, value)
			}
			changes += teamCommand("profile", "describe", role.Name, "--text", role.Description)
			changes += soulWrite(role.Name, role.Soul)
			if defaultSkillDiscovery != false {
				changes += teamCommand("-p", role.Name, "skills", "trust", "/workspace")
			}
			continue
		}
		if exists {
			// Exact historical SOUL alone is insufficient evidence of
			// ownership: description and managed configuration must match too.
			prior, customized, compatible, e := classifyRole(run, root, role)
			if e != nil {
				return teamPlan{}, e
			}
			if !compatible {
				plan.Drift = append(plan.Drift, role.Name)
				continue
			}
			if customized {
				plan.Customized = append(plan.Customized, role.Name)
				continue
			}
			if prior != role.Soul {
				changes += soulWrite(role.Name, role.Soul)
			}
			skillDiscovery, e := configValue(run, role.Name, "skills.project_discovery")
			if e != nil {
				return teamPlan{}, e
			}
			if skillDiscovery != false {
				changes += teamCommand("-p", role.Name, "skills", "trust", "/workspace")
			}
			continue
		}
		if role.Name == "default" {
			plan.Drift = append(plan.Drift, role.Name)
			continue
		}
		changes += teamCommand("profile", "create", role.Name, "--clone", "--clone-from", "default", "--no-alias", "--description", role.Description)
		changes += soulWrite(role.Name, role.Soul)
		for _, rel := range []string{"memories/MEMORY.md", "memories/USER.md", "MEMORY.md", "USER.md"} {
			changes += "rm -f -- " + shellQuote("/opt/data/profiles/"+role.Name+"/"+rel) + "\n"
		}
		for key, value := range expectedTeamFields(role) {
			changes += teamSet(role.Name, key, value)
		}
		for _, channel := range channels {
			if channel != "cli" {
				changes += teamSet(role.Name, "platform_toolsets."+channel, role.Toolsets)
			}
		}
		changes += teamCommand("profile", "describe", role.Name, "--text", role.Description)
		if defaultSkillDiscovery != false {
			changes += teamCommand("-p", role.Name, "skills", "trust", "/workspace")
		}
	}
	if len(plan.Drift) > 0 {
		// Missing managed configuration means the roster can no longer be
		// proved. Preserve all state and let the owner inspect it first.
		plan.Status = "drift"
		return plan, nil
	}
	plan.Script = bootstrapScript + "\n" + guard + changes
	plan.Script += teamCommand("profile", "list")
	for _, role := range roles {
		plan.Script += teamCommand("profile", "show", role.Name)
	}
	return plan, nil
}

// Recheck the exact files that informed Go's decision after the container lock
// is acquired. A concurrent native setup must not turn a safe plan into a write
// against newer owner state.
func teamStateGuard(root *os.Root, roles []team.Role) (string, error) {
	var script strings.Builder
	for _, role := range roles {
		base := ".hermes"
		container := "/opt/data"
		if role.Name != "default" {
			base = filepath.Join(base, "profiles", role.Name)
			container += "/profiles/" + role.Name
		}
		for _, name := range []string{"config.yaml", "SOUL.md", "profile.yaml"} {
			path := filepath.Join(base, name)
			into := container + "/" + name
			info, err := root.Lstat(path)
			if os.IsNotExist(err) {
				script.WriteString("[ ! -e " + shellQuote(into) + " ] && [ ! -L " + shellQuote(into) + " ]\n")
				continue
			}
			if err != nil || !info.Mode().IsRegular() || info.Size() > 1024*1024 {
				return "", errors.New("native team state cannot be qualified")
			}
			content, err := root.ReadFile(path)
			if err != nil {
				return "", errors.New("native team state cannot be read")
			}
			digest := sha256.Sum256(content)
			script.WriteString("[ ! -L " + shellQuote(into) + " ]\n")
			script.WriteString("printf '%s  %s\\n' '" + fmt.Sprintf("%x", digest) + "' " + shellQuote(into) + " | sha256sum -c --status\n")
		}
	}
	return script.String(), nil
}
func teamResult(output string) (TeamReport, error) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "REPOKIT_TEAM=") {
			continue
		}
		var result struct {
			Status     string   `json:"status"`
			Drift      []string `json:"drift"`
			Customized []string `json:"customized"`
			Deferred   []string `json:"deferred"`
		}
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "REPOKIT_TEAM=")), &result) != nil {
			return TeamReport{}, errors.New("invalid team provisioning result")
		}
		if result.Status == "pending-setup" {
			return TeamReport{}, ErrTeamPending
		}
		if result.Status != "configured" && result.Status != "drift" {
			return TeamReport{}, errors.New("invalid team provisioning status")
		}
		known := map[string]bool{}
		for _, role := range team.Roster() {
			known[role.Name] = true
		}
		for _, names := range [][]string{result.Drift, result.Customized, result.Deferred} {
			for _, name := range names {
				if !known[name] {
					return TeamReport{}, errors.New("native team result names an unknown profile")
				}
			}
		}
		if len(result.Drift) > 0 {
			return TeamReport{}, fmt.Errorf("managed configuration drift preserved: %s; inspect native profiles before reconciliation", strings.Join(result.Drift, ", "))
		}
		if result.Status == "drift" {
			return TeamReport{}, errors.New("native team drift result incomplete")
		}
		return TeamReport{Customized: result.Customized, Deferred: result.Deferred}, nil
	}
	return TeamReport{}, errors.New("native team provisioning result missing; inspect existing profiles before retrying")
}

// defaultChannelTools enables Kanban and memory for default on every saved
// human-facing channel that lacks them, through native `tools enable`.
func defaultChannelTools(run teamCLI) (string, error) {
	value, err := configValue(run, "default", "platform_toolsets")
	if err != nil {
		return "", err
	}
	configured, _ := value.(map[string]any)
	names := make([]string, 0, len(configured))
	for name := range configured {
		names = append(names, name)
	}
	sort.Strings(names)
	script := ""
	for _, channel := range names {
		switch channel {
		case "cli", "acp", "api_server", "cron", "webhook":
			continue
		}
		if len(channel) == 0 || len(channel) > 64 || strings.IndexFunc(channel, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-')
		}) >= 0 {
			return "", errors.New("native channel name differs")
		}
		have := map[string]bool{}
		tools, _ := configured[channel].([]any)
		for _, t := range tools {
			if s, ok := t.(string); ok {
				have[s] = true
			}
		}
		if !have["kanban"] || !have["memory"] {
			script += teamCommand("-p", "default", "tools", "enable", "kanban", "memory", "--platform", channel)
		}
	}
	return script, nil
}

func stockSoul(soul string) bool {
	sum := sha256.Sum256([]byte(strings.TrimSuffix(soul, "\n")))
	return hex.EncodeToString(sum[:]) == qualification.NativeDefaultSoulSHA256
}

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

var ErrTeamPending = errors.New("team pending: Hermes private setup has not chosen a default model yet; run repokit setup")

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

// configValue reads one resolved key. It asks Hermes for the key's top-level
// section, which resolves defaults the same way, and walks to the key: each
// Hermes CLI call costs about a second of start-up, and with cachedConfig one
// section serves every key beneath it. An unset key is nil, as nativeTeamCLI
// reports a direct `config get` of one.
func configValue(run teamCLI, name, key string) (any, error) {
	path := strings.Split(key, ".")
	raw, err := run("-p", name, "config", "get", path[0], "--json")
	if err != nil {
		return nil, err
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil, errors.New("native configuration JSON unavailable")
	}
	if object, ok := value.(map[string]any); ok {
		if v, found := object["value"]; found && len(object) == 1 {
			value = v
		}
	}
	for _, part := range path[1:] {
		object, _ := value.(map[string]any)
		value = object[part] // nil when the section or key is unset
	}
	return value, nil
}

// cachedConfig answers repeated `config get` reads from one call each. Only a
// read-only pass may use it: nothing it serves reflects a later write.
func cachedConfig(run teamCLI) teamCLI {
	type reply struct {
		out []byte
		err error
	}
	cache := map[string]reply{}
	return func(args ...string) ([]byte, error) {
		if len(args) != 6 || args[0] != "-p" || args[2] != "config" || args[3] != "get" || args[5] != "--json" {
			return run(args...)
		}
		key := args[1] + "\x00" + args[4]
		if r, ok := cache[key]; ok {
			return r.out, r.err
		}
		out, err := run(args...)
		cache[key] = reply{out, err}
		return out, err
	}
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

// matchingSoul reports RepoKit's current managed SOUL; RepoKit recognizes no
// earlier generation, so any other SOUL is owner state.
// matchingSoul reports a RepoKit-managed SOUL: the current one, or the exact
// SOUL the previous release installed (which install upgrades in place).
func matchingSoul(soul string, role team.Role) bool {
	return soul == role.Soul || previousSoul(soul, role)
}

// previousSoul reports an untouched SOUL from team.PreviousRelease that differs
// from the current one.
func previousSoul(soul string, role team.Role) bool {
	return role.PreviousSoul != "" && soul == role.PreviousSoul && soul != role.Soul
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

// requiredTeamFields is expectedTeamFields with each role's Required toolsets
// instead of its granted baseline: what an existing profile must still hold
// to count as compatible. The rest of the baseline (memory, broader tools) is
// granted at creation or reset and is the owner's to remove.
func requiredTeamFields(role team.Role) map[string]any {
	fields := expectedTeamFields(role)
	for _, key := range []string{"toolsets", "platform_toolsets.cli"} {
		if _, ok := fields[key]; ok {
			fields[key] = role.Required
		}
	}
	return fields
}

// skillsWrite installs a role's granted official skills. The native install
// exits 0 even when it installs nothing (unknown, blocked or offline), so the
// installed SKILL.md is checked instead, and a missing one is reported without
// failing the run: skills are granted extras, not readiness.
func skillsWrite(role team.Role) string {
	dir := "/opt/data"
	if role.Name != "default" {
		dir += "/profiles/" + role.Name
	}
	script := ""
	for _, skill := range role.Skills {
		script += "hermes -p " + shellQuote(role.Name) + " skills install " + shellQuote(skill) + " --yes >/dev/null 2>&1 || true\n"
		installed := dir + "/skills/" + strings.TrimPrefix(skill, "official/") + "/SKILL.md"
		script += "[ -f " + shellQuote(installed) + " ] || printf '%s\\n' " + shellQuote("REPOKIT_SKILL_MISSING="+role.Name+" "+skill) + "\n"
	}
	return script
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

// roleCheck compares one existing roster profile with its managed role.
// Config names the managed configuration keys that no longer hold; values are
// never recorded because native configuration may contain private state.
type roleCheck struct {
	soul        string
	soulManaged bool
	descManaged bool
	config      []string
}

// customized reports an owner-changed identity: a SOUL that is neither the
// current nor the previous release's, or a different description.
func (c roleCheck) customized() bool { return !c.soulManaged || !c.descManaged }

// previous reports an untouched previous-release SOUL awaiting upgrade.
func (c roleCheck) previous(role team.Role) bool { return previousSoul(c.soul, role) }

// compatible reports that every managed configuration value the team depends
// on still holds.
func (c roleCheck) compatible() bool { return len(c.config) == 0 }

// differs names what separates the profile from RepoKit's current baseline.
func (c roleCheck) differs(role team.Role) []string {
	out := []string{}
	if c.soul != role.Soul {
		out = append(out, "SOUL")
	}
	if !c.descManaged {
		out = append(out, "description")
	}
	return append(out, c.config...)
}

func classifyRole(run teamCLI, root *os.Root, role team.Role) (roleCheck, error) {
	soul, err := readSoul(root, role.Name)
	if err != nil {
		return roleCheck{}, err
	}
	desc, err := profileDescription(run, role.Name)
	if err != nil {
		return roleCheck{}, err
	}
	check := roleCheck{soul: soul, soulManaged: matchingSoul(soul, role), descManaged: desc == role.Description}
	fields := requiredTeamFields(role)
	if role.Name == "default" {
		// After activation default carries the complete managed dispatch
		// policy; anything between off and that policy is owner drift.
		value, err := configValue(run, role.Name, "kanban")
		if err != nil {
			return roleCheck{}, err
		}
		if kanban, ok := value.(map[string]any); ok && OperationalPolicy(kanban) {
			for _, field := range DispatchPolicy() {
				fields["kanban."+field.Key] = field.Value
			}
		}
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		got, err := configValue(run, role.Name, key)
		if err != nil {
			return roleCheck{}, err
		}
		if key == "terminal.backend" && got == nil {
			got = "local"
		}
		if !holdsTeamValue(got, fields[key]) {
			check.config = append(check.config, key)
		}
	}
	return check, nil
}

// inspectRole reports whether a profile is RepoKit-managed and compatible.
func inspectRole(run teamCLI, root *os.Root, role team.Role) (string, bool, error) {
	check, err := classifyRole(run, root, role)
	return check.soul, err == nil && !check.customized() && check.compatible(), err
}
func soulWrite(name, soul string) string {
	path := "/opt/data/SOUL.md"
	if name != "default" {
		path = "/opt/data/profiles/" + name + "/SOUL.md"
	}
	return "printf '%s' " + shellQuote(base64.StdEncoding.EncodeToString([]byte(soul))) + " | base64 -d > " + shellQuote(path) + "\nchmod 600 " + shellQuote(path) + "\n"
}

// teamPlan is one convergence decision. Each roster profile is classified on
// its own: current, missing (created) or owner-customized (preserved and
// reported, never a failure). Drift names profiles lacking managed
// configuration the team depends on; it blocks every write because the roster
// can no longer be proved.
type teamPlan struct {
	Script     string
	Status     string
	Roles      []RoleStatus
	Drift      []string
	Customized []string
}

// RoleStatus is one plan row: a roster profile, its convergence state and the
// action install takes. Differs names what separates the profile from
// RepoKit's current baseline (SOUL, description or managed configuration
// keys); values are never reported because they may be private.
type RoleStatus struct {
	Profile string   `json:"profile"`
	State   string   `json:"state"`
	Action  string   `json:"action"`
	Differs []string `json:"differs,omitempty"`
}

var roleActions = map[string]string{
	"current":    "none",
	"upgrade":    "rewrite the untouched " + team.PreviousRelease + " SOUL",
	"deferred":   "upgrade the " + team.PreviousRelease + " SOUL after running work finishes",
	"missing":    "create",
	"adopt":      "claim stock default",
	"customized": "preserve",
	"drift":      "preserve; blocks this run",
	"reset":      "replace with RepoKit baseline; back up prior files",
}

func (p *teamPlan) role(name, state string, differs []string) {
	p.Roles = append(p.Roles, RoleStatus{Profile: name, State: state, Action: roleActions[state], Differs: differs})
	switch state {
	case "drift":
		p.Drift = append(p.Drift, name)
	case "customized":
		p.Customized = append(p.Customized, name)
	}
}

// resetWrite backs up a profile's native files, then restores RepoKit's
// baseline. default's configuration carries the dispatch policy that setup
// activation owns, so a default reset restores identity only.
func resetWrite(role team.Role) string {
	dir := "/opt/data"
	if role.Name != "default" {
		dir += "/profiles/" + role.Name
	}
	script := "stamp=$(date -u +%Y%m%dT%H%M%SZ)\n"
	for _, name := range []string{"SOUL.md", "config.yaml", "profile.yaml"} {
		path := shellQuote(dir + "/" + name)
		script += "if [ -f " + path + " ]; then cp -p " + path + " " + path + ".before-reset-\"$stamp\"; fi\n"
	}
	script += soulWrite(role.Name, role.Soul)
	script += teamCommand("profile", "describe", role.Name, "--text", role.Description)
	if role.Name != "default" {
		for key, value := range expectedTeamFields(role) {
			script += teamSet(role.Name, key, value)
		}
	}
	return script + skillsWrite(role)
}

// teamScript plans convergence. reset, when set, names one roster profile the
// owner explicitly asked to return to RepoKit's baseline.
func teamScript(id target.Identity, afterSetup bool, reset string, run teamCLI, root *os.Root) (teamPlan, error) {
	// Planning only reads native configuration; the plan's script writes later.
	run = cachedConfig(run)
	roles := team.ForRepository(id)
	if reset != "" && !knownRole(reset) {
		return teamPlan{}, errors.New("reset target is not a roster profile")
	}
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
		// An operational team is reconciled while no card runs: missing roles
		// are created and a requested reset applies. An owner-changed policy,
		// or a busy board, is only observed.
		if OperationalPolicy(kanban) {
			running, err := runningWork(run)
			busy = err != nil || running
			if busy && reset != "" {
				return teamPlan{}, errors.New("a card is running or Kanban is unreadable; profile reset waits for idle workers")
			}
			guard += idleGuard
		}
		observe = busy || !OperationalPolicy(kanban)
		if observe && reset != "" {
			return teamPlan{}, errors.New("owner-changed dispatch policy is only observed; profile reset is unavailable")
		}
	}
	if observe {
		// Only default's own channel tools are completed so every channel can
		// reach Kanban.
		plan := teamPlan{Status: "configured"}
		for _, role := range roles {
			check, e := classifyRole(run, root, role)
			switch {
			case e != nil:
				plan.role(role.Name, "drift", nil)
			case !check.compatible():
				plan.role(role.Name, "drift", check.differs(role))
			case check.customized():
				plan.role(role.Name, "customized", check.differs(role))
			case check.previous(role):
				// Never rewrite an identity under a running worker.
				plan.role(role.Name, "deferred", []string{"SOUL"})
			default:
				plan.role(role.Name, "current", nil)
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
		plan := teamPlan{Status: "drift"}
		plan.role("default", "drift", []string{"SOUL"})
		return plan, nil
	}
	// Native CLI owns configuration semantics. The global fallback is the only
	// publicly inspectable default tool boundary; channel discovery is separate.
	// Kanban is the only required tool: it is added, never replaced, and only
	// when missing. Memory and every other tool are the owner's choice.
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
	if !holdsTeamValue(tools, []string{"kanban"}) {
		selected = append(selected, "kanban")
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
	if !holdsTeamValue(configured["cli"], []string{"kanban"}) {
		changes += teamCommand("-p", "default", "tools", "enable", "kanban", "--platform", "cli")
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
			plan.role(role.Name, "adopt", nil)
			for key, value := range expectedTeamFields(role) {
				changes += teamSet(role.Name, key, value)
			}
			changes += teamCommand("profile", "describe", role.Name, "--text", role.Description)
			changes += soulWrite(role.Name, role.Soul)
			if defaultSkillDiscovery != false {
				changes += teamCommand("-p", role.Name, "skills", "trust", "/workspace")
			}
			changes += skillsWrite(role)
			continue
		}
		if exists {
			// Exact historical SOUL alone is insufficient evidence of
			// ownership: description and managed configuration must match too.
			check, e := classifyRole(run, root, role)
			if e != nil {
				return teamPlan{}, e
			}
			resetConfig := role.Name == reset && role.Name != "default"
			if !check.compatible() && !resetConfig {
				plan.role(role.Name, "drift", check.differs(role))
				continue
			}
			switch {
			case role.Name == reset:
				plan.role(role.Name, "reset", check.differs(role))
				changes += resetWrite(role)
			case check.customized():
				plan.role(role.Name, "customized", check.differs(role))
				continue
			case check.previous(role):
				plan.role(role.Name, "upgrade", []string{"SOUL"})
				changes += soulWrite(role.Name, role.Soul)
			default:
				plan.role(role.Name, "current", nil)
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
			plan.role(role.Name, "drift", []string{"SOUL"})
			continue
		}
		plan.role(role.Name, "missing", nil)
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
		changes += skillsWrite(role)
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
func knownRole(name string) bool {
	for _, role := range team.Roster() {
		if role.Name == name {
			return true
		}
	}
	return false
}

func grantedSkill(name, skill string) bool {
	for _, role := range team.Roster() {
		if role.Name == name {
			for _, s := range role.Skills {
				if s == skill {
					return true
				}
			}
		}
	}
	return false
}

func teamResult(output string) (TeamReport, error) {
	var missing []string
	for _, line := range strings.Split(output, "\n") {
		if entry, ok := strings.CutPrefix(line, "REPOKIT_SKILL_MISSING="); ok {
			// Output is untrusted: accept only a roster profile paired with
			// one of that profile's own granted skills.
			if name, skill, found := strings.Cut(entry, " "); found && grantedSkill(name, skill) {
				missing = append(missing, name+" "+skill)
			}
			continue
		}
		if !strings.HasPrefix(line, "REPOKIT_TEAM=") {
			continue
		}
		var result struct {
			Status     string   `json:"status"`
			Drift      []string `json:"drift"`
			Customized []string `json:"customized"`
			Reset      []string `json:"reset"`
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
		for _, names := range [][]string{result.Drift, result.Customized, result.Reset} {
			for _, name := range names {
				if !knownRole(name) {
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
		return TeamReport{Customized: result.Customized, Reset: result.Reset, MissingSkills: missing}, nil
	}
	return TeamReport{}, errors.New("native team provisioning result missing; inspect existing profiles before retrying")
}

// defaultChannelTools enables Kanban for default on every saved human-facing
// channel that lacks it, through native `tools enable`. Other tools on the
// channel, memory included, are the owner's and are left alone.
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
		if !have["kanban"] {
			script += teamCommand("-p", "default", "tools", "enable", "kanban", "--platform", channel)
		}
	}
	return script, nil
}

func stockSoul(soul string) bool {
	sum := sha256.Sum256([]byte(strings.TrimSuffix(soul, "\n")))
	return hex.EncodeToString(sum[:]) == qualification.NativeDefaultSoulSHA256
}

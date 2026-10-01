package native

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

// readKeys are the dotted configuration keys RepoKit reads.
func readKeys() []string {
	seen := map[string]bool{"kanban.dispatch_in_gateway": true, "model.default": true, "skills.project_discovery": true}
	for _, role := range team.Roster() {
		for key := range expectedTeamFields(role) {
			seen[key] = true
		}
	}
	for _, field := range DispatchPolicy() {
		seen["kanban."+field.Key] = true
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	return keys
}

// sectioned answers `config get <section>` the way Hermes does, from a fake
// written per dotted key: a fake's direct answer for the section wins;
// otherwise the section is assembled from the fake's answers for each key
// beneath it, and an empty section is unset (null).
func sectioned(fake teamCLI) teamCLI {
	return func(args ...string) ([]byte, error) {
		if len(args) != 6 || args[2] != "config" || args[3] != "get" || strings.Contains(args[4], ".") {
			return fake(args...)
		}
		decode := func(raw []byte) any {
			var v any
			if json.Unmarshal(raw, &v) != nil {
				return nil
			}
			if o, ok := v.(map[string]any); ok && len(o) == 1 {
				if inner, found := o["value"]; found {
					return inner
				}
			}
			return v
		}
		raw, err := fake(args...)
		if err != nil {
			return raw, err
		}
		if direct := decode(raw); direct != nil {
			return raw, nil
		}
		section := map[string]any{}
		for _, key := range readKeys() {
			rest, ok := strings.CutPrefix(key, args[4]+".")
			if !ok {
				continue
			}
			call := append([]string{}, args...)
			call[4] = key
			out, err := fake(call...)
			if err != nil {
				return out, err
			}
			if v := decode(out); v != nil {
				node := section
				parts := strings.Split(rest, ".")
				for _, p := range parts[:len(parts)-1] {
					next, _ := node[p].(map[string]any)
					if next == nil {
						next = map[string]any{}
						node[p] = next
					}
					node = next
				}
				node[parts[len(parts)-1]] = v
			}
		}
		if len(section) == 0 {
			return []byte("null"), nil
		}
		return json.Marshal(section)
	}
}

func TestConfigSectionServesEveryKeyWithOneCall(t *testing.T) {
	calls := 0
	run := cachedConfig(func(args ...string) ([]byte, error) {
		calls++
		if args[4] == "kanban" {
			return []byte(`{"max_in_progress":1,"nested":{"deep":true}}`), nil
		}
		return []byte("null"), nil
	})
	for key, want := range map[string]any{"kanban.max_in_progress": float64(1), "kanban.nested.deep": true, "kanban.absent": nil, "kanban.max_in_progress.below": nil, "model.default": nil} {
		if got, err := configValue(run, "default", key); err != nil || got != want {
			t.Errorf("%s: %v %v", key, got, err)
		}
	}
	if calls != 2 {
		t.Fatalf("%d native calls for two sections", calls)
	}
	if _, err := configValue(func(...string) ([]byte, error) { return []byte("Traceback"), nil }, "default", "kanban.x"); err == nil {
		t.Fatal("unreadable configuration accepted")
	}
}

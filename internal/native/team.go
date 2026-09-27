package native

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

//go:embed team.py
var teamScript string

func initializationScript(afterSetup bool) string {
	payload, _ := json.Marshal(struct {
		Roles      []team.Role `json:"roles"`
		AfterSetup bool        `json:"after_setup"`
	}{team.Roster(), afterSetup})
	// Only generated text enters this heredoc. No credentials or owner content.
	return bootstrapScript + "\npython - <<'REPOKIT_TEAM_PY'\n" + teamScript +
		"\nimport base64\ntry:\n    main(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\nexcept Exception:\n    print('RepoKit team provisioning failed; existing profiles preserved.', file=sys.stderr)\n    sys.exit(1)\nREPOKIT_TEAM_PY\n"
}

func teamResult(output string) error {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "REPOKIT_TEAM=") {
			continue
		}
		var result struct {
			Status string   `json:"status"`
			Drift  []string `json:"drift"`
		}
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "REPOKIT_TEAM=")), &result) != nil {
			return fmt.Errorf("invalid team provisioning result")
		}
		if result.Status == "drift" {
			// Do not echo arbitrary subprocess content, even in purported profile names.
			var names []string
			for _, r := range team.Roster() {
				for _, n := range result.Drift {
					if r.Name == n {
						names = append(names, n)
					}
				}
			}
			return fmt.Errorf("profile drift preserved: %s; inspect native profiles before reconciliation", strings.Join(names, ", "))
		}
	}
	return nil
}

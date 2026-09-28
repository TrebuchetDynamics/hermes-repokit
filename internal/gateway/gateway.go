// Package gateway provides one-shot native gateway convergence diagnostics.
// Nothing in the generated Hermes runtime imports or invokes this package.
package gateway

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

//go:embed state.py
var StateScript string

//go:embed catalog.py
var catalogScript string

//go:embed channels.py
var channelsScript string

//go:embed dispatch.py
var DispatchScript string

// SupportScript supplies passive helpers without invoking an entry point.
func SupportScript() string {
	return catalogScript + "\n" + channelsScript + "\n" + StateScript + "\n" + DispatchScript
}

// Script includes only generated public role metadata, never owner secrets.
func Script(id target.Identity, apply bool) string {
	payload, _ := json.Marshal(map[string]any{"roles": team.ForRepository(id), "repo_id": id.Project, "apply": apply})
	return SupportScript() + "\nimport base64\nmain(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\n"
}

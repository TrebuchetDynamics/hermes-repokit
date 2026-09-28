// Package maintenance embeds the optional native restart broker. The Go binary
// installs these sources once; the broker never calls RepoKit at runtime.
package maintenance

import _ "embed"

const Name = "repokit_maintenance"

//go:embed __init__.py
var implementation string

//go:embed plugin.yaml
var manifest string

func Files() map[string]string {
	return map[string]string{"__init__.py": implementation, "plugin.yaml": manifest}
}

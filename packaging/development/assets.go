// Package development embeds the immutable development image recipe.
package development

import "embed"

//go:embed Dockerfile repokit-browser-use-requirements.txt repokit-hermes-requirements.txt repokit-stop-gateways
var Assets embed.FS

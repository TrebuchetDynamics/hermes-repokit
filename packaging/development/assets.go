// Package development embeds the immutable development image recipe.
package development

import "embed"

//go:embed Dockerfile repokit-ddgs-requirements.txt repokit-browser-use-requirements.txt repokit-stop-gateways
var Assets embed.FS

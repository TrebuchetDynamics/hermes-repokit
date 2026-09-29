// Package development embeds the immutable development image recipe.
package development

import "embed"

//go:embed Dockerfile repokit-openviking openviking-run openviking-finish
var Assets embed.FS

// Package development embeds the immutable development image recipe.
package development

import "embed"

//go:embed Dockerfile
var Assets embed.FS

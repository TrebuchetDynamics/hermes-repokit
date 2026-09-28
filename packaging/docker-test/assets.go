// Package dockertest embeds the helper for the generated development image.
package dockertest

import "embed"

//go:embed repokit-docker-test
var Assets embed.FS

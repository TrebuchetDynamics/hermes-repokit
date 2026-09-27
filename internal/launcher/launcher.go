// Package launcher generates a shell/Docker-only native command.
package launcher

import (
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"path/filepath"
	"strings"
)

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func Render(id target.Identity, context string) ([]byte, error) {
	if !filepath.IsAbs(id.Compose) || context == "" || strings.ContainsAny(context, "\n\r\x00") {
		return nil, fmt.Errorf("absolute Compose path and Docker context required")
	}
	prefix := "docker --context " + quote(context) + " compose --env-file /dev/null -f " + quote(id.Compose)
	return []byte("#!/bin/sh\n" +
		"# Standalone native launcher; no RepoKit runtime dependency.\n" +
		"unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES COMPOSE_PATH_SEPARATOR COMPOSE_DISABLE_ENV_FILE COMPOSE_CONVERT_WINDOWS_PATHS COMPOSE_EXPERIMENTAL COMPOSE_MENU COMPOSE_ANSI COMPOSE_PROGRESS\n" +
		"if [ -t 0 ] && [ -t 1 ]; then\n  exec " + prefix + " exec --workdir /workspace hermes hermes \"$@\"\nfi\n" +
		"exec " + prefix + " exec -T --workdir /workspace hermes hermes \"$@\"\n"), nil
}

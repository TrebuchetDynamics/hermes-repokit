// Package launcher generates a shell/Docker-only native command.
package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const clearSelectors = "unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES COMPOSE_PATH_SEPARATOR COMPOSE_DISABLE_ENV_FILE COMPOSE_CONVERT_WINDOWS_PATHS COMPOSE_EXPERIMENTAL COMPOSE_MENU COMPOSE_ANSI COMPOSE_PROGRESS"

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func Render(id target.Identity, context string) ([]byte, error) {
	if !filepath.IsAbs(id.Compose) || context == "" || strings.ContainsAny(context, "\n\r\x00") {
		return nil, fmt.Errorf("absolute Compose path and Docker context required")
	}
	encoded, _ := json.Marshal(context)
	prefix := "docker --context " + quote(context) + " compose --env-file /dev/null -f " + quote(id.Compose)
	return []byte("#!/bin/sh\n" +
		"# Standalone native launcher; no RepoKit runtime dependency.\n" +
		"# Docker context: " + string(encoded) + "\n" +
		clearSelectors + "\n" +
		"if [ \"$#\" -eq 0 ]; then set -- -p default; fi\n" +
		"if [ -t 0 ] && [ -t 1 ]; then\n  exec " + prefix + " exec --workdir /workspace hermes hermes \"$@\"\nfi\n" +
		"exec " + prefix + " exec -T --workdir /workspace hermes hermes \"$@\"\n"), nil
}

// Context inspects only a byte-for-byte recognized launcher. Owner modifications
// are allowed but make automatic context qualification unknown.
func Context(id target.Identity) (string, error) {
	root, e := os.OpenRoot(id.Root)
	if e != nil {
		return "", e
	}
	defer root.Close()
	f, e := root.Open(".hermes/bin/" + id.Container)
	if e != nil {
		return "", e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("launcher is not a regular file")
	}
	data, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil || len(data) > 65536 {
		return "", fmt.Errorf("launcher cannot be inspected within limit")
	}
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 3 || !bytes.HasPrefix(lines[2], []byte("# Docker context: ")) {
		return "", fmt.Errorf("launcher context unknown")
	}
	var context string
	if e = json.Unmarshal(bytes.TrimPrefix(lines[2], []byte("# Docker context: ")), &context); e != nil {
		return "", e
	}
	expected, e := Render(id, context)
	if e != nil || !bytes.Equal(expected, data) {
		return "", fmt.Errorf("owner-edited launcher: context unknown")
	}
	return context, nil
}

// StartCommand is an operator-facing command with the launcher's routing.
func StartCommand(compose, context string) string {
	return "(" + clearSelectors + "; docker --context " + quote(context) + " compose --env-file /dev/null -f " + quote(compose) + " up -d hermes)"
}

// StartMemoryCommand leaves service lifecycle with ordinary Compose.
func StartMemoryCommand(compose, context string) string {
	return "(" + clearSelectors + "; docker --context " + quote(context) + " compose --env-file /dev/null -f " + quote(compose) + " up -d openviking)"
}

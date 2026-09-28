// Package laya carries the qualified offline image recipe into standalone installs.
package laya

import "embed"

// Assets contains only the exact inputs needed by ordinary Docker Compose builds.
//
//go:embed Dockerfile requirements.lock model-manifest.json download_model.py
var Assets embed.FS

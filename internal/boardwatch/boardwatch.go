// Package boardwatch owns RepoKit's board watch: the Hermes cron job that
// wakes default when a quiet board has an open goal or a stuck card.
package boardwatch

import _ "embed"

// Script is the cron job's monitor script.
//
//go:embed repokit-board-watch.py
var Script []byte

package team

import _ "embed"

// KanbanPolicy is pure configuration logic shared by bootstrap and observation.
// It performs no file access, plugin imports, model calls, or runtime callbacks.
//
//go:embed kanban.py
var KanbanPolicy string

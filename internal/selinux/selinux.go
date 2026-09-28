// Package selinux reports host SELinux state so RepoKit can request Docker's
// native bind-mount relabeling. It never runs getenforce, changes policy, or
// touches global SELinux configuration.
package selinux

import (
	"os"
	"runtime"
	"strings"
)

type State string

const (
	Enforcing   State = "enforcing"
	Permissive  State = "permissive"
	Disabled    State = "disabled"
	Unavailable State = "unavailable"
)

// Detect reads kernel/system SELinux state without requiring an external
// package. A present but unreadable enforce node is treated as enforcing so the
// conservative relabel path is still selected.
func Detect() State {
	if runtime.GOOS != "linux" {
		return Unavailable
	}
	raw, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		if _, statErr := os.Stat("/sys/fs/selinux"); statErr == nil {
			return Enforcing
		}
		return Disabled
	}
	switch strings.TrimSpace(string(raw)) {
	case "1":
		return Enforcing
	case "0":
		return Permissive
	default:
		return Disabled
	}
}

// Enabled reports whether SELinux is active, so Docker bind relabeling has an
// effect. Enforcing and permissive both relabel.
func (s State) Enabled() bool { return s == Enforcing || s == Permissive }

// RelabelMode is the Docker Compose bind SELinux option for the private
// single-container topology. It is empty when relabeling is meaningless, so the
// generated mounts are unchanged on hosts without SELinux.
func (s State) RelabelMode() string {
	if s.Enabled() {
		return "Z"
	}
	return ""
}

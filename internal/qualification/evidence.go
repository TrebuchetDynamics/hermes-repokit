// Package qualification checks the completeness of version-bound native
// operation evidence. It does not authenticate references or grant execution.
package qualification

import "strings"

type Operation string

const (
	Image               Operation = "image"
	ProfileDescriptions Operation = "profile-descriptions"
	Toolsets            Operation = "toolsets"
	OpenViking          Operation = "openviking"
	Nerve               Operation = "nerve"
	Laya                Operation = "laya"
	SameCardReview      Operation = "same-card-review"
	NativeChat          Operation = "native-chat"
	Setup               Operation = "setup"
	Profiles            Operation = "profiles"
	Plugins             Operation = "plugins"
	Kanban              Operation = "kanban"
	OfficialExecShim    Operation = "official-exec-shim"
	ReadOnlyProbes      Operation = "read-only-probes"
)

type Verdict uint8

const (
	Unknown Verdict = iota
	Unsupported
	Supported
)

// Evidence records a claim for one operation at one selected Hermes commit.
// References are slash-separated relative paths whose segments start with an
// ASCII letter or digit and then contain only ASCII letters, digits, dot,
// underscore or hyphen. Evaluate checks their syntax, not their existence or
// authenticity. They must identify sanitized records, never raw native output.
type Evidence struct {
	Operation        Operation
	SelectedRevision string
	Verdict          Verdict
	SourceReference  string
	RuntimeReference string
}

// Evaluate returns Unknown unless the evidence is complete for this exact
// operation and immutable Hermes source revision.
func Evaluate(selectedRevision string, operation Operation, evidence Evidence) Verdict {
	if !validRevision(selectedRevision) || !validOperation(operation) ||
		evidence.Operation != operation || evidence.SelectedRevision != selectedRevision ||
		!validReference(evidence.SourceReference) {
		return Unknown
	}
	switch evidence.Verdict {
	case Unsupported:
		if evidence.RuntimeReference != "" && !validReference(evidence.RuntimeReference) {
			return Unknown
		}
		return Unsupported
	case Supported:
		if validReference(evidence.RuntimeReference) {
			return Supported
		}
	}
	return Unknown
}

func validOperation(operation Operation) bool {
	switch operation {
	case Image, ProfileDescriptions, Toolsets, OpenViking, Nerve, Laya, SameCardReview, NativeChat, Setup, Profiles, Plugins, Kanban, OfficialExecShim, ReadOnlyProbes:
		return true
	default:
		return false
	}
}

func validRevision(revision string) bool {
	if len(revision) != 40 {
		return false
	}
	for i := range len(revision) {
		if !('0' <= revision[i] && revision[i] <= '9') && !('a' <= revision[i] && revision[i] <= 'f') {
			return false
		}
	}
	return true
}

func validReference(reference string) bool {
	if reference == "" {
		return false
	}
	for _, segment := range strings.Split(reference, "/") {
		if segment == "" || segment == "." || segment == ".." || !isAlphaNumeric(segment[0]) {
			return false
		}
		for i := range len(segment) {
			c := segment[i]
			if !isAlphaNumeric(c) && c != '.' && c != '-' && c != '_' {
				return false
			}
		}
	}
	return true
}

func isAlphaNumeric(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// Package plugins enforces admission of immutable upstream plugins.
// It contains no plugin runtime or replacement scanner.
package plugins

import (
	"fmt"
	"regexp"
)

type Scan struct{ Revision, Verdict, FindingsDigest string }
type Approval struct{ Revision, FindingsDigest string }

var revision = regexp.MustCompile(`^[a-f0-9]{40}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func Admit(s Scan, a Approval) error {
	if !revision.MatchString(s.Revision) || !digest.MatchString(s.FindingsDigest) {
		return fmt.Errorf("plugin scanner evidence missing or malformed")
	}
	switch s.Verdict {
	case "safe":
		return nil
	case "caution":
		if a.Revision == s.Revision && a.FindingsDigest == s.FindingsDigest {
			return nil
		}
		return fmt.Errorf("plugin caution requires explicit approval of exact revision and findings")
	default:
		return fmt.Errorf("plugin scanner verdict refuses installation: %s", s.Verdict)
	}
}

package qualification

import (
	"fmt"
	"regexp"
	"strings"
)

// Qualification carries version-specific facts, never guesses or credentials.
// Only reviewed, built-in records may authorize install operations.
type Qualification struct {
	Supported bool              `json:"supported"`
	Source    string            `json:"source"`
	Revision  string            `json:"revision"`
	Contract  map[string]string `json:"contract"`
}

func (q Qualification) Require(keys ...string) error {
	if !q.Supported || !validRevision(q.Revision) || !validReference(q.Source) {
		return fmt.Errorf("unsupported: incomplete upstream qualification")
	}
	for _, k := range keys {
		if strings.TrimSpace(q.Contract[k]) == "" {
			return fmt.Errorf("unsupported: missing qualified contract %s", k)
		}
	}
	return nil
}

var imagePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9./:_-]*@sha256:[a-f0-9]{64}$`)

func ImmutableImage(image string) bool { return imagePattern.MatchString(image) }

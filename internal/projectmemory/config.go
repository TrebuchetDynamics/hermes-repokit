// Package projectmemory defines the public native shared-memory configuration.
// It neither provisions a service nor proves that extraction or recall works.
package projectmemory

import (
	"fmt"
	"regexp"
	"strings"
)

const Endpoint = "http://openviking:1933"
const Account = "repokit"

var nativeIdentity = regexp.MustCompile(`^[A-Za-z0-9_.@-]+$`)

// NativeConfig returns the native Hermes memory section for every team profile.
// repoID is the caller's stable deployment identity (currently target.Identity.Project),
// never a profile name. Secrets and environment/linked-config conflicts must be
// handled separately through native setup and inspection before activation.
func NativeConfig(repoID string) (map[string]any, error) {
	if !nativeIdentity.MatchString(repoID) || strings.Count(repoID, "@") > 1 {
		return nil, fmt.Errorf("invalid project memory user identity")
	}
	return map[string]any{
		"provider":             "openviking",
		"memory_enabled":       true,
		"user_profile_enabled": true,
		"openviking": map[string]any{
			"endpoint": Endpoint,
			"account":  Account,
			"user":     repoID,
		},
	}, nil
}

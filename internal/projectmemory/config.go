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

// Image is the official v0.4.21 service qualified for native configuration and
// pending-setup behavior. It does not establish model-backed memory acceptance.
const Image = "ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05"

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

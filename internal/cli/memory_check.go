package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

const memoryCLIConfig = "/opt/data/.openviking/ovcli.conf.repository"

// memoryCheck is an explicit, bounded mutation using the public OpenViking CLI.
// It deliberately does not import or initialize Hermes's memory provider.
func (a App) memoryCheck(out io.Writer) int {
	if a.Runner == nil {
		a.Runner = process.Runner{Timeout: 90 * time.Second}
	}
	probes := []verify.Probe{
		{Component: "memory-check", Status: verify.Unqualified, Detail: "preflight incomplete; no memory mutation attempted"},
		{Component: "memory-recall-same-profile", Status: verify.Unqualified, Detail: "Hermes agent recall was not invoked; native file read is a separate layer"},
		{Component: "memory-recall-cross-profile", Status: verify.Unqualified, Detail: "Hermes agent cross-profile recall remains NOT VERIFIED"},
		{Component: "memory-extraction", Status: verify.Unqualified, Detail: "asynchronous viking_remember extraction is separate acceptance and was not invoked"},
	}
	finish := func() int {
		if json.NewEncoder(out).Encode(probes) != nil {
			return 1
		}
		for _, p := range probes {
			if p.Status != verify.Healthy {
				return 1
			}
		}
		return 0
	}
	blocked := func(reason string) int {
		probes[0].Detail = "preflight incomplete; no memory mutation attempted: " + reason
		return finish()
	}
	id, err := target.Resolve(a.Directory)
	if err != nil {
		return blocked("repository identity unavailable")
	}
	if len(target.Inspect(id, "")) != 0 || len(a.gitIssues(context.Background(), id)) != 0 {
		return blocked("repository or private-state protection unverified")
	}
	dc, err := launcher.Context(id)
	if err != nil {
		return blocked("recognized deployment launcher unavailable")
	}
	selected, ok := compose.DevelopmentSelected(id)
	if !ok || selected.OpenVikingImage == "" {
		return blocked("embedded OpenViking deployment not selected")
	}
	ready, err := a.nativeRuntimeReady(id, dc)
	if err != nil || !ready {
		return blocked("pinned running Hermes container not verified")
	}
	for _, p := range verify.OpenViking(context.Background(), id, a.Runner) {
		if (p.Component == "openviking-config" || p.Component == "openviking-container" || p.Component == "openviking-runtime") && p.Status != verify.Healthy {
			return blocked("embedded OpenViking health or configuration incomplete")
		}
	}
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return blocked("repository cannot be inspected safely")
	}
	info, err := root.Lstat(".hermes/.openviking/ovcli.conf.repository")
	root.Close()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return blocked("private repository OpenViking CLI configuration unavailable")
	}

	runExec := func(env string, args ...string) process.Result {
		argv := []string{"--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes"}
		if env != "" {
			argv = append(argv, "--env", env)
		}
		argv = append(argv, "hermes")
		argv = append(argv, args...)
		return a.Runner.Run(context.Background(), "docker", argv...)
	}
	native := func(args ...string) process.Result { return runExec("", args...) }
	for _, role := range team.Roster() {
		for key, expected := range map[string]any{
			"memory.provider":                     "openviking",
			"memory.memory_enabled":               true,
			"memory.openviking.use_ovcli_config":  true,
			"memory.openviking.ovcli_config_path": memoryCLIConfig,
		} {
			got := native("hermes", "-p", role.Name, "config", "get", key, "--json")
			if got.Err != nil || got.Truncated || !nativeValueMatches(got.Output, expected) {
				return blocked("six native Hermes profile memory settings are not aligned")
			}
		}
	}
	probes[0].Detail = "six profiles use the native shared OpenViking CLI configuration; exact-file check pending"
	probes = append(probes,
		verify.Probe{Component: "memory-profile-config", Status: verify.Healthy, Detail: "six native Hermes profile settings point to the same private CLI configuration; agent recall not exercised"},
		verify.Probe{Component: "memory-profile-default", Status: verify.Unqualified, Detail: "NOT VERIFIED: no deterministic non-agent profile-scoped read/search interface"},
		verify.Probe{Component: "memory-profile-researcher", Status: verify.Unqualified, Detail: "NOT VERIFIED: no deterministic non-agent profile-scoped read/search interface"},
		verify.Probe{Component: "memory-profile-reviewer", Status: verify.Unqualified, Detail: "NOT VERIFIED: no deterministic non-agent profile-scoped read/search interface"},
	)
	ov := func(args ...string) process.Result {
		argv := append([]string{"/app/.venv/bin/ov", "-o", "json"}, args...)
		return runExec("OPENVIKING_CLI_CONFIG_FILE="+memoryCLIConfig, argv...)
	}
	if r := ov("health"); r.Err != nil || r.Truncated || !nativeHealthMatches(r.Output, id.Project) {
		return blocked("OpenViking repository user identity not verified")
	}
	probes = append(probes, verify.Probe{Component: "memory-identity", Status: verify.Healthy, Detail: "native OpenViking health identifies a non-admin repository user key"})
	if r := ov("stat", "viking://~/memories/repokit-selfcheck"); r.Err != nil || r.Truncated || !validNativeJSON(r.Output) {
		return blocked("OpenViking diagnostic namespace unavailable")
	}
	probes = append(probes, verify.Probe{Component: "memory-server", Status: verify.Healthy, Detail: "authenticated native OpenViking CLI and diagnostic namespace respond"})
	probes = append(probes, runMemoryCanary(ov)...)
	probes[0].Detail = "deterministic native OpenViking file check attempted; Hermes agent recall remains NOT VERIFIED"
	return finish()
}

// runMemoryCanary never initializes Hermes. On any uncertain write it attempts
// deletion of only its cryptographically unique file; it never retries a write.
func runMemoryCanary(ov func(...string) process.Result) []verify.Probe {
	var probes []verify.Probe
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return []verify.Probe{{Component: "memory-write", Status: verify.Unqualified, Detail: "canary nonce unavailable; nothing written"}}
	}
	marker := "REPOKIT_SELFCHECK_" + hex.EncodeToString(nonce[:])
	basename := "repokit-selfcheck-" + hex.EncodeToString(nonce[:]) + ".md"
	uri := "viking://~/memories/repokit-selfcheck/" + basename
	// Create mode prevents replacing owner data. Never retry an uncertain write.
	write := ov("write", uri, "--mode", "create", "--content", marker, "--wait", "--timeout", "60")
	if write.Err != nil || write.Truncated || !validNativeJSON(write.Output) || !writeReceiptMatches(write.Output, uri) {
		// The command may have committed before failing. Attempt only an exact
		// cleanup of the nonce URI and report uncertainty if it fails.
		removed := ov("rm", uri, "--wait", "--timeout", "60")
		probes = append(probes, verify.Probe{Component: "memory-cleanup", Status: verify.Unqualified, Detail: "uncertain write; exact cleanup attempted but absence cannot be proven"})
		if removed.Err != nil {
			probes[len(probes)-1].Detail = "uncertain write and exact cleanup failed; inspect the reported canary URI: " + uri
		}
		return probes
	}
	probes = append(probes, verify.Probe{Component: "memory-write", Status: verify.Healthy, Detail: "native OpenViking create-mode write completed at a unique exact URI"})
	read := ov("read", uri)
	if read.Err == nil && !read.Truncated && validNativeJSON(read.Output) && strings.Contains(read.Output, marker) {
		probes = append(probes, verify.Probe{Component: "memory-read", Status: verify.Healthy, Detail: "native exact-URI read returned the canary"})
	} else {
		probes = append(probes, verify.Probe{Component: "memory-read", Status: verify.Degraded, Detail: "native exact-URI read did not return the canary"})
	}
	find := ov("find", marker, "-u", "viking://~/memories/repokit-selfcheck", "--context-type", "memory")
	if find.Err == nil && !find.Truncated && validNativeJSON(find.Output) && strings.Contains(find.Output, basename) {
		probes = append(probes, verify.Probe{Component: "memory-search", Status: verify.Healthy, Detail: "native search returned the exact canary URI"})
	} else {
		probes = append(probes, verify.Probe{Component: "memory-search", Status: verify.Degraded, Detail: "native search did not return the exact canary URI"})
	}
	removed := ov("rm", uri, "--wait", "--timeout", "60")
	if removed.Err != nil || removed.Truncated || !validNativeJSON(removed.Output) {
		probes = append(probes, verify.Probe{Component: "memory-cleanup", Status: verify.Degraded, Detail: "exact cleanup failed or is uncertain; inspect canary URI: " + uri})
		return probes
	}
	after := ov("read", uri)
	if after.Err == nil || !nativeNotFound(after.Output) {
		probes = append(probes, verify.Probe{Component: "memory-cleanup", Status: verify.Degraded, Detail: "exact canary absence not proven after delete; inspect canary URI: " + uri})
		return probes
	}
	afterSearch := ov("find", marker, "-u", "viking://~/memories/repokit-selfcheck", "--context-type", "memory")
	if afterSearch.Err != nil || afterSearch.Truncated || !validNativeJSON(afterSearch.Output) || strings.Contains(afterSearch.Output, basename) {
		probes = append(probes, verify.Probe{Component: "memory-cleanup", Status: verify.Degraded, Detail: "canary remained in search or search absence was not proven; inspect canary URI: " + uri})
		return probes
	}
	probes = append(probes, verify.Probe{Component: "memory-cleanup", Status: verify.Healthy, Detail: "exact canary URI deleted and native read reports not found; extraction artifacts were not created"})
	return probes
}

func writeReceiptMatches(output, requested string) bool {
	var value any
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &value) != nil {
		return false
	}
	var uris []string
	var visit func(any)
	visit = func(v any) {
		switch item := v.(type) {
		case map[string]any:
			for key, child := range item {
				if key == "uri" || key == "canonical_uri" {
					if s, ok := child.(string); ok {
						uris = append(uris, s)
					}
				}
				visit(child)
			}
		case []any:
			for _, child := range item {
				visit(child)
			}
		}
	}
	visit(value)
	for _, got := range uris {
		if got == requested {
			return true
		}
		const alias = "viking://~/memories/repokit-selfcheck/"
		const canonical = "viking://repokit/user/memories/repokit-selfcheck/"
		if strings.HasPrefix(requested, alias) && strings.HasPrefix(got, canonical) &&
			strings.TrimPrefix(requested, alias) == strings.TrimPrefix(got, canonical) {
			return true
		}
	}
	return false
}

func nativeValueMatches(output string, expected any) bool {
	var value any
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &value) != nil {
		return false
	}
	if obj, ok := value.(map[string]any); ok {
		value = obj["value"]
	}
	return value == expected
}

func validNativeJSON(output string) bool {
	var value any
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &value) != nil || value == nil {
		return false
	}
	if obj, ok := value.(map[string]any); ok {
		if obj["error"] != nil || obj["status"] == "error" || obj["success"] == false {
			return false
		}
	}
	return true
}

func nativeNotFound(output string) bool {
	value := strings.ToLower(output)
	return strings.Contains(value, "not found") || strings.Contains(value, "404")
}

func nativeHealthMatches(output, repo string) bool {
	var health map[string]any
	if !validNativeJSON(output) || json.Unmarshal([]byte(output), &health) != nil {
		return false
	}
	if result, ok := health["result"].(map[string]any); ok {
		health = result
	}
	return health["auth_mode"] == "api_key" && health["role"] == "user" && health["account_id"] == "repokit" && health["user_id"] == repo
}

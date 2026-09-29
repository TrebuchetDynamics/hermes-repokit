package native

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
)

// DispatchPolicy is the native Kanban policy RepoKit configures on default.
// Native Hermes owns claims, workers, review dispatch and notifications; RepoKit
// only sets these public configuration values and restarts the gateway once.
// dispatch_in_gateway is written last so a partial write leaves dispatch off.
func DispatchPolicy() []struct {
	Key   string
	Value any
} {
	profiles := []any{}
	for _, role := range team.Roster() {
		profiles = append(profiles, role.Name)
	}
	return []struct {
		Key   string
		Value any
	}{
		{"review_dispatch", true},
		{"max_in_progress", float64(1)},
		{"auto_decompose", false},
		{"orchestrator_profile", "default"},
		{"dispatch_profiles", profiles},
		{"dispatch_in_gateway", true},
	}
}

// OperationalPolicy reports whether a decoded native `kanban` config section
// holds the complete managed dispatch policy.
func OperationalPolicy(kanban map[string]any) bool {
	for _, field := range DispatchPolicy() {
		if !reflect.DeepEqual(kanban[field.Key], field.Value) {
			return false
		}
	}
	return true
}

var (
	gatewayRunning = regexp.MustCompile(`Gateway is running \(PID: ([0-9]{1,10})\)`)
	gatewayStopped = regexp.MustCompile(`(?i)gateway is not running|gateway not running`)
)

// GatewayPID parses public `hermes gateway status` output: a PID when running,
// 0 when the status explicitly says it is not running, and an error otherwise.
func GatewayPID(status string) (int, error) {
	if m := gatewayRunning.FindStringSubmatch(status); m != nil {
		return strconv.Atoi(m[1])
	}
	if gatewayStopped.MatchString(status) {
		return 0, nil
	}
	return 0, errors.New("gateway status unrecognized")
}

func gatewayPID(run teamCLI) (int, error) {
	out, err := run("-p", "default", "gateway", "status")
	if err != nil {
		return 0, err
	}
	return GatewayPID(string(out))
}

func runningWork(run teamCLI) (bool, error) {
	raw, err := run("-p", "default", "kanban", "stats", "--json")
	if err != nil {
		return false, err
	}
	var stats struct {
		ByStatus map[string]int `json:"by_status"`
	}
	if json.Unmarshal(raw, &stats) != nil {
		return false, errors.New("native Kanban stats unavailable")
	}
	return stats.ByStatus["running"] > 0, nil
}

// ConvergeGateway enables native automatic dispatch on the default gateway.
// It never overwrites an owner-changed dispatch policy, never restarts while a
// card is running, and never claims that a worker has executed: live dispatch
// is proved separately by `verify --dispatch-check` or real reviewed work.
// A stopped gateway is started through `hermes gateway start` only when start
// is true (explicit setup); Hermes then keeps it running across restarts.
// States: "current" (already operational, untouched), "restarted" (policy set
// and gateway replaced), "started" (gateway was stopped and is now running) or
// "not-running" (gateway stopped and left stopped).
func ConvergeGateway(ctx context.Context, id target.Identity, dc string, r InputRunner, start bool) (string, error) {
	return convergeGateway(ctx, id, dc, r, start, 90, time.Second)
}

func startGateway(ctx context.Context, id target.Identity, dc string, r InputRunner, run teamCLI, attempts int, pause time.Duration) (string, error) {
	if _, err := runBootstrap(ctx, id, dc, false, bootstrapScript+teamCommand("-p", "default", "gateway", "start"), r); err != nil {
		return "", errors.New("native `gateway start` failed; inspect `gateway status`")
	}
	for i := 0; i < attempts; i++ {
		if pid, err := gatewayPID(run); err == nil && pid != 0 {
			return "started", nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pause):
		}
	}
	return "", errors.New("gateway start requested, but a running gateway was not observed; inspect `gateway status`")
}

func convergeGateway(ctx context.Context, id target.Identity, dc string, r InputRunner, start bool, attempts int, pause time.Duration) (string, error) {
	run := nativeTeamCLI(ctx, id, dc, r)
	value, err := configValue(run, "default", "kanban")
	if err != nil {
		return "", errors.New("native dispatch configuration unavailable")
	}
	kanban, ok := value.(map[string]any)
	if !ok {
		return "", errors.New("native dispatch configuration unavailable")
	}
	before, err := gatewayPID(run)
	if err != nil {
		return "", errors.New("native gateway status unavailable")
	}
	switch kanban["dispatch_in_gateway"] {
	case true:
		if !OperationalPolicy(kanban) {
			return "", errors.New("owner-changed dispatch policy preserved; inspect `kanban` configuration on default")
		}
		if before == 0 {
			if start {
				return startGateway(ctx, id, dc, r, run, attempts, pause)
			}
			return "not-running", nil
		}
		return "current", nil
	case false, nil:
	default:
		return "", errors.New("ambiguous native dispatch setting preserved")
	}
	if busy, err := runningWork(run); err != nil || busy {
		return "", errors.New("a card is running or Kanban is unreadable; dispatch left off so active work is not interrupted")
	}
	script := bootstrapScript + "\n# Refuse under the lock if work started since Go observed the board.\n" +
		"if hermes -p default kanban stats --json | grep -q '\"running\"'; then exit 3; fi\n"
	for _, field := range DispatchPolicy() {
		script += teamSet("default", "kanban."+field.Key, field.Value)
	}
	if before != 0 {
		script += teamCommand("-p", "default", "gateway", "restart")
	}
	if _, err := runBootstrap(ctx, id, dc, false, script, r); err != nil {
		return "", errors.New("native dispatch configuration or gateway restart failed; inspect `kanban` configuration and gateway status")
	}
	if before == 0 {
		if start {
			return startGateway(ctx, id, dc, r, run, attempts, pause)
		}
		return "not-running", nil
	}
	for i := 0; i < attempts; i++ {
		if pid, err := gatewayPID(run); err == nil && pid != 0 && pid != before {
			return "restarted", nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pause):
		}
	}
	return "", fmt.Errorf("dispatch configured, but a replacement gateway was not observed; inspect `gateway status`")
}

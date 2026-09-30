package native

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

//go:embed bootstrap.sh
var bootstrapScript string

// InputRunner sends one-shot installer input; it must not print captured output.
type InputRunner interface {
	RunInput(context.Context, io.Reader, string, ...string) process.Result
}

// TeamReport names roster profiles whose owner identity was preserved, and
// untouched earlier RepoKit SOULs whose upgrade waits for idle workers. Roles
// is the per-profile plan that was applied, in roster order.
type TeamReport struct {
	Customized []string
	Deferred   []string
	Reset      []string
	Roles      []RoleStatus
}

// TeamStatus previews team convergence: the decision install would make, read
// only through public native commands. Nothing is written.
type TeamStatus struct {
	Status   string       `json:"status"`
	Profiles []RoleStatus `json:"profiles"`
}

// PlanTeam returns the per-profile plan install would apply. reset previews an
// explicit `install --reset-profile`.
func PlanTeam(ctx context.Context, id target.Identity, dockerContext, reset string, r InputRunner) (TeamStatus, error) {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return TeamStatus{}, fmt.Errorf("native team state unavailable")
	}
	defer root.Close()
	plan, err := teamScript(id, false, reset, nativeTeamCLI(ctx, id, dockerContext, r), root)
	if err != nil {
		return TeamStatus{}, err
	}
	if plan.Roles == nil {
		plan.Roles = []RoleStatus{}
	}
	return TeamStatus{Status: plan.Status, Profiles: plan.Roles}, nil
}

// Initialize uses the single existing, source-qualified Hermes runtime. Caller
// must first verify its image, project, mounts and running state. The lock is
// held INSIDE that container, so killing the Docker client cannot release it
// while a daemon-owned native subprocess is still writing state.
//
// reset names one roster profile the owner explicitly returns to RepoKit's
// baseline; it is empty for ordinary convergence.
func Initialize(ctx context.Context, id target.Identity, dockerContext string, afterSetup bool, reset string, r InputRunner) (TeamReport, error) {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return TeamReport{}, fmt.Errorf("native team state unavailable")
	}
	defer root.Close()
	plan, err := teamScript(id, afterSetup, reset, nativeTeamCLI(ctx, id, dockerContext, r), root)
	if err != nil {
		return TeamReport{}, err
	}
	script := plan.Script
	if script == "" {
		script = bootstrapScript + "\n"
	}
	status := plan.Status
	if len(plan.Drift) > 0 {
		status = "drift"
	}
	marker, _ := json.Marshal(struct {
		Status     string   `json:"status"`
		Drift      []string `json:"drift"`
		Customized []string `json:"customized"`
		Deferred   []string `json:"deferred"`
		Reset      []string `json:"reset"`
	}{status, plan.Drift, plan.Customized, plan.Deferred, resetRoles(plan)})
	script += "printf '%s\\n' 'REPOKIT_TEAM=" + string(marker) + "'\n"
	result, err := runBootstrap(ctx, id, dockerContext, afterSetup, script, r)
	if err != nil {
		return TeamReport{}, err
	}
	report, err := teamResult(result.Output)
	if err == nil {
		report.Roles = plan.Roles
	}
	return report, err
}

func runBootstrap(ctx context.Context, id target.Identity, dockerContext string, afterSetup bool, script string, r InputRunner) (process.Result, error) {
	paths := []string{id.Root, filepath.Join(id.Root, ".hermes"), filepath.Join(id.Root, ".hermes-repokit.lock")}
	identities := make([]string, 0, len(paths))
	if issues := target.Inspect(id, ""); len(issues) > 0 {
		return process.Result{}, fmt.Errorf("unsafe native initialization target")
	}
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return process.Result{}, fmt.Errorf("native initialization identity unavailable")
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || info.Mode()&os.ModeSymlink != 0 {
			return process.Result{}, fmt.Errorf("unsafe native initialization identity")
		}
		identities = append(identities, fmt.Sprintf("%d:%d", st.Dev, st.Ino))
	}
	args := []string{"--context", dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes", "--env", "HOME=/opt/data", "--workdir", "/workspace", "hermes", "/usr/bin/flock", "-n", "/workspace/.hermes-repokit.lock", "/bin/sh", "-s", "--", "/workspace", "/opt/data"}
	args = append(args, identities...)
	args = append(args, strconv.FormatBool(afterSetup))
	result := r.RunInput(ctx, strings.NewReader(script), "docker", args...)
	if result.Err != nil || result.Truncated {
		return process.Result{}, fmt.Errorf("native initialization failed or was interrupted; preserve state, inspect with native commands, then rerun install")
	}
	return result, nil
}

func resetRoles(plan teamPlan) []string {
	out := []string{}
	for _, role := range plan.Roles {
		if role.State == "reset" {
			out = append(out, role.Profile)
		}
	}
	return out
}

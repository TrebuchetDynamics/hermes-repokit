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

// Initialize uses the single existing, source-qualified Hermes runtime. Caller
// must first verify its image, project, mounts and running state. The lock is
// held INSIDE that container, so killing the Docker client cannot release it
// while a daemon-owned native subprocess is still writing state.
func Initialize(ctx context.Context, id target.Identity, dockerContext string, afterSetup bool, r InputRunner) error {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return fmt.Errorf("native team state unavailable")
	}
	defer root.Close()
	script, status, drift, err := teamScript(id, afterSetup, nativeTeamCLI(ctx, id, dockerContext, r), root)
	if err != nil {
		return err
	}
	if script == "" {
		script = bootstrapScript + "\n"
	}
	if len(drift) > 0 {
		status = "drift"
	}
	marker, _ := json.Marshal(struct {
		Status string   `json:"status"`
		Drift  []string `json:"drift"`
	}{status, drift})
	script += "printf '%s\\n' 'REPOKIT_TEAM=" + string(marker) + "'\n"
	result, err := runBootstrap(ctx, id, dockerContext, afterSetup, script, r)
	if err != nil {
		return err
	}
	return teamResult(result.Output)
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

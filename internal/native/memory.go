package native

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed memory.py
var memoryScript string

//go:embed memory_server.py
var memoryServerScript string

// SetupMemory inherits the operator's terminal for all private native setup.
// The caller verifies the generated deployment and profile readiness first.
func SetupMemory(id target.Identity, dockerContext string, stdin io.Reader, stdout, stderr io.Writer) int {
	_, err := os.Lstat(filepath.Join(id.Root, ".hermes/openviking/ov.conf"))
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(stderr, "cannot inspect native OpenViking configuration")
		return 1
	}
	return memorySetup(id, dockerContext, err == nil, func(cmd *exec.Cmd) int {
		cmd.Env = process.CleanEnvironment(os.Environ())
		return runTerminal(cmd, stdin, stdout, stderr)
	}, func() error {
		return configureMemory(context.Background(), id, dockerContext, process.Runner{Timeout: 2 * time.Minute})
	}, stderr)
}

func memorySetup(id target.Identity, dockerContext string, configured bool, run func(*exec.Cmd) int, configure func() error, out io.Writer) int {
	compose := func(args ...string) *exec.Cmd {
		return exec.Command("docker", append([]string{"--context", dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose}, args...)...)
	}
	native := func(command string) *exec.Cmd {
		return compose("exec", "openviking", "openviking-server", command)
	}
	if !configured {
		fmt.Fprintln(out, "In native OpenViking init, select remote binding 0.0.0.0:1933 with API-key auth and persistent workspace /app/.openviking/data. Keep the root key in native server state; Hermes needs a separate normal repository user key.")
		fmt.Fprintln(out, "Enter embedding and extraction-model credentials only in native setup. Decline 'Start the server now?'; the container entrypoint manages the server.")
		if code := run(native("init")); code != 0 {
			return code
		}
	}
	if code := run(native("doctor")); code != 0 {
		return code
	}
	if code := run(compose("exec", "-T", "openviking", "python", "-c", memoryServerScript, "validate")); code != 0 {
		return code
	}
	// The entrypoint notices a first config, but an existing server only reads
	// model/storage changes at startup. Reload before the native health gate.
	if code := run(compose("restart", "openviking")); code != 0 {
		return code
	}
	if code := run(compose("exec", "-T", "openviking", "python", "-c", memoryServerScript, "health")); code != 0 {
		return code
	}
	fmt.Fprintln(out, "Use Custom URL http://openviking:1933 and a normal user key for account repokit and this repository. Choose Mirror to OpenViking store to share the native connection with all six profiles.")
	if code := run(exec.Command(id.Launcher, "-p", "default", "memory", "setup", "openviking")); code != 0 {
		return code
	}
	if err := configure(); err != nil {
		fmt.Fprintln(out, err)
		return 1
	}
	fmt.Fprintln(out, "Native shared memory connection checked. Cross-profile recall and repository isolation still require live acceptance.")
	return 0
}

func configureMemory(ctx context.Context, id target.Identity, dockerContext string, r InputRunner) error {
	var roles []string
	for _, role := range team.Roster() {
		roles = append(roles, role.Name)
	}
	payload, _ := json.Marshal(struct {
		Roles  []string `json:"roles"`
		RepoID string   `json:"repo_id"`
	}{roles, id.Project})
	script := bootstrapScript + "\npython - <<'REPOKIT_MEMORY_PY'\n" + memoryScript + "\nimport base64\ntry:\n    main(json.loads(base64.b64decode('" + base64.StdEncoding.EncodeToString(payload) + "')))\nexcept Exception:\n    print('Shared memory activation incomplete; inspect native connection, user-key identity and profile drift. Profile linking may be partial.',file=sys.stderr)\n    sys.exit(1)\nREPOKIT_MEMORY_PY\n"
	_, err := runBootstrap(ctx, id, dockerContext, false, script, r)
	if err != nil {
		return fmt.Errorf("shared memory activation incomplete; inspect native setup, repository user identity and preserved profile configuration")
	}
	return nil
}

package native

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/team"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
)

const repositoryOVCLIConfig = "/opt/data/.openviking/ovcli.conf.repository"

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
		return validateServerConfig(id)
	}, func() error {
		return configureMemory(context.Background(), id, dockerContext, process.Runner{Timeout: 2 * time.Minute})
	}, stderr)
}

func memorySetup(id target.Identity, dockerContext string, configured bool, run func(*exec.Cmd) int, validate, configure func() error, out io.Writer) int {
	compose := func(args ...string) *exec.Cmd {
		return exec.Command("docker", append([]string{"--context", dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose}, args...)...)
	}
	native := func(command string) *exec.Cmd {
		return compose("exec", "--user", "hermes", "hermes", "repokit-openviking", "server", command)
	}
	if !configured {
		fmt.Fprintln(out, "In native OpenViking init, select Remote mode with API-key auth on port 1933; the installed wrapper forces the actual listener to 127.0.0.1. Select persistent workspace /opt/data/openviking/data. Keep the root key in native server state; Hermes needs a separate normal repository user key.")
		fmt.Fprintln(out, "Enter embedding and extraction-model credentials only in native setup. Decline 'Start the server now?'; the Hermes container supervisor manages the server.")
		if code := run(native("init")); code != 0 {
			return code
		}
	}
	if code := run(native("doctor")); code != 0 {
		return code
	}
	if err := validate(); err != nil {
		fmt.Fprintln(out, "OpenViking server configuration is not qualified; inspect private native setup.")
		return 1
	}
	// The entrypoint notices a first config, but an existing server only reads
	// model/storage changes at startup. Reload before the native health gate.
	if code := run(compose("exec", "-T", "--user", "hermes", "hermes", "repokit-openviking", "restart")); code != 0 {
		return code
	}
	if code := run(compose("exec", "-T", "--user", "hermes", "hermes", "repokit-openviking", "health")); code != 0 {
		return code
	}
	fmt.Fprintln(out, "Use Custom URL http://127.0.0.1:1933 and a normal user key for account repokit and this repository. Choose Mirror to OpenViking store to share the native connection with all six profiles.")
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
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return fmt.Errorf("shared memory activation incomplete: private state unavailable")
	}
	defer root.Close()
	info, err := root.Lstat(".hermes/.openviking/ovcli.conf.repository")
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("shared memory activation incomplete: private repository user connection unavailable")
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Getuid() {
		return fmt.Errorf("shared memory activation incomplete: user connection ownership unverified")
	}
	run := func(env string, args ...string) process.Result {
		argv := []string{"--context", dockerContext, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes"}
		if env != "" {
			argv = append(argv, "--env", env)
		}
		argv = append(argv, "hermes")
		argv = append(argv, args...)
		return r.RunInput(ctx, nil, "docker", argv...)
	}
	var health map[string]any
	status := run("OPENVIKING_CLI_CONFIG_FILE="+repositoryOVCLIConfig, "/app/.venv/bin/ov", "-o", "json", "health")
	if status.Err != nil || status.Truncated || json.Unmarshal([]byte(status.Output), &health) != nil {
		return fmt.Errorf("shared memory activation incomplete: native repository user health unavailable")
	}
	if result, ok := health["result"].(map[string]any); ok {
		health = result
	}
	if health["auth_mode"] != "api_key" || health["role"] != "user" || health["account_id"] != "repokit" || health["user_id"] != id.Project {
		return fmt.Errorf("shared memory activation incomplete: native repository user identity differs")
	}
	type update struct {
		role       string
		connection map[string]any
		configHash string
	}
	var pending []update
	for _, role := range team.Roster() {
		name := role.Name
		get := func(key string) (any, bool) {
			value := run("", "hermes", "-p", name, "config", "get", key, "--json")
			if value.Err != nil || value.Truncated {
				return nil, false
			}
			var decoded any
			if json.Unmarshal([]byte(strings.TrimSpace(value.Output)), &decoded) != nil {
				return nil, false
			}
			if obj, ok := decoded.(map[string]any); ok {
				decoded = obj["value"]
			}
			return decoded, true
		}
		provider, ok := get("memory.provider")
		if !ok && name == "default" {
			return fmt.Errorf("shared memory activation incomplete: native default memory setup missing")
		}
		if ok && provider != "" && provider != "builtin" && provider != "openviking" {
			return fmt.Errorf("shared memory activation incomplete: owner memory provider differs")
		}
		priorValue, hasPrior := get("memory.openviking")
		prior := map[string]any{}
		if hasPrior && priorValue != nil {
			var valid bool
			prior, valid = priorValue.(map[string]any)
			if !valid {
				return fmt.Errorf("shared memory activation incomplete: owner memory connection differs")
			}
		}
		for key, allowed := range map[string][]any{
			"agent": {nil, ""}, "api_key": {nil, ""},
			"endpoint": {nil, "", "http://127.0.0.1:1933"},
			"account":  {nil, "", "repokit"}, "user": {nil, "", id.Project},
			"ovcli_config_path": {nil, "", repositoryOVCLIConfig},
			"use_ovcli_config":  {nil, false, true},
		} {
			value := prior[key]
			matched := false
			for _, candidate := range allowed {
				if reflect.DeepEqual(value, candidate) {
					matched = true
				}
			}
			if !matched {
				return fmt.Errorf("shared memory activation incomplete: owner memory connection differs")
			}
		}
		if provider == "openviking" && prior["ovcli_config_path"] != repositoryOVCLIConfig {
			return fmt.Errorf("shared memory activation incomplete: active owner memory connection differs")
		}
		connection := make(map[string]any, len(prior)+2)
		for key, value := range prior {
			connection[key] = value
		}
		connection["use_ovcli_config"] = true
		connection["ovcli_config_path"] = repositoryOVCLIConfig
		configPath := ".hermes/config.yaml"
		if name != "default" {
			configPath = ".hermes/profiles/" + name + "/config.yaml"
		}
		configInfo, err := root.Lstat(configPath)
		if err != nil || !configInfo.Mode().IsRegular() {
			return fmt.Errorf("shared memory activation incomplete: native profile configuration unavailable")
		}
		configBytes, err := root.ReadFile(configPath)
		if err != nil || len(configBytes) > 1024*1024 {
			return fmt.Errorf("shared memory activation incomplete: native profile configuration cannot be read safely")
		}
		digest := sha256.Sum256(configBytes)
		pending = append(pending, update{name, connection, hex.EncodeToString(digest[:])})
	}
	keyBytes, err := root.ReadFile(".hermes/.openviking/ovcli.conf.repository")
	if err != nil || len(keyBytes) == 0 || len(keyBytes) > 1024*1024 {
		return fmt.Errorf("shared memory activation incomplete: repository user key cannot be read safely")
	}
	keyDigest := sha256.Sum256(keyBytes)
	script := bootstrapScript + "\n"
	checkHash := func(digest, path string) {
		script += "[ ! -L '" + path + "' ]\nprintf '%s  %s\\n' '" + digest + "' '" + path + "' | sha256sum -c --status\n"
	}
	checkHash(hex.EncodeToString(keyDigest[:]), repositoryOVCLIConfig)
	for _, next := range pending {
		path := "/opt/data/config.yaml"
		if next.role != "default" {
			path = "/opt/data/profiles/" + next.role + "/config.yaml"
		}
		checkHash(next.configHash, path)
	}
	for _, next := range pending {
		payload, _ := json.Marshal(next.connection)
		script += "connection=$(printf %s '" + base64.StdEncoding.EncodeToString(payload) + "' | base64 -d)\n"
		script += "hermes -p '" + next.role + "' config set memory.openviking \"$connection\" >/dev/null\n"
		for _, entry := range [][2]string{{"memory.memory_enabled", "true"}, {"memory.user_profile_enabled", "true"}, {"memory.provider", "openviking"}} {
			script += "hermes -p '" + next.role + "' config set '" + entry[0] + "' '" + entry[1] + "' >/dev/null\n"
		}
		script += "hermes -p '" + next.role + "' memory status >/dev/null\n"
	}
	script += "printf 'REPOKIT_MEMORY=configured\\n'\n"
	result, err := runBootstrap(ctx, id, dockerContext, false, script, r)
	if err != nil || !strings.Contains(result.Output, "REPOKIT_MEMORY=configured") {
		return fmt.Errorf("shared memory activation partial or unverified; inspect native profile status before retrying")
	}
	return nil
}

func validateServerConfig(id target.Identity) error {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	info, err := root.Lstat(".hermes/openviking/ov.conf")
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("private OpenViking configuration unavailable")
	}
	data, err := root.ReadFile(".hermes/openviking/ov.conf")
	if err != nil || len(data) == 0 || len(data) > 1024*1024 {
		return fmt.Errorf("private OpenViking configuration cannot be read safely")
	}
	var config struct {
		Storage struct {
			Workspace string `json:"workspace"`
		} `json:"storage"`
		Server struct {
			Host       string `json:"host"`
			Port       int    `json:"port"`
			RootAPIKey string `json:"root_api_key"`
		} `json:"server"`
		Memory struct {
			ExtractionEnabled *bool `json:"extraction_enabled"`
		} `json:"memory"`
	}
	if json.Unmarshal(data, &config) != nil || config.Storage.Workspace != "/opt/data/openviking/data" || (config.Server.Host != "127.0.0.1" && config.Server.Host != "0.0.0.0") || config.Server.Port != 1933 || config.Server.RootAPIKey == "" || (config.Memory.ExtractionEnabled != nil && !*config.Memory.ExtractionEnabled) {
		return fmt.Errorf("native OpenViking configuration does not meet repository storage and API-key requirements")
	}
	return nil
}

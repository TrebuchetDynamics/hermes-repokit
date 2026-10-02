package native

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/boardwatch"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// EnsureBoardWatch installs or upgrades RepoKit's board watch cron job on
// default through public Hermes commands, and leaves alone a job the owner
// created, paused, edited or removed.
func EnsureBoardWatch(ctx context.Context, id target.Identity, dc string, r InputRunner) (boardwatch.Decision, error) {
	state, err := os.OpenRoot(filepath.Join(id.Root, ".hermes"))
	if err != nil {
		return "", err
	}
	defer state.Close()
	job, marker, disk, err := boardwatch.Read(state)
	if err != nil {
		return "", err
	}
	run := nativeTeamCLI(ctx, id, dc, r)
	deliver := "local"
	if platforms := chatPlatforms(run); len(platforms) > 0 {
		deliver = platforms[0]
	}
	want := boardwatch.Desired(deliver)
	decision := boardwatch.Decide(job, marker, disk, want)
	if decision != boardwatch.Create && decision != boardwatch.Upgrade {
		return decision, nil
	}
	if err := writeState(ctx, id, dc, r, "/opt/data/scripts/"+boardwatch.ScriptName, boardwatch.Script); err != nil {
		return "", err
	}
	fields := []string{"--prompt", want.Prompt, "--monitor-script", want.MonitorScript, "--deliver", want.Deliver, "--failure-deliver", "local"}
	if decision == boardwatch.Create {
		args := append([]string{"-p", "default", "cron", "create", want.Schedule, "--name", boardwatch.JobName}, fields[2:]...)
		if _, err := run(append(args, want.Prompt)...); err != nil {
			return "", errors.New("board watch job not created")
		}
		if job, _, _, err = boardwatch.Read(state); err != nil || job == nil {
			return "", errors.New("board watch job not found after creation")
		}
	} else {
		// An empty --workdir clears one an earlier release set.
		args := append([]string{"-p", "default", "cron", "edit", job.ID, "--schedule", want.Schedule, "--workdir", want.Workdir}, fields...)
		if _, err := run(args...); err != nil {
			return "", errors.New("board watch job not upgraded")
		}
	}
	data, _ := json.Marshal(boardwatch.Marker{JobID: job.ID, Spec: want})
	if err := writeState(ctx, id, dc, r, "/opt/data/"+boardwatch.MarkerName, data); err != nil {
		return "", err
	}
	return decision, nil
}

// writeState writes one private file under default's state, 0600, atomically.
func writeState(ctx context.Context, id target.Identity, dc string, r InputRunner, path string, content []byte) error {
	script := `umask 077; mkdir -p "$(dirname "$1")" && cat > "$1.tmp" && chmod 0600 "$1.tmp" && mv -f "$1.tmp" "$1"`
	res := r.RunInput(ctx, bytes.NewReader(content), "docker", "--context", dc, "compose", "--env-file", "/dev/null", "-f", id.Compose, "exec", "-T", "--user", "hermes", "hermes", "sh", "-c", script, "sh", path)
	if res.Err != nil {
		return errors.New("could not write " + path)
	}
	return nil
}

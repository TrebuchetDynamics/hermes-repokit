package native

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"time"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// DispatchCheckBody asks for evidence the checker can compute independently.
const DispatchCheckBody = "Use read_file to read line 1 of /workspace/README.md. Do not modify any files. " +
	"Copy the first physical line exactly, excluding only its newline; preserve spaces, Markdown markers and HTML. " +
	"For an empty file or blank first line use an empty string. Only if the file does not exist use README_MISSING; " +
	"permission or tool failures must block the task. Complete with a nonempty summary and structured metadata " +
	`{"first_line": "the exact first line", "changed_files": []}.`

var taskID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

// ReadmeFirstLine is the expected dispatch-check answer for a repository.
func ReadmeFirstLine(id target.Identity) (string, error) {
	root, err := os.OpenRoot(id.Root)
	if err != nil {
		return "", err
	}
	defer root.Close()
	info, err := root.Lstat("README.md")
	if os.IsNotExist(err) {
		return "README_MISSING", nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return "", errors.New("README.md is not a bounded regular file")
	}
	data, err := root.ReadFile("README.md")
	if err != nil {
		return "", err
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	return string(bytes.TrimSuffix(line, []byte("\r"))), nil
}

type taskRecord struct {
	Task struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"task"`
	Runs []struct {
		Profile  string          `json:"profile"`
		Status   string          `json:"status"`
		Outcome  string          `json:"outcome"`
		Summary  string          `json:"summary"`
		Metadata json.RawMessage `json:"metadata"`
	} `json:"runs"`
}

func runMetadata(raw json.RawMessage) map[string]any {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	if s, ok := value.(string); ok && json.Unmarshal([]byte(s), &value) != nil {
		return nil
	}
	m, _ := value.(map[string]any)
	return m
}

// DispatchCheckPassed accepts only a default run that completed the card
// with the independently computed README line and no changed files.
func DispatchCheckPassed(record taskRecord, expected string) bool {
	if record.Task.Status != "done" || len(record.Runs) == 0 {
		return false
	}
	run := record.Runs[len(record.Runs)-1]
	meta := runMetadata(run.Metadata)
	changed, _ := meta["changed_files"].([]any)
	return run.Profile == "default" && run.Status == "done" && run.Outcome == "completed" &&
		run.Summary != "" && meta["first_line"] == expected && meta["changed_files"] != nil && len(changed) == 0
}

// DispatchCheck is an explicit, bounded mutation: it creates one default
// card and waits for the running gateway to claim and complete it without any
// manual dispatch. A passing card is archived; a failing one is preserved.
func DispatchCheck(ctx context.Context, id target.Identity, dc string, r InputRunner, claimWithin, finishWithin, pause time.Duration) (string, error) {
	run := nativeTeamCLI(ctx, id, dc, r)
	value, err := configValue(run, "default", "kanban")
	kanban, ok := value.(map[string]any)
	if err != nil || !ok || !OperationalPolicy(kanban) {
		return "", errors.New("automatic dispatch is not configured; run setup first")
	}
	if pid, err := gatewayPID(run); err != nil || pid == 0 {
		return "", errors.New("default gateway is not running")
	}
	expected, err := ReadmeFirstLine(id)
	if err != nil {
		return "", errors.New("README.md cannot be read safely for the expected answer")
	}
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	raw, err := run("-p", "default", "kanban", "create", "RepoKit dispatch check",
		"--assignee", "default", "--workspace", "dir:/workspace", "--created-by", "default",
		"--idempotency-key", "repokit-dispatch-check-"+hex.EncodeToString(nonce),
		"--max-runtime", "180", "--max-retries", "1", "--priority", "100",
		"--completion-contract", "local-only", "--body", DispatchCheckBody, "--json")
	var created struct {
		ID string `json:"id"`
	}
	if err != nil || json.Unmarshal(raw, &created) != nil || !taskID.MatchString(created.ID) {
		return "", errors.New("native card creation failed")
	}
	start := time.Now()
	for {
		raw, err := run("-p", "default", "kanban", "show", created.ID, "--json")
		var record taskRecord
		if err == nil && json.Unmarshal(raw, &record) == nil && record.Task.ID == created.ID {
			if DispatchCheckPassed(record, expected) {
				if _, err := run("-p", "default", "kanban", "archive", created.ID); err != nil {
					return created.ID, errors.New("check passed but the card could not be archived")
				}
				return created.ID, nil
			}
			switch record.Task.Status {
			case "done", "blocked", "archived":
				return created.ID, errors.New("default did not return the expected no-write evidence; card preserved")
			}
			if len(record.Runs) == 0 && time.Since(start) > claimWithin {
				return created.ID, errors.New("gateway did not claim the card; card preserved")
			}
		}
		if time.Since(start) > finishWithin {
			return created.ID, errors.New("card did not finish in time; card preserved")
		}
		select {
		case <-ctx.Done():
			return created.ID, ctx.Err()
		case <-time.After(pause):
		}
	}
}

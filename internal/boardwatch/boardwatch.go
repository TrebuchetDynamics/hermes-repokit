// Package boardwatch owns RepoKit's board watch: the Hermes cron job that
// wakes default when a quiet board has an open goal or a stuck card.
package boardwatch

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

// Script is the cron job's monitor script.
//
//go:embed repokit-board-watch.py
var Script []byte

const (
	JobName    = "repokit-board-watch"
	ScriptName = "repokit-board-watch.py"
	MarkerName = "repokit-board-watch.json"
	Schedule   = "every 5m"
	Workdir    = "/workspace"
)

// Prompt is what default reads when the watch wakes it; the monitor digest
// follows it in the run context.
const Prompt = `RepoKit board watch woke you: this board is idle and the digest below names open goal cards (title "Goal:"), cards in triage and blocked cards. Do one bounded pass, then stop.
1. A triage card whose cause is now resolved (an owner decision recorded on it, a capability added): return it with "hermes kanban specify <id>", then immediately restore its original title and body with "hermes kanban edit <id> --title <original title> --body <original body>". Otherwise leave it.
2. Each goal card: read it and its comments. If its acceptance is met with evidence, complete it with that evidence. If its budget is spent, block it with one question for the owner. Otherwise create the next card toward it (after checking the board for one that already covers the work), then link it as a parent of the goal card with kanban_link.
3. Blocked cards waiting on the owner: once a day, report them in one short message, skipping any whose comments snooze them past today.
Never create work that no goal card asks for. If nothing needs doing or saying, reply exactly [SILENT].`

// Spec is what RepoKit manages on the job and its script.
type Spec struct {
	Prompt        string `json:"prompt"`
	Schedule      string `json:"schedule"`
	MonitorScript string `json:"monitor_script"`
	Workdir       string `json:"workdir"`
	Deliver       string `json:"deliver"`
	ScriptSHA     string `json:"script_sha256"`
}

// Job is the native cron job named JobName.
type Job struct {
	ID     string
	Name   string
	Paused bool
	Spec   Spec
}

// Marker records what RepoKit last wrote; its presence means RepoKit created
// the job, so a missing job is the owner's removal.
type Marker struct {
	JobID string `json:"job_id"`
	Spec  Spec   `json:"spec"`
}

type Decision string

const (
	Create        Decision = "create"
	Upgrade       Decision = "upgrade"
	Current       Decision = "current"
	Paused        Decision = "paused"
	OwnerModified Decision = "owner-modified"
	OptedOut      Decision = "opted-out"
	OwnerJob      Decision = "owner-job"
)

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// Desired is the job this release installs, delivering to deliver ("local"
// when the deployment has no messaging platform).
func Desired(deliver string) Spec {
	return Spec{Prompt: Prompt, Schedule: Schedule, MonitorScript: ScriptName, Workdir: Workdir, Deliver: deliver, ScriptSHA: sha(Script)}
}

// Decide never rewrites what the owner changed: a job RepoKit did not create,
// a paused or edited job, an edited script, or a removed job.
func Decide(job *Job, marker *Marker, diskSHA string, want Spec) Decision {
	switch {
	case marker == nil && job == nil:
		return Create
	case marker == nil:
		return OwnerJob
	case job == nil || job.ID != marker.JobID:
		return OptedOut
	case job.Paused:
		return Paused
	}
	installed := job.Spec
	installed.ScriptSHA = diskSHA
	if installed != marker.Spec {
		return OwnerModified
	}
	if marker.Spec != want {
		return Upgrade
	}
	return Current
}

// Read observes the job, marker and script from the host's .hermes root.
func Read(state *os.Root) (*Job, *Marker, string, error) {
	var job *Job
	if data, err := state.ReadFile("cron/jobs.json"); err == nil {
		var store struct {
			Jobs []struct {
				ID, Name, Prompt, Workdir, Deliver, State string
				ScheduleDisplay                           string `json:"schedule_display"`
				MonitorScript                             string `json:"monitor_script"`
				Enabled                                   *bool
			}
		}
		if err := json.Unmarshal(data, &store); err != nil {
			return nil, nil, "", errors.New("cron job store unreadable")
		}
		for _, j := range store.Jobs {
			if j.Name == JobName {
				paused := j.State == "paused" || (j.Enabled != nil && !*j.Enabled)
				job = &Job{ID: j.ID, Name: j.Name, Paused: paused, Spec: Spec{Prompt: j.Prompt, Schedule: j.ScheduleDisplay, MonitorScript: j.MonitorScript, Workdir: j.Workdir, Deliver: j.Deliver}}
				break
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	var marker *Marker
	if data, err := state.ReadFile(MarkerName); err == nil {
		marker = &Marker{}
		if json.Unmarshal(data, marker) != nil {
			return nil, nil, "", errors.New("board watch marker unreadable")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	disk := ""
	if data, err := state.ReadFile("scripts/" + ScriptName); err == nil {
		disk = sha(data)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	if job != nil {
		job.Spec.ScriptSHA = disk
	}
	return job, marker, disk, nil
}

package boardwatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecide(t *testing.T) {
	want := Desired("telegram")
	ours := &Marker{JobID: "j1", Spec: want}
	job := &Job{ID: "j1", Name: JobName, Spec: want}
	older := want
	older.Prompt = "old prompt"
	olderMarker := &Marker{JobID: "j1", Spec: older}
	olderJob := &Job{ID: "j1", Name: JobName, Spec: older}
	edited := *job
	edited.Spec.Schedule = "every 1h"
	paused := *job
	paused.Paused = true
	for _, tc := range []struct {
		name   string
		job    *Job
		marker *Marker
		disk   string
		want   Decision
	}{
		{"fresh", nil, nil, "", Create},
		{"owner's own job", job, nil, want.ScriptSHA, OwnerJob},
		{"current", job, ours, want.ScriptSHA, Current},
		{"owner removed it", nil, ours, want.ScriptSHA, OptedOut},
		{"owner paused it", &paused, ours, want.ScriptSHA, Paused},
		{"owner edited job", &edited, ours, want.ScriptSHA, OwnerModified},
		{"owner edited script", job, ours, "deadbeef", OwnerModified},
		{"older release", olderJob, olderMarker, older.ScriptSHA, Upgrade},
		{"delivery target changed", job, ours, want.ScriptSHA, Current},
	} {
		w := want
		if tc.name == "delivery target changed" {
			w = Desired("discord")
			if got := Decide(tc.job, tc.marker, tc.disk, w); got != Upgrade {
				t.Errorf("%s: got %s want %s", tc.name, got, Upgrade)
			}
			continue
		}
		if got := Decide(tc.job, tc.marker, tc.disk, w); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func TestReadHostState(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cron"), 0700)
	os.MkdirAll(filepath.Join(dir, "scripts"), 0700)
	jobs := `{"jobs":[{"id":"x1","name":"other"},{"id":"j1","name":"repokit-board-watch","prompt":"p","schedule_display":"every 5m","monitor_script":"repokit-board-watch.py","workdir":"/workspace","deliver":"telegram","enabled":false,"state":"paused"}]}`
	os.WriteFile(filepath.Join(dir, "cron", "jobs.json"), []byte(jobs), 0600)
	os.WriteFile(filepath.Join(dir, "scripts", ScriptName), Script, 0600)
	os.WriteFile(filepath.Join(dir, MarkerName), []byte(`{"job_id":"j1","spec":{"prompt":"p"}}`), 0600)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	job, marker, sha, err := Read(root)
	if err != nil || job == nil || job.ID != "j1" || !job.Paused || job.Spec.Deliver != "telegram" || marker == nil || marker.JobID != "j1" || sha != Desired("").ScriptSHA {
		t.Fatalf("read: %+v %+v %q %v", job, marker, sha, err)
	}
	empty, _ := os.OpenRoot(t.TempDir())
	if job, marker, sha, err := Read(empty); job != nil || marker != nil || sha != "" || err != nil {
		t.Fatalf("absent state: %+v %+v %q %v", job, marker, sha, err)
	}
}

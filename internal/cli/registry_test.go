package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// install records its repository, list shows it with live state from
// anywhere, and remove forgets it.
func TestInstallRecordsListShowsAndRemoveForgets(t *testing.T) {
	a, r := installedForRemoval(t)
	records, err := loadRegistry()
	d, ok := records[r.id.Root]
	if err != nil || !ok || d.Name != r.id.Name || d.Container != r.id.Container || d.LastResult != "ok" || d.RepoKit != Version || d.LastInstall.IsZero() {
		t.Fatalf("install not recorded: %+v %v", records, err)
	}
	anywhere := a
	anywhere.Directory = t.TempDir()
	code, out, diag := invoke(t, anywhere, "list")
	if code != 0 || !strings.Contains(out, r.id.Name) || !strings.Contains(out, ".hermes present") || !strings.Contains(out, "LAST INSTALL") {
		t.Fatalf("list: code=%d\n%s%s", code, out, diag)
	}
	code, out, _ = invoke(t, anywhere, "list", "--json")
	var rows []map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &rows) != nil || len(rows) != 1 || rows[0]["root"] != r.id.Root {
		t.Fatalf("list --json: %d %s", code, out)
	}
	a.Confirm = func(string) (string, error) { return r.id.Name, nil }
	if code, _, diag := invoke(t, a, "remove"); code != 0 {
		t.Fatalf("remove: %s", diag)
	}
	if records, _ := loadRegistry(); len(records) != 0 {
		t.Fatalf("removed deployment still recorded: %+v", records)
	}
}

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// Catch a canary falling through to runtime execution before its lifecycle is
// qualified, or presenting an unsupported check as health/recall success.
func TestMemoryCheckBlocksBeforeRuntimeAndPreservesState(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, ".hermes")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, "pending-session.json")
	original := []byte(`{"session_id":"unrelated-pending-session"}`)
	if err := os.WriteFile(state, original, 0600); err != nil {
		t.Fatal(err)
	}
	r := &fakeRunner{}
	var out, diagnostics bytes.Buffer
	code := (App{Directory: root, Runner: r}).Run([]string{"verify", "--memory-check"}, &out, &diagnostics)
	if code != 1 || diagnostics.Len() != 0 {
		t.Fatalf("want qualification failure, got code=%d stderr=%q stdout=%q", code, diagnostics.String(), out.String())
	}
	var probes []verify.Probe
	if err := json.Unmarshal(out.Bytes(), &probes); err != nil {
		t.Fatalf("structured qualification report: %v; output=%q", err, out.String())
	}
	want := map[string]verify.Status{
		"memory-check":                verify.Unsupported,
		"memory-write":                verify.Unqualified,
		"memory-extraction":           verify.Unqualified,
		"memory-recall-same-profile":  verify.Unqualified,
		"memory-recall-cross-profile": verify.Unqualified,
		"memory-cleanup":              verify.Unqualified,
	}
	got := make(map[string]verify.Status)
	for _, p := range probes {
		if _, duplicate := got[p.Component]; duplicate {
			t.Fatalf("duplicate evidence layer: %s", p.Component)
		}
		got[p.Component] = p.Status
		if p.Detail == "" {
			t.Fatalf("missing explanation for %s", p.Component)
		}
		if p.Component == "memory-check" {
			for _, reason := range []string{"isolation", "cleanup", "pending-session"} {
				if !strings.Contains(p.Detail, reason) {
					t.Errorf("missing blocker %q: %q", reason, p.Detail)
				}
			}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence layers = %v, want %v", got, want)
	}
	if len(r.calls) != 0 {
		t.Fatalf("unsupported lifecycle must block before subprocess execution: %v", r.calls)
	}
	data, err := os.ReadFile(state)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("unrelated state changed: %q, %v", data, err)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 1 {
		t.Fatalf("check created native state: %v, %v", entries, err)
	}
	entries, err = os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("check created repository state: %v, %v", entries, err)
	}
}

// Explicitly declining the canary must retain the ordinary read-only report.
func TestMemoryCheckFalsePreservesOrdinaryVerify(t *testing.T) {
	root := t.TempDir()
	fRoot = root
	var baseline, baselineErr, out, diagnostics bytes.Buffer
	ordinaryRunner, optedOutRunner := &fakeRunner{}, &fakeRunner{}
	baselineCode := (App{Directory: root, Runner: ordinaryRunner}).Run([]string{"verify"}, &baseline, &baselineErr)
	code := (App{Directory: root, Runner: optedOutRunner}).Run([]string{"verify", "--memory-check=false"}, &out, &diagnostics)
	if code != baselineCode || out.String() != baseline.String() || diagnostics.String() != baselineErr.String() || !reflect.DeepEqual(ordinaryRunner.calls, optedOutRunner.calls) {
		t.Fatalf("opt-out changed ordinary verify: code=%d/%d stdout=%q/%q stderr=%q/%q", code, baselineCode, out.String(), baseline.String(), diagnostics.String(), baselineErr.String())
	}
}

func TestMemoryCheckRejectsInvalidRequestsBeforeRuntime(t *testing.T) {
	for _, args := range [][]string{
		{"plan", "--memory-check"}, {"install", "--memory-check"}, {"setup", "--memory-check"},
		{"verify", "--memory-check=secret-invalid"}, {"verify", "--memory-check", "secret-positional"},
		{"verify", "--memory-check", "--"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			r := &fakeRunner{}
			var out, diagnostics bytes.Buffer
			code := (App{Directory: t.TempDir(), Runner: r}).Run(args, &out, &diagnostics)
			if code != 2 || out.Len() != 0 || len(r.calls) != 0 || strings.Contains(diagnostics.String(), "secret-") {
				t.Fatalf("invalid request reached verification or leaked input: code=%d stdout=%q stderr=%q calls=%v", code, out.String(), diagnostics.String(), r.calls)
			}
		})
	}
}

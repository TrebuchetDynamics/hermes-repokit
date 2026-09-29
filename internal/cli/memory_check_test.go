package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/verify"
)

// An unconfigured repository must not reach a memory mutation or claim recall.
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
		"memory-check":                verify.Unqualified,
		"memory-recall-same-profile":  verify.Unqualified,
		"memory-recall-cross-profile": verify.Unqualified,
		"memory-extraction":           verify.Unqualified,
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
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence layers = %v, want %v", got, want)
	}
	for _, call := range r.calls {
		if strings.HasPrefix(call, "docker ") || strings.HasPrefix(call, "ov ") || strings.HasPrefix(call, "hermes ") {
			t.Fatalf("unconfigured repository must block before runtime/provider execution: %v", r.calls)
		}
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

func TestNativeMemoryCanaryDeletesExactFileAndChecksIndex(t *testing.T) {
	var uri string
	var marker string
	var calls []string
	searches := 0
	deleted := false
	ov := func(args ...string) process.Result {
		calls = append(calls, args[0])
		switch args[0] {
		case "write":
			uri = args[1]
			if !strings.Contains(strings.Join(args, " "), "--mode create") {
				t.Fatal("write must be create-only")
			}
			for i := range args {
				if args[i] == "--content" {
					marker = args[i+1]
				}
			}
			return process.Result{Output: `{"result":{"uri":"` + uri + `"}}`}
		case "read":
			if args[1] != uri {
				t.Fatal("read escaped exact canary URI")
			}
			if deleted {
				return process.Result{Output: `not found`, Err: errors.New("not found")}
			}
			return process.Result{Output: `{"content":"` + marker + `"}`}
		case "find":
			searches++
			if searches == 1 {
				return process.Result{Output: `{"results":[{"uri":"` + strings.Replace(uri, "viking://~/", "viking://repokit/user/", 1) + `"}]}`}
			}
			return process.Result{Output: `{"results":[]}`}
		case "rm":
			if args[1] != uri {
				t.Fatal("delete escaped exact canary URI")
			}
			deleted = true
			return process.Result{Output: `{}`}
		}
		t.Fatalf("unexpected OpenViking command: %v", args)
		return process.Result{}
	}
	probes := runMemoryCanary(ov)
	if !strings.HasPrefix(uri, "viking://~/memories/repokit-selfcheck/") || searches != 2 || !reflect.DeepEqual(calls, []string{"write", "read", "find", "rm", "read", "find"}) {
		t.Fatalf("incomplete exact-file lifecycle: uri=%q calls=%v", uri, calls)
	}
	for _, p := range probes {
		if p.Status != verify.Healthy {
			t.Fatalf("%s: %s", p.Component, p.Detail)
		}
	}
}

func TestNativeMemoryCanaryUncertainWriteDoesNotRetry(t *testing.T) {
	var calls []string
	probes := runMemoryCanary(func(args ...string) process.Result {
		calls = append(calls, args[0])
		if args[0] == "write" {
			return process.Result{Err: errors.New("timeout")}
		}
		return process.Result{Output: `{}`}
	})
	if !reflect.DeepEqual(calls, []string{"write", "rm"}) || probes[0].Status != verify.Unqualified {
		t.Fatalf("uncertain write must have one exact cleanup and no retry: %v %v", calls, probes)
	}
}

func TestMemoryCheckNativeResponsesFailClosed(t *testing.T) {
	good := `{"result":{"auth_mode":"api_key","role":"user","account_id":"repokit","user_id":"repo-a"}}`
	if !nativeHealthMatches(good, "repo-a") || nativeHealthMatches(good, "repo-b") ||
		nativeHealthMatches(`{"result":{"auth_mode":"api_key","role":"admin","account_id":"repokit","user_id":"repo-a"}}`, "repo-a") {
		t.Fatal("health identity did not enforce repository user authority")
	}
	if validNativeJSON(`{"status":"error","error":"authentication failed"}`) || nativeValueMatches(`{"value":"builtin"}`, "openviking") {
		t.Fatal("native error or mismatched provider was accepted")
	}
}

func TestWriteReceiptMustIdentifyExactCanary(t *testing.T) {
	requested := "viking://~/memories/repokit-selfcheck/id.md"
	if !writeReceiptMatches(`{"result":{"uri":"`+requested+`"}}`, requested) {
		t.Fatal("exact receipt rejected")
	}
	if !writeReceiptMatches(`{"result":{"uri":"viking://repokit/user/memories/repokit-selfcheck/id.md"}}`, requested) {
		t.Fatal("documented home alias canonicalization rejected")
	}
	for _, response := range []string{`{}`, `{"uri":"viking://~/memories/repokit-selfcheck/other.md"}`, `{"uri":"viking://~/memories/repokit-selfcheck"}`} {
		if writeReceiptMatches(response, requested) {
			t.Fatalf("accepted nonmatching write receipt: %s", response)
		}
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

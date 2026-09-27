package native

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupDelegatesExactlyAndPreservesStoppedError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "launcher")
	os.WriteFile(p, []byte("#!/bin/sh\n[ \"$#\" = 3 ] && [ \"$1\" = -p ] && [ \"$2\" = default ] && [ \"$3\" = setup ] || exit 99\nIFS= read -r x\nprintf '%s' \"$x\"\nprintf 'service hermes is not running' >&2\nexit 17\n"), 0700)
	var out, errout bytes.Buffer
	code := Setup(p, "/tmp/repo/.hermes/compose.yaml", "repo-local", strings.NewReader("private terminal input\n"), &out, &errout)
	if code != 17 || out.String() != "private terminal input" || !strings.Contains(errout.String(), "service hermes is not running") || !strings.Contains(errout.String(), "up -d hermes") {
		t.Fatalf("%d %q %q", code, out.String(), errout.String())
	}
}

func TestRecoveryCommandUsesContextAndClearsSelectors(t *testing.T) {
	bin := t.TempDir()
	record := filepath.Join(bin, "record")
	os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n[ -z \"${COMPOSE_PROJECT_NAME+x}\" ] || exit 99\nprintf '%s\\n' \"$@\" > \"$RECORD\"\n"), 0700)
	launcherPath := filepath.Join(bin, "launcher")
	os.WriteFile(launcherPath, []byte("#!/bin/sh\nexit 17\n"), 0700)
	var out, diagnostics bytes.Buffer
	Setup(launcherPath, "/tmp/compose.yaml", "repo-local", strings.NewReader(""), &out, &diagnostics)
	line := strings.TrimSpace(strings.Split(diagnostics.String(), "start it with: ")[1])
	cmd := exec.Command("sh", "-c", line)
	cmd.Env = append(os.Environ(), "PATH="+bin, "RECORD="+record, "COMPOSE_PROJECT_NAME=wrong")
	if err := cmd.Run(); err != nil {
		t.Fatalf("unsafe recovery: %v %s", err, line)
	}
	raw, _ := os.ReadFile(record)
	if !strings.HasPrefix(string(raw), "--context\nrepo-local\ncompose\n") {
		t.Fatalf("context missing: %q", raw)
	}
}

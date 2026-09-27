package native

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupDelegatesExactlyAndPreservesStoppedError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "launcher")
	os.WriteFile(p, []byte("#!/bin/sh\n[ \"$#\" = 1 ] && [ \"$1\" = setup ] || exit 99\nIFS= read -r x\nprintf '%s' \"$x\"\nprintf 'service hermes is not running' >&2\nexit 17\n"), 0700)
	var out, errout bytes.Buffer
	code := Setup(p, "/tmp/repo/.hermes/compose.yaml", strings.NewReader("private terminal input\n"), &out, &errout)
	if code != 17 || out.String() != "private terminal input" || !strings.Contains(errout.String(), "service hermes is not running") || !strings.Contains(errout.String(), "up -d hermes") {
		t.Fatalf("%d %q %q", code, out.String(), errout.String())
	}
}

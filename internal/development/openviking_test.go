package development

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Exercise the pinned entrypoint's TERM-while-waiting failure without Docker,
// sockets, credentials or a model. The child records that it fully drained.
func TestMemoryEntrypointWaitsForServerDrain(t *testing.T) {
	files, _ := Recipe(Requirements{})
	patch, ok := files["patch-openviking-entrypoint.py"]
	if !ok {
		t.Fatal("missing qualified shutdown repair")
	}
	root := t.TempDir()
	patchPath := filepath.Join(root, "patch.py")
	os.WriteFile(patchPath, patch, 0600)
	script := `import os, pathlib, signal, subprocess, sys, time
root=pathlib.Path(sys.argv[1])
entry=root/'entrypoint'
child=root/'child.py'
child.write_text('''import pathlib, signal, sys, time
p=pathlib.Path(sys.argv[1])
def stop(*args):
    time.sleep(0.3)
    (p/'drained').touch()
    raise SystemExit(0)
signal.signal(signal.SIGTERM,stop)
(p/'ready').touch()
while True: time.sleep(0.02)
''')
entry.write_text('''#!/bin/sh
set -eu
forward_signal() { kill "$SERVER_PID"; }
trap 'forward_signal' INT TERM
python3 "$1/child.py" "$1" &
SERVER_PID=$!
wait "$SERVER_PID" || SERVER_STATUS=$?
exit "${SERVER_STATUS:-0}"
''')
subprocess.run([sys.executable,str(root/'patch.py'),str(entry)],check=True)
p=subprocess.Popen(['sh',str(entry),str(root)],start_new_session=True)
try:
    until=time.monotonic()+5
    while not (root/'ready').exists():
        assert time.monotonic()<until, 'child did not start'
        time.sleep(0.01)
    p.send_signal(signal.SIGTERM)
    assert p.wait(timeout=5)==0
    assert (root/'drained').exists(), 'supervisor acknowledged down before child drain'
finally:
    try: os.killpg(p.pid, signal.SIGKILL)
    except ProcessLookupError: pass
before=entry.read_bytes()
assert subprocess.run([sys.executable,str(root/'patch.py'),str(entry)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode!=0
assert entry.read_bytes()==before, 'unexpected upstream revision was modified'
`
	cmd := exec.Command("python3", "-c", script, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shutdown regression: %v %s", err, out)
	}
}

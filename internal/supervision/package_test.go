package supervision

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// This explicitly gated probe loads the built image with no mounts or network.
// It catches dependency/cache leakage from the former mounted-venv fixture.
func TestLayaImageSelfContained(t *testing.T) {
	image := os.Getenv("REPOKIT_LAYA_IMAGE")
	if image == "" {
		t.Skip("set REPOKIT_LAYA_IMAGE to an explicitly built image ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "--context", "default", "run", "--rm", "--pull=never", "--network", "none", "--read-only", "--tmpfs", "/tmp:rw,nosuid,nodev,size=256m", "--user", "1000:1000", "--entrypoint", "python", image, "-c", `import hashlib,importlib.metadata,json
from pathlib import Path
import torch,transformers,laya
assert torch.__version__ == '2.8.0+cpu'
assert transformers.__version__ == '4.57.6'
assert importlib.metadata.version('laya') == '0.3.3'
import hermes_nerve.reflex.laya_service
manifest=json.loads(Path('/opt/laya/model-manifest.json').read_text())
for name,digest in manifest['files'].items():
    with (Path('/model')/name).open('rb') as f: assert hashlib.file_digest(f,'sha256').hexdigest()==digest
print('self-contained pinned runtime and model bytes verified')`)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standalone image: %v %s", err, out)
	}
}

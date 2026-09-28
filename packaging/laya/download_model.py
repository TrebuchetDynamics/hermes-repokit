"""Build-time acquisition only; the final image runs the upstream service offline."""
import hashlib
import json
from pathlib import Path
import urllib.request

manifest = json.loads(Path('/build/model-manifest.json').read_text())
for name, expected in manifest['files'].items():
    target = Path('/model') / name
    target.parent.mkdir(parents=True, exist_ok=True)
    url = f"https://huggingface.co/{manifest['repository']}/resolve/{manifest['revision']}/{name}"
    urllib.request.urlretrieve(url, target)
    with target.open('rb') as stream:
        if hashlib.file_digest(stream, 'sha256').hexdigest() != expected:
            raise SystemExit(f'Pinned model hash mismatch: {name}')

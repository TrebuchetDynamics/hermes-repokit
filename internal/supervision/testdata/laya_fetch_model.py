"""Explicit ~843 MB public download into a new disposable model directory."""

import hashlib
from pathlib import Path
import sys
import urllib.request

REVISION = "1a793eb568e6718f15941d08f85432581df534e3"
FILES = {
    "encoder/config.json": "5268d24ad3b77c8151de5dcb0762ba4391619aad9ab0bda33e36fb083cfeae6d",
    "model.safetensors": "4fa56de72383a9d3efa9cfa78955733c81b9fc8067a587ca4beb82c78107a24e",
    "rl_agent_config.json": "ebf0cd524d92342a6be5e48e9fca3d7c2babfb5a56ccd79d2171ef5d8c7f7be8",
    "tokenizer/tokenizer.json": "6c8aaa9a542084f2457eab775d4eeb51f92a70c0fd9de28d5edb0ddec3c08d30",
    "tokenizer/tokenizer_config.json": "08d4cf3ac4dca381759441b85b91a6d40e688471dcd33d15d6649eb0a9a854d1",
}

destination = Path(sys.argv[1])
verify_only = len(sys.argv) == 3 and sys.argv[2] == "--verify"
if not verify_only:
    destination.mkdir(parents=True, exist_ok=False)
for filename, expected in FILES.items():
    target = destination / filename
    if not verify_only:
        target.parent.mkdir(parents=True, exist_ok=True)
        urllib.request.urlretrieve(
            f"https://huggingface.co/convaiinnovations/laya-typed-decisions/resolve/{REVISION}/{filename}",
            target,
        )
    with target.open("rb") as stream:
        actual = hashlib.file_digest(stream, "sha256").hexdigest()
    if actual != expected:
        raise SystemExit(f"SHA256 mismatch for {filename}")
    print(f"verified {filename}: {actual}", flush=True)

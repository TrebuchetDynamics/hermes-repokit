"""Bounded repair for the pinned upstream shell entrypoint's TERM wait race.

s6 must not report the service down before its server child releases storage.
Refuse an unexpected upstream script rather than applying a speculative patch.
"""
from pathlib import Path
import sys

path = Path(sys.argv[1])
source = path.read_text()
old = "trap 'forward_signal' INT TERM"
new = "trap 'trap \"\" INT TERM; forward_signal; wait \"${SERVER_PID}\" || true; exit 0' INT TERM"
if source.count(old) != 1:
    raise SystemExit("Unqualified OpenViking entrypoint shutdown contract")
path.write_text(source.replace(old, new))

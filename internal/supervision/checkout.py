"""Read-only native checkout verification. Never execute plugin hooks or filters."""
import hashlib
import os
from pathlib import Path
import re
import stat
import subprocess


def _checkout_read_config(path):
    with open(path, "rb") as source:
        value = source.read(262145)
    if len(value) > 262144:
        raise ValueError("oversize Git config")
    return value.decode("utf-8")


def _checkout_metadata_safe(plugin, root):
    """Validate native clone metadata BEFORE invoking Git, even rev-parse.

    git status runs clean filters while refreshing the index even with optional
    locks disabled. Only ordinary clone config is admitted here; includes,
    filters, hooks overrides, worktree extensions and other settings fail closed.
    Object/metadata indirection must not lead Git outside this native checkout.
    """
    try:
        gitdir = plugin / ".git"
        if plugin.resolve() != plugin.absolute() or not plugin.resolve().is_relative_to(root.resolve()):
            return False
        if gitdir.is_symlink() or not gitdir.is_dir():
            return False
        forbidden = {"commondir", "gitdir", "config.worktree", "objects/info/alternates", "objects/info/http-alternates"}
        count = 0
        for directory, directories, files in os.walk(gitdir, followlinks=False):
            for name in directories + files:
                path = Path(directory) / name
                info = path.lstat()
                count += 1
                if count > 100000 or str(path.relative_to(gitdir)) in forbidden:
                    return False
                if stat.S_ISLNK(info.st_mode) or not (stat.S_ISDIR(info.st_mode) or stat.S_ISREG(info.st_mode)):
                    return False
                if stat.S_ISREG(info.st_mode) and info.st_nlink != 1:
                    return False
        section = None
        seen = set()
        core = {"repositoryformatversion": {"0"}, "filemode": {"true", "false"},
                "bare": {"false"}, "logallrefupdates": {"true", "false"},
                "ignorecase": {"true", "false"}, "symlinks": {"true", "false"},
                "precomposeunicode": {"true", "false"}}
        for raw in _checkout_read_config(gitdir / "config").splitlines():
            line = raw.strip()
            if not line or line.startswith(("#", ";")):
                continue
            if line.startswith("["):
                if line == "[core]" or line == '[remote "origin"]' or re.fullmatch(r'\[branch "[A-Za-z0-9._/-]+"\]', line):
                    section = line
                    continue
                return False
            match = re.fullmatch(r"([a-zA-Z]+)\s*=\s*([^\r\n]+)", line)
            if not match or section is None:
                return False
            key, value = match.group(1).lower(), match.group(2).strip()
            if (section, key) in seen:
                return False
            seen.add((section, key))
            if section == "[core]":
                if value not in core.get(key, set()):
                    return False
            elif section == '[remote "origin"]':
                if key == "url" and value in ("https://github.com/keeltrace/hermes-nerve", "https://github.com/keeltrace/hermes-nerve.git"):
                    continue
                if key == "fetch" and re.fullmatch(r"\+refs/heads/[A-Za-z0-9._/*-]+:refs/remotes/origin/[A-Za-z0-9._/*-]+", value):
                    continue
                return False
            elif not ((key == "remote" and value == "origin") or (key == "merge" and re.fullmatch(r"refs/heads/[A-Za-z0-9._/-]+", value))):
                return False
        return ("[core]", "repositoryformatversion") in seen and ("[core]", "bare") in seen
    except (OSError, ValueError):
        return False


def verify_checkout(plugin, root, revision):
    """Compare native plugin files against pinned Git blobs, ignoring the index.

    Only metadata-only Git operations run after raw config/path validation.
    Worktree hashing is ours, so attributes, clean filters, assume-unchanged and
    skip-worktree cannot hide edits or execute processes. The sole native extra
    file is the catalog convenience copy; imported Python may also create pyc.
    """
    try:
        plugin, root = Path(plugin), Path(root)
        if not re.fullmatch(r"[0-9a-f]{40}", revision) or not _checkout_metadata_safe(plugin, root):
            return False
        git_env = {"PATH": "/usr/bin:/bin", "GIT_OPTIONAL_LOCKS": "0", "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_NO_REPLACE_OBJECTS": "1"}
        def git(*args):
            result = subprocess.run(["git", "-c", "core.fsmonitor=false", "-C", str(plugin), *args], env=git_env, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=5, check=True).stdout
            if len(result) > 8388608:
                raise ValueError("oversize Git metadata")
            return result
        if git("rev-parse", "HEAD").decode().strip() != revision:
            return False
        tree = git("ls-tree", "-r", "-z", "--full-tree", revision)
        tracked = {}
        for entry in tree.split(b"\0"):
            if not entry:
                continue
            attributes, encoded_path = entry.split(b"\t", 1)
            mode, kind, oid = attributes.decode("ascii").split()
            path = encoded_path.decode("utf-8")
            if mode not in ("100644", "100755") or kind != "blob" or not re.fullmatch(r"[0-9a-f]{40}", oid):
                return False
            if not path or path.startswith("/") or any(part in ("", ".", "..", ".git") for part in path.split("/")) or path in tracked:
                return False
            tracked[path] = (mode, oid)
        if not tracked or len(tracked) > 100000:
            return False
        seen = set()
        total = 0
        for directory, directories, files in os.walk(plugin, followlinks=False):
            if Path(directory) == plugin:
                directories.remove(".git")
            for name in directories + files:
                path = Path(directory) / name
                info = path.lstat()
                if stat.S_ISLNK(info.st_mode) or not (stat.S_ISREG(info.st_mode) or stat.S_ISDIR(info.st_mode)):
                    return False
                if stat.S_ISDIR(info.st_mode):
                    continue
                if info.st_nlink != 1:
                    return False
                relative = path.relative_to(plugin).as_posix()
                if relative not in tracked:
                    if relative == ".hermes-catalog.json" or (path.parent.name == "__pycache__" and path.suffix == ".pyc"):
                        continue
                    return False
                mode, oid = tracked[relative]
                if bool(info.st_mode & 0o111) != (mode == "100755"):
                    return False
                total += info.st_size
                if info.st_size > 33554432 or total > 134217728:
                    return False
                digest = hashlib.sha1(b"blob " + str(info.st_size).encode("ascii") + b"\0")
                count = 0
                fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK | os.O_NOFOLLOW)
                with os.fdopen(fd, "rb") as source:
                    opened = os.fstat(source.fileno())
                    if not stat.S_ISREG(opened.st_mode) or opened.st_ino != info.st_ino or opened.st_dev != info.st_dev:
                        return False
                    while chunk := source.read(65536):
                        count += len(chunk)
                        if count > info.st_size:
                            return False
                        digest.update(chunk)
                if count != info.st_size or digest.hexdigest() != oid:
                    return False
                seen.add(relative)
        return seen == set(tracked)
    except (OSError, ValueError, subprocess.SubprocessError):
        return False

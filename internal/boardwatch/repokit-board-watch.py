#!/usr/bin/env python3
"""RepoKit board watch: monitor script of the Hermes cron job repokit-board-watch.

Hermes runs it every tick and wakes default only when the output changes. It
prints exactly "busy" while nothing needs default, so a busy or settled board
costs no model call; otherwise a stable digest of open goals and stuck cards
plus an idle back-off bucket. It also subscribes each goal card's chat to the
work cards linked to that goal, so their completions wake the chat coordinator.
"""
import json
import os
import subprocess
import sys
import time

HERMES = os.environ.get("HERMES_BIN") or "/opt/hermes/.venv/bin/hermes"
GOAL_PREFIX = "Goal:"
CHAT_QUIET_SECONDS = 15 * 60
BUCKETS = [(30 * 60, "30m"), (3600, "1h"), (2 * 3600, "2h"), (4 * 3600, "4h"), (8 * 3600, "8h")]
NOT_CHATS = {"cli", "cron", "kanban", "oneshot", "acp", "api_server", "tui", "webhook"}


def bucket(idle):
    """None under 30 minutes, then 30m..8h, then whole days."""
    if idle >= 86400:
        return "%dd" % (idle // 86400)
    name = None
    for seconds, label in BUCKETS:
        if idle >= seconds:
            name = label
    return name


def is_goal(task):
    return (task.get("title") or "").startswith(GOAL_PREFIX)


def decide(tasks, last_chat, now):
    work = [t for t in tasks if not is_goal(t)]
    goals = sorted(t["id"] for t in tasks if is_goal(t) and t["status"] == "blocked")
    triage = sorted(t["id"] for t in work if t["status"] == "triage")
    blocked = sorted(t["id"] for t in work if t["status"] == "blocked")
    active = any(
        t["status"] == "running" or (t["status"] in ("ready", "todo", "review") and t.get("assignee"))
        for t in work
    )
    if active or now - last_chat < CHAT_QUIET_SECONDS or not (goals or triage or blocked):
        return "busy"
    last = max([last_chat] + [t.get(k) or 0 for t in tasks for k in ("created_at", "started_at", "completed_at")])
    idle = bucket(now - last)
    if idle is None:
        return "busy"
    return "idle " + json.dumps({"idle": idle, "goals": goals, "triage": triage, "blocked": blocked}, sort_keys=True)


def hermes(*args):
    out = subprocess.run([HERMES, *args], capture_output=True, text=True, timeout=60)
    if out.returncode != 0:
        raise RuntimeError("hermes %s: %s" % (args[:2], out.stderr.strip()[:200]))
    return out.stdout


def last_chat_activity():
    try:
        sys.path.insert(0, "/opt/hermes")
        from hermes_state import SessionDB
        rows = SessionDB().list_sessions_rich(limit=50)
    except Exception:
        return 0
    return max([float(r.get("last_active") or 0) for r in rows if r.get("source") not in NOT_CHATS] + [0])


def task_id(entry):
    return entry if isinstance(entry, str) else (entry or {}).get("id")


def subscribe_goal_work(goals):
    for goal in goals:
        targets = json.loads(hermes("kanban", "notify-list", goal, "--json") or "[]")
        if not targets:
            continue
        parents = json.loads(hermes("kanban", "show", goal, "--json")).get("parents") or []
        for work in filter(None, map(task_id, parents)):
            have = {(s.get("platform"), str(s.get("chat_id")))
                    for s in json.loads(hermes("kanban", "notify-list", work, "--json") or "[]")}
            for t in targets:
                if (t.get("platform"), str(t.get("chat_id"))) in have:
                    continue
                args = ["kanban", "notify-subscribe", "--platform", t["platform"],
                        "--chat-id", str(t["chat_id"]), "--delivery-mode", "notify+wake"]
                for flag, key in (("--thread-id", "thread_id"), ("--user-id", "user_id"), ("--chat-type", "chat_type")):
                    if t.get(key):
                        args += [flag, str(t[key])]
                hermes(*args, work)


def main():
    snapshot = os.environ.get("REPOKIT_WATCH_SNAPSHOT")
    if snapshot:
        with open(snapshot) as f:
            s = json.load(f)
        print(decide(s["tasks"] or [], s["last_chat"], s["now"]))
        return
    try:
        tasks = json.loads(hermes("kanban", "list", "--json"))
    except Exception as exc:
        print("repokit-board-watch: %s" % exc, file=sys.stderr)
        print("busy")
        return
    out = decide(tasks, last_chat_activity(), time.time())
    try:
        subscribe_goal_work(sorted(t["id"] for t in tasks if is_goal(t) and t["status"] == "blocked"))
    except Exception as exc:
        print("repokit-board-watch: %s" % exc, file=sys.stderr)
    print(out)


if __name__ == "__main__":
    main()

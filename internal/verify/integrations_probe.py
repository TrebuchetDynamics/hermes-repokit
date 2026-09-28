"""Bounded, read-only native config observation; never import a Hermes plugin.

Only closed status codes leave this process. In particular parser errors, Git
diagnostics, credentials, linked identities and peer configuration stay private.
"""
import io
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import urllib.request

import yaml
from dotenv import dotenv_values

ROLES = ("default", "researcher", "planner", "executor", "reviewer", "steward")
EXPECTED = {
    "nerve_profile": "lean",
    "reflex_backend": "laya",
    "reflex_laya_base_url": "http://127.0.0.1:8765",
    "reflex_laya_model": "/model",
    "reflex_laya_timeout_seconds": 120,
}


def read(path, root):
    path = Path(path)
    # Refuse symlink escapes and nonregular files, including FIFOs.
    if not path.resolve().is_relative_to(root.resolve()):
        raise ValueError("outside native home")
    fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as source:
        meta = os.fstat(source.fileno())
        if not stat.S_ISREG(meta.st_mode) or meta.st_size > 262144:
            raise ValueError("invalid native file")
        value = source.read(262145)
    if len(value) > 262144:
        raise ValueError("oversize native file")
    return value.decode("utf-8-sig")


def mapping(value):
    if not isinstance(value, dict):
        raise ValueError("invalid mapping")
    return value


def env_for(home, root):
    values = {}
    for name in (".op.env", ".env"):
        try:
            values.update(dotenv_values(stream=io.StringIO(read(home / name, root)), interpolate=False))
        except FileNotFoundError:
            pass
    return values


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def user_health(api_key, project):
    # Fixed read-only native endpoint: never /ready, /initialize or a provider.
    client = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    request = urllib.request.Request("http://openviking:1933/health", method="GET", headers={
        "X-API-Key": api_key, "Authorization": "Bearer " + api_key,
    })
    with client.open(request, timeout=3) as response:
        data = response.read(16385)
        if response.status != 200 or len(data) > 16384:
            return False
        health = mapping(json.loads(data))
    return tuple(health.get(key) for key in ("auth_mode", "role", "account_id", "user_id")) == ("api_key", "user", "repokit", project)


def proxy_routes_loopback(env):
    effective = dict(os.environ)
    effective.update({key: value for key, value in env.items() if value is not None})
    def value(name):
        return effective.get(name.lower(), effective.get(name.upper(), ""))
    if not (value("http_proxy") or value("all_proxy")):
        return False
    return not urllib.request.proxy_bypass_environment("127.0.0.1:8765", {"no": value("no_proxy")})


def managed_configuration(env):
    managed = Path("/etc/hermes")
    return managed.is_symlink() or any((managed / name).exists() or (managed / name).is_symlink() for name in ("config.yaml", ".env")) or any("HERMES_MANAGED_DIR" in source for source in (env, os.environ))



def memory_state(config, env, root, project, service_available, health_cache):
    if managed_configuration(env):
        return "degraded"
    memory = mapping(config.get("memory", {}))
    if memory.get("provider") != "openviking" or memory.get("memory_enabled") is not True or memory.get("user_profile_enabled") is not True:
        return "inactive"
    if config.get("secrets"):
        # External secret providers require evaluation; verification cannot
        # hydrate them or claim its raw-file view is the effective identity.
        return "degraded"
    ov = mapping(memory.get("openviking", {}))
    linked = {}
    if ov.get("use_ovcli_config"):
        path = ov.get("ovcli_config_path")
        # Native path resolution permits env/tilde paths. Only the explicit
        # repository-scoped linked path can be positively observed here.
        if not isinstance(path, str) or not Path(path).is_absolute():
            return "degraded"
        if not Path(path).resolve().is_relative_to((root / ".openviking").resolve()):
            return "degraded"
        linked = mapping(json.loads(read(path, root)))
    if any(name in env or name in os.environ for name in ("OPENVIKING_CLI_CONFIG_FILE",)):
        return "degraded"
    values = {
        "endpoint": linked.get("url") or ov.get("endpoint"),
        "account": linked.get("account") or linked.get("account_id") or ov.get("account"),
        "user": linked.get("user") or linked.get("user_id") or ov.get("user"),
        "agent": linked.get("actor_peer_id") or linked.get("agent_id") or ov.get("agent") or "",
    }
    # Native user keys derive identity server-side. Never treat a root key or
    # client-provided account/user fields as proof of repository isolation.
    api_key = linked.get("api_key")
    if not isinstance(api_key, str) or not api_key.strip():
        return "inactive"
    api_key = api_key.strip()
    if api_key == linked.get("root_api_key"):
        return "degraded"
    for key in values:
        name = "OPENVIKING_" + key.upper()
        for source in (os.environ, env):
            if name in source:
                value = source[name]
                if value is not None and (key in ("account", "user") or value):
                    values[key] = value
    effective_key = env.get("OPENVIKING_API_KEY", os.environ.get("OPENVIKING_API_KEY", api_key))
    if not isinstance(effective_key, str) or effective_key.strip() != api_key:
        return "degraded"
    if values["endpoint"] != "http://openviking:1933" or values["account"] not in (None, "", "repokit") or values["user"] not in (None, "", project) or values["agent"]:
        return "degraded"
    if not service_available:
        return "inactive"
    # All six profiles normally share one key. Cache only within this one-shot
    # invocation, keeping both credentials and response identities private.
    if api_key not in health_cache:
        try:
            health_cache[api_key] = user_health(api_key, project)
        except Exception:
            health_cache[api_key] = False
    return "active" if health_cache[api_key] else "degraded"


def nerve_state(config, env, home, root, revision):
    if managed_configuration(env):
        return "degraded"
    plugins = mapping(config.get("plugins", {}))
    enabled, disabled = plugins.get("enabled", []), plugins.get("disabled", [])
    if not isinstance(enabled, list) or not isinstance(disabled, list):
        return "degraded"
    if "nerve" not in enabled or "nerve" in disabled:
        return "inactive"
    if config.get("secrets"):
        return "degraded"
    entry = mapping(mapping(plugins.get("entries", {})).get("nerve", {}))
    settings = mapping(entry.get("settings", {}))
    if settings != EXPECTED:
        return "degraded"
    # Overrides require native semantic qualification, not guessing precedence.
    if any(key.startswith(("HERMES_NERVE_", "HERMES_REFLEX_", "NERVE_")) for source in (env, os.environ) for key in source):
        return "degraded"
    if proxy_routes_loopback(env):
        return "degraded"
    if (home / "nerve/profile.json").exists():
        return "degraded"
    plugin = home / "plugins/nerve"
    if not plugin.resolve().is_relative_to(root.resolve()):
        return "degraded"
    manifest = mapping(yaml.safe_load(read(plugin / "plugin.yaml", root)))
    if manifest.get("name") != "nerve":
        return "degraded"
    record = mapping(mapping(json.loads(read(home / "plugins/.install-metadata.json", root))).get("nerve", {}))
    if record.get("revision") != revision or record.get("pinned") is not True or record.get("source") not in ("https://github.com/keeltrace/hermes-nerve", "https://github.com/keeltrace/hermes-nerve.git"):
        return "degraded"
    if not verify_checkout(plugin, root, revision):
        return "degraded"
    return "configured"


def aggregate(values, positive):
    if "degraded" in values:
        return "degraded"
    return positive if all(value == positive for value in values) else "inactive"


def main():
    root, project, revision = Path(sys.argv[1]), sys.argv[2], sys.argv[3]
    service_available = len(sys.argv) == 5 and sys.argv[4] == "memory-service-matched"
    health_cache = {}
    memory, nerve = [], []
    for role in ROLES:
        home = root if role == "default" else root / "profiles" / role
        try:
            config = mapping(yaml.safe_load(read(home / "config.yaml", root)))
            env = env_for(home, root)
        except FileNotFoundError:
            memory.append("inactive")
            nerve.append("inactive")
            continue
        except Exception:
            memory.append("degraded")
            nerve.append("degraded")
            continue
        for values, check in ((memory, lambda: memory_state(config, env, root, project, service_available, health_cache)), (nerve, lambda: nerve_state(config, env, home, root, revision))):
            try:
                values.append(check())
            except Exception:
                values.append("degraded")
    print(json.dumps({"memory": aggregate(memory, "active"), "nerve": aggregate(nerve, "configured")}))


if __name__ == "__main__":
    try:
        main()
    except Exception:
        print('{"memory":"degraded","nerve":"degraded"}')

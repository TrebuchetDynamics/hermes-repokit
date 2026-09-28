import contextlib
import copy
import hashlib
import io
import json
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

import yaml

scope = {"__name__": "fixture"}
exec(sys.argv[1], scope)
with tempfile.TemporaryDirectory() as temporary:
    root = Path(temporary)
    project = "repokit-fixture"
    linked = root / ".openviking/ovcli.conf"
    linked.parent.mkdir()
    native_link = {"url": "http://127.0.0.1:1933", "account_id": "repokit", "user_id": project, "api_key": "secret-user-key"}
    linked.write_text(json.dumps(native_link))
    config = {"memory": {"provider": "openviking", "memory_enabled": True, "user_profile_enabled": True, "openviking": {"use_ovcli_config": True, "ovcli_config_path": str(linked)}}}
    for role in scope["ROLES"]:
        home = root if role == "default" else root / "profiles" / role
        home.mkdir(parents=True, exist_ok=True)
        (home / "config.yaml").write_text(yaml.safe_dump(config))
    original = (root / "config.yaml").read_text()

    def snapshot():
        return {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in root.rglob("*") if p.is_file()}

    health_response = {"auth_mode": "api_key", "role": "user", "account_id": "repokit", "user_id": project}
    health_requests = []
    def open_health(request, timeout):
        assert request.full_url == "http://127.0.0.1:1933/health"
        assert request.method == "GET" and request.data is None
        assert request.get_header("X-api-key") == native_link["api_key"]
        assert request.get_header("Authorization") == "Bearer " + native_link["api_key"]
        health_requests.append(request)
        response = io.BytesIO(json.dumps(health_response).encode())
        response.status = 200
        return response

    def opener(*handlers):
        assert handlers[0].proxies == {}
        assert handlers[1].redirect_request() is None
        return SimpleNamespace(open=open_health)

    def observe(memory, service=True):
        before = snapshot()
        output = io.StringIO()
        health_requests.clear()
        with patch.object(sys, "argv", ["probe", str(root), project, "memory-service-matched" if service else "unavailable"]), patch.object(scope["urllib"].request, "build_opener", side_effect=opener), contextlib.redirect_stdout(output):
            scope["main"]()
        assert snapshot() == before, "verification mutated native files"
        assert json.loads(output.getvalue()) == {"memory": memory}, output.getvalue()
        assert "secret" not in output.getvalue()
        assert len(health_requests) <= 1, "shared user key should be probed once"

    observe("active")
    observe("inactive", service=False)
    assert not health_requests
    for role in ("root", "admin"):
        health_response["role"] = role
        observe("degraded")
    health_response["role"] = "user"
    health_response["user_id"] = "secret-other-repository"
    observe("degraded")
    health_response["user_id"] = project
    health_response["account_id"] = "secret-other-account"
    observe("degraded")
    health_response["account_id"] = "repokit"
    health_response["oversized"] = "x" * 16385
    observe("degraded")
    del health_response["oversized"]
    changed = copy.deepcopy(config)
    changed["secrets"] = {"sources": ["external"]}
    (root / "config.yaml").write_text(yaml.safe_dump(changed))
    observe("degraded")
    (root / "config.yaml").write_text(original)
    for field in ("actor_peer_id", "agent_id", "user_id", "account_id"):
        native = dict(native_link)
        native[field] = "secret-other-identity"
        linked.write_text(json.dumps(native))
        observe("degraded")
    linked.write_text(json.dumps(native_link))
    (root / ".env").write_text("OPENVIKING_API_KEY=secret-user-key\n")
    observe("active")
    (root / ".env").write_text("OPENVIKING_API_KEY=secret-wrong-key\n")
    observe("degraded")
    (root / ".env").write_text("OPENVIKING_USER=secret-other-identity\n")
    observe("degraded")
    (root / ".env").unlink()
    (root / ".env").write_text("HERMES_MANAGED_DIR=/secret-managed\n")
    observe("degraded")
    (root / ".env").unlink()
    (root / ".env").write_text("HTTP_PROXY=http://secret-proxy\nNO_PROXY=localhost\n")
    observe("active")
    (root / ".env").write_text("HTTP_PROXY=http://secret-proxy\nNO_PROXY=127.0.0.1\n")
    observe("active")
    (root / ".env").unlink()
    (root / "config.yaml").write_text("malformed: [secret\n")
    observe("degraded")
    (root / "config.yaml").unlink()
    observe("inactive")
    (root / "config.yaml").write_text("memory: {}\n")
    observe("inactive")
    # A nonempty native board/receipt never establishes memory or review.
    (root / "kanban.db").write_text("secret-not-a-database")
    (root / "repokit-install.json").write_text('{"accepted": true}')
    observe("inactive")
print("read-only parser fixtures passed")

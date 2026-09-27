"""Explicit live qualification probe; run inside a disposable Hermes container.

Requires an already admitted/enabled Nerve and local sidecar. Calls the real
Hermes plugin loader and registry, performs inference, and writes Nerve receipts.
It does not install, enable, configure, or silently substitute a provider.
"""

import json
import math
import os
from pathlib import Path
import subprocess
import urllib.request

import yaml

NERVE_REVISION = "b9e78dd5e00cf1117c563cada4d563a56f66e609"
home = Path(os.environ["HERMES_HOME"])
config = yaml.safe_load((home / "config.yaml").read_text())
settings = config["plugins"]["entries"]["nerve"]["settings"]
assert settings["reflex_backend"] == "laya", "Refusing non-Laya backend"
assert settings["reflex_laya_base_url"] == "http://127.0.0.1:8765"
assert settings["reflex_laya_model"] == "/model"
revision = subprocess.check_output(
    ["git", "-C", str(home / "plugins/nerve"), "rev-parse", "HEAD"], text=True
).strip()
assert revision == NERVE_REVISION
if os.environ.get("LAYA_EXPECT_UNAVAILABLE") == "1":
    from hermes_cli.plugins import discover_plugins, get_plugin_manager
    from tools.registry import registry

    discover_plugins()
    failed = registry.dispatch("nerve_decide", {
        "state": "Synthetic outage test", "instructions": "Choose a route",
        "choices": ["billing", "technical"],
    }, scope=get_plugin_manager().scope_key)
    failed = json.loads(failed) if isinstance(failed, str) else failed
    assert failed.get("ok") is False and "Laya sidecar connection failed" in failed["error"], failed
    print(json.dumps({"outage_failed_closed": True, "result": failed}, indent=2))
    raise SystemExit(0)
with urllib.request.urlopen("http://127.0.0.1:8765/healthz", timeout=5) as response:
    health = json.load(response)
assert health == {
    "ok": True, "provider": "Laya", "model": "/model", "transport": "laya-local-http"
}

from hermes_cli.plugins import discover_plugins, get_plugin_manager
from tools.registry import registry

discover_plugins()
manager = get_plugin_manager()


def call(name, arguments):
    result = registry.dispatch(name, arguments, scope=manager.scope_key)
    if isinstance(result, str):
        result = json.loads(result)
    assert isinstance(result, dict) and not result.get("error"), result
    return result


stats = call("nerve_stats", {"section": "summary", "include_recent": False})
typed_payload = {
    "state": {"message": "Please refund a duplicate invoice payment."},
    "questions": {
        "department": {
            "type": "choice", "instructions": "Which department handles this request?",
            "criteria": {"billing": "Invoices and refunds", "technical": "Software bugs"},
        },
        "urgency": {
            "type": "score", "instructions": "How urgent is this request?",
            "criteria": ["routine", "soon", "critical"],
        },
        "refund": {"type": "noul", "instructions": "Does the customer request a refund?"},
    },
}
request = urllib.request.Request("http://127.0.0.1:8765/v1/systemone",
    data=json.dumps(typed_payload).encode(), headers={"Content-Type": "application/json"})
with urllib.request.urlopen(request, timeout=120) as response:
    typed_result = json.load(response)
assert typed_result["provider"] == "Laya" and typed_result["model"] == "/model"
answers = typed_result["answers"]
assert answers["department"]["choice"] in {"billing", "technical"}
for name in ("department", "urgency"):
    assert all(math.isfinite(p) and 0 <= p <= 1 for p in answers[name]["probabilities"].values())
assert 0 <= answers["urgency"]["score"] <= 2
assert 0 <= answers["refund"]["noul"] <= 1
decision = call("nerve_decide", {
    "state": typed_payload["state"],
    "instructions": "Which department handles this request?",
    "choices": ["billing", "technical"],
    "criteria": typed_payload["questions"]["department"]["criteria"],
})
assert decision["value"] in {"billing", "technical"}
assert decision["provider"] == "Laya"
assert decision["model"] == "/model"
assert decision["provenance_status"] == "LOCAL_ONLY"
assert decision["execution"]["live_provider_call"] is False
assert decision["receipt_id"]
print(json.dumps({"nerve_revision": revision, "health": health,
                  "stats": stats, "typed_result": typed_result,
                  "decision": decision}, indent=2, sort_keys=True))

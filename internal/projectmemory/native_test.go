package projectmemory

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// These probes execute native code in cached pinned images without network or
// host mounts. They prove configuration semantics, not model-backed memory.
func TestPinnedNativeConfiguration(t *testing.T) {
	if os.Getenv("REPOKIT_TEST_MEMORY_CONFIG_DOCKER") != "1" {
		t.Skip("set REPOKIT_TEST_MEMORY_CONFIG_DOCKER=1 with cached pinned images")
	}
	cfg, err := NativeConfig("repokit-qualification-a")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct{ name, image, python, script string }{
		{"hermes", "nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db", "/opt/hermes/.venv/bin/python", hermesProbe},
		{"openviking", "ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05", "python", openvikingProbe},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--pull=never", "--network", "none", "--entrypoint", check.python, check.image, "-c", check.script, string(payload))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("native probe: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "REPOKIT_NATIVE_CONFIG_OK") {
				t.Fatalf("missing result: %s", out)
			}
			t.Log(string(out))
		})
	}
}

const hermesProbe = `
import json, sys
from unittest.mock import patch
import plugins.memory.openviking as ov
from tools.memory_tool import get_builtin_memory_store_flags
config = json.loads(sys.argv[1])
assert get_builtin_memory_store_flags({'memory':config}) == (True,True)
for profile in ('default','researcher','planner','executor','reviewer','steward'):
    with patch.object(ov, 'get_secret', return_value=None), patch.object(ov, '_ovcli_values_for', return_value={}):
        settings = ov._resolve_connection_settings(config['openviking'])
    assert settings == {'endpoint':'http://127.0.0.1:1933', 'api_key':'', 'account':'repokit', 'user':'repokit-qualification-a', 'agent':''}, (profile,settings)
# Verify the actual linked-config parser, including both peer aliases.
for key in ('actor_peer_id','agent_id'):
    assert ov._connection_values_from_ovcli({key:'private-peer'})['agent'] == 'private-peer'
with patch.object(ov, 'get_secret', return_value=None), patch.object(ov, '_ovcli_values_for', return_value={'agent':'private-peer'}):
    assert ov._resolve_connection_settings(config['openviking'])['agent'] == 'private-peer'
with patch.object(ov, 'get_secret', side_effect=lambda name: '' if name=='OPENVIKING_USER' else None), patch.object(ov, '_ovcli_values_for', return_value={}):
    assert ov._resolve_connection_settings(config['openviking'])['user'] == ''
assert 'agent' not in config['openviking']
print('REPOKIT_NATIVE_CONFIG_OK Hermes public config, built-in flags, peer override and empty-user override')
`

const openvikingProbe = `
from openviking_cli.utils.config.embedding_config import EmbeddingModelConfig
from openviking_cli.utils.config.vlm_config import VLMConfig
from openviking_cli.utils.config.memory_config import MemoryConfig
assert MemoryConfig().extraction_enabled is True
assert VLMConfig().is_available() is False
try:
    EmbeddingModelConfig()
except ValueError as exc:
    assert 'model name is required' in str(exc)
else:
    raise AssertionError('empty embedding accepted')
# Schema-only samples have deliberately unreachable loopback endpoints; no calls.
EmbeddingModelConfig(provider='openai',model='schema-probe-only',api_base='http://127.0.0.1:1/v1',dimension=3,input='text')
try:
    VLMConfig(provider='openai',model='schema-probe-only',api_base='http://127.0.0.1:1/v1')
except ValueError as exc:
    assert "requires 'api_key'" in str(exc)
else:
    raise AssertionError('missing VLM credential accepted')
print('REPOKIT_NATIVE_CONFIG_OK OpenViking extraction default and provider schema requirements; no inference')
`

package supervision

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestLocalLayaSettingsAreNativeAndProfileIsolated(t *testing.T) {
	first := LocalLayaSettings()
	first["reflex_backend"] = "jev"
	second := LocalLayaSettings()
	if second["reflex_backend"] != "laya" {
		t.Fatal("one profile can mutate the next profile's backend")
	}
	endpoint, err := url.Parse(second["reflex_laya_base_url"].(string))
	if err != nil || endpoint.Scheme != "http" || endpoint.Host != "127.0.0.1:8765" {
		t.Fatalf("expected the qualified loopback endpoint, got %v", endpoint)
	}
	if second["reflex_laya_model"] != "/model" || second["reflex_laya_timeout_seconds"] != 120 {
		t.Fatal("settings no longer match the qualified fixed-model CPU sidecar")
	}
	// This object is passed to the native config setter, not a complete config.
	// Serializing it must preserve the native key names and must not enable Nerve.
	data, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["reflex_backend"] != "laya" || wire["nerve_profile"] != "lean" {
		t.Fatalf("invalid native settings: %s", data)
	}
	if _, ok := wire["enabled"]; ok {
		t.Fatal("settings must not enable a plugin before sidecar verification")
	}
}

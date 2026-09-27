package projectmemory

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativeConfigSharedIdentity(t *testing.T) {
	cfg, err := NativeConfig("repo-0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if cfg["provider"] != "openviking" || cfg["memory_enabled"] != true || cfg["user_profile_enabled"] != true {
		t.Fatal(cfg)
	}
	ov := cfg["openviking"].(map[string]any)
	if ov["endpoint"] != "http://openviking:1933" || ov["account"] != "repokit" || ov["user"] != "repo-0123456789abcdef" {
		t.Fatal(ov)
	}
	encoded, _ := json.Marshal(cfg)
	for _, forbidden := range []string{"agent", "peer", "api_key"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("unexpected %s", forbidden)
		}
	}
	ov["user"] = "changed"
	second, _ := NativeConfig("repo-0123456789abcdef")
	if second["openviking"].(map[string]any)["user"] != "repo-0123456789abcdef" {
		t.Fatal("shared mutable configuration")
	}
}

func TestNativeConfigRejectsUnsafeIdentity(t *testing.T) {
	for _, id := range []string{"", " repo", "repo ", "repo/other", "repo\nOPENVIKING_AGENT=private", "rêpo", "a@b@c"} {
		if _, err := NativeConfig(id); err == nil {
			t.Errorf("accepted %q", id)
		}
	}
}

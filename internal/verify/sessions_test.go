package verify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A messaging conversation that started before default's current SOUL still
// acts on the earlier identity; verify names the platform, never the chat.
func TestChannelSessionsFlagConversationsOlderThanTheIdentity(t *testing.T) {
	id, base := integrationFixture(t)
	soul := filepath.Join(id.Root, ".hermes", "SOUL.md")
	if err := os.WriteFile(soul, []byte("identity"), 0600); err != nil {
		t.Fatal(err)
	}
	changed := time.Date(2026, 9, 30, 4, 29, 0, 0, time.UTC)
	if err := os.Chtimes(soul, changed, changed); err != nil {
		t.Fatal(err)
	}
	listing := func(id string) string {
		return "Title   Preview            Last Active   ID\nPrivate chat title   secret message   39m ago   " + id + "\n"
	}
	for _, c := range []struct {
		telegram, discord string
		want              Status
		named             string
	}{
		{"20260929_192739_30656db1", "20260930_120000_aaaaaaaa", Degraded, "telegram"},
		{"20260930_050000_30656db1", "20260930_120000_aaaaaaaa", Healthy, ""},
	} {
		r := &hermesRunner{integrationRunner: base, replies: map[string]string{
			"send --list --json":                        `{"platforms":{"telegram":[{"id":"6586915095","name":"Private Name"}],"discord":[{"id":"1"}],"cron":[]}}`,
			"sessions list --source telegram --limit 1": listing(c.telegram),
			"sessions list --source discord --limit 1":  listing(c.discord),
		}}
		probes := ChannelSessions(context.Background(), id, r)
		if len(probes) != 1 || probes[0].Status != c.want {
			t.Fatalf("sessions: %+v", probes)
		}
		detail := probes[0].Detail
		if c.named != "" && (!strings.Contains(detail, c.named) || strings.Contains(detail, "discord") || !strings.Contains(detail, "/new")) {
			t.Fatalf("stale platform not named precisely: %s", detail)
		}
		for _, private := range []string{"6586915095", "Private", "secret", "title"} {
			if strings.Contains(detail, private) {
				t.Fatalf("private chat data reported: %s", detail)
			}
		}
		for _, call := range r.calls {
			if strings.Contains(strings.Join(call, " "), "sessions list --source cron") {
				t.Fatal("non-interactive platform inspected")
			}
		}
	}
}

// Without a matching runtime (a recipe change awaiting recreation) the probe
// reports unknown like every other runtime probe instead of disappearing.
func TestChannelSessionsReportsUnavailableRuntime(t *testing.T) {
	id, base := integrationFixture(t)
	if err := os.WriteFile(filepath.Join(id.Root, ".hermes", "SOUL.md"), []byte("identity"), 0600); err != nil {
		t.Fatal(err)
	}
	base.hermes = "" // container inspect finds no matching runtime
	probes := ChannelSessions(context.Background(), id, base)
	if len(probes) != 1 || probes[0].Component != "sessions" || probes[0].Status != Unknown {
		t.Fatalf("unavailable runtime hidden: %+v", probes)
	}
}

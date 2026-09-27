package process

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProbeBoundsOutputAndKillsHangingProcessGroup(t *testing.T) {
	r := Runner{Limit: 64, Timeout: 100 * time.Millisecond}
	start := time.Now()
	result := r.Run(context.Background(), "sh", "-c", "printf '%0100d' 1; sleep 20 & wait")
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout did not kill child")
	}
	if !result.Truncated || len(result.Output) != 64 || result.Err == nil {
		t.Fatalf("%+v", result)
	}
}
func TestProbeSanitizesComposeSelectorsAndPreservesArgv(t *testing.T) {
	t.Setenv("COMPOSE_FILE", "bad")
	r := Runner{Limit: 1000, Timeout: time.Second}
	got := r.Run(context.Background(), "sh", "-c", `[ -z "${COMPOSE_FILE+x}" ] || exit 4; printf '%s' "$1"`, "sh", "$x; ' \n")
	if got.Err != nil || got.Output != "$x; ' \n" {
		t.Fatalf("%+v", got)
	}
	for _, v := range CleanEnvironment(os.Environ()) {
		if strings.HasPrefix(v, "COMPOSE_") {
			t.Fatal("selector remains")
		}
	}
}

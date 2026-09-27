package process

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
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

func TestProbeReapsDescendantAfterParentExits(t *testing.T) {
	r := Runner{Timeout: 2 * time.Second, Limit: 1000}
	result := r.Run(context.Background(), "sh", "-c", "sleep 30 & echo $!")
	pid, e := strconv.Atoi(strings.TrimSpace(result.Output))
	if e != nil {
		t.Fatal(e)
	}
	defer syscall.Kill(pid, syscall.SIGKILL)
	for i := 0; i < 100; i++ {
		data, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if os.IsNotExist(e) || strings.Contains(string(data), ") Z ") {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatal("descendant survived completed probe")
}

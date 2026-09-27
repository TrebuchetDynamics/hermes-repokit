package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainEntryPoint(t *testing.T) {
	if raw := os.Getenv("REPOKIT_TEST_ENTRY_ARGS"); raw != "" {
		var args []string
		if e := json.Unmarshal([]byte(raw), &args); e != nil {
			panic(e)
		}
		os.Args = append([]string{"hermes-repokit"}, args...)
		main()
		return
	}
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, tt := range []struct {
		args     []string
		code     int
		contains string
	}{{[]string{"--help"}, 0, "plan|install|setup|verify"}, {[]string{"daemon"}, 2, "usage error"}} {
		encoded, _ := json.Marshal(tt.args)
		cmd := exec.Command(executable, "-test.run=^TestMainEntryPoint$")
		cmd.Env = append(os.Environ(), "REPOKIT_TEST_ENTRY_ARGS="+string(encoded))
		out, e := cmd.CombinedOutput()
		code := 0
		if e != nil {
			exit, ok := e.(*exec.ExitError)
			if !ok {
				t.Fatal(e)
			}
			code = exit.ExitCode()
		}
		if code != tt.code || !strings.Contains(string(out), tt.contains) {
			t.Fatalf("%v: %d %q", tt.args, code, out)
		}
	}
}

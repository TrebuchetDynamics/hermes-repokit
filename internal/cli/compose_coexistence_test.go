package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAndPlanCoexistWithOwnerCompose(t *testing.T) {
	a, r := foundationApp(t)
	owner := filepath.Join(a.Directory, "docker-compose.yml")
	content := []byte("name: owner-stack\nservices:\n  owner:\n    image: busybox\n")
	if err := os.WriteFile(owner, content, 0644); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(a.Directory, "compose.override.yaml")
	if err := os.Symlink("missing-owner-override", override); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"plan", "install", "plan", "install"} {
		code, out, diag := invoke(t, a, command)
		if code != 0 {
			t.Fatalf("%s refused owner Compose: %s %s", command, out, diag)
		}
		if command == "plan" {
			var plan Plan
			if err := json.Unmarshal([]byte(out), &plan); err != nil || len(plan.Collisions) != 0 {
				t.Fatalf("bad plan: %s", out)
			}
		}
		if got, err := os.ReadFile(owner); err != nil || !bytes.Equal(got, content) {
			t.Fatal("owner Compose modified")
		}
		if link, err := os.Readlink(override); err != nil || link != "missing-owner-override" {
			t.Fatal("owner override modified")
		}
	}
	generated, err := os.ReadFile(r.id.Compose)
	if err != nil || !strings.Contains(string(generated), "name: "+`"`+r.id.Project+`"`) || strings.Contains(string(generated), "owner-stack") {
		t.Fatal("generated project not isolated", err)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n[ -z \"${COMPOSE_FILE+x}${COMPOSE_PROJECT_NAME+x}\" ] || exit 99\nprintf '%s\\n' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(r.id.Launcher, "profile", "list")
	cmd.Dir = a.Directory
	cmd.Env = append(os.Environ(), "PATH="+bin, "COMPOSE_FILE="+owner, "COMPOSE_PROJECT_NAME=owner-stack")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "--env-file\n/dev/null\n-f\n"+r.id.Compose+"\n") || strings.Contains(string(out), owner) {
		t.Fatalf("launcher selected owner Compose: %v %s", err, out)
	}
}

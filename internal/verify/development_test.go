package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type devRunner struct {
	*integrationRunner
	toolData string
	derived  string
}

func (r *devRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	call := strings.Join(args, " ")
	if strings.HasSuffix(call, "/usr/local/bin/repokit-openviking health") {
		r.calls = append(r.calls, append([]string{p}, args...))
		return process.Result{Output: r.health}
	}
	if strings.Contains(call, developmentProbe) {
		return process.Result{Output: r.toolData}
	}
	if strings.Contains(call, "image inspect") {
		if args[len(args)-1] == qualification.FoundationImage {
			return process.Result{Output: `["sha256:` + strings.Repeat("e", 64) + `"]`}
		}
		return process.Result{Output: r.derived}
	}
	return r.integrationRunner.Run(ctx, p, args...)
}
func TestDevelopmentReadinessRequiresRealToolsAndProfileWorkdirs(t *testing.T) {
	id, base := integrationFixture(t)
	r := &devRunner{integrationRunner: base}
	os.WriteFile(filepath.Join(id.Root, "go.mod"), []byte("module fixture\ngo 1.26.0\n"), 0600)
	tools := map[string]any{}
	for _, name := range []string{"git", "bash", "curl", "jq", "rg", "python", "node", "npm", "make", "gcc", "g++", "docker"} {
		tools[name] = map[string]any{"ok": true, "version": "fixture version"}
	}
	tools["compose"] = map[string]any{"ok": true, "version": development.ComposeVersion}
	tools["buildx"] = map[string]any{"ok": true, "version": "github.com/docker/buildx v" + development.BuildxVersion}
	tools["go"] = map[string]any{"ok": true, "version": "go version go" + development.GoVersion + " linux/amd64"}
	profiles := map[string]bool{}
	for _, name := range []string{"default", "researcher", "planner", "executor", "reviewer", "steward"} {
		profiles[name] = true
	}
	data := map[string]any{"tools": tools, "workspace": true, "runtime_commands": true, "certificates": true, "profiles": profiles}
	encode := func() { b, _ := json.Marshal(data); r.toolData = string(b) }
	encode()
	if got := Development(context.Background(), id, r); got[0].Status != Healthy {
		t.Fatal(got)
	}
	data["runtime_commands"] = false
	encode()
	if got := Development(context.Background(), id, r); got[0].Status != Degraded {
		t.Fatal("missing login-shell entrypoints accepted", got)
	}
	data["runtime_commands"] = true
	tools["go"] = map[string]any{"ok": false, "version": "unavailable"}
	encode()
	if got := Development(context.Background(), id, r); got[0].Status != Degraded {
		t.Fatal("missing Go accepted", got)
	}
	tools["go"] = map[string]any{"ok": true, "version": "go version go" + development.GoVersion + " linux/amd64"}
	profiles["executor"] = false
	encode()
	if got := Development(context.Background(), id, r); got[0].Status != Degraded {
		t.Fatal("wrong profile workdir accepted", got)
	}
	profiles["executor"] = true
	os.WriteFile(filepath.Join(id.Root, "Cargo.toml"), []byte("[package]\n"), 0600)
	encode()
	if got := Development(context.Background(), id, r); got[0].Status != Degraded || !strings.Contains(got[0].Detail, "rust") {
		t.Fatal("unsupported compiler accepted", got)
	}
}
func TestDerivedImageRequiresRecipeContentIDAndBaseLayers(t *testing.T) {
	id, base := integrationFixture(t)
	req := development.Requirements{Go: true}
	data, _ := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid(), Development: &req})
	os.WriteFile(id.Compose, data, 0600)
	os.Mkdir(filepath.Join(id.Root, ".hermes/development-image"), 0700)
	recipe, _ := development.Recipe(req)
	for name, data := range recipe {
		os.WriteFile(filepath.Join(id.Root, ".hermes/development-image", name), data, 0600)
	}
	imageID := "sha256:" + strings.Repeat("d", 64)
	r := &devRunner{integrationRunner: base, derived: fmt.Sprintf(`{"id":%q,"os":"linux","arch":"amd64","recipe":%q,"base":%q,"layers":["sha256:%s","sha256:%s"]}`, imageID, development.Fingerprint(req), qualification.FoundationImage, strings.Repeat("e", 64), strings.Repeat("f", 64))}
	check := func() bool {
		return HermesImageMatches(context.Background(), id, "fixture", development.ImageName(id.Project, req), imageID, r)
	}
	if !check() {
		t.Fatal("qualified image refused")
	}
	good := r.derived
	for _, bad := range []string{strings.Replace(good, imageID, "sha256:"+strings.Repeat("a", 64), 1), strings.Replace(good, development.Fingerprint(req), "wrong", 1), strings.Replace(good, strings.Repeat("e", 64), strings.Repeat("b", 64), 1), strings.Replace(good, "amd64", "arm64", 1)} {
		r.derived = bad
		if check() {
			t.Fatal("mismatched derived image admitted")
		}
	}
	r.derived = good
	os.WriteFile(filepath.Join(id.Root, ".hermes/development-image/owner.sh"), []byte("owner"), 0600)
	if check() {
		t.Fatal("owner recipe addition accepted")
	}
}

func TestBuildxVersionUsesExactToken(t *testing.T) {
	if containsVersionToken("github.com/docker/buildx v0.37.10 abc", "v0.37.1") {
		t.Fatal("accepted different plugin version")
	}
	if !containsVersionToken("github.com/docker/buildx v0.37.1 abc", "v0.37.1") {
		t.Fatal("rejected exact version")
	}
}

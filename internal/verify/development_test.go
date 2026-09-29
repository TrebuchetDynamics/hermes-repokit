package verify

import (
	"context"
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
	derived string
}

func (r *devRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	call := strings.Join(args, " ")
	if strings.Contains(call, "image inspect") {
		if args[len(args)-1] == qualification.FoundationImage {
			return process.Result{Output: `["sha256:` + strings.Repeat("e", 64) + `"]`}
		}
		return process.Result{Output: r.derived}
	}
	return r.integrationRunner.Run(ctx, p, args...)
}
func TestDevelopmentDoesNotClaimCodingAcceptanceFromToolPresence(t *testing.T) {
	id, base := integrationFixture(t)
	r := &devRunner{integrationRunner: base}
	probes := Development(context.Background(), id, r)
	if probes[0].Status == Healthy {
		t.Fatalf("tool presence certified coding: %+v", probes[0])
	}
	for _, call := range r.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, " -c ") {
			t.Fatalf("custom Python probe executed: %s", joined)
		}
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
		return HermesImageMatches(context.Background(), id, "fixture", development.ImageName(id.Container, req), imageID, r)
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

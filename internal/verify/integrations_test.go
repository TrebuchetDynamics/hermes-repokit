package verify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

type integrationRunner struct {
	hermes, service, image, config, health string
	calls                                  [][]string
}

func (r *integrationRunner) Run(_ context.Context, p string, args ...string) process.Result {
	r.calls = append(r.calls, append([]string{p}, args...))
	call := strings.Join(args, " ")
	switch {
	case strings.Contains(call, "container ls"):
		return process.Result{Output: "123456abcdef"}
	case strings.Contains(call, "container inspect"):
		if args[len(args)-1] == "123456abcdef" {
			return process.Result{Output: r.service}
		}
		return process.Result{Output: r.hermes}
	case strings.Contains(call, "image inspect"):
		return process.Result{Output: r.image}
	default:
		return process.Result{Err: fmt.Errorf("unexpected command")}
	}
}

func integrationFixture(t *testing.T) (target.Identity, *integrationRunner) {
	t.Helper()
	id, err := target.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(id.Launcher), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := launcher.Render(id, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(id.Launcher, data, 0700); err != nil {
		t.Fatal(err)
	}
	data, err = compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(id.Compose, data, 0600); err != nil {
		t.Fatal(err)
	}
	r := &integrationRunner{
		hermes: fmt.Sprintf(`{"id":"%s","status":"running","image":%q,"project":%q,"service":"hermes","workspace":%q,"home":%q}`, strings.Repeat("a", 64), qualification.FoundationImage, id.Project, id.Root, filepath.Join(id.Root, ".hermes")),
		config: `{"memory":"active"}`, health: "healthy\n",
	}
	return id, r
}

package cli

import (
	"context"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/process"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/supervision"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type layaRunner struct {
	*foundationRunner
	image   string
	invalid bool
}

func (r layaRunner) Run(ctx context.Context, p string, args ...string) process.Result {
	if strings.Contains(strings.Join(args, " "), "image inspect") {
		if r.invalid {
			return process.Result{Output: `{"id":"wrong"}`}
		}
		return process.Result{Output: fmt.Sprintf(`{"id":%q,"os":"linux","arch":"amd64","recipe":"1","nerve":%q,"model":%q}`, r.image, supervision.NerveRevision, supervision.LayaModelRevision)}
	}
	return r.foundationRunner.Run(ctx, p, args...)
}
func TestSelectedLayaInstallUpgradeAndRerun(t *testing.T) {
	a, r := foundationApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	old, _ := os.ReadFile(r.id.Compose)
	image := "sha256:" + strings.Repeat("b", 64)
	a.Runner = layaRunner{r, image, false}
	if c, _, d := invoke(t, a, "install", "--laya-image", image); c != 0 {
		t.Fatal(d)
	}
	if compose.SelectedLaya(r.id) != image {
		t.Fatal("selected image not persisted in native Compose")
	}
	backup, _ := os.ReadFile(filepath.Join(a.Directory, ".hermes/compose.before-laya-image.yaml"))
	if string(backup) != string(old) {
		t.Fatal("prior Compose backup missing")
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal("rerun lost selection:", d)
	}
}
func TestLayaRejectsUnqualifiedImageBeforePublication(t *testing.T) {
	a, r := foundationApp(t)
	image := "sha256:" + strings.Repeat("b", 64)
	a.Runner = layaRunner{r, image, true}
	if c, _, _ := invoke(t, a, "install", "--laya-image", image); c == 0 {
		t.Fatal("unqualified image accepted")
	}
	if _, err := os.Stat(filepath.Join(a.Directory, ".hermes")); !os.IsNotExist(err) {
		t.Fatal("published before image qualification")
	}
}

func TestLayaRecreationCommandPreservesRepositoryPath(t *testing.T) {
	a, r := foundationApp(t)
	renamed := filepath.Join(filepath.Dir(a.Directory), "folder up -d hermes)")
	if err := os.Rename(a.Directory, renamed); err != nil {
		t.Fatal(err)
	}
	a.Directory = renamed
	var err error
	r.id, err = target.Resolve(renamed)
	if err != nil {
		t.Fatal(err)
	}
	image := "sha256:" + strings.Repeat("b", 64)
	a.Runner = layaRunner{r, image, false}
	code, out, diag := invoke(t, a, "install", "--laya-image", image)
	if code != 0 {
		t.Fatal(diag)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, " up -d --force-recreate hermes laya)") {
			if !strings.Contains(line, "-f '"+r.id.Compose+"'") {
				t.Fatalf("recreation command changed repository path: %s", line)
			}
			return
		}
	}
	t.Fatal("missing coordinated recreation command")
}

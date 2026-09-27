package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentityUsesFullNormalizedNameAndCanonicalPath(t *testing.T) {
	root := privateDir(t)
	path := filepath.Join(root, "My Project!!")
	os.Mkdir(path, 0700)
	got, err := Resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "my-project" || got.Container != "hermes-my-project" || got.Launcher != filepath.Join(path, ".hermes/bin/hermes-my-project") {
		t.Fatalf("%+v", got)
	}
	link := filepath.Join(root, "alias")
	os.Symlink(path, link)
	alias, err := Resolve(link)
	if err != nil || alias != got {
		t.Fatalf("alias: %+v %v", alias, err)
	}
	other := filepath.Join(root, "nested", "My Project!!")
	os.MkdirAll(other, 0700)
	second, _ := Resolve(other)
	if second.Container != got.Container || second.Project == got.Project {
		t.Fatal("public names must collide, projects must differ")
	}
}
func TestUnsupportedNamesRefuseWithoutTruncation(t *testing.T) {
	for _, name := range []string{"!!!", strings.Repeat("a", 240), "é"} {
		p := filepath.Join(privateDir(t), name)
		os.Mkdir(p, 0700)
		if _, err := Resolve(p); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
}
func TestInspectFindsDanglingLinksRootComposeAndPermissions(t *testing.T) {
	for _, name := range []string{"compose.yaml", "compose.override.yml", ".hermes"} {
		t.Run(name, func(t *testing.T) {
			p := privateDir(t)
			os.Symlink("missing", filepath.Join(p, name))
			id, _ := Resolve(p)
			if issues := Inspect(id, ""); len(issues) == 0 {
				t.Fatal("missed dangling collision")
			}
		})
	}
	p := privateDir(t)
	os.Mkdir(filepath.Join(p, ".hermes"), 0755)
	id, _ := Resolve(p)
	if len(Inspect(id, "")) == 0 {
		t.Fatal("public state permitted")
	}
}
func TestPathCollisionAndPrivateNativeState(t *testing.T) {
	p := privateDir(t)
	id, _ := Resolve(p)
	bin := privateDir(t)
	os.Symlink("absent", filepath.Join(bin, id.Container))
	if len(Inspect(id, bin)) == 0 {
		t.Fatal("missed PATH dangling link")
	}
	os.Mkdir(filepath.Join(p, ".hermes"), 0700)
	os.WriteFile(filepath.Join(p, ".hermes", "owner.txt"), []byte("keep"), 0600)
	if got := Inspect(id, ""); len(got) != 0 {
		info, _ := os.Stat(p)
		t.Fatalf("valid private state: %v info=%+v uid=%d", got, info.Sys(), os.Geteuid())
	}
	os.Symlink("absent", filepath.Join(p, ".hermes", "config.yaml"))
	if len(Inspect(id, "")) == 0 {
		t.Fatal("missed native symlink")
	}
}

func privateDir(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	if err := os.Chmod(p, 0700); err != nil {
		t.Fatal(err)
	}
	return p
}

package target

import (
	"fmt"
	"io/fs"
	"net"
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
func TestInspectFindsDanglingNativeLinksAndPermissions(t *testing.T) {
	for _, name := range []string{".hermes"} {
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
func TestInspectIgnoresExistingRootComposeFiles(t *testing.T) {
	for _, name := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml", "compose.override.yaml", "compose.override.yml", "docker-compose.override.yaml", "docker-compose.override.yml"} {
		for _, kind := range []string{"regular", "malformed", "dangling-link"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				root := privateDir(t)
				path := filepath.Join(root, name)
				content := []byte("name: owner-stack\nservices:\n  owner:\n    image: busybox\n")
				if kind == "malformed" {
					content = []byte("services: [broken owner YAML")
				}
				var err error
				if kind == "dangling-link" {
					err = os.Symlink("missing-owner-file", path)
				} else {
					err = os.WriteFile(path, content, 0644)
				}
				if err != nil {
					t.Fatal(err)
				}
				before, err := os.Lstat(path)
				if err != nil {
					t.Fatal(err)
				}
				id, err := Resolve(root)
				if err != nil {
					t.Fatal(err)
				}
				if issues := Inspect(id, ""); len(issues) != 0 {
					t.Fatalf("owner Compose is outside native state: %v", issues)
				}
				after, err := os.Lstat(path)
				if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.ModTime() != after.ModTime() {
					t.Fatal("owner entry changed")
				}
				if kind == "dangling-link" {
					if link, err := os.Readlink(path); err != nil || link != "missing-owner-file" {
						t.Fatal("owner symlink changed")
					}
				} else if got, err := os.ReadFile(path); err != nil || string(got) != string(content) {
					t.Fatal("owner contents changed")
				}
			})
		}
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

func TestNativeReadableChildrenStayPrivateBehindRoot(t *testing.T) {
	p := privateDir(t)
	state := filepath.Join(p, ".hermes")
	os.Mkdir(state, 0700)
	os.Mkdir(filepath.Join(state, "profiles"), 0755)
	os.WriteFile(filepath.Join(state, "config.yaml"), []byte("native config"), 0644)
	id, _ := Resolve(p)
	if issues := Inspect(id, ""); len(issues) != 0 {
		t.Fatalf("private root protects native children: %v", issues)
	}
}

func TestNativeToolInstallDoesNotInvalidatePrivateDeployment(t *testing.T) {
	p := privateDir(t)
	state := filepath.Join(p, ".hermes")
	links := map[string]string{
		"home/.cache/uv/wheels-v6/pypi/edge-tts/revision": "../../../archive-v0/revision",
		".cache/uv/wheels-v6/pypi/browser-use/revision":   "../../../archive-v0/revision",
		".local/share/uv/tools/browser-use/bin/python":    "/usr/local/bin/python3",
		".local/bin/cua-driver":                           "/opt/data/.cua-driver/packages/current/cua-driver",
		".cua-driver/packages/current":                    "releases/0.30.2-linux",
		"bin/browser-use":                                 "/opt/data/.local/share/uv/tools/browser-use/bin/browser-use",
	}
	for rel, dest := range links {
		path := filepath.Join(state, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(dest, path); err != nil {
			t.Fatal(err)
		}
	}
	locks := []string{".cache/uv/.lock", ".local/share/uv/tools/.lock", "home/.cache/uv/.lock", "lazy-packages/.lock"}
	for _, rel := range locks {
		path := filepath.Join(state, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("native lock"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0666); err != nil {
			t.Fatal(err)
		}
	}
	id, _ := Resolve(p)
	if issues := Inspect(id, ""); len(issues) != 0 {
		t.Fatalf("native setup rejected: %v", issues)
	}
	for rel, dest := range links {
		if got, err := os.Readlink(filepath.Join(state, rel)); err != nil || got != dest {
			t.Fatalf("native link changed: %s", rel)
		}
	}
	for _, rel := range locks {
		info, err := os.Stat(filepath.Join(state, rel))
		if err != nil || info.Mode().Perm() != 0666 {
			t.Fatalf("native lock changed: %s", rel)
		}
	}
}

func TestNativeToolExceptionsDoNotPermitRedirectedManagedPaths(t *testing.T) {
	for _, rel := range []string{
		"compose.yaml", "config.yaml", ".env", "auth.json", "kanban.db", "profiles", "bin",
		"development-image", ".cache", ".cache/uv", ".local", ".local/bin",
		".local/share", ".local/share/uv", ".local/share/uv/tools", ".cua-driver", ".cua-driver/packages",
		"profiles/executor/config.yaml", "development-image/Dockerfile",
		"home", "home/.cache", "home/.cache/uv", "lazy-packages", "gateway.sock", "state/gateway.loop-tick.123.sock",
	} {
		t.Run(rel, func(t *testing.T) {
			p := privateDir(t)
			path := filepath.Join(p, ".hermes", rel)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(privateDir(t), path); err != nil {
				t.Fatal(err)
			}
			id, _ := Resolve(p)
			if len(Inspect(id, "")) == 0 {
				t.Fatalf("accepted redirected %s", rel)
			}
		})
	}
	p := privateDir(t)
	id, _ := Resolve(p)
	path := filepath.Join(p, ".hermes", "bin", id.Container)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/opt/data/bin/browser-use", path); err != nil {
		t.Fatal(err)
	}
	if len(Inspect(id, "")) == 0 {
		t.Fatal("accepted redirected generated launcher")
	}
}

func TestNativeGatewaySocketsRemainInspectable(t *testing.T) {
	for _, tc := range []struct {
		rel     string
		allowed bool
	}{
		{"gateway.sock", true},
		{"state/gateway.loop-tick.123.sock", true},
		{"unknown.sock", false},
		{"config.yaml", false},
		{"state/gateway.loop-tick.bad.sock", false},
		{"state/gateway.loop-tick.123.sock.extra", false},
	} {
		t.Run(tc.rel, func(t *testing.T) {
			p, err := os.MkdirTemp("", "rksock-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(p)
			path := filepath.Join(p, ".hermes", tc.rel)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("unix", path)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err := os.Chmod(path, 0755); err != nil {
				t.Fatal(err)
			}
			id, _ := Resolve(p)
			if got := len(Inspect(id, "")) == 0; got != tc.allowed {
				t.Fatalf("socket allowed=%v, want %v", got, tc.allowed)
			}
			if err := os.Chmod(path, 0777); err != nil {
				t.Fatal(err)
			}
			if len(Inspect(id, "")) == 0 {
				t.Fatal("accepted writable socket")
			}
		})
	}
}

func TestNativeCodeKernelSocketsRemainInspectable(t *testing.T) {
	for _, tc := range []struct {
		rel     string
		allowed bool
	}{
		{"cache/scratch/hermes_rpc_9975be3cdc9b4889816ce660e36880a6.sock", true},
		{"profiles/researcher/cache/scratch/hermes_rpc_9975be3cdc9b4889816ce660e36880a6.sock", true},
		{"cache/scratch/hermes_rpc_bad.sock", false},
		{"cache/hermes_rpc_9975be3cdc9b4889816ce660e36880a6.sock", false},
		{"cache/scratch/other.sock", false},
	} {
		t.Run(tc.rel, func(t *testing.T) {
			// Bind relative to keep AF_UNIX paths below the kernel length limit.
			root, err := os.MkdirTemp("/tmp", "rk-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(root)
			path := filepath.Join(root, ".hermes", tc.rel)
			if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			listener, err := net.Listen("unix", filepath.Join(".hermes", tc.rel))
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err = os.Chmod(path, 0600); err != nil {
				t.Fatal(err)
			}
			id, err := Resolve(root)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(Inspect(id, "")) == 0; got != tc.allowed {
				t.Fatalf("socket allowed=%v, want %v", got, tc.allowed)
			}
			if err = os.Chmod(path, 0660); err != nil {
				t.Fatal(err)
			}
			if len(Inspect(id, "")) == 0 {
				t.Fatal("accepted exposed RPC socket")
			}
		})
	}
}

func TestNativeToolExceptionsKeepModesAndPrefixesStrict(t *testing.T) {
	for _, rel := range []string{"config.yaml", "compose.yaml", "development-image/Dockerfile", ".cache/uv/module.py", ".local/share/uv/tools/module.py", ".cache/uv-other/.lock", "bin/browser-use"} {
		t.Run(rel, func(t *testing.T) {
			p := privateDir(t)
			path := filepath.Join(p, ".hermes", rel)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("owner data"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0666); err != nil {
				t.Fatal(err)
			}
			id, _ := Resolve(p)
			if len(Inspect(id, "")) == 0 {
				t.Fatalf("accepted writable native file: %s", rel)
			}
		})
	}
	for _, rel := range []string{"bin", ".cache/uv", ".cache/uv/archive", ".local/share/uv/tools"} {
		t.Run("directory/"+rel, func(t *testing.T) {
			p := privateDir(t)
			path := filepath.Join(p, ".hermes", rel)
			if err := os.MkdirAll(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0777); err != nil {
				t.Fatal(err)
			}
			id, _ := Resolve(p)
			if len(Inspect(id, "")) == 0 {
				t.Fatalf("accepted writable directory: %s", rel)
			}
		})
	}
}

func TestHuggingFaceModelCacheMetadataPreservesNativeArtifacts(t *testing.T) {
	model := "models--owner--speech-model"
	hash := strings.Repeat("a", 64)
	revision := strings.Repeat("b", 40)
	base := ".cache/huggingface/hub/"
	for _, tc := range []struct {
		path string
		link bool
		mode os.FileMode
		want bool
	}{
		{base + model + "/blobs/" + hash, true, 0, true},
		{base + model + "/snapshots/" + revision + "/model.bin", true, 0, true},
		{base + model + "/snapshots/" + revision + "/nested/config.json", true, 0, true},
		{base + ".locks/" + model + "/" + hash + ".lock", false, 0664, true},
		{base + ".locks/" + model + "/" + revision + ".lock", false, 0664, true},
		{base + "blobs/aa/" + hash + ".lock", false, 0666, true},
		{base + "blobs/aa/" + hash + ".refs", false, 0666, true},
		{base + "blobs/aa/" + hash, false, 0666, false},
		{base + "blobs/aa/" + hash + ".refs", true, 0, false},
		{base + "blobs/aa/" + hash + ".lock", false, 0777, false},
		{base + ".locks/" + model + "/owner.py", false, 0666, false},
		{base + ".locks/" + model + "/unknown.lock", false, 0666, false},
		{base + model + "/snapshots/" + revision + "/owner.py", false, 0666, false},
		{base + model + "/snapshots/unknown/model.bin", true, 0, false},
		{base + model + "/refs/main", true, 0, false},
		{base + model + "/snapshots/" + revision, true, 0, false},
		{base + model + "/blobs", true, 0, false},
		{base + model, true, 0, false},
		{".cache/huggingface/hub", true, 0, false},
		{".cache/huggingface-other/hub/" + model + "/blobs/" + hash, true, 0, false},
	} {
		t.Run(tc.path+fmt.Sprint(tc.link, tc.mode), func(t *testing.T) {
			root := privateDir(t)
			path := filepath.Join(root, ".hermes", tc.path)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if tc.link {
				// A dangling target verifies inspection never follows cache pointers.
				if err := os.Symlink("/missing-native-model-blob", path); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, []byte("cache metadata"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, tc.mode); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			id, err := Resolve(root)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(Inspect(id, "")) == 0; got != tc.want {
				t.Fatalf("allowed=%v, want %v", got, tc.want)
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Fatal("inspection changed native cache")
			}
			if tc.want {
				if err := os.Chmod(filepath.Dir(path), 0777); err != nil {
					t.Fatal(err)
				}
				if len(Inspect(id, "")) == 0 {
					t.Fatal("accepted writable cache ancestor")
				}
			}
		})
	}
}

// vanishedEntry is a directory listing whose file was deleted before Info.
type vanishedEntry struct{ dir bool }

func (v vanishedEntry) Name() string               { return "gone" }
func (v vanishedEntry) IsDir() bool                { return v.dir }
func (v vanishedEntry) Type() fs.FileMode          { return 0 }
func (v vanishedEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrNotExist }

// A running Hermes deletes short-lived files while the native state is walked;
// an entry that vanished is skipped, but the state root and real errors are not.
func TestNativeWalkSkipsEntriesThatVanished(t *testing.T) {
	state := "/r/.hermes"
	if info, err := walkedInfo(state+"/tmp.yaml", state, vanishedEntry{}, nil); info != nil || err != nil {
		t.Fatalf("vanished file must be skipped: %v %v", info, err)
	}
	if _, err := walkedInfo(state+"/scratch", state, vanishedEntry{dir: true}, fs.ErrNotExist); err != filepath.SkipDir {
		t.Fatalf("vanished directory must be skipped: %v", err)
	}
	if _, err := walkedInfo(state, state, nil, fs.ErrNotExist); err == nil {
		t.Fatal("missing state root must fail")
	}
	if _, err := walkedInfo(state+"/locked", state, vanishedEntry{dir: true}, fs.ErrPermission); err == nil {
		t.Fatal("unreadable entry must still fail inspection")
	}
}

func TestRepeatedPathDirectoryIsOneCollision(t *testing.T) {
	root := t.TempDir()
	id, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, id.Container), []byte("other"), 0700); err != nil {
		t.Fatal(err)
	}
	path := strings.Join([]string{bin, bin + "/", bin}, string(os.PathListSeparator))
	count := 0
	for _, issue := range Inspect(id, path) {
		if strings.HasPrefix(issue, "PATH collision") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("repeated PATH directory reported %d collisions", count)
	}
}

func TestRepoKitBootstrapOnPathIsNotACollision(t *testing.T) {
	id, err := Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	command := filepath.Join(bin, id.Container)
	for data, collides := range map[string]bool{
		"\x7fELF...usage: hermes-repokit <plan|install|setup|verify|start|stop|remove>...": false,
		"#!/bin/sh\necho owner tool\n": true,
	} {
		if err := os.WriteFile(command, []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
		found := strings.Contains(strings.Join(Inspect(id, bin), "; "), "PATH collision")
		if found != collides {
			t.Fatalf("%q: collision=%v", data[:12], found)
		}
	}
}

// Ordinary granted-tool activity (npx package links, the browser harness's
// private control socket) must not make RepoKit refuse its own deployment.
func TestInspectAllowsNpmLinksAndPrivateBrowserSocket(t *testing.T) {
	// Unix socket paths are limited to about 108 bytes, so use a short root.
	root, err := os.MkdirTemp("/tmp", "rk")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	id, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(root, ".hermes")
	bin := filepath.Join(state, "home/.npm/_npx/abc/node_modules/.bin")
	runtimeDir := filepath.Join(state, "home/.config/browser-harness/runtime")
	for _, dir := range []string{bin, filepath.Join(state, "home/.npm/_npx/abc/node_modules/agent-browser/bin"), runtimeDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(state, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../agent-browser/bin/agent-browser.js", filepath.Join(bin, "agent-browser")); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(runtimeDir, "bu-default.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := os.Chmod(socket, 0700); err != nil {
		t.Fatal(err)
	}
	for _, issue := range Inspect(id, "") {
		if strings.Contains(issue, ".npm") || strings.Contains(issue, "browser-harness") {
			t.Fatalf("ordinary tool state refused: %s", issue)
		}
	}
	if err := os.Chmod(socket, 0777); err != nil {
		t.Fatal(err)
	}
	flagged := false
	for _, issue := range Inspect(id, "") {
		flagged = flagged || strings.Contains(issue, "bu-default.sock")
	}
	if !flagged {
		t.Fatal("world-writable browser socket accepted")
	}
}

package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func namingApp(t *testing.T) (App, *foundationRunner) {
	t.Helper()
	a, r := foundationApp(t)
	path := filepath.Join(filepath.Dir(a.Directory), "hermes-repokit")
	if err := os.Rename(a.Directory, path); err != nil {
		t.Fatal(err)
	}
	a.Directory = path
	var err error
	r.id, err = target.Resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	return a, r
}

func TestNameMigrationRefusesConflictsWithoutChangingState(t *testing.T) {
	for _, kind := range []string{"old-edit", "old-link", "old-noexec", "new-edit", "new-link", "new-noexec", "recipe-drift", "compose-drift", "backup-conflict", "host-command", "new-container", "old-container"} {
		t.Run(kind, func(t *testing.T) {
			a, r := namingApp(t)
			previous := previousNames(t, a, r)
			state := filepath.Join(a.Directory, ".hermes")
			old := filepath.Join(state, "bin/hermes-hermes-repokit")
			current := filepath.Join(state, "bin/hermes-repokit")
			launch, err := os.ReadFile(old)
			if err != nil {
				t.Fatal(err)
			}
			write := func(path string, data []byte, mode os.FileMode) {
				t.Helper()
				if err := os.WriteFile(path, data, mode); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "old-edit":
				write(old, []byte("#!/bin/sh\n# owner command\n"), 0700)
			case "old-link":
				if err := os.Rename(old, filepath.Join(state, "saved-launcher")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../saved-launcher", old); err != nil {
					t.Fatal(err)
				}
			case "old-noexec":
				if err := os.Chmod(old, 0600); err != nil {
					t.Fatal(err)
				}
			case "new-edit":
				write(current, []byte("#!/bin/sh\n# owner command\n"), 0700)
			case "new-link":
				if err := os.Symlink(old, current); err != nil {
					t.Fatal(err)
				}
			case "new-noexec":
				write(current, launch, 0600)
			case "recipe-drift":
				write(filepath.Join(state, "development-image/Dockerfile"), []byte("# owner recipe"), 0600)
			case "compose-drift":
				write(r.id.Compose, append(previous, []byte("# owner compose\n")...), 0600)
			case "backup-conflict":
				write(filepath.Join(state, "compose.before-names.yaml"), []byte("owner backup"), 0600)
			case "host-command":
				a.Path = filepath.Join(os.Getenv("HOME"), ".local/bin")
				write(filepath.Join(a.Path, "hermes-repokit"), []byte("#!/bin/sh\n# owner executable\n"), 0700)
			case "new-container", "old-container":
				r.names = "hermes-repokit"
				if kind == "old-container" {
					r.names = "hermes-hermes-repokit"
				}
				r.runtime = `{"project":"other","workspace":"/other","home":"/other/.hermes"}`
			}
			before := namingSnapshot(t, state)
			if c, _, _ := invoke(t, a, "install"); c == 0 {
				t.Fatal("accepted conflict", kind)
			}
			after := namingSnapshot(t, state)
			if len(before) != len(after) {
				t.Fatal("changed files on refusal")
			}
			for name, data := range before {
				if !bytes.Equal(data, after[name]) {
					t.Fatal("changed on refusal", name)
				}
			}
		})
	}
}

func namingSnapshot(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			files[path] = []byte("link:" + link)
			return err
		}
		data, err := os.ReadFile(path)
		files[path] = data
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestNameMigrationResumesAfterNewLauncherPublished(t *testing.T) {
	a, r := namingApp(t)
	previous := previousNames(t, a, r)
	state := filepath.Join(a.Directory, ".hermes")
	launch, err := os.ReadFile(filepath.Join(state, "bin/hermes-hermes-repokit"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "compose.before-names.yaml"), previous, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.id.Launcher, launch, 0700); err != nil {
		t.Fatal(err)
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal("rerun", d)
	}
}

func TestRunningImageRenameWaitsForComposeRecreation(t *testing.T) {
	for _, kind := range []string{"qualified", "qualified-new-docker", "wrong-mount", "wrong-image-id", "wrong-project"} {
		t.Run(kind, func(t *testing.T) {
			a, r := foundationApp(t) // Non-prefixed repository keeps its container name.
			if c, _, d := invoke(t, a, "install"); c != 0 {
				t.Fatal(d)
			}
			data, err := os.ReadFile(r.id.Compose)
			if err != nil {
				t.Fatal(err)
			}
			current := development.ImageName(r.id.Container, development.Requirements{})
			old := r.id.Project + "-hermes-dev:" + development.Fingerprint(development.Requirements{})[:24]
			r.runtime = strings.Replace(developmentRuntimeFixture(r.id), current, old, 1)
			if err := os.WriteFile(r.id.Compose, bytes.Replace(data, []byte(current), []byte(old), 1), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "wrong-mount":
				r.runtime = strings.ReplaceAll(r.runtime, r.id.Root, "/other")
			case "wrong-image-id":
				r.runtime = strings.ReplaceAll(r.runtime, strings.Repeat("d", 64), strings.Repeat("b", 64))
			case "wrong-project":
				r.runtime = strings.ReplaceAll(r.runtime, r.id.Project, "other-project")
			}
			for attempt := 0; attempt < 2; attempt++ {
				args := []string{"install"}
				if kind == "qualified-new-docker" && attempt == 0 {
					args = append(args, "--docker-tests")
				}
				code, out, diag := invoke(t, a, args...)
				if strings.HasPrefix(kind, "qualified") {
					if code != 0 || !strings.Contains(out, "initialization pending") || len(r.composeCalls) == 0 || !strings.HasSuffix(r.composeCalls[len(r.composeCalls)-1], " up -d --build hermes") {
						t.Fatalf("install did not recreate the renamed deployment: %d %v %s %s", code, r.composeCalls, out, diag)
					}
				} else if code == 0 {
					t.Fatal("unqualified previous runtime accepted", kind)
				}
			}
		})
	}
}

func TestInstallReadableRuntimeNames(t *testing.T) {
	a, r := namingApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	data, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("container_name: \"hermes-repokit\"")) || !bytes.Contains(data, []byte("image: \"repokit/hermes-repokit:")) {
		t.Fatalf("runtime names are not readable:\n%s", data)
	}
	link := filepath.Join(os.Getenv("HOME"), ".local/bin/hermes-repokit")
	if got, err := os.Readlink(link); err != nil || got != filepath.Join(a.Directory, ".hermes/bin/hermes-repokit") {
		t.Fatalf("host command: %q %v", got, err)
	}
}

// Model the exact pre-naming publication, keeping all native state intact.
func previousNames(t *testing.T, a App, r *foundationRunner) []byte {
	t.Helper()
	args := []string{"install"}
	if a.DockerTests {
		args = append(args, "--docker-tests")
	}
	if c, _, d := invoke(t, a, args...); c != 0 {
		t.Fatal(d)
	}
	data, err := os.ReadFile(r.id.Compose)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("container_name: \"hermes-repokit\""), []byte("container_name: \"hermes-hermes-repokit\""), 1)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "    image:") {
			// Keep the current recipe fingerprint verbatim while reconstructing
			// the historical tag independently of the migration renderer.
			tag := strings.TrimSuffix(line[strings.LastIndex(line, ":")+1:], "\"")
			lines[i] = fmt.Sprintf("    image: %q", r.id.Project+"-hermes-dev:"+tag)
			break
		}
	}
	data = []byte(strings.Join(lines, "\n"))
	if err := os.WriteFile(r.id.Compose, data, 0600); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(a.Directory, ".hermes/bin/hermes-hermes-repokit")
	if r.id.Launcher != old {
		if err := os.Rename(r.id.Launcher, old); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(os.Getenv("HOME"), ".local/bin/hermes-repokit")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(old, filepath.Join(os.Getenv("HOME"), ".local/bin/hermes-hermes-repokit")); err != nil {
			t.Fatal(err)
		}
	}
	return data
}

func TestNameMigrationRetainsDevelopmentAndDockerSelections(t *testing.T) {
	for _, goTool := range []bool{false, true} {
		for _, dockerTests := range []bool{false, true} {
			t.Run(fmt.Sprintf("go=%v/docker=%v", goTool, dockerTests), func(t *testing.T) {
				a, r := namingApp(t)
				a.DockerTests = dockerTests
				a.HostSELinux = selinux.Enforcing
				if goTool {
					if err := os.WriteFile(filepath.Join(a.Directory, "go.mod"), []byte("module example.test/demo\n\ngo 1.26.0\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				previousNames(t, a, r)
				a.DockerTests = false // Existing opt-in must be recovered without the flag.
				if c, _, d := invoke(t, a, "install"); c != 0 {
					t.Fatal(d)
				}
				selected, ok := compose.DevelopmentInstallSelected(r.id)
				if !ok || selected.Development.Go != goTool || selected.DockerTests != dockerTests || !selected.SELinux.Enabled() {
					t.Fatalf("lost existing options: %+v", selected)
				}
			})
		}
	}
}

func TestInstallMigratesNamesPreservingNativeStateAndOldCommand(t *testing.T) {
	a, r := namingApp(t)
	previous := previousNames(t, a, r)
	private := filepath.Join(a.Directory, ".hermes/owner-data")
	if err := os.WriteFile(private, []byte("owner state"), 0600); err != nil {
		t.Fatal(err)
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	data, _ := os.ReadFile(r.id.Compose)
	if !bytes.Contains(data, []byte("container_name: \"hermes-repokit\"")) || !bytes.Contains(data, []byte("image: \"repokit/hermes-repokit:")) {
		t.Fatal("names not migrated")
	}
	backup, _ := os.ReadFile(filepath.Join(a.Directory, ".hermes/compose.before-names.yaml"))
	if !bytes.Equal(backup, previous) {
		t.Fatal("old Compose not backed up")
	}
	for _, command := range []string{"hermes-repokit", "hermes-hermes-repokit"} {
		resolved, err := filepath.EvalSymlinks(filepath.Join(os.Getenv("HOME"), ".local/bin", command))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(resolved)
		if err != nil || !bytes.Contains(b, []byte(r.id.Compose)) {
			t.Fatalf("lost command %s", command)
		}
	}
	if b, _ := os.ReadFile(private); string(b) != "owner state" {
		t.Fatal("owner state changed")
	}
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal("rerun", d)
	}
}

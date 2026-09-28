package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// Public generated preimage from ce7b6c8; never read or adopt private state.
func legacyLayaCompose(t *testing.T, id target.Identity) []byte {
	t.Helper()
	base, err := compose.Render(id, compose.Options{HermesImage: qualification.FoundationImage, OpenVikingImage: projectmemory.Image, UID: os.Getuid(), GID: os.Getgid()})
	if err != nil {
		t.Fatal(err)
	}
	return append(base, []byte(fmt.Sprintf(`  laya:
    build:
      context: ./laya-image
    platform: linux/amd64
    network_mode: service:hermes
    depends_on:
      hermes:
        condition: service_started
        restart: true
    restart: unless-stopped
    user: "%d:%d"
    read_only: true
    tmpfs:
      - /tmp:rw,nosuid,nodev,size=256m
    cpus: 4
    mem_limit: 6g
    environment:
      HF_HOME: /cache/huggingface
      TORCHINDUCTOR_CACHE_DIR: /cache/torchinductor
    volumes:
      - type: bind
        source: "./laya"
        target: /cache
        bind:
          create_host_path: false
`, os.Getuid(), os.Getgid()))...)
}

func TestInstallMigratesExactLegacySidecarsPreservingOwnerState(t *testing.T) {
	for _, drift := range []bool{false, true} {
		t.Run(fmt.Sprint(drift), func(t *testing.T) {
			a, _ := legacyMemoryFixture(t)
			id, err := target.Resolve(a.Directory)
			if err != nil {
				t.Fatal(err)
			}
			prior := legacyLayaCompose(t, id)
			if drift {
				prior = append(prior, []byte("# owner customization\n")...)
			}
			if err = os.WriteFile(id.Compose, prior, 0600); err != nil {
				t.Fatal(err)
			}
			state := filepath.Join(a.Directory, ".hermes")
			preserved := map[string]string{"config.yaml": "owner-config\n", "openviking/ov.conf": "private memory config", "laya/model": "owner model", "profiles/owner/SOUL.md": "owner identity", "kanban.db": "existing board"}
			for name, data := range preserved {
				path := filepath.Join(state, name)
				if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			code, out, diag := invoke(t, a, "install")
			if (code != 0) != drift {
				t.Fatalf("install: %d %s %s", code, out, diag)
			}
			if drift {
				got, _ := os.ReadFile(id.Compose)
				if string(got) != string(prior) {
					t.Fatal("changed unrecognized Compose")
				}
			} else {
				backup, err := os.ReadFile(filepath.Join(state, "compose.before-core.yaml"))
				if err != nil || string(backup) != string(prior) {
					t.Fatalf("missing original Compose backup: %v", err)
				}
				current, _ := os.ReadFile(id.Compose)
				if strings.Contains(string(current), "  laya:") || strings.Contains(string(current), "  openviking:") || !strings.Contains(string(current), "./development-image") {
					t.Fatal("replacement is not core runtime")
				}
				if code, _, diag = invoke(t, a, "install"); code != 0 {
					t.Fatal(diag)
				}
			}
			for name, want := range preserved {
				got, err := os.ReadFile(filepath.Join(state, name))
				if err != nil || string(got) != want {
					t.Fatalf("changed owner state %s: %v", name, err)
				}
			}
		})
	}
}

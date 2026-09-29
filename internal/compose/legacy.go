package compose

import (
	"bytes"
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

func LegacyLayaBuildSelected(id target.Identity) bool {
	expected, err := LegacyLayaBuild(id, os.Getuid(), os.Getgid())
	return err == nil && bytes.Equal(generatedData(id), expected)
}

// LegacyLayaBuild is the exact ce7b6c8 generated preimage, for migration only.
// It does not add a Laya selection to new installations or adopt edited stacks.
func LegacyLayaBuild(id target.Identity, uid, gid int) ([]byte, error) {
	base, err := Render(id, Options{HermesImage: qualification.FoundationImage, UID: uid, GID: gid})
	if err != nil {
		return nil, err
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
`, uid, gid))...), nil
}

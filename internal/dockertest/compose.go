// Package dockertest renders the explicitly selected disposable Docker daemon.
package dockertest

import "fmt"

// Image is the official Docker 29.8.0 DinD multi-platform index. Provenance is
// recorded in packaging/docker-test/README.md; never resolve a mutable tag.
const Image = "docker:29.8.0-dind@sha256:5efed980cba3fc126cf54e21a5a6ff8849d05b6e0623d6e7612f48e9cd6cd17e"

// DaemonCommand is shared by generation and read-only runtime identity checks.
func DaemonCommand(uid, gid int) string {
	return fmt.Sprintf("mkdir -p /docker-test/run /docker-tests; chown 0:%d /docker-test/run; chmod 0750 /docker-test/run; chown %d:%d /docker-tests; chmod 0700 /docker-tests; exec dockerd-entrypoint.sh dockerd --host=unix:///docker-test/run/docker.sock --group=%d --label=io.repokit.docker-test=1", gid, uid, gid, gid)
}

// EmitServices returns service entries only. The explicit dockerd argument is
// essential: the upstream entrypoint otherwise adds a TCP listener by default.
func EmitServices(uid, gid int) (string, error) {
	if uid <= 0 || gid <= 0 {
		return "", fmt.Errorf("Docker test service requires non-root host UID/GID")
	}
	return fmt.Sprintf(`  docker-test:
    image: %q
    profiles: [docker-tests]
    privileged: true
    restart: unless-stopped
    networks: [docker-test]
    environment:
      DOCKER_TLS_CERTDIR: ""
    entrypoint: ["sh", "-ec"]
    command:
      - %q
    healthcheck:
      test: ["CMD", "docker", "--host=unix:///docker-test/run/docker.sock", "info", "--format", "{{json .ID}}"]
      interval: 5s
      timeout: 3s
      retries: 20
      start_period: 10s
    volumes:
      - type: volume
        source: docker-test-run
        target: /docker-test/run
      - type: volume
        source: docker-test-work
        target: /docker-tests
      - type: volume
        source: docker-test-data
        target: /var/lib/docker
`, Image, DaemonCommand(uid, gid)), nil
}

// Mounts are appended only to an explicitly selected Hermes service. No host
// daemon socket, checkout or native state is mounted into the test daemon.
func Mounts() string {
	return `      - type: volume
        source: docker-test-run
        target: /docker-test/run
        read_only: true
      - type: volume
        source: docker-test-work
        target: /docker-tests
`
}

// Resources uses Compose's project-scoped names, never shared external volumes.
func Resources() string {
	return `volumes:
  docker-test-run:
  docker-test-work:
  docker-test-data:
networks:
  docker-test:
`
}

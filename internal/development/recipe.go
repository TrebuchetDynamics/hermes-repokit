package development

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/projectmemory"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	assets "github.com/TrebuchetDynamics/hermes-repokit/packaging/development"
	dockertest "github.com/TrebuchetDynamics/hermes-repokit/packaging/docker-test"
)

// These are pinned recipe candidates, not a claim of live coding acceptance.
// Node/npm/Python come from the immutable Hermes base and are asserted at build.
const GoVersion = "1.26.6"
const NodeVersion = "26.5.1"
const NPMVersion = "11.17.0"
const PythonVersion = "3.13.5"
const JQVersion = "1.8.2"
const ComposeVersion = "5.5.1"
const BuildxVersion = "0.37.1"

const goInstall = `RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) arch=amd64; go_sha=708effb774be8237570d0add163225abbdfaf4fca28b2611df167beba4feef89 ;; \
      arm64) arch=arm64; go_sha=d0507e9e9d7fe012aae570108cbd76c15de879e17130ab8cb90d4d7445cb1f2e ;; \
      *) echo 'Unqualified Go architecture' >&2; exit 1 ;; \
    esac; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 300 \
      "https://go.dev/dl/go1.26.6.linux-${arch}.tar.gz" -o /tmp/repokit-go.tar.gz; \
    printf '%s  %s\n' "$go_sha" /tmp/repokit-go.tar.gz | sha256sum -c -; \
    test ! -e /usr/local/go; \
    tar -C /usr/local -xzf /tmp/repokit-go.tar.gz; \
    rm /tmp/repokit-go.tar.gz; \
    ln -s /usr/local/go/bin/go /usr/local/bin/go; \
    ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt; \
    /usr/local/go/bin/go version | grep -F 'go1.26.6 linux/'; \
    mkdir /tmp/repokit-go-smoke; \
    printf '%s\n' 'package smoke' 'import "testing"' 'func TestSmoke(t *testing.T) { if 2+2 != 4 { t.Fatal("compiler") } }' > /tmp/repokit-go-smoke/smoke_test.go; \
    cd /tmp/repokit-go-smoke; \
    GOTOOLCHAIN=local GO111MODULE=off GOCACHE=/tmp/repokit-go-cache /usr/local/go/bin/go test -race; \
    rm -rf /tmp/repokit-go-smoke /tmp/repokit-go-cache
ENV PATH="/usr/local/go/bin:${PATH}" \
    GOTOOLCHAIN=local \
    GOCACHE=/opt/data/development/go-build \
    GOMODCACHE=/opt/data/development/go-mod
`

func recipeInputs(req Requirements) map[string][]byte {
	template, err := assets.Assets.ReadFile("Dockerfile")
	if err != nil {
		panic("missing embedded development Dockerfile")
	}
	helper, err := dockertest.Assets.ReadFile("repokit-docker-test")
	if err != nil {
		panic("missing embedded Docker test helper")
	}
	goSteps := ""
	if req.Go {
		goSteps = goInstall
	}
	content := strings.NewReplacer("{{HERMES_IMAGE}}", qualification.FoundationImage, "{{OPENVIKING_IMAGE}}", projectmemory.Image, "{{GO_INSTALL}}", goSteps).Replace(string(template))
	files := map[string][]byte{"Dockerfile": []byte(content), "repokit-docker-test": helper, ".dockerignore": []byte("*\n!Dockerfile\n!repokit-docker-test\n!repokit-openviking\n!openviking-run\n!openviking-finish\n!patch-openviking-entrypoint.py\n")}
	for _, name := range []string{"repokit-openviking", "openviking-run", "openviking-finish", "patch-openviking-entrypoint.py"} {
		data, err := assets.Assets.ReadFile(name)
		if err != nil {
			panic("missing embedded memory asset")
		}
		files[name] = data
	}
	return files
}

// Fingerprint identifies the complete recipe including the isolated-test helper.
// The generation label itself is excluded to avoid a recursive hash.
func Fingerprint(req Requirements) string {
	files := recipeInputs(req)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(files[name]))
		h.Write(files[name])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func Recipe(req Requirements) (map[string][]byte, error) {
	files := recipeInputs(req)
	files["Dockerfile"] = []byte(strings.ReplaceAll(string(files["Dockerfile"]), "{{RECIPE_HASH}}", Fingerprint(req)))
	return files, nil
}

func ImageName(container string, req Requirements) string {
	return "repokit/" + container + ":" + Fingerprint(req)[:24]
}

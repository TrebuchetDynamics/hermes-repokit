package development

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	assets "github.com/TrebuchetDynamics/hermes-repokit/packaging/development"
	dockertest "github.com/TrebuchetDynamics/hermes-repokit/packaging/docker-test"
)

// These are pinned recipe candidates, not a claim of live coding acceptance.
// Node/npm/Python come from the immutable Hermes base and are asserted at build.
const GoVersion = "1.26.6"
const StaticcheckVersion = "2026.2.1"
const RustVersion = "1.98.1"
const FlutterVersion = "3.47.5"
const DartVersion = "3.13.4"
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
# staticcheck: Go static analysis beyond go vet, from the checksum-pinned release.
RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) arch=amd64; sc_sha=91186205a78db3f2d40efb3c102749aef66f85c2204793de7488d163b655aa7c ;; \
      arm64) arch=arm64; sc_sha=594421f28ba620ea14b98377cb84a309cf73cc420ba0f52f30c7fa4d92fd2b0e ;; \
      *) echo 'Unqualified staticcheck architecture' >&2; exit 1 ;; \
    esac; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 300 \
      "https://github.com/dominikh/go-tools/releases/download/2026.2.1/staticcheck_linux_${arch}.tar.gz" -o /tmp/repokit-staticcheck.tar.gz; \
    printf '%s  %s\n' "$sc_sha" /tmp/repokit-staticcheck.tar.gz | sha256sum -c -; \
    tar -C /tmp -xzf /tmp/repokit-staticcheck.tar.gz staticcheck/staticcheck; \
    install -m 0755 /tmp/staticcheck/staticcheck /usr/local/bin/staticcheck; \
    rm -rf /tmp/repokit-staticcheck.tar.gz /tmp/staticcheck; \
    /usr/local/bin/staticcheck -version | grep -F '2026.2.1'
# Go caches live on the project's toolchain-cache volume, never in the
# repository's .hermes. Docker copies this sticky, world-writable directory
# into the empty volume, so the runtime-remapped hermes user can write there.
RUN install -d -m 1777 /var/cache/repokit
ENV PATH="/usr/local/go/bin:${PATH}" \
    GOTOOLCHAIN=local \
    GOCACHE=/var/cache/repokit/go-build \
    GOMODCACHE=/var/cache/repokit/go-mod
`

// rustInstall is the official standalone Rust release (dated, immutable URL),
// checksum-pinned, with cargo, clippy, rustfmt and rust-analyzer. Cargo's
// registry and git caches live on the toolchain-cache volume.
const rustInstall = `RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) arch=x86_64; rust_sha=5326b36c53de11d148c8f8dab6553a3d1006c2cfd32123683073fad3c302605b ;; \
      arm64) arch=aarch64; rust_sha=0b514a8cc1cbcd939bff0f151661fe58b6ea5c7a7f645a5098c69e32e8c1e0a2 ;; \
      *) echo 'Unqualified Rust architecture' >&2; exit 1 ;; \
    esac; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 900 \
      "https://static.rust-lang.org/dist/2026-09-03/rust-1.98.1-${arch}-unknown-linux-gnu.tar.xz" -o /tmp/repokit-rust.tar.xz; \
    printf '%s  %s\n' "$rust_sha" /tmp/repokit-rust.tar.xz | sha256sum -c -; \
    mkdir /tmp/repokit-rust; \
    tar -xJf /tmp/repokit-rust.tar.xz -C /tmp/repokit-rust --strip-components=1; \
    /tmp/repokit-rust/install.sh --prefix=/usr/local --disable-ldconfig \
      --components=rustc,rust-std-${arch}-unknown-linux-gnu,cargo,rustfmt-preview,clippy-preview,rust-analyzer-preview; \
    rm -rf /tmp/repokit-rust /tmp/repokit-rust.tar.xz; \
    rustc --version | grep -F 'rustc 1.98.1 '; \
    cargo clippy --version; rustfmt --version; \
    cd /tmp; CARGO_HOME=/tmp/repokit-cargo cargo new --quiet --vcs none repokit-rust-smoke; \
    cd /tmp/repokit-rust-smoke; CARGO_HOME=/tmp/repokit-cargo cargo test --quiet --offline; \
    rm -rf /tmp/repokit-rust-smoke /tmp/repokit-cargo
RUN install -d -m 1777 /var/cache/repokit
ENV CARGO_HOME=/var/cache/repokit/cargo
`

// flutterInstall is the official stable Flutter SDK (with its Dart), pinned by
// the checksum in Flutter's release manifest. Flutter publishes Linux SDKs for
// x86_64 only; elsewhere the step is skipped and verify reports it missing.
// Flutter unpacks downloaded artifacts with unzip, which the base lacks; it
// comes from Debian's own package (permanent snapshot URL, checksum from the
// signed trixie index), never from apt.
// A smoke project runs analyze, test and a web build at build time so their
// artifacts are cached; Flutter writes into its own SDK, so the SDK is opened
// to the runtime-remapped user in the same layer. Pub's cache lives on the
// toolchain-cache volume. Flutter's and Dart's telemetry are suppressed.
const flutterInstall = `RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) ;; \
      *) echo 'Flutter publishes Linux SDKs for x86_64 only; Flutter is not installed' >&2; exit 0 ;; \
    esac; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 300 \
      "https://snapshot.debian.org/file/b8536b5816fa245c8e1c3df6f84d812f15d51b7b" -o /tmp/repokit-unzip.deb; \
    printf '%s  %s\n' fc11f8736bdec6fc46a6864f75bce0c8206a790df37dbcab5ddb25b1bf0f58f0 /tmp/repokit-unzip.deb | sha256sum -c -; \
    dpkg -i /tmp/repokit-unzip.deb; rm /tmp/repokit-unzip.deb; unzip -v | head -1; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 1800 \
      "https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_3.47.5-stable.tar.xz" -o /tmp/repokit-flutter.tar.xz; \
    printf '%s  %s\n' 2132e990f236f8d22e7c6314b29a191a95b10d7cbcfec9b4e2e303d996652cbb /tmp/repokit-flutter.tar.xz | sha256sum -c -; \
    test ! -e /opt/flutter; \
    tar -xJf /tmp/repokit-flutter.tar.xz -C /opt; \
    rm /tmp/repokit-flutter.tar.xz; \
    git config --system --add safe.directory /opt/flutter; \
    ln -s /opt/flutter/bin/flutter /usr/local/bin/flutter; \
    ln -s /opt/flutter/bin/dart /usr/local/bin/dart; \
    export PUB_CACHE=/tmp/repokit-pub-cache FLUTTER_SUPPRESS_ANALYTICS=true DASH__SUPPRESS_ANALYTICS=true; \
    flutter --version | grep -F 'Flutter 3.47.5 '; \
    cd /tmp; flutter create --project-name repokit_smoke --platforms web repokit_smoke >/dev/null; \
    cd /tmp/repokit_smoke; flutter analyze; flutter test; flutter build web >/dev/null; \
    cd /; rm -rf /tmp/repokit_smoke /tmp/repokit-pub-cache /root/.config/flutter /root/.dart-tool /root/.flutter; \
    chmod -R a+rwX /opt/flutter
RUN install -d -m 1777 /var/cache/repokit
# Flutter and Dart send telemetry by default for each new user; agents run
# unattended, so it stays off without anyone accepting Flutter's notice.
ENV PUB_CACHE=/var/cache/repokit/pub-cache \
    FLUTTER_SUPPRESS_ANALYTICS=true \
    DASH__SUPPRESS_ANALYTICS=true
`

// flutterLinuxInstall adds what Flutter's Linux desktop target builds and
// tests with (clang, ninja, GTK 3 headers) plus a virtual display, for Flutter
// apps that carry a linux/ runner. Debian's packages come from apt pointed
// only at a fixed snapshot.debian.org date, verified by Debian's archive key,
// so a rebuild installs the same versions. A smoke project builds the Linux
// bundle and runs its widget test under Xvfb.
const flutterLinuxInstall = `RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) ;; \
      *) echo 'Flutter publishes Linux SDKs for x86_64 only; the Linux desktop toolchain is not installed' >&2; exit 0 ;; \
    esac; \
    mkdir /tmp/repokit-apt; \
    printf '%s\n' 'Types: deb' \
      'URIs: https://snapshot.debian.org/archive/debian/20261001T000000Z' \
      'Suites: trixie trixie-updates' 'Components: main' \
      'Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg' '' \
      'Types: deb' 'URIs: https://snapshot.debian.org/archive/debian-security/20261001T000000Z' \
      'Suites: trixie-security' 'Components: main' \
      'Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg' > /tmp/repokit-apt/snapshot.sources; \
    apt-get -o Dir::Etc::sourcelist=/dev/null -o Dir::Etc::sourceparts=/tmp/repokit-apt \
      -o Acquire::Check-Valid-Until=false -o Acquire::Retries=3 update; \
    DEBIAN_FRONTEND=noninteractive apt-get -o Dir::Etc::sourcelist=/dev/null -o Dir::Etc::sourceparts=/tmp/repokit-apt \
      -o Acquire::Check-Valid-Until=false -o Acquire::Retries=3 install -y --no-install-recommends \
      clang ninja-build libgtk-3-dev liblzma-dev xvfb xauth; \
    rm -rf /tmp/repokit-apt /var/lib/apt/lists/*; \
    clang --version | head -1; ninja --version; \
    export PUB_CACHE=/tmp/repokit-pub-cache FLUTTER_SUPPRESS_ANALYTICS=true DASH__SUPPRESS_ANALYTICS=true; \
    cd /tmp; flutter create --project-name repokit_linux_smoke --platforms linux repokit_linux_smoke >/dev/null; \
    cd /tmp/repokit_linux_smoke; flutter build linux >/dev/null; xvfb-run -a flutter test; \
    cd /; rm -rf /tmp/repokit_linux_smoke /tmp/repokit-pub-cache /root/.config/flutter /root/.dart-tool /root/.flutter; \
    chmod -R a+rwX /opt/flutter
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
	if req.Rust {
		goSteps += rustInstall
	}
	if req.Flutter {
		goSteps += flutterInstall
		if req.FlutterLinux {
			goSteps += flutterLinuxInstall
		}
	}
	if req.Godot != "" {
		goSteps += godotInstall(req.Godot)
	}
	content := strings.NewReplacer("{{HERMES_IMAGE}}", qualification.FoundationImage, "{{GO_INSTALL}}", goSteps).Replace(string(template))
	browser, err := assets.Assets.ReadFile("repokit-browser-use-requirements.txt")
	if err != nil {
		panic("missing embedded browser-use requirements")
	}
	hermesPackages, err := assets.Assets.ReadFile("repokit-hermes-requirements.txt")
	if err != nil {
		panic("missing embedded Hermes package requirements")
	}
	stopGateways, err := assets.Assets.ReadFile("repokit-stop-gateways")
	if err != nil {
		panic("missing embedded gateway stop hook")
	}
	files := map[string][]byte{"Dockerfile": []byte(content), "repokit-docker-test": helper, "repokit-browser-use-requirements.txt": browser, "repokit-hermes-requirements.txt": hermesPackages, "repokit-stop-gateways": stopGateways, ".dockerignore": []byte("*\n!Dockerfile\n!repokit-docker-test\n!repokit-browser-use-requirements.txt\n!repokit-hermes-requirements.txt\n!repokit-stop-gateways\n")}
	return files
}

// Fingerprint identifies the complete recipe including the isolated-test helper.
// The generation label itself is excluded to avoid a recursive hash.
func Fingerprint(req Requirements) string {
	return hashRecipe(recipeInputs(req))
}

func hashRecipe(files map[string][]byte) string {
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

// RecipeRequirements reports the toolchains a generated recipe installs, which
// determine the selection its Compose was rendered with.
func RecipeRequirements(files map[string][]byte) Requirements {
	return Requirements{
		Go:           bytes.Contains(files["Dockerfile"], []byte("https://go.dev/dl/go")),
		Rust:         bytes.Contains(files["Dockerfile"], []byte("https://static.rust-lang.org/dist/")),
		Flutter:      bytes.Contains(files["Dockerfile"], []byte("https://storage.googleapis.com/flutter_infra_release/")),
		FlutterLinux: bytes.Contains(files["Dockerfile"], []byte("repokit_linux_smoke")),
		Godot:        recipeGodot(files["Dockerfile"]),
	}
}

var godotRecipe = regexp.MustCompile(`(?m)^# RepoKit Godot (\d+\.\d+)$`)

func recipeGodot(dockerfile []byte) string {
	if m := godotRecipe.FindSubmatch(dockerfile); m != nil {
		return string(m[1])
	}
	return ""
}

// Toolchains lists every toolchain selection a generated Compose can carry.
func Toolchains() []Requirements {
	var all []Requirements
	for i := 0; i < 16; i++ {
		req := Requirements{Go: i&1 != 0, Rust: i&2 != 0, Flutter: i&4 != 0, FlutterLinux: i&8 != 0}
		if req.FlutterLinux && !req.Flutter {
			continue
		}
		all = append(all, req)
		for _, minor := range GodotMinors() {
			withGodot := req
			withGodot.Godot = minor
			all = append(all, withGodot)
		}
	}
	return all
}

// SameToolchains reports whether two selections install the same toolchains.
func SameToolchains(a, b Requirements) bool {
	return a.Go == b.Go && a.Rust == b.Rust && a.Flutter == b.Flutter && a.FlutterLinux == b.FlutterLinux && a.Godot == b.Godot
}

var recipeLabel = regexp.MustCompile(`org\.repokit\.development\.recipe=([0-9a-f]{64})`)

// GeneratedRecipe recognizes an unmodified RepoKit-generated recipe of any
// version: its Dockerfile label is the fingerprint of the files themselves.
// This lets upgrades replace old generated recipes without freezing old bytes,
// while any owner edit changes the hash and is preserved as drift.
func GeneratedRecipe(files map[string][]byte) (string, bool) {
	dockerfile := files["Dockerfile"]
	labels := recipeLabel.FindAllSubmatch(dockerfile, -1)
	if len(labels) != 1 {
		return "", false
	}
	inputs := make(map[string][]byte, len(files))
	for name, data := range files {
		inputs[name] = data
	}
	inputs["Dockerfile"] = bytes.Replace(dockerfile, labels[0][1], []byte("{{RECIPE_HASH}}"), 1)
	fingerprint := string(labels[0][1])
	return fingerprint, hashRecipe(inputs) == fingerprint
}

func Recipe(req Requirements) (map[string][]byte, error) {
	files := recipeInputs(req)
	files["Dockerfile"] = []byte(strings.ReplaceAll(string(files["Dockerfile"]), "{{RECIPE_HASH}}", Fingerprint(req)))
	return files, nil
}

func ImageName(container string, req Requirements) string {
	return "repokit/" + container + ":" + Fingerprint(req)[:24]
}

// ReadGeneratedRecipe reads a bounded development-image directory and reports
// whether it is an unmodified generated recipe, returning its fingerprint.
func ReadGeneratedRecipe(dir string) (map[string][]byte, string, bool) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", false
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil || len(entries) == 0 || len(entries) > 32 {
		return nil, "", false
	}
	files := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		info, err := root.Lstat(entry.Name())
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return nil, "", false
		}
		data, err := root.ReadFile(entry.Name())
		if err != nil {
			return nil, "", false
		}
		files[entry.Name()] = data
	}
	fingerprint, ok := GeneratedRecipe(files)
	return files, fingerprint, ok
}

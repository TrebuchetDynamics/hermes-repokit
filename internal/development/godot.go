package development

import (
	"fmt"
	"regexp"
	"strings"
)

// godotRelease is one official Godot stable release, checksum-pinned by the
// SHA-512 that Godot publishes beside it (SHA512-SUMS.txt on the release).
type godotRelease struct{ Version, AMD64SHA512, ARM64SHA512 string }

// godotReleases are the qualified Godot 4 stable releases, oldest first.
// A project declares only its minor version (config/features "4.5"), and
// game repositories commonly pin an exact patch, so every patch of the
// declared minor is installed under its upstream name.
var godotReleases = []godotRelease{
	{"4.4", "3d56a7698bdf7027e41e5e0a930b6b93f89a204eb95b9cd0b8ae25180231994029e211984863a5ebb59759b3e602ae1d042e745ab1b7cbbfeda62652c3936f60", "d321c91363c37093fde025f97063d8d1b52be56cc1822f0b3325a279668670d10cb205d607d8e8fcc5008688a728b358d414aebef868853eae4acd7ff5257eff"},
	{"4.4.1", "ef4e76880a514257175544952c61191106fdef3095b909bafed9fcbeb230c3e5533920a0f3012882dd4bbde83028a67549825794e2d2c3cf76eba7918b71370e", "8af79fe5fbc4cc42f5a7e4a52bd124973e084218f4d299cb5400595c6083ca800a59770c9c289b622dfc164fc3c6f369b362134ea2ac44223361657ea7941b09"},
	{"4.5", "b5bd5d8a4dd3f44de1d123361beabaafa4825dd05b5e20fd2bfa540f32594f81d82e7c4e86fae420f4faee77d9def573b570f8b156e9c84a4dc2d5123d09e852", "36a9f21358f3521832d83e692f0f5364e0486f7ddeb1008a99973774a3fc26953ebf80965e9140ae6a2595b6a568b8e0d5f8efb7cb37609b01ea8f8e16b8b3c4"},
	{"4.5.1", "5bccbed65a94b82c7c319fdb15719ee8113a6e503976cc54e16f1c61fe95f3d74e5e40b8449b5bb89ff7f424574c20af01a4f5ef08b389e4dc338b245185b0b9", "9d55e9cdccaa4b6528d0b39824d51c47987667babffbab7f83b05fba2ece842a59752b84128332421c9d261fe06846b88480f3b0171af255ea99383d3bc96b2c"},
	{"4.5.2", "e3ce6194b6d4d2dcef5e5b5136752c084d9d8b9071c10bc2d2011b947ec7439a257c8d23c541cb30699f16d9edea6dff36e6e84d2bec14a8fe9b4e9aacbe5a8d", "fde4d772a578f0978a10eaf06a460a4ad867778e3edb5ed3be97adae07a7547915208b1582123a9cfbc0757f7f5aff6ffac966d83f996ded3bdcdac404322c91"},
	{"4.6", "0c1bc5e8dca8f892a9a5fd0628b742f399fbd520e5f0051ecac021c0aa4ea5a8f0d237c8ed6b767b2afda7f7a4b32001118f1b04a82e335a5e35b947a1217940", "e36bcc57a16aba669e20db22b0f0fce777301fea799d754bb4f928834854c17289bb29966fbe0f66450638c1d5a4dffd7929e0ad55a97ae37b936105ec95579e"},
	{"4.6.1", "a76fd0fe1d44a2dd6c065b6f7b434ad75f5593c07bda3d3017f8304f2d069acbcf0f39cb5d0976f0434b56e9ea852032ddbcbdb7e0ce1c75a47e1dacb6794bd7", "7301207e346dc2064f3f6b474c199ebbc904b045c0620a98580ee6ff3743e185c3ff24b414c974cb9b4e79494356149c52c0614c9a778444de5d01f42c18160c"},
	{"4.6.2", "b6e4d5a716085e9649905be2afe77f723f97853544fb33392ce3d32594c730a95d9eb4d1042ed51508904c9e1d996bd36b7c7a2bf4f93f5b1885e98d81b792e7", "47f6101e6df4f3022c752d465689fc2dcddf6ca99cf5d0e7c672f72b03a7412e4a2d9c6cbc38302f59b96870982d71ff8f249ee797c760640ba9e0aec9120147"},
	{"4.6.3", "a035258da32b77f966a5376f9fa29c30a6adde826a85ba918e1605bd1fc9823eba7d85f1dd5e748956bd2ba72827c0025ffa11bb82aec91128c407a2e723c99c", "447381de9ccc68aa02f37e279322289f7ddf88ce9b839ed88a97c73e01cdcda46e026897e5d88722e08491f71b3d74f72dfeb22ec7e3add6fd3e9bfbbdad6751"},
	{"4.7", "b639ca9c1ddea39bb3df89bd5283a51ca6047467abe6b25e9436566f2b2082ede633025073989ecf39c7d5d3c2493d80ea13e3af6dd5e261bbf89e462d6d2214", "8ff111327573eb8a91bbf9d1915e467687788d766402b3cdd4bf3da5f1254fc35df18e5e0a406410f3fd8645b13a4b3cac7288ec36f3733b4b0f04dc71d9cd4a"},
	{"4.7.1", "4ccdab7a48eeccbe8819a2fc1f6262f8d72065d98601bcb3743fcbd7ebd39f373758a788ee3293a05ec5b2c48538266c437404312e372225cd2df273945a2de9", "de64efe4d936ac0403769e078a73d961a9c647cab04168c5fb5a7fe33728e200a67324ed99368eeb27964e205e72a61e48efb63b52d5de34d12dd6a95ca0fc45"},
	{"4.7.2", "9aa00f7a605200940bce3027a567b782f49bd8e940dd06ae9e987bd65aee1b1467edd56ed84fcdcbdd44354bf613bdbb4e5d2913e925850368e150c59ed54c65", "dd59918da086bd49bde2f5450b5e567ff8650cbde9abbd7b8f4ca1197ff8c609baa38834666d032deafb47099078d7822279e2a0e06e5665745468f26533e7e2"},
}

// GodotMinors lists the qualified Godot minor versions, oldest first.
func GodotMinors() []string {
	var minors []string
	for _, r := range godotReleases {
		if minor := godotMinor(r.Version); len(minors) == 0 || minors[len(minors)-1] != minor {
			minors = append(minors, minor)
		}
	}
	return minors
}

// GodotVersions lists the qualified releases of one minor, oldest first.
func GodotVersions(minor string) []string {
	var versions []string
	for _, r := range godotReleases {
		if godotMinor(r.Version) == minor {
			versions = append(versions, r.Version)
		}
	}
	return versions
}

func godotMinor(version string) string {
	parts := strings.SplitN(version, ".", 3)
	return parts[0] + "." + parts[1]
}

var godotConfigVersion = regexp.MustCompile(`(?m)^config_version\s*=\s*(\d+)\s*$`)
var godotFeatures = regexp.MustCompile(`(?m)^config/features\s*=\s*PackedStringArray\(([^)]*)\)\s*$`)
var godotFeatureVersion = regexp.MustCompile(`^"(\d+\.\d+)"$`)

// godotRequirement reads the minor version a project.godot declares. It
// returns the minor when it is qualified, otherwise a problem naming rel.
func godotRequirement(rel, data string) (string, string) {
	if m := godotConfigVersion.FindStringSubmatch(data); m == nil || m[1] != "5" {
		return "", rel + " is not a Godot 4 project"
	}
	m := godotFeatures.FindStringSubmatch(data)
	if m == nil {
		return "", rel + " declares no Godot version"
	}
	for _, feature := range strings.Split(m[1], ",") {
		v := godotFeatureVersion.FindStringSubmatch(strings.TrimSpace(feature))
		if v == nil {
			continue
		}
		if len(GodotVersions(v[1])) == 0 {
			return "", rel + " requires Godot " + v[1] + ", which is not qualified"
		}
		return v[1], ""
	}
	return "", rel + " declares no Godot version"
}

// addGodot records one project's Godot minor. A repository gets one Godot
// minor; a project needing another is reported, never silently served.
func (r *Requirements) addGodot(rel, data string) {
	minor, problem := godotRequirement(rel, data)
	switch {
	case problem != "":
		r.Unsupported = append(r.Unsupported, problem)
	case r.Godot == "":
		r.Godot = minor
	case r.Godot != minor:
		r.Unsupported = append(r.Unsupported, fmt.Sprintf("%s requires Godot %s; this repository is provisioned with Godot %s", rel, minor, r.Godot))
	}
}

// godotInstall installs every qualified patch of one minor from Godot's
// official release archives, verified against Godot's published SHA-512,
// under its upstream name (Godot_v4.5.1-stable_linux.x86_64) on PATH, with
// godot naming the newest. The base has no unzip; Python's zipfile extracts.
// A headless import and script run prove the newest engine works here.
func godotInstall(minor string) string {
	versions := GodotVersions(minor)
	var b strings.Builder
	fmt.Fprintf(&b, "# RepoKit Godot %s\n", minor)
	for _, v := range versions {
		var rel godotRelease
		for _, r := range godotReleases {
			if r.Version == v {
				rel = r
			}
		}
		fmt.Fprintf(&b, `RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) arch=x86_64; godot_sha=%s ;; \
      arm64) arch=arm64; godot_sha=%s ;; \
      *) echo 'Unqualified Godot architecture' >&2; exit 1 ;; \
    esac; \
    name="Godot_v%s-stable_linux.${arch}"; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 900 \
      "https://github.com/godotengine/godot/releases/download/%s-stable/${name}.zip" -o /tmp/repokit-godot.zip; \
    printf '%%s  %%s\n' "$godot_sha" /tmp/repokit-godot.zip | sha512sum -c -; \
    install -d /opt/godot; \
    python3 -c 'import sys, zipfile; zipfile.ZipFile(sys.argv[1]).extract(sys.argv[2], "/opt/godot")' /tmp/repokit-godot.zip "$name"; \
    rm /tmp/repokit-godot.zip; \
    chmod 0755 "/opt/godot/$name"; \
    ln -s "/opt/godot/$name" "/usr/local/bin/$name"; \
    case "$("/opt/godot/$name" --version)" in '%s.stable.official.'*) ;; *) echo "unexpected $name version" >&2; exit 1 ;; esac
`, rel.AMD64SHA512, rel.ARM64SHA512, v, v, v)
	}
	newest := versions[len(versions)-1]
	fmt.Fprintf(&b, `RUN set -eu; \
    ln -s "$(ls /opt/godot/Godot_v%s-stable_linux.*)" /usr/local/bin/godot; \
    mkdir -p /tmp/repokit-godot-smoke; cd /tmp/repokit-godot-smoke; \
    printf '%%s\n' 'config_version=5' '[application]' 'config/name="repokit_smoke"' 'config/features=PackedStringArray("%s")' > project.godot; \
    printf '%%s\n' 'extends SceneTree' 'func _init():' '	quit(0 if 2 + 2 == 4 else 1)' > smoke.gd; \
    HOME=/tmp/repokit-godot-home godot --headless --path . --import; \
    HOME=/tmp/repokit-godot-home godot --headless --path . -s smoke.gd; \
    cd /; rm -rf /tmp/repokit-godot-smoke /tmp/repokit-godot-home
`, newest, minor)
	return b.String()
}

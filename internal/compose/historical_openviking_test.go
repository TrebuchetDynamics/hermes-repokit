package compose

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/qualification"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/selinux"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// The golden files are the exact Compose the last OpenViking-era revision
// (0bd3b14^) rendered for this identity. Deployments are recognized for
// upgrade only by exact bytes, so every historical render must reproduce them.
func TestHistoricalOpenVikingRendersAreExact(t *testing.T) {
	id := target.Identity{Name: "atlas", Container: "hermes-atlas", Project: "repokit-0123456789abcdef01234567", Root: "/w/atlas", Compose: "/w/atlas/.hermes/compose.yaml"}
	fp := strings.Repeat("ab", 32)
	golden := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata/historical-openviking", name+".yaml"))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	check := func(name string, got []byte, err error) {
		t.Helper()
		if err != nil || !bytes.Equal(got, golden(name)) {
			t.Errorf("%s no longer reproduces the historical Compose (err=%v)", name, err)
		}
	}
	for _, g := range []bool{false, true} {
		for _, tests := range []bool{false, true} {
			for _, sel := range []selinux.State{selinux.Disabled, selinux.Enforcing} {
				req := development.Requirements{Go: g}
				o := Options{HermesImage: qualification.FoundationImage, HistoricalOpenViking: true, Development: &req, DockerTests: tests, UID: 1000, GID: 1000, SELinux: sel}
				tag := fmt.Sprintf("go%v-tests%v-%s", g, tests, sel)
				// A release's own current render carried that release's recipe
				// fingerprint; today it is reached through the older-recipe
				// renders with the fingerprint read from the deployed recipe.
				b, err := OlderRecipe(id, o, tagFingerprint(t, golden("render-"+tag)))
				check("render-"+tag, b, err)
				b, err = LegacyDevelopment(id, o, tagFingerprint(t, golden("names-"+tag)))
				check("names-"+tag, b, err)
				b, err = LegacyDevelopment(id, o, fp)
				check("legacydev-"+tag, b, err)
				b, err = OlderRecipe(id, o, fp)
				check("older-"+tag, b, err)
			}
		}
	}
	b, err := PreviousNames(id, Options{HermesImage: qualification.FoundationImage, HistoricalOpenViking: true, UID: 1000, GID: 1000})
	check("names-sidecar", b, err)
	b, err = LegacyLayaBuild(target.PreviousNames(id), 1000, 1000)
	check("laya", b, err)
}

func TestCurrentRenderNeverCarriesOpenViking(t *testing.T) {
	id := target.Identity{Name: "atlas", Container: "hermes-atlas", Project: "repokit-0123456789abcdef01234567"}
	req := development.Requirements{}
	for _, o := range []Options{{HermesImage: qualification.FoundationImage, Development: &req, UID: 1000, GID: 1000}, {HermesImage: qualification.FoundationImage, UID: 1000, GID: 1000}} {
		b, err := Render(id, o)
		if err != nil || bytes.Contains(b, []byte("openviking")) || bytes.Contains(b, []byte("OPENVIKING")) {
			t.Fatalf("current render carries OpenViking: %v\n%s", err, b)
		}
	}
}

// tagFingerprint recovers the recipe fingerprint prefix from a golden's image
// tag, padded to the full length the older renders require.
func tagFingerprint(t *testing.T, golden []byte) string {
	t.Helper()
	for _, line := range strings.Split(string(golden), "\n") {
		if image, ok := strings.CutPrefix(strings.TrimSpace(line), "image: "); ok {
			tag := strings.Trim(image, `"`)
			tag = tag[strings.LastIndex(tag, ":")+1:]
			if len(tag) == 24 {
				return tag + strings.Repeat("0", 40)
			}
		}
	}
	t.Fatal("golden has no recipe image tag")
	return ""
}

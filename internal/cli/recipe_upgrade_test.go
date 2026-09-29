package cli

import (
	"bytes"
	"testing"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/compose"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/development"
)

func TestOlderGeneratedImageIsRecreatePendingNotRefused(t *testing.T) {
	a, r := foundationApp(t)
	if c, _, d := invoke(t, a, "install"); c != 0 {
		t.Fatal(d)
	}
	current := developmentRuntimeFixture(r.id)
	o, _ := compose.DevelopmentSelected(r.id)
	image := development.ImageName(r.id.Container, *o.Development)
	for older, wantErr := range map[string]bool{
		"repokit/" + r.id.Container + ":0123456789abcdef01234567": false,
		r.id.Project + "-hermes-dev:0123456789abcdef01234567":     false,
		"foreign/" + r.id.Container + ":0123456789abcdef01234567": true,
	} {
		r.runtime = string(bytes.Replace([]byte(current), []byte(image), []byte(older), 1))
		ready, err := a.nativeRuntimeReady(r.id, r.context)
		if ready || (err != nil) != wantErr {
			t.Fatalf("%s: ready=%v err=%v", older, ready, err)
		}
	}
}

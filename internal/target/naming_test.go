package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicNameDoesNotDuplicateHermesPrefix(t *testing.T) {
	for _, tc := range []struct{ repo, want string }{
		{"hermes-repokit", "hermes-repokit"},
		{"Hermes Repokit", "hermes-repokit"},
		{"my-project", "hermes-my-project"},
		{"hermescope", "hermes-hermescope"},
		{"hermes", "hermes-hermes"},
		{"hermes-" + strings.Repeat("a", 121), "hermes-" + strings.Repeat("a", 121)},
	} {
		t.Run(tc.repo, func(t *testing.T) {
			path := filepath.Join(privateDir(t), tc.repo)
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			id, err := Resolve(path)
			if err != nil {
				t.Fatal(err)
			}
			if id.Container != tc.want || id.Launcher != filepath.Join(path, ".hermes/bin", tc.want) {
				t.Fatalf("unexpected public identity: %+v", id)
			}
		})
	}
}

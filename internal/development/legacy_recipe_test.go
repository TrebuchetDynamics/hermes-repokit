package development

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

func TestLegacyRecipeFrozenFingerprint(t *testing.T) {
	for _, goTool := range []bool{false, true} {
		req := Requirements{Go: goTool}
		files, err := LegacyRecipe(req)
		if err != nil {
			t.Fatal(err)
		}
		fingerprint := LegacyFingerprint(req)
		label := []byte("org.repokit.development.recipe=" + fingerprint)
		if bytes.Count(files["Dockerfile"], label) != 1 {
			t.Fatal("missing exact generation label")
		}
		files["Dockerfile"] = bytes.Replace(files["Dockerfile"], label, []byte("org.repokit.development.recipe={{RECIPE_HASH}}"), 1)
		names := make([]string, 0, len(files))
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)
		hash := sha256.New()
		for _, name := range names {
			fmt.Fprintf(hash, "%s\x00%d\x00", name, len(files[name]))
			hash.Write(files[name])
		}
		if got := hex.EncodeToString(hash.Sum(nil)); got != fingerprint {
			t.Fatalf("frozen recipe changed: got %s want %s", got, fingerprint)
		}
		if Fingerprint(req) == fingerprint {
			t.Fatal("PATH fix did not change recipe fingerprint")
		}
	}
}

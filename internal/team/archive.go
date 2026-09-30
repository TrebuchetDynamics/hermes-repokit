package team

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// archive holds the exact SOUL every earlier RepoKit revision wrote, as
// templates with {{REPOKIT_NAME}} and {{REPOKIT_PROJECT}} in place of the
// repository identity. Each file is named by the SHA-256 of its bytes, so a
// changed file fails the tests: provenance is recognized only by exact match,
// and an edited archive silently turns RepoKit's own output into "owner
// customized". Add files for a retired generation; never edit or remove one.
//
//go:embed souls/archive
var archive embed.FS

const (
	archiveName    = "{{REPOKIT_NAME}}"
	archiveProject = "{{REPOKIT_PROJECT}}"
)

// archivedSouls renders every archived generation of one role for a
// repository. One pass substitutes both fields, so an identity containing a
// placeholder cannot expand into the other.
func archivedSouls(id target.Identity, role string) []string {
	entries, err := fs.ReadDir(archive, path.Join("souls/archive", role))
	if err != nil {
		return nil
	}
	fill := strings.NewReplacer(archiveName, id.Name, archiveProject, id.Project)
	out := []string{}
	for _, entry := range entries {
		data, err := archive.ReadFile(path.Join("souls/archive", role, entry.Name()))
		if err == nil {
			out = append(out, fill.Replace(string(data)))
		}
	}
	sort.Strings(out)
	return out
}

// archiveDigestsMatch reports whether every archive file still has the bytes
// its name records.
func archiveDigestsMatch() (bad []string) {
	fs.WalkDir(archive, "souls/archive", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := archive.ReadFile(p)
		sum := sha256.Sum256(data)
		if strings.TrimSuffix(path.Base(p), ".md") != hex.EncodeToString(sum[:]) {
			bad = append(bad, p)
		}
		return nil
	})
	return bad
}

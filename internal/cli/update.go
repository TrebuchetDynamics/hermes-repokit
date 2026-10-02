package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

// Version is the RepoKit release this binary was built from. install.sh
// stamps it (a release tag, a branch, or "checkout"); a plain go build says
// "dev".
var Version = "dev"

const (
	latestReleaseAPI = "https://api.github.com/repos/TrebuchetDynamics/hermes-repokit/releases/latest"
	installerURL     = "https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/%s/install.sh"
)

var releaseTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// update replaces this RepoKit binary with the latest release (or main) by
// running the official installer pinned to it. Deployments are untouched until
// the owner runs install in each repository.
func (a App) update(toMain bool, stdout, stderr io.Writer) int {
	u := newUI(stdout, stderr)
	fetch := a.Fetch
	if fetch == nil {
		fetch = httpGet
	}
	ref := "main"
	if !toMain {
		body, err := fetch(latestReleaseAPI)
		var release struct {
			Tag string `json:"tag_name"`
		}
		if err != nil || json.Unmarshal(body, &release) != nil || !releaseTag.MatchString(release.Tag) {
			u.fail("cannot find the latest RepoKit release; check your network, or run %s update --main", self())
			return 1
		}
		ref = release.Tag
		if Version == ref {
			u.ok("RepoKit", ref+" is the latest release; nothing to update")
			return 0
		}
	}
	u.working("RepoKit", "updating "+Version+" → "+ref)
	script, err := fetch(fmt.Sprintf(installerURL, ref))
	if err != nil || !bytes.HasPrefix(script, []byte("#!/bin/sh")) {
		u.fail("cannot download the RepoKit installer for %s", ref)
		return 1
	}
	run := a.RunInstaller
	if run == nil {
		run = runInstaller
	}
	if err := run(script, ref, stdout, stderr); err != nil {
		u.fail("the installer failed; RepoKit %s is unchanged", Version)
		return 1
	}
	u.ok("RepoKit", "updated to "+ref)
	u.note("run " + self() + " install in each repository to bring its deployment up to date")
	return 0
}

// runInstaller pipes the installer to sh exactly as `curl … | sh` does, so it
// builds from the downloaded source of ref: $0 must be plain "sh", not a
// readable file, or the installer takes itself for a checkout. An empty
// directory keeps a stray file named sh from doing the same.
func runInstaller(script []byte, ref string, stdout, stderr io.Writer) error {
	dir, err := os.MkdirTemp("", "repokit-update.")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	cmd := exec.Command("/bin/sh", "-s")
	cmd.Args[0] = "sh"
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(script)
	cmd.Env = append(os.Environ(), "REPOKIT_REF="+ref)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

func httpGet(url string) ([]byte, error) {
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

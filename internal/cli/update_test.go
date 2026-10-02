package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func updateApp(t *testing.T, latest string, fail bool) (App, *[]string) {
	t.Helper()
	var ran []string
	a := App{Directory: t.TempDir()}
	a.Fetch = func(url string) ([]byte, error) {
		switch {
		case fail:
			return nil, errors.New("offline")
		case url == latestReleaseAPI:
			return []byte(`{"tag_name":"` + latest + `"}`), nil
		case strings.HasSuffix(url, "/install.sh"):
			return []byte("#!/bin/sh\necho installer for " + url + "\n"), nil
		}
		return nil, errors.New("unexpected " + url)
	}
	a.RunInstaller = func(script []byte, ref string, stdout, _ io.Writer) error {
		ran = append(ran, ref+" "+strings.TrimSpace(strings.SplitN(string(script), "\n", 3)[1]))
		return nil
	}
	return a, &ran
}

func runUpdate(a App, args ...string) (int, string) {
	var out, errOut bytes.Buffer
	code := a.Run(args, &out, &errOut)
	return code, out.String() + errOut.String()
}

// update replaces the binary with the latest release through the installer
// pinned to that tag, or does nothing when already current; --main follows
// main; it works outside any repository.
func TestUpdateInstallsTheLatestReleaseOrMain(t *testing.T) {
	defer func(v string) { Version = v }(Version)
	Version = "v0.2.5"
	a, ran := updateApp(t, "v0.2.6", false)
	if code, out := runUpdate(a, "update"); code != 0 || len(*ran) != 1 || (*ran)[0] != "v0.2.6 echo installer for https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.2.6/install.sh" || !strings.Contains(out, "v0.2.5 → v0.2.6") || !strings.Contains(out, "install in each repository") {
		t.Fatalf("update: code=%d ran=%v\n%s", code, *ran, out)
	}
	Version = "v0.2.6"
	a, ran = updateApp(t, "v0.2.6", false)
	if code, out := runUpdate(a, "update"); code != 0 || len(*ran) != 0 || !strings.Contains(out, "nothing to update") {
		t.Fatalf("current: code=%d ran=%v\n%s", code, *ran, out)
	}
	a, ran = updateApp(t, "v0.2.6", false)
	if code, _ := runUpdate(a, "update", "--main"); code != 0 || len(*ran) != 1 || !strings.HasPrefix((*ran)[0], "main ") {
		t.Fatalf("--main: code=%d ran=%v", code, *ran)
	}
	a, ran = updateApp(t, "", true)
	if code, out := runUpdate(a, "update"); code == 0 || len(*ran) != 0 || !strings.Contains(out, "cannot find the latest RepoKit release") {
		t.Fatalf("offline: code=%d\n%s", code, out)
	}
	a, _ = updateApp(t, "not-a-tag", false)
	if code, _ := runUpdate(a, "update"); code == 0 {
		t.Fatal("a malformed release tag was trusted")
	}
	if code, out := runUpdate(a, "version"); code != 0 || out != "repokit v0.2.6\n" {
		t.Fatalf("version: %d %q", code, out)
	}
}

// The installer sees itself piped (as from curl), not run from a checkout.
func TestRunInstallerLooksPiped(t *testing.T) {
	var out bytes.Buffer
	script := []byte("#!/bin/sh\nif [ -f \"$0\" ]; then echo checkout; else echo piped \"$REPOKIT_REF\"; fi\n")
	if err := runInstaller(script, "v9.9.9", &out, &out); err != nil || out.String() != "piped v9.9.9\n" {
		t.Fatalf("installer saw %q (%v)", out.String(), err)
	}
}

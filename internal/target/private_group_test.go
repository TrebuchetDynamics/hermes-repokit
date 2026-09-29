package target

import (
	"os"
	"strings"
	"testing"
)

const passwdFixture = "root:x:0:0:root:/root:/bin/bash\nxel:x:1000:1000:Xel:/home/xel:/bin/bash\nbob:x:1001:1001:Bob:/home/bob:/bin/bash\n"

func TestPrivateGroupRecognizesOnlyTheOwnersPersonalGroup(t *testing.T) {
	for _, tc := range []struct {
		name, groups, passwd string
		gid                  uint32
		want                 bool
	}{
		{"user private group", "root:x:0:\nxel:x:1000:\nbob:x:1001:\n", passwdFixture, 1000, true},
		{"owner listed as its own member", "xel:x:1000:xel\n", passwdFixture, 1000, true},
		{"another member", "xel:x:1000:bob\n", passwdFixture, 1000, false},
		{"another account's primary group", "xel:x:1000:\n", passwdFixture + "eve:x:1002:1000::/home/eve:/bin/sh\n", 1000, false},
		{"shared group not named after the user", "users:x:1000:\n", passwdFixture, 1000, false},
		{"not the user's primary group", "docker:x:123:xel\n", passwdFixture, 123, false},
		{"group only in a directory service", "root:x:0:\n", passwdFixture, 1000, false},
		{"duplicate gid entries", "xel:x:1000:\nstaff:x:1000:\n", passwdFixture, 1000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := privateGroupFrom([]byte(tc.groups), []byte(tc.passwd), "xel", "1000", "1000", tc.gid); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestInspectAcceptsGroupWritableRootOnlyForPrivateGroup(t *testing.T) {
	for _, tc := range []struct {
		mode    os.FileMode
		private bool
		want    string
	}{
		{0755, false, ""},
		{0775, true, ""},
		{0775, false, "group-writable by a group that is not your private group; run chmod g-w"},
		{0777, true, "world-writable; run chmod o-w"},
		{0757, true, "world-writable; run chmod o-w"},
	} {
		saved := PrivateGroup
		PrivateGroup = func(uint32) bool { return tc.private }
		dir := t.TempDir()
		if err := os.Chmod(dir, tc.mode); err != nil {
			t.Fatal(err)
		}
		id, err := Resolve(dir)
		if err != nil {
			t.Fatal(err)
		}
		issues := strings.Join(Inspect(id, ""), "; ")
		PrivateGroup = saved
		if tc.want == "" && strings.Contains(issues, "writable") {
			t.Fatalf("%o private=%v refused: %s", tc.mode, tc.private, issues)
		}
		if tc.want != "" && !strings.Contains(issues, tc.want+" "+dir) {
			t.Fatalf("%o private=%v: want %q with the path, got %s", tc.mode, tc.private, tc.want, issues)
		}
	}
}

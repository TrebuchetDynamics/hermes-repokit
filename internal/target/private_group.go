package target

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

// PrivateGroup reports whether gid is the current user's personal group: the
// user's primary group, named after the user, with no other members in
// /etc/group and no other account using it as a primary group in /etc/passwd.
// Under this convention (Debian, Ubuntu, Fedora and others) umask 0002 makes
// fresh clones group-writable without granting anyone else access. Anything
// that cannot be verified from the local files (for example directory-service
// groups) is not private.
var PrivateGroup = func(gid uint32) bool {
	current, err := user.Current()
	if err != nil {
		return false
	}
	groups, err1 := readBounded("/etc/group")
	passwd, err2 := readBounded("/etc/passwd")
	if err1 != nil || err2 != nil {
		return false
	}
	return privateGroupFrom(groups, passwd, current.Username, current.Uid, current.Gid, gid)
}

func readBounded(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	const limit = 4 << 20
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%s too large", path)
	}
	return data, nil
}

func privateGroupFrom(groups, passwd []byte, username, uid, primaryGID string, gid uint32) bool {
	want := strconv.FormatUint(uint64(gid), 10)
	if primaryGID != want {
		return false
	}
	found := false
	for _, fields := range colonRecords(groups, 4) {
		if fields[2] != want {
			continue
		}
		if found || fields[0] != username {
			return false // duplicate gid or not the user's own group
		}
		found = true
		for _, member := range strings.Split(fields[3], ",") {
			if member = strings.TrimSpace(member); member != "" && member != username {
				return false
			}
		}
	}
	if !found {
		return false
	}
	for _, fields := range colonRecords(passwd, 7) {
		if fields[3] == want && fields[2] != uid {
			return false // another account's primary group
		}
	}
	return true
}

func colonRecords(data []byte, n int) [][]string {
	var out [][]string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) >= n {
			out = append(out, fields)
		}
	}
	return out
}

// WritableByOthers explains why a directory owned by the current user is
// writable by someone else, or returns "" when only the owner can write it.
func WritableByOthers(info fs.FileInfo) string {
	perm := info.Mode().Perm()
	if perm&0002 != 0 {
		return "world-writable; run chmod o-w"
	}
	if perm&0020 == 0 {
		return ""
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if ok && PrivateGroup(st.Gid) {
		return ""
	}
	return "group-writable by a group that is not your private group; run chmod g-w"
}

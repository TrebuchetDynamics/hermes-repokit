package locking

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockIsExclusiveAndStaleFileDoesNotBlock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lock")
	os.WriteFile(p, []byte("999999 stale"), 0600)
	one, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	if two, e := Acquire(p); e == nil {
		two.Close()
		t.Fatal("concurrent lock acquired")
	}
	one.Close()
	two, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	two.Close()
}
func TestLockRefusesSymlink(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lock")
	os.Symlink("missing", p)
	if lock, e := Acquire(p); e == nil {
		lock.Close()
		t.Fatal("followed dangling symlink")
	}
}

package locking

import (
	"io"
	"os"
	"os/exec"
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

func TestChildRetainingDescriptorKeepsLock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lock")
	lock, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	read, write, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer read.Close()
	defer write.Close()
	readyR, readyW, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer readyR.Close()
	defer readyW.Close()
	cmd := exec.Command("sh", "-c", "printf ready >&4; read -r line")
	cmd.Stdin = read
	cmd.ExtraFiles = []*os.File{lock, readyW}
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer cmd.Process.Kill()
	buf := make([]byte, 5)
	if _, e = io.ReadFull(readyR, buf); e != nil {
		t.Fatal(e)
	}
	lock.Close()
	if second, e := Acquire(p); e == nil {
		second.Close()
		t.Fatal("released child-held lock")
	}
	write.Close()
	_ = cmd.Wait()
	second, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	second.Close()
}

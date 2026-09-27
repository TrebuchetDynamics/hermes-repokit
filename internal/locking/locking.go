// Package locking uses kernel locks, never PID guesses or stale-file removal.
package locking

import (
	"fmt"
	"os"
	"syscall"
)

func Acquire(path string) (*os.File, error) {
	fd, e := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	info, e := f.Stat()
	if e != nil {
		f.Close()
		return nil, e
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || st.Nlink != 1 {
		f.Close()
		return nil, fmt.Errorf("unsafe installer lock")
	}
	if e = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("installer is already running: %w", e)
	}
	// Close releases only this descriptor. A deliberately inherited descriptor
	// retains the same kernel lock until the last holder closes it.
	return f, nil
}

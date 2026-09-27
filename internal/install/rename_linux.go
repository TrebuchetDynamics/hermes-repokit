package install

import (
	"os"
	"syscall"
	"unsafe"
)

// Both paths are relative to an opened directory; RENAME_NOREPLACE makes
// concurrent owner-created files/directories a refusal, never an overwrite.
func publishDirectory(root *os.Root, stage string) error {
	dir, e := root.Open(".")
	if e != nil {
		return e
	}
	defer dir.Close()
	old, e := syscall.BytePtrFromString(stage)
	if e != nil {
		return e
	}
	next, e := syscall.BytePtrFromString(".hermes")
	if e != nil {
		return e
	}
	_, _, errno := syscall.Syscall6(renameat2, dir.Fd(), uintptr(unsafe.Pointer(old)), dir.Fd(), uintptr(unsafe.Pointer(next)), 1, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

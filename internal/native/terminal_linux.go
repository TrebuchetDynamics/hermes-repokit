package native

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

// InteractiveInput excludes pipes and /dev/null: Hermes setup returns zero
// after guidance on nonterminals, which cannot authorize post-setup cloning.
func InteractiveInput(input io.Reader) bool {
	f, ok := input.(*os.File)
	if !ok {
		return false
	}
	var state syscall.Termios
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&state)))
	return err == 0
}

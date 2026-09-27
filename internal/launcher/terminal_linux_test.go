package launcher

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func terminal(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	master, e := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if e != nil {
		t.Fatal(e)
	}
	var unlock int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock)))
	if errno != 0 {
		master.Close()
		t.Fatal(errno)
	}
	var number uint32
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&number)))
	if errno != 0 {
		master.Close()
		t.Fatal(errno)
	}
	slave, e := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|syscall.O_NOCTTY, 0)
	if e != nil {
		master.Close()
		t.Fatal(e)
	}
	t.Cleanup(func() { slave.Close(); master.Close() })
	return master, slave
}
func TestTTYAllocatedOnlyForTerminalInputAndOutput(t *testing.T) {
	for _, tt := range []struct{ in, out bool }{{true, true}, {false, true}, {true, false}, {false, false}} {
		t.Run(fmt.Sprint(tt), func(t *testing.T) {
			_, tty := terminal(t)
			file, bin, record := fixture(t, `printf '%s\000' "$@" > "$RECORD"`)
			cmd := exec.Command(file, "setup")
			cmd.Env = append(os.Environ(), "PATH="+bin, "RECORD="+record)
			if tt.in {
				cmd.Stdin = tty
			} else {
				cmd.Stdin = strings.NewReader("")
			}
			if tt.out {
				cmd.Stdout = tty
			} else {
				cmd.Stdout = &bytes.Buffer{}
			}
			if e := cmd.Run(); e != nil {
				t.Fatal(e)
			}
			raw, _ := os.ReadFile(record)
			noTTY := bytes.Contains(raw, []byte("\x00-T\x00"))
			if noTTY == (tt.in && tt.out) {
				t.Fatalf("wrong terminal mode: %q", raw)
			}
		})
	}
}

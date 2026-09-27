// Package native delegates terminal operations to the standalone native launcher.
package native

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func Setup(launcher, compose string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := exec.Command(launcher, "setup")
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(stderr, "cannot execute native launcher:", err)
		return 1
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case s := <-signals:
				_ = cmd.Process.Signal(s)
			case <-done:
				return
			}
		}
	}()
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	fmt.Fprintf(stderr, "\nIf Hermes is stopped, start it with: docker compose --env-file /dev/null -f '%s' up -d hermes\n", strings.ReplaceAll(compose, "'", "'\"'\"'"))
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if s, ok := exit.Sys().(syscall.WaitStatus); ok && s.Signaled() {
			return 128 + int(s.Signal())
		}
		return exit.ExitCode()
	}
	return 1
}

// Package native delegates terminal operations to the standalone native launcher.
package native

import (
	"errors"
	"fmt"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/launcher"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func Setup(launcherPath, compose, dockerContext string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := exec.Command(launcherPath, "-p", "default", "setup")
	code := runTerminal(cmd, stdin, stdout, stderr)
	if code != 0 {
		if dockerContext != "" {
			fmt.Fprintf(stderr, "\nIf Hermes is stopped, start it with: %s\n", launcher.StartCommand(compose, dockerContext))
		} else {
			fmt.Fprintln(stderr, "Native launcher was modified; inspect its Docker context before starting Compose.")
		}
	}
	return code
}

func runTerminal(cmd *exec.Cmd, stdin io.Reader, stdout, stderr io.Writer) int {
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
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if s, ok := exit.Sys().(syscall.WaitStatus); ok && s.Signaled() {
			return 128 + int(s.Signal())
		}
		return exit.ExitCode()
	}
	return 1
}

// Package process provides bounded, shell-free inspection subprocesses.
package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Result struct {
	Output    string
	Err       error
	Truncated bool
}
type Runner struct {
	Timeout time.Duration
	Limit   int
}
type bounded struct {
	mu        sync.Mutex
	data      []byte
	limit     int
	truncated bool
}

func (b *bounded) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	remaining := b.limit - len(b.data)
	if len(p) > remaining {
		b.truncated = true
		p = p[:remaining]
	}
	b.data = append(b.data, p...)
	return n, nil
}
func CleanEnvironment(env []string) []string {
	out := make([]string, 0, len(env))
	for _, v := range env {
		if !strings.HasPrefix(v, "COMPOSE_") {
			out = append(out, v)
		}
	}
	return out
}
func (r Runner) Run(parent context.Context, program string, args ...string) Result {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	limit := r.Limit
	if limit <= 0 {
		limit = 64 * 1024
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Env = CleanEnvironment(os.Environ())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 200 * time.Millisecond
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		e := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(e, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return e
	}
	out := &bounded{limit: limit}
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return Result{string(out.data), err, out.truncated}
}

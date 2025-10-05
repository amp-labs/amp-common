// Package cmd provides a fluent API for building and executing shell commands.
// It wraps Go's exec.Cmd with a chainable interface and enhanced error handling.
package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Cmd wraps an exec.Cmd with a fluent interface for configuration.
// It also tracks observers that should be called after command execution.
type Cmd struct {
	cmd      *exec.Cmd
	finished []func()
}

// New creates a new Cmd that will execute the given command with args.
// The command inherits the current process's environment variables.
// The provided context can be used to cancel the command.
func New(ctx context.Context, cmd string, args ...string) *Cmd {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = os.Environ()

	return &Cmd{
		cmd: c,
	}
}

// SetDir sets the working directory for the command.
// If dir is empty, the command runs in the current process's directory.
func (c *Cmd) SetDir(dir string) *Cmd {
	c.cmd.Dir = dir

	return c
}

// SetStdin sets the command's standard input source.
func (c *Cmd) SetStdin(in io.Reader) *Cmd {
	c.cmd.Stdin = in

	return c
}

// SetStdinBytes sets the command's standard input from a byte slice.
func (c *Cmd) SetStdinBytes(input []byte) *Cmd {
	c.cmd.Stdin = bytes.NewReader(input)

	return c
}

// SetStdout sets the command's standard output destination.
func (c *Cmd) SetStdout(out io.Writer) *Cmd {
	c.cmd.Stdout = out

	return c
}

// SetStdoutObserver captures stdout to a buffer and calls the observer function
// with the buffered output after the command finishes.
func (c *Cmd) SetStdoutObserver(f func([]byte)) *Cmd {
	var buf bytes.Buffer

	c.cmd.Stdout = &buf
	c.finished = append(c.finished, func() {
		f(buf.Bytes())
	})

	return c
}

// SetStderr sets the command's standard error destination.
func (c *Cmd) SetStderr(out io.Writer) *Cmd {
	c.cmd.Stderr = out

	return c
}

// SetStderrObserver captures stderr to a buffer and calls the observer function
// with the buffered output after the command finishes.
func (c *Cmd) SetStderrObserver(f func([]byte)) *Cmd {
	var buf bytes.Buffer

	c.cmd.Stderr = &buf
	c.finished = append(c.finished, func() {
		f(buf.Bytes())
	})

	return c
}

// SetStdoutAndStderr sets both stdout and stderr to the same destination.
func (c *Cmd) SetStdoutAndStderr(out io.Writer) *Cmd {
	c.cmd.Stdout = out
	c.cmd.Stderr = out

	return c
}

// SetStdoutAndStderrObserver captures both stdout and stderr to a shared buffer
// and calls the observer function with the combined output after the command finishes.
func (c *Cmd) SetStdoutAndStderrObserver(f func([]byte)) *Cmd {
	var buf bytes.Buffer

	c.cmd.Stdout = &buf
	c.cmd.Stderr = &buf

	c.finished = append(c.finished, func() {
		f(buf.Bytes())
	})

	return c
}

// SetForeground configures the command to run in the foreground process group.
// This is typically used for interactive commands that need terminal control.
func (c *Cmd) SetForeground() *Cmd {
	modSysProcAttr(c.cmd, func(sa *syscall.SysProcAttr) {
		sa.Foreground = true
	})

	return c
}

// SetSameProcessGroup places the command in the same process group as the parent.
// This is useful for ensuring signals are delivered to both parent and child processes.
func (c *Cmd) SetSameProcessGroup() *Cmd {
	modSysProcAttr(c.cmd, func(sa *syscall.SysProcAttr) {
		sa.Setpgid = true
		sa.Pgid = os.Getpid()
	})

	return c
}

// ReplaceEnv replaces the command's entire environment with the provided map.
// This clears any previously inherited or set environment variables.
func (c *Cmd) ReplaceEnv(env map[string]string) *Cmd {
	c.cmd.Env = nil
	for k, v := range env {
		c.cmd.Env = append(c.cmd.Env, k+"="+v)
	}

	return c
}

// AppendEnv adds a single environment variable to the command's environment.
// This does not affect variables that were previously set.
func (c *Cmd) AppendEnv(key, value string) *Cmd {
	c.cmd.Env = append(c.cmd.Env, key+"="+value)

	return c
}

// exitStatus is an interface for extracting exit codes from errors.
type exitStatus interface {
	ExitStatus() int
}

// status extracts the exit code from an error returned by exec.Cmd.Run().
// It returns (0, nil) for successful commands.
// For failed commands, it returns the exit code and optionally wraps stderr in the error.
func status(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	if e, ok := err.(exitStatus); ok {
		return e.ExitStatus(), nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if len(exitErr.Stderr) == 0 {
			return exitErr.ExitCode(), nil
		} else {
			code := exitErr.ExitCode()
			if code == 0 {
				return code, nil
			}

			// Dynamic error wraps command stderr for debugging
			return code, fmt.Errorf("command failed: %s", string(exitErr.Stderr)) //nolint:err113
		}
	}

	return 1, err
}

// Run executes the command and waits for it to complete.
// It returns the exit code and any error that occurred.
// All registered observer functions are called after the command finishes.
// The command and environment are logged at debug level.
func (c *Cmd) Run() (int, error) {
	cmd := strings.Join(c.cmd.Args, " ")
	env := strings.Join(c.cmd.Env, " ")
	slog.Debug("run cmd", "cmd", cmd, "env", env)

	err := c.cmd.Run()

	st, err := status(err)

	for _, f := range c.finished {
		f()
	}

	return st, err
}

// modSysProcAttr safely modifies the SysProcAttr of a command.
// It creates a new SysProcAttr if one doesn't exist, then applies the modifier function.
func modSysProcAttr(cmd *exec.Cmd, f func(sa *syscall.SysProcAttr)) {
	if cmd == nil {
		return
	}

	if cmd.SysProcAttr == nil {
		a := &syscall.SysProcAttr{}
		f(a)
		cmd.SysProcAttr = a
	} else {
		f(cmd.SysProcAttr)
	}
}

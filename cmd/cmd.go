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

type Cmd struct {
	cmd      *exec.Cmd
	finished []func()
}

func New(ctx context.Context, cmd string, args ...string) *Cmd {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = os.Environ()

	return &Cmd{
		cmd: c,
	}
}

func (c *Cmd) SetDir(dir string) *Cmd {
	c.cmd.Dir = dir

	return c
}

func (c *Cmd) SetStdin(in io.Reader) *Cmd {
	c.cmd.Stdin = in

	return c
}

func (c *Cmd) SetStdinBytes(input []byte) *Cmd {
	c.cmd.Stdin = bytes.NewReader(input)

	return c
}

func (c *Cmd) SetStdout(out io.Writer) *Cmd {
	c.cmd.Stdout = out

	return c
}

func (c *Cmd) SetStdoutObserver(f func([]byte)) *Cmd {
	var buf bytes.Buffer

	c.cmd.Stdout = &buf
	c.finished = append(c.finished, func() {
		f(buf.Bytes())
	})

	return c
}

func (c *Cmd) SetStderr(out io.Writer) *Cmd {
	c.cmd.Stderr = out

	return c
}

func (c *Cmd) SetStderrObserver(f func([]byte)) *Cmd {
	var buf bytes.Buffer

	c.cmd.Stderr = &buf
	c.finished = append(c.finished, func() {
		f(buf.Bytes())
	})

	return c
}

func (c *Cmd) SetStdoutAndStderr(out io.Writer) *Cmd {
	c.cmd.Stdout = out
	c.cmd.Stderr = out

	return c
}

func (c *Cmd) SetStdoutAndStderrObserver(f func([]byte)) *Cmd {
	var buf bytes.Buffer

	c.cmd.Stdout = &buf
	c.cmd.Stderr = &buf

	c.finished = append(c.finished, func() {
		f(buf.Bytes())
	})

	return c
}

func (c *Cmd) SetForeground() *Cmd {
	modSysProcAttr(c.cmd, func(sa *syscall.SysProcAttr) {
		sa.Foreground = true
	})

	return c
}

func (c *Cmd) SetSameProcessGroup() *Cmd {
	modSysProcAttr(c.cmd, func(sa *syscall.SysProcAttr) {
		sa.Setpgid = true
		sa.Pgid = os.Getpid()
	})

	return c
}

func (c *Cmd) ReplaceEnv(env map[string]string) *Cmd {
	c.cmd.Env = nil
	for k, v := range env {
		c.cmd.Env = append(c.cmd.Env, k+"="+v)
	}

	return c
}

func (c *Cmd) AppendEnv(key, value string) *Cmd {
	c.cmd.Env = append(c.cmd.Env, key+"="+value)

	return c
}

type exitStatus interface {
	ExitStatus() int
}

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

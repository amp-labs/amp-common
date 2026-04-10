//go:build linux || darwin

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

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

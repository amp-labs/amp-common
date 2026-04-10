//go:build windows

package cmd

func (c *Cmd) SetForeground() *Cmd {
	return c
}

func (c *Cmd) SetSameProcessGroup() *Cmd {
	return c
}

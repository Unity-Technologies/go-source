package source

import (
	"fmt"
	"strings"
)

// Cmd represents a source rcon command.
type Cmd struct {
	cmd  string
	args []interface{}
}

// NewCmd creates a new Cmd.
func NewCmd(cmd string) *Cmd {
	return &Cmd{cmd: cmd}
}

// WithArgs sets the command Args.
func (c *Cmd) WithArgs(args ...interface{}) *Cmd {
	c.args = args
	return c
}

func (c *Cmd) String() string {
	args := append([]interface{}{c.cmd}, c.args...)
	// We use fmt.Sprintln + fmt.TrimSuffix as fmt.Sprintln guarantees all args
	// are space separated, which is what we want, where as fmt.Sprint doesn't.
	return strings.TrimSuffix(fmt.Sprintln(args...), "\n")
}

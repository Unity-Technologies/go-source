package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd(t *testing.T) {
	tests := []struct {
		name   string
		cmd    *Cmd
		expect string
	}{
		{"status", NewCmd("status"), "status"},
		{"echo", NewCmd("echo").WithArgs("test me"), "echo test me"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, tc.cmd.String())
		})
	}
}

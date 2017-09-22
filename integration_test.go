// +build integration

package source

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	serverAddress  = flag.String("server-address", "127.0.0.1", "sets the servers address for integration tests")
	serverPassword = flag.String("server-password", "", "sets the rcon password for integration tests")
	serverFlavour  = flag.String("server-flavour", "source", "configures the flavour <source|minecraft|starbound> of integration tests")
)

type subtest struct {
	name string
	f    func(t *testing.T)
}

func sourceTests(c *Client) []subtest {
	return []subtest{
		{"source-echo", func(t *testing.T) {
			arg := "my test"
			r, err := c.ExecCmd(NewCmd("echo").WithArgs(arg))
			assert.NoError(t, err)
			assert.Contains(t, r, arg)
		}},
		{"source-status", func(t *testing.T) {
			r, err := c.Exec("status")
			assert.NoError(t, err)
			assert.NotEmpty(t, r)
		}},
	}
}

func minecraftTests(c *Client) []subtest {
	return []subtest{
		{"minecraft-help", func(t *testing.T) {
			r, err := c.ExecCmd(NewCmd("/help"))
			assert.NoError(t, err)
			assert.Contains(t, r, "Showing help")
		}},
		{"minecraft-say", func(t *testing.T) {
			r, err := c.ExecCmd(NewCmd("/say").WithArgs("go-source test"))
			assert.NoError(t, err)
			assert.Empty(t, r)
		}},
	}
}

func starboundTests(c *Client) []subtest {
	return []subtest{
		{"starbound-help", func(t *testing.T) {
			r, err := c.ExecCmd(NewCmd("help"))
			assert.NoError(t, err)
			assert.Contains(t, r, "Basic commands")
		}},
		{"starbound-echo", func(t *testing.T) {
			msg := "go-source test"
			r, err := c.ExecCmd(NewCmd("echo").WithArgs(msg))
			assert.NoError(t, err)
			assert.Equal(t, r, msg)
		}},
	}
}

func TestIntegration(t *testing.T) {
	opts := []func(*Client) error{Timeout(time.Second * 10)}
	if *serverPassword != "" {
		opts = append(opts, Password(*serverPassword))
	}

	switch *serverFlavour {
	case "source":
	case "minecraft", "starbound":
		opts = append(opts, DisableMultiPacket())
	default:
		t.Fatal("unsupported flavour", *serverFlavour)
	}

	c, err := NewClient(*serverAddress, opts...)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		assert.NoError(t, c.Close())
	}()

	var tests []subtest
	switch *serverFlavour {
	case "source":
		tests = sourceTests(c)
	case "minecraft":
		tests = minecraftTests(c)
	case "starbound":
		tests = starboundTests(c)
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.f)
	}
}

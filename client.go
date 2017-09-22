// Package source provides a client which can talk to game servers which
// support the source RCON protocol:
// https://developer.valvesoftware.com/wiki/Source_RCON_Protocol
package source

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	// DefaultPort is the default source RCON port.
	DefaultPort = 27015

	// maxPkt is the maximum size of a response packet.
	maxPkt = 4096
)

var (
	// DefaultTimeout is the default read / write / dial timeout for Clients.
	DefaultTimeout = time.Second * 10

	// responseBody is the expected response body for the second response reply.
	responseBody = []byte{0x00, 0x01, 0x00, 0x00}
)

// Client is a source rcon client.
type Client struct {
	conn    net.Conn
	addr    string
	pwd     string
	timeout time.Duration
	reader  *bufio.Reader
	reqID   int32
	read    func(expectedID int32) (string, error)
	write   func(pktType int32, body string) error
}

// Timeout sets read / write / dial timeout for a source rcon Client.
func Timeout(timeout time.Duration) func(*Client) error {
	return func(c *Client) error {
		c.timeout = timeout
		return nil
	}
}

// Password sets authentication password for a source rcon Client.
func Password(pwd string) func(*Client) error {
	return func(c *Client) error {
		c.pwd = pwd
		return nil
	}
}

// DisableMultiPacket disables multi-packet support, which not all servers support.
// This is required for Minecraft and Starbound servers.
func DisableMultiPacket() func(*Client) error {
	return func(c *Client) error {
		c.read = c.readSingle
		c.write = c.writePkt
		return nil
	}
}

// NewClient returns a new source rcon client connected to addr.
// If addr doesn't include a port the DefaultPort will be used.
func NewClient(addr string, options ...func(c *Client) error) (c *Client, err error) {
	c = &Client{timeout: DefaultTimeout, addr: addr}
	c.read = c.readMulti
	c.write = c.writeMulti
	for _, f := range options {
		if f == nil {
			return nil, ErrNilOption
		}
		if err = f(c); err != nil {
			return nil, err
		}
	}

	if !strings.Contains(c.addr, ":") {
		c.addr = fmt.Sprintf("%v:%v", c.addr, DefaultPort)
	}

	if c.conn, err = net.DialTimeout("tcp", c.addr, c.timeout); err != nil {
		return nil, err
	}

	c.reader = bufio.NewReaderSize(c.conn, maxPkt)

	if err = c.auth(); err != nil {
		c.conn.Close() // nolint: errcheck
		return nil, err
	}

	return c, nil
}

// auth authenticates with the server if a password is set, otherwise its a no-op.
func (c *Client) auth() error {
	if c.pwd == "" {
		return nil
	}

	if err := c.writePkt(auth, c.pwd); err != nil {
		return err
	}

	p, err := c.readPkt()
	if err != nil {
		return err
	}

	if p.ID != 0 {
		return ErrAuthFailure
	}

	// The official spec says we should get a responseValue followed by authResponse
	// however Minecraft doesn't send the responseValue packet so we deal with that
	// case too.
	switch {
	case p.Type == responseValue:
		if p, err = c.readPkt(); err != nil {
			return err
		}

		if p.ID != 0 || p.Type != authResponse {
			return ErrAuthFailure
		}
	case p.Type != authResponse:
		return ErrAuthFailure
	}

	return nil
}

// Exec creates a new Cmd from cmd and calls ExecCmd with it.
// If cmd contains non-ASCII characters it returns ErrNonASCII.
func (c *Client) Exec(cmd string) (string, error) {
	return c.ExecCmd(NewCmd(cmd))
}

// ExecCmd executes cmd on the server and returns the response.
// If cmd contains non-ASCII characters it returns ErrNonASCII.
func (c *Client) ExecCmd(cmd *Cmd) (resp string, err error) {
	body := cmd.String()

	// Validate body is ASCII only
	for _, r := range body {
		if r >= 0x80 {
			return "", ErrNonASCII
		}
	}

	expectedID := c.reqID
	if err = c.write(execCommand, body); err != nil {
		return "", err
	}

	return c.read(expectedID)
}

// Close closes the connection to the server.
func (c *Client) Close() error {
	return c.conn.Close()
}

// readSingle reads a single packet, validates its ID matches expectedID and returns its body.
func (c *Client) readSingle(expectedID int32) (string, error) {
	p, err := c.readPkt()
	if err != nil {
		return "", err
	}

	if p.ID != expectedID {
		return "", ErrMalformedResponse(fmt.Sprintf("unexpected packet id %v", p.ID))
	}

	return p.Body(), nil
}

// readMulti reads responses packets from the server, combines multi-packet
// response bodies and returns the result.
func (c *Client) readMulti(expectedID int32) (body string, err error) {
	var buf bytes.Buffer
	var cnt int
	for {
		p, err := c.readPkt()
		if err != nil {
			return "", err
		}
		if p.Type != responseValue {
			return "", ErrMalformedResponse("unexpected type")
		}

		switch p.ID {
		case expectedID:
			// Command response packets, one or more expected.
			if _, err = buf.Write(p.body); err != nil {
				return "", err
			}
		case expectedID + 1:
			// Response response packets, exactly two expected.
			cnt++
			switch cnt {
			case 1:
				// Echoed response packet.
				if len(p.body) != 0 {
					return "", ErrMalformedResponse("non-empty body")
				}
			case 2:
				// Response packet response.
				if !bytes.Equal(p.body, responseBody) {
					return "", ErrMalformedResponse(fmt.Sprintf("unexpected body %q", p.Body()))
				}
				return buf.String(), nil
			}
		default:
			return "", ErrMalformedResponse(fmt.Sprintf("unexpected packet id %v", p.ID))
		}
	}
}

// readPkt reads a single packet from the server and returns it.
func (c *Client) readPkt() (*pkt, error) {
	if err := c.setDeadline(); err != nil {
		return nil, err
	}

	p := &pkt{}
	if _, err := p.ReadFrom(c.reader); err != nil {
		return nil, err
	}

	return p, nil
}

// writeMulti writes a packet with type t and body followed by a empty body
// responseValue type packet, so that we can easily decode multi-packet responses.
// https://developer.valvesoftware.com/wiki/Source_RCON_Protocol#Multiple-packet_Responses
func (c *Client) writeMulti(pktType int32, body string) error {
	if err := c.writePkt(pktType, body); err != nil {
		return err
	}

	// Now send an empty server response packet which will be echoed back, allowing
	// us to easily determine if we are processing a multi packet response.
	return c.writePkt(responseValue, "")
}

// writePkt writes a single packet to the server.
func (c *Client) writePkt(pktType int32, body string) error {
	p := newPkt(pktType, c.reqID, body)
	c.reqID++

	if err := c.setDeadline(); err != nil {
		return err
	}

	_, err := p.WriteTo(c.conn)
	return err
}

// setDeadline updates the deadline on the connection based on the clients configured timeout.
func (c *Client) setDeadline() error {
	return c.conn.SetDeadline(time.Now().Add(c.timeout))
}

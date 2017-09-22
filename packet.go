package source

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	// responseValue is the packet type returned in response to an execCommand.
	responseValue = int32(0)

	// execCommand is the packet type which represents a command issued to the server by the client.
	execCommand = int32(2)

	// auth is the packet type which is used to authenticate the connection with the server.
	auth = int32(3)

	// authResponse is the packet type which represents the connections current auth status.
	authResponse = int32(2)
)

// pkt represents an rcon packet
type pkt struct {
	Size int32
	ID   int32
	Type int32
	body []byte
}

// newPkt returns a new pkt for the given details.
func newPkt(t, id int32, body string) *pkt {
	return &pkt{Type: t, Size: int32(len(body) + 10), ID: id, body: []byte(body)}
}

// Body returns the packet body as a string.
func (p *pkt) Body() string {
	return string(p.body)
}

// WriteTo implements io.WriterTo.
func (p *pkt) WriteTo(w io.Writer) (n int64, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, p.Size+4))

	// Size of the packet not including the size field itself.
	if err := binary.Write(buf, binary.LittleEndian, p.Size); err != nil {
		return 0, err
	}

	// ID
	if err := binary.Write(buf, binary.LittleEndian, p.ID); err != nil {
		return 0, err
	}

	// Type
	if err := binary.Write(buf, binary.LittleEndian, p.Type); err != nil {
		return 0, err
	}

	// Body + null terminator + empty string null terminator
	if _, err := buf.Write(append(p.body, 0x00, 0x00)); err != nil {
		return 0, err
	}

	return buf.WriteTo(w)
}

// ReadFrom implements io.ReaderFrom, reading a packet from r.
func (p *pkt) ReadFrom(r io.Reader) (n int64, err error) {
	if err = binary.Read(r, binary.LittleEndian, &p.Size); err != nil {
		return n, err
	}
	n += 4
	if p.Size < 10 {
		return n, ErrMalformedResponse("size too small")
	}

	if err = binary.Read(r, binary.LittleEndian, &p.ID); err != nil {
		return n, err
	}
	n += 4

	if err = binary.Read(r, binary.LittleEndian, &p.Type); err != nil {
		return n, err
	}
	n += 4

	// We can't use ReadString(0x00) here as even though the spec says this
	// should be null terminated string, said string can actually include null
	// characters, which is the case in response to a responseValue packet.
	var i int32
	p.body = make([]byte, p.Size-8)
	for i < p.Size-8 {
		n2, err2 := r.Read(p.body[i:])
		if err != nil {
			return n + int64(n2) + int64(i), err2
		}
		i += int32(n2)
	}
	n += int64(i)

	if !bytes.Equal(p.body[len(p.body)-2:], []byte{0x00, 0x00}) {
		return n, ErrMalformedResponse("invalid trailer")
	}
	p.body = p.body[0 : len(p.body)-2]

	return n, nil
}

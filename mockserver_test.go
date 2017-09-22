package source

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	commands = map[string][]*pkt{
		fmt.Sprintf("%v:echo test me", execCommand): {newPkt(responseValue, 0, "test me")},
		fmt.Sprintf("%v:", responseValue): {
			newPkt(responseValue, 1, ""),
			newPkt(responseValue, 1, string(responseBody)),
		},
	}
)

// newLockListener creates a new listener on the local IP.
func newLocalListener() (net.Listener, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// server is a mock source rcon server
type server struct {
	Addr     string
	Listener net.Listener

	t        *testing.T
	conns    map[net.Conn]struct{}
	done     chan struct{}
	wg       sync.WaitGroup
	failConn bool
	mtx      sync.Mutex
}

// sconn represents a server connection
type sconn struct {
	id int
	net.Conn
}

// newServer returns a running server or nil if an error occurred.
func newServer(t *testing.T) *server {
	s := newServerStopped(t)
	s.Start()

	return s
}

// newServerStopped returns a stopped servers or nil if an error occurred.
func newServerStopped(t *testing.T) *server {
	l, err := newLocalListener()
	if !assert.NoError(t, err) {
		return nil
	}

	s := &server{
		Listener: l,
		conns:    make(map[net.Conn]struct{}),
		done:     make(chan struct{}),
		t:        t,
	}
	s.Addr = s.Listener.Addr().String()
	return s
}

// Start starts the server.
func (s *server) Start() {
	s.wg.Add(1)
	go s.serve()
}

// server processes incoming requests until signaled to stop with Close.
func (s *server) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if s.running() {
				assert.NoError(s.t, err)
			}
			return
		}
		s.wg.Add(1)
		go s.handle(conn)
	}
}

// write writes msg to conn.
func (s *server) write(conn net.Conn, id int32, pkts []*pkt) error {
	for _, p := range pkts {
		p.ID = id
		_, err := p.WriteTo(conn)
		if s.running() {
			assert.NoError(s.t, err)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// running returns true unless Close has been called, false otherwise.
func (s *server) running() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

// handle handles a client connection.
func (s *server) handle(conn net.Conn) {
	s.mtx.Lock()
	s.conns[conn] = struct{}{}
	s.mtx.Unlock()
	defer func() {
		s.closeConn(conn)
		s.wg.Done()
	}()

	if s.failConn {
		return
	}

	c := &sconn{Conn: conn}
	for {
		p := &pkt{}
		if _, err := p.ReadFrom(conn); err != nil {
			return
		}

		cmd := fmt.Sprintf("%v:%v", p.Type, p.Body())
		resp, ok := commands[cmd]
		if !ok {
			resp = []*pkt{newPkt(responseValue, p.ID, fmt.Sprintf("unknown command %v", cmd))}
		}

		if err := s.write(c, p.ID, resp); err != nil {
			return
		}
	}
}

// closeConn closes a client connection and removes it from our map of connections.
func (s *server) closeConn(conn net.Conn) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	conn.Close() // nolint: errcheck
	delete(s.conns, conn)
}

// Close cleanly shuts down the server.
func (s *server) Close() error {
	close(s.done)
	err := s.Listener.Close()
	s.mtx.Lock()
	for c := range s.conns {
		if err2 := c.Close(); err2 != nil && err == nil {
			err = err2
		}
	}
	s.mtx.Unlock()
	s.wg.Wait()

	return err
}

package dns //nolint:dupl

import (
	"net"
	"sync"
	"time"
)

// udpConnPool is a small bounded pool of reusable UDP connections to a single
// DNS server. The buffered conns channel both stores idle connections and caps
// how many are retained: Put discards a connection when the channel is full,
// and Get dials a fresh one when it is empty. It is safe for concurrent use.
type udpConnPool struct {
	addr    string
	timeout time.Duration
	size    int
	conns   chan *net.UDPConn
	mu      sync.Mutex
	closed  bool
	dialer  *net.Dialer
}

// newConnPool creates a UDP connection pool for addr retaining up to size idle
// connections (defaulting to 4 when size is non-positive).
func newConnPool(addr string, timeout time.Duration, size int) *udpConnPool {
	if size <= 0 {
		size = 4
	}

	pool := &udpConnPool{
		addr:    addr,
		timeout: timeout,
		size:    size,
		conns:   make(chan *net.UDPConn, size),
		dialer: &net.Dialer{
			Timeout: timeout,
		},
	}

	return pool
}

// Get returns an idle pooled connection if one is available, otherwise it dials
// a new one. It returns [net.ErrClosed] if the pool has been closed.
func (p *udpConnPool) Get() (*net.UDPConn, error) {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()

		return nil, net.ErrClosed
	}

	p.mu.Unlock()

	select {
	case conn := <-p.conns:
		if conn != nil {
			return conn, nil
		}
	default:
	}

	raddr, err := net.ResolveUDPAddr("udp", p.addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Put returns a connection to the pool for reuse. If the pool is full or
// closed, or conn is nil, the connection is closed instead.
func (p *udpConnPool) Put(conn *net.UDPConn) {
	if conn == nil {
		return
	}

	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()

		_ = conn.Close()

		return
	}

	p.mu.Unlock()

	select {
	case p.conns <- conn:

	default:
		_ = conn.Close()
	}
}

// Close marks the pool closed and closes every idle connection. It is
// idempotent and safe to call concurrently with Get/Put.
func (p *udpConnPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	close(p.conns)

	for conn := range p.conns {
		if conn != nil {
			_ = conn.Close()
		}
	}

	return nil
}

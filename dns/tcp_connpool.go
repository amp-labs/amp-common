package dns //nolint:dupl

import (
	"net"
	"sync"
	"time"
)

// tcpConnPool is a small bounded pool of reusable TCP connections to a single
// DNS server. The buffered conns channel both stores idle connections and caps
// how many are retained: Put discards a connection when the channel is full,
// and Get dials a fresh one when it is empty. It is safe for concurrent use.
type tcpConnPool struct {
	addr    string
	timeout time.Duration
	size    int
	conns   chan *net.TCPConn
	mu      sync.Mutex
	closed  bool
	dialer  *net.Dialer
}

// newTCPConnPool creates a TCP connection pool for addr retaining up to size
// idle connections (defaulting to 4 when size is non-positive).
func newTCPConnPool(addr string, timeout time.Duration, size int) *tcpConnPool {
	if size <= 0 {
		size = 4
	}

	pool := &tcpConnPool{
		addr:    addr,
		timeout: timeout,
		size:    size,
		conns:   make(chan *net.TCPConn, size),
		dialer: &net.Dialer{
			Timeout: timeout,
		},
	}

	return pool
}

// Get returns an idle pooled connection if one is available, otherwise it dials
// a new one. It returns [net.ErrClosed] if the pool has been closed.
func (p *tcpConnPool) Get() (*net.TCPConn, error) {
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

	raddr, err := net.ResolveTCPAddr("tcp", p.addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Put returns a connection to the pool for reuse. If the pool is full or
// closed, or conn is nil, the connection is closed instead.
func (p *tcpConnPool) Put(conn *net.TCPConn) {
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
func (p *tcpConnPool) Close() error {
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

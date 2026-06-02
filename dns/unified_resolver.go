package dns

import (
	"context"
	"net"
	"time"
)

// unifiedResolver queries a server over UDP first and transparently retries
// over TCP when the UDP response is truncated. This is the standard DNS
// behavior: try the cheap datagram path, fall back to TCP only when the answer
// doesn't fit. It is the base resolver [NewDialer] wraps for each address.
type unifiedResolver struct {
	addr string
	udp  *udpResolver
	tcp  *tcpResolver
}

// newUnifiedResolver creates a resolver for addr (defaulting to port 53 when
// none is given) backed by both a UDP and a TCP resolver sharing the same
// timeout and pool size.
func newUnifiedResolver(addr string, timeout time.Duration, poolSize int) *unifiedResolver {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, "53")
	}

	return &unifiedResolver{
		addr: addr,
		udp:  newUDPResolver(addr, timeout, poolSize),
		tcp:  newTCPResolver(addr, timeout, poolSize),
	}
}

// ResolveType resolves host over UDP and, if the response was truncated,
// retries the same query over TCP and returns that result instead. Any other
// UDP error (or a clean UDP answer) is returned as-is without a TCP retry.
func (r *unifiedResolver) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
) ([]Record, TruncationStatus, error) {
	records, truncated, err := r.udp.ResolveType(ctx, host, qtype)
	if err == nil && truncated == TruncationStatusOK {
		return records, truncated, nil
	}

	if truncated == TruncationStatusTruncated {
		return r.tcp.ResolveType(ctx, host, qtype)
	}

	return records, truncated, err
}

// Name returns the resolver's "host:port" address.
func (r *unifiedResolver) Name() string {
	return r.addr
}

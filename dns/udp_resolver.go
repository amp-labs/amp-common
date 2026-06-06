package dns //nolint:dupl

import (
	"context"
	"fmt"
	"net"
	"time"

	"codeberg.org/miekg/dns"
)

// udpResolver issues DNS queries over UDP to a single server using a pooled
// connection. UDP is the common case but is size-limited, so a truncated
// response is reported (via [TruncationStatusTruncated]) rather than retried
// here; unifiedResolver handles the TCP retry.
type udpResolver struct {
	addr     string
	timeout  time.Duration
	connPool *udpConnPool
}

// newUDPResolver creates a UDP resolver for addr (defaulting to port 53 when
// none is given), with the given per-query timeout and connection pool size.
func newUDPResolver(addr string, timeout time.Duration, poolSize int) *udpResolver {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, "53")
	}

	return &udpResolver{
		addr:     addr,
		timeout:  timeout,
		connPool: newConnPool(addr, timeout, poolSize),
	}
}

// ResolveType sends a single UDP query for host of the given type and parses
// the answer section into [Record] values. A truncated response returns
// [TruncationStatusTruncated] so the caller can retry over TCP; a non-success
// rcode or an empty answer is reported as an error.
func (r *udpResolver) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
) ([]Record, TruncationStatus, error) {
	msg := dns.NewMsg(host, uint16(qtype))
	if msg == nil {
		return nil, TruncationStatusUnknown, fmt.Errorf("%w: %q", errUnsupportedQueryType, qtype.String())
	}

	msg.UDPSize = 4096

	response, err := r.exchangeUDP(ctx, msg)
	if err != nil {
		return nil, TruncationStatusUnknown, err
	}

	if response.Truncated {
		logError(ctx, "udp response truncated",
			"host", host,
			"type", qtype.String())

		return nil, TruncationStatusTruncated, errTruncatedUDP
	}

	if response.Rcode != dns.RcodeSuccess { //nolint:dupl
		return nil, TruncationStatusOK, fmt.Errorf("%w: %s", errDNSResponse, dns.RcodeToString[response.Rcode])
	}

	records := make([]Record, 0, len(response.Answer))

	for _, ans := range response.Answer { //nolint:dupl
		record := Record{
			Name: ans.Header().Name,
			Type: RecordType(dns.RRToType(ans)),
			TTL:  ans.Header().TTL,
		}

		switch answer := ans.(type) {
		case *dns.A:
			record.Value = answer.Addr.String()
		case *dns.AAAA:
			record.Value = answer.Addr.String()
		case *dns.CNAME:
			record.Value = answer.Target
		case *dns.MX:
			record.Value = fmt.Sprintf("%d %s", answer.Preference, answer.Mx)
		case *dns.NS:
			record.Value = answer.Ns
		case *dns.TXT:
			record.Value = fmt.Sprintf("%v", answer.Txt)
		case *dns.SOA:
			record.Value = fmt.Sprintf("%s %s %d %d %d %d %d",
				answer.Ns, answer.Mbox, answer.Serial, answer.Refresh, answer.Retry, answer.Expire, answer.Minttl)
		case *dns.PTR:
			record.Value = answer.Ptr
		case *dns.SRV:
			record.Value = fmt.Sprintf("%d %d %d %s",
				answer.Priority, answer.Weight, answer.Port, answer.Target)
		default:
			record.Value = ans.String()
		}

		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, TruncationStatusOK, ErrNoRecords
	}

	return records, TruncationStatusOK, nil
}

// Name returns the resolver's "host:port" address.
func (r *udpResolver) Name() string {
	return r.addr
}

// newClient builds a dns.Client whose timeouts are the smaller of the
// configured timeout and the time remaining on the context deadline, so a query
// never outlives its caller's deadline.
func (r *udpResolver) newClient(ctx context.Context) *dns.Client {
	timeout := r.timeout

	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			if remaining := time.Until(deadline); remaining < timeout {
				timeout = remaining
			}
		}
	}

	return &dns.Client{
		Transport: &dns.Transport{
			Dialer:       &net.Dialer{Timeout: timeout},
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
		},
	}
}

// exchangeUDP performs the request/response exchange over a pooled connection,
// returning the connection to the pool on success and closing it on error.
func (r *udpResolver) exchangeUDP(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	conn, err := r.connPool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	response, _, err := r.newClient(ctx).ExchangeWithConn(ctx, msg, conn)
	if err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("query failed: %w", err)
	}

	r.connPool.Put(conn)

	return response, nil
}

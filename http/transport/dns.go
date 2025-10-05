package transport

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/rs/dnscache"
)

// dnsResolver is a package-level DNS cache resolver used to reduce DNS lookup overhead.
// It is initialized once and shared across all transports that enable DNS caching.
var dnsResolver *dnscache.Resolver

func init() {
	dnsResolver = &dnscache.Resolver{}
}

// useDNSCacheDialer modifies the given http.Transport to use a DNS caching dialer.
// This helps reduce DNS lookups and improve performance, especially under load.
func useDNSCacheDialer(trans *http.Transport, timeout, keepAlive time.Duration) {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}

	// Under load the DNS resolver can sometimes time out (resulting in "DNS timeout errors" when sending webhooks),
	// so we'll use a caching resolver to reduce the amount of DNS traffic.
	trans.DialContext = func(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := dnsResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				break
			}
		}

		return
	}
}

package dns

import (
	"context"
	"fmt"
	"net"

	"github.com/amp-labs/amp-common/retry"
	"github.com/amp-labs/amp-common/spans"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Dialer resolves hostnames using its configured resolvers and strategy, then
// dials the resulting IP addresses. Its [Dialer.DialContext] method matches the
// signature of [net.Dialer.DialContext], so it can be assigned directly to an
// [net/http.Transport]'s DialContext field. Build one with [NewDialer].
type Dialer struct {
	// lookup performs the hostname-to-IP resolution (strategy, caching, filtering)
	lookup *LookupCoordinator

	// dialer opens the final connection to a resolved IP (see WithDialer)
	dialer *net.Dialer

	// retryOptions configures how a failed dial of an individual IP is retried
	// (see WithDialerRetryOptions); empty means each IP is tried once
	retryOptions []retry.Option
}

// NewDialer builds a [Dialer] from the given options. It returns
// [ErrNoResolvers] if no resolvers were configured via [WithResolvers].
func NewDialer(opts ...Option) (*Dialer, error) {
	o := newOptions()

	for _, opt := range opts {
		opt(o)
	}

	return o.createDialer()
}

// DialContext resolves the host portion of addr (unless it is already an IP)
// and dials the resulting addresses for the requested network, returning the
// first connection that succeeds. The network is honored when selecting between
// IPv4 and IPv6 results: "tcp4"/"udp4" use only IPv4, "tcp6"/"udp6" only IPv6,
// and the generic "tcp"/"udp" try IPv4 first then IPv6. It mirrors the
// signature of [net.Dialer.DialContext].
func (r *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	attrs := []spans.Option{
		spans.WithSpanKind(trace.SpanKindClient),
		spans.WithAttribute("network", attribute.StringValue(network)),
		spans.WithAttribute("addr", attribute.StringValue(addr)),
	}

	return spans.StartValErr[net.Conn](ctx, "dialAddress", attrs...).
		Enter(func(ctx context.Context, span trace.Span) (net.Conn, error) {
			ips, port, err := r.lookup.Lookup(ctx, network, addr)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())

				return nil, err
			}

			var lastErr error

			for _, ipAddr := range ips {
				conn, err := retry.DoValue[net.Conn](ctx, func(ctx context.Context) (net.Conn, error) {
					ipAttrs := []spans.Option{
						spans.WithSpanKind(trace.SpanKindClient),
						spans.WithAttribute("network", attribute.StringValue(network)),
						spans.WithAttribute("ip", attribute.StringValue(ipAddr.String())),
						spans.WithAttribute("port", attribute.StringValue(port)),
						spans.WithAttribute("attempt", attribute.Int64Value(int64(retry.Attempt(ctx)))), //nolint:gosec
					}

					return spans.StartValErr[net.Conn](ctx, "dialIP", ipAttrs...).
						Enter(func(ctx context.Context, span trace.Span) (net.Conn, error) {
							ipAddrStr := net.JoinHostPort(ipAddr.String(), port)

							conn, err := r.dialer.DialContext(ctx, network, ipAddrStr)
							if err != nil {
								span.SetStatus(codes.Error, err.Error())

								return nil, err
							}

							span.SetStatus(codes.Ok, "connection established")
							span.SetAttributes(attribute.String("local", getAddrStr(conn.LocalAddr())))
							span.SetAttributes(attribute.String("remote", getAddrStr(conn.RemoteAddr())))

							return conn, nil
						})
				}, r.retryOptions...)
				if err == nil {
					return conn, nil
				}

				lastErr = err

				logDebug(ctx, "connection failed, trying next IP",
					"ip", ipAddr.String(),
					"error", err.Error())
			}

			return nil, fmt.Errorf("failed to connect to %q (network: %q): %w", addr, network, lastErr)
		})
}

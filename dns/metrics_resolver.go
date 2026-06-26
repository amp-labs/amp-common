package dns

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/amp-labs/amp-common/spans"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// metricsResolver decorates another Resolver, recording Prometheus metrics
// for each query: lookup count, error count, and latency, all labeled by the
// DNS server's address. It sits directly above the transport-level resolver
// (see createLookupCoordinator), so every query that actually hits the server
// is counted -- including the follow-up queries cnameResolver makes while
// chasing CNAME chains -- while purely local outcomes (filter rejections) are
// not.
type metricsResolver struct {
	addr     string
	proto    string
	resolver Resolver
}

// newMetricsResolver wraps resolver in a metricsResolver. addr (defaulting to
// port 53 when none is given) is used as the resolver's Name and as the
// "server" label on the recorded metrics.
func newMetricsResolver(addr, proto string, resolver Resolver) *metricsResolver {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, "53")
	}

	return &metricsResolver{
		addr:     addr,
		proto:    proto,
		resolver: resolver,
	}
}

// ResolveType resolves host via the wrapped resolver, recording the query
// count, error count, and latency under the server's address. Canceled
// queries are not recorded at all: under the [Race] strategy every lookup
// cancels the losing resolvers, and counting those as errors (or their
// time-to-cancellation as latency) would say nothing about the server while
// drowning out genuine failures. Timeouts are recorded as errors.
func (m *metricsResolver) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
) ([]Record, TruncationStatus, error) {
	attrs := []spans.Option{
		spans.WithSpanKind(trace.SpanKindClient),
		spans.WithAttribute("query", attribute.StringValue(host)),
		spans.WithAttribute("type", attribute.StringValue(qtype.String())),
		spans.WithAttribute("server", attribute.StringValue(m.addr)),
		spans.WithAttribute("protocol", attribute.StringValue(m.proto)),
	}

	var (
		start, end time.Time
		err        error
		records    []Record
		trunc      TruncationStatus
	)

	spans.Start(ctx, "dnsQuery", attrs...).Enter(func(ctx context.Context, span trace.Span) {
		start = time.Now()

		records, trunc, err = m.resolver.ResolveType(ctx, host, qtype)

		end = time.Now()

		switch {
		case trunc == TruncationStatusTruncated:
			span.SetStatus(codes.Error, "truncated")
		case err != nil:
			span.SetStatus(codes.Error, err.Error())
		default:
			ipStrs := make([]string, 0, len(records))

			for _, ip := range records {
				ipStrs = append(ipStrs, ip.String())
			}

			span.SetStatus(codes.Ok, "ok")
			span.SetAttributes(attribute.StringSlice("results", ipStrs))
		}
	})

	if errors.Is(err, context.Canceled) {
		return records, trunc, err
	}

	dur := end.Sub(start)

	lookupsTotal.WithLabelValues(m.addr, m.proto).Inc()
	lookupDuration.WithLabelValues(m.addr, m.proto).Observe(float64(dur.Milliseconds()))

	if err != nil {
		lookupErrorsTotal.WithLabelValues(m.addr, m.proto).Inc()
	}

	return records, trunc, err
}

// Name returns the resolver address, identifying the underlying server.
func (m *metricsResolver) Name() string {
	return m.addr
}

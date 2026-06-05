package dns

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// serverLabel is the metric label holding the DNS server's "host:port"
// address, so metrics can be grouped by which server was consulted.
const (
	serverLabel = "server"
	protoLabel  = "protocol"
)

var (
	// lookupsTotal counts every DNS query sent to a server, labeled by the
	// server's "host:port" address. Each query through a unifiedResolver counts
	// once, even when a truncated UDP response forces a TCP retry. Queries
	// canceled before completing (typically resolvers that lost a [Race]) are
	// not counted; see metricsResolver.
	lookupsTotal = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "dns_lookups_total",
		Help: "The total number of DNS queries sent, per DNS server",
	}, []string{serverLabel, protoLabel})

	// lookupErrorsTotal counts DNS queries that returned an error, labeled by
	// the server's "host:port" address. Cancellation is not an error here (the
	// server did nothing wrong); timeouts are.
	lookupErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "dns_lookup_errors_total",
		Help: "The total number of DNS queries that failed, per DNS server",
	}, []string{serverLabel, protoLabel})

	// lookupDuration tracks per-query latency in milliseconds, labeled by the
	// server's "host:port" address. Buckets cover sub-millisecond LAN answers
	// through multi-second timeouts.
	lookupDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{ //nolint:gochecknoglobals
		Name: "dns_lookup_duration_millis",
		Help: "DNS query latency in milliseconds, per DNS server",
		Buckets: []float64{
			0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000,
		},
	}, []string{serverLabel, protoLabel})
)

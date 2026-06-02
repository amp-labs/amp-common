package dns

import (
	"context"
)

// TruncationStatus reports whether a DNS response was truncated. UDP responses
// are limited in size, so a server signals truncation to ask the client to
// retry over TCP. It is returned alongside records so wrappers (such as
// unifiedResolver) can decide whether a TCP retry is warranted.
type TruncationStatus int

const (
	// TruncationStatusUnknown means truncation could not be determined, usually
	// because the query failed before a response was parsed.
	TruncationStatusUnknown TruncationStatus = iota
	// TruncationStatusOK means the response was complete and not truncated.
	TruncationStatusOK
	// TruncationStatusTruncated means the server set the TC bit; the answer is
	// incomplete and should be re-fetched over TCP.
	TruncationStatusTruncated
)

// Resolver performs a single typed DNS query against one DNS server.
//
// Implementations are layered (UDP/TCP transport, CNAME following, filtering),
// each wrapping the next, so that callers can compose behavior. All
// implementations must be safe for concurrent use, because strategies query
// several resolvers in parallel.
type Resolver interface {
	// ResolveType resolves host for the given record type, returning the matching
	// records, whether the response was truncated, and any error.
	ResolveType(ctx context.Context, host string, qtype RecordType) ([]Record, TruncationStatus, error)
	// Name returns a stable identifier for the resolver (typically its
	// "host:port" address) used for logging and consensus grouping.
	Name() string
}

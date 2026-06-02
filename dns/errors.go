package dns

import (
	"errors"
)

// Sentinel errors returned by the package. They are exported so callers can
// match against them with [errors.Is].
var (
	// ErrNoRecords is returned when a query succeeds but yields no usable records.
	ErrNoRecords = errors.New("no records found")
	// ErrNoConsensus is returned by the [Consensus] strategy when no answer
	// reaches the required agreement threshold.
	ErrNoConsensus = errors.New("consensus not reached")
	// ErrNoResolvers is returned by [NewDialer] when no resolvers were configured.
	ErrNoResolvers = errors.New("no resolvers found")
	// ErrCNAMELoop is returned when a CNAME chain points back to a name already
	// visited.
	ErrCNAMELoop = errors.New("CNAME loop detected")
	// ErrCNAMEChainTooLong is returned when a CNAME chain exceeds maxCNAMEDepth
	// hops.
	ErrCNAMEChainTooLong = errors.New("CNAME chain too long")
	// errUnsupportedQueryType is returned when a query type cannot be turned into
	// a DNS message.
	errUnsupportedQueryType = errors.New("unsupported query type")
	// errDNSResponse is returned when a resolver replies with a non-success rcode.
	errDNSResponse = errors.New("dns error")
	// errNoSuitableIPs is returned when resolution yields no addresses matching
	// the requested network.
	errNoSuitableIPs = errors.New("no suitable IP addresses found")
	// errNoIPAddresses is returned when a lookup yields no A/AAAA addresses.
	errNoIPAddresses = errors.New("no IP addresses found")
)

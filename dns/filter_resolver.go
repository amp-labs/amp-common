package dns

import (
	"context"
	"net"
)

// filterResolver decorates another Resolver, dropping any returned record the
// configured [Filter] rejects. If filtering removes every record the query is
// treated as having no records ([ErrNoRecords]).
type filterResolver struct {
	addr     string
	resolver Resolver
	filter   Filter
}

func newFilterResolver(addr string, resolver Resolver, filter Filter) *filterResolver {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "53")
	}

	return &filterResolver{
		addr:     addr,
		resolver: resolver,
		filter:   filter,
	}
}

// ResolveType resolves host via the wrapped resolver and returns only the
// records accepted by the filter. An error from the wrapped resolver is
// propagated unchanged; an empty result after filtering becomes [ErrNoRecords].
func (f *filterResolver) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
) ([]Record, TruncationStatus, error) {
	records, trunc, err := f.resolver.ResolveType(ctx, host, qtype)
	if err != nil {
		return nil, trunc, err
	}

	if len(records) == 0 {
		return nil, trunc, nil
	}

	var filtered []Record

	for _, record := range records {
		if f.filter.Accept(host, record) {
			filtered = append(filtered, record)
		}
	}

	if len(filtered) == 0 {
		return nil, TruncationStatusOK, ErrNoRecords
	}

	return filtered, trunc, nil
}

// Name returns the resolver address, identifying the underlying server.
func (f *filterResolver) Name() string {
	return f.addr
}

package dns

import (
	"context"
	"sync/atomic"
	"time"
)

// stubResolver is a configurable Resolver for tests. It returns a fixed answer
// (records/trunc/err), optionally after a delay, and counts how many times it
// was queried. Each stub carries its own name so strategies that key by
// resolver name (Consensus, Compare) can tell them apart.
type stubResolver struct {
	name    string
	records []Record
	trunc   TruncationStatus
	err     error
	delay   time.Duration
	calls   atomic.Int32
}

func (s *stubResolver) ResolveType(
	ctx context.Context,
	_ string,
	_ RecordType,
) ([]Record, TruncationStatus, error) {
	s.calls.Add(1)

	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, TruncationStatusUnknown, ctx.Err()
		}
	}

	return s.records, s.trunc, s.err
}

func (s *stubResolver) Name() string { return s.name }

// fakeStrategy is a Strategy stand-in for Dialer tests. It returns canned
// records per query type without touching the network, and counts invocations
// so cache behavior can be asserted.
type fakeStrategy struct {
	byType map[RecordType][]Record
	err    error
	calls  atomic.Int32
}

func (s *fakeStrategy) ResolveType(
	_ context.Context,
	_ string,
	qtype RecordType,
	_ []Resolver,
) ([]Record, error) {
	s.calls.Add(1)

	if s.err != nil {
		return nil, s.err
	}

	return s.byType[qtype], nil
}

package dns

import (
	"context"
	"time"
)

// ResolveType queries every resolver concurrently and returns the answer from
// the first one to succeed, then cancels the rest. If all resolvers fail it
// returns the last error seen. The context is cancelled on return so in-flight
// queries that lost the race are not left running.
func (s Race) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
	resolvers []Resolver,
) ([]Record, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		records  []Record
		err      error
		resolver string
		latency  time.Duration
	}

	results := make(chan result, len(resolvers))

	for _, res := range resolvers {
		go func(r Resolver) {
			start := time.Now()
			records, _, err := r.ResolveType(ctx, host, qtype)
			results <- result{
				records:  records,
				err:      err,
				resolver: r.Name(),
				latency:  time.Since(start),
			}
		}(res)
	}

	var lastErr error

	for i := 0; i < len(resolvers); i++ {
		r := <-results

		if r.err == nil {
			logDebug(ctx, "resolver won race",
				"resolver", r.resolver,
				"latency", r.latency,
				"type", qtype.String())

			cancel()

			return r.records, nil
		}

		lastErr = r.err
	}

	return nil, lastErr
}

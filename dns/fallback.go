package dns

import (
	"context"
)

// ResolveType tries the resolvers in order and returns the first successful
// answer. It moves to the next resolver only when the current one errors, and
// returns the last error if they all fail.
func (s Fallback) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
	resolvers []Resolver,
) ([]Record, error) {
	var lastErr error

	for _, res := range resolvers {
		records, _, err := res.ResolveType(ctx, host, qtype)
		if err == nil {
			logDebug(ctx, "resolver succeeded",
				"resolver", res.Name(),
				"type", qtype.String())

			return records, nil
		}

		lastErr = err

		logDebug(ctx, "resolver failed, trying next",
			"resolver", res.Name(),
			"type", qtype.String(),
			"error", err)
	}

	return nil, lastErr
}

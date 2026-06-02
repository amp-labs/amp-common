package dns

import (
	"context"
)

// ResolveType queries every resolver and returns the first answer received,
// but if the resolvers disagree it logs the discrepancy and invokes
// OnDiscrepancy (when set) with each resolver's result keyed by name. It is
// meant for auditing resolver behavior; for picking a trustworthy answer use
// [Consensus]. Resolvers that error are omitted from the comparison.
func (s Compare) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
	resolvers []Resolver,
) ([]Record, error) {
	results := make(map[string][]Record)

	for _, res := range resolvers {
		records, _, err := res.ResolveType(ctx, host, qtype)
		if err == nil {
			results[res.Name()] = records
		}
	}

	var first []Record

	allMatch := true

	for _, records := range results {
		if first == nil {
			first = records
		} else if !recordsEqual(first, records, s.IgnoreTTL) {
			allMatch = false

			break
		}
	}

	if !allMatch {
		logInfo(ctx, "discrepancy detected in record type query",
			"host", host,
			"type", qtype.String())

		if s.OnDiscrepancy != nil {
			s.OnDiscrepancy(host, qtype, results)
		}
	}

	return first, nil
}

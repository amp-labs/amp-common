package dns

import (
	"context"
	"fmt"
)

// ResolveType queries every resolver, groups the answers by equality, and
// returns the first group whose size meets MinAgreement. When MinAgreement is
// unset it defaults to a strict majority of the resolvers. Resolvers that error
// simply don't contribute to any group. If no group reaches the threshold it
// returns [ErrNoConsensus].
func (s Consensus) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
	resolvers []Resolver,
) ([]Record, error) {
	if s.MinAgreement <= 0 {
		s.MinAgreement = (len(resolvers) / 2) + 1
	}

	// resultGroup collects resolvers that returned an identical answer (under the
	// IgnoreTTL comparison) along with how many of them did so.
	type resultGroup struct {
		records []Record
		count   int
	}

	var groups []resultGroup

	for _, res := range resolvers {
		records, _, err := res.ResolveType(ctx, host, qtype)
		if err != nil {
			continue
		}

		matched := false

		for i := range groups {
			if recordsEqual(groups[i].records, records, s.IgnoreTTL) {
				groups[i].count++
				matched = true

				break
			}
		}

		if !matched {
			groups = append(groups, resultGroup{
				records: records,
				count:   1,
			})
		}
	}

	for _, group := range groups {
		if group.count >= s.MinAgreement {
			logDebug(ctx, "consensus reached",
				"agreements", group.count,
				"required", s.MinAgreement,
				"type", qtype.String())

			return group.records, nil
		}
	}

	return nil, fmt.Errorf("%w: required %d agreements", ErrNoConsensus, s.MinAgreement)
}

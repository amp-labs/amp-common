package dns

// recordKey is the comparable identity of a record for set-equality purposes.
// Name and Type are deliberately excluded: callers compare answer sets for the
// same query, where the value (and optionally the TTL) is what distinguishes
// one answer from another.
type recordKey struct {
	value string
	ttl   uint32
}

// recordsEqual reports whether a and b contain the same records with the same
// multiplicities, treating the slices as multisets (order-independent). When
// ignoreTTL is true, records are compared by value only, so the same data with
// differing TTLs counts as equal -- useful because each resolver counts its
// TTLs down independently.
func recordsEqual(a, b []Record, ignoreTTL bool) bool {
	// Fast path: if lengths differ, they can't be equal
	if len(a) != len(b) {
		return false
	}

	// Build a frequency map for slice 'a'. This counts how many times each
	// unique record appears. For example, if 'a' contains [X, X, Y], the map
	// will be {X: 2, Y: 1}.
	aMap := make(map[recordKey]int)
	for _, r := range a {
		key := recordKey{
			value: r.Value,
			ttl:   r.TTL,
		}
		if ignoreTTL {
			// Normalize TTL to 0 when comparing. This treats records with different
			// TTLs but the same value as equal. Important because:
			// 1. TTLs count down independently at each resolver
			// 2. Resolvers may have cached the record at different times
			// 3. We care about "is this the same data" not "same data with exact same TTL"
			key.ttl = 0
		}
		aMap[key]++
	}

	// Check that slice 'b' has the exact same frequency of each record.
	// For each record in 'b', decrement its count in the map. If we encounter
	// a record that's not in the map or has count 0, the slices aren't equal.
	for _, r := range b {
		key := recordKey{
			value: r.Value,
			ttl:   r.TTL,
		}
		if ignoreTTL {
			key.ttl = 0
		}
		count, exists := aMap[key]
		if !exists || count == 0 {
			// Either this record isn't in 'a', or 'b' has more copies of it than 'a' does
			return false
		}
		aMap[key]--
	}

	// If we get here, both slices contain the same records with the same frequencies
	return true
}

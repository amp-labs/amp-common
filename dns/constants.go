package dns

const (
	// recordsPerTypeHint estimates how many records each query type returns; it
	// is used only to pre-size the aggregated results slice.
	recordsPerTypeHint = 4

	// maxCachedTTLSeconds caps the TTL used when caching resolved IPs, before the
	// cache applies its own min/max bounds.
	maxCachedTTLSeconds = 300

	// maxCNAMEDepth bounds how many CNAME hops we follow before giving up. It guards
	// against CNAME loops (a -> b -> a) and pathologically long chains. RFC 1034
	// doesn't mandate a specific limit; 16 is far more than any legitimate chain yet
	// still terminates quickly when a resolver hands us something abusive.
	maxCNAMEDepth = 16
)

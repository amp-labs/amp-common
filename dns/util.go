package dns

import (
	"fmt"
	"net"
	"strconv"
)

// recordKey is the comparable identity of a record for set-equality purposes.
// Name and Type are deliberately excluded: callers compare answer sets for the
// same query, where the value (and optionally the TTL) is what distinguishes
// one answer from another.
type recordKey struct {
	value string
	ttl   uint32
}

// recordsEqual reports whether first and second contain the same records with
// the same multiplicities, treating the slices as multisets (order-independent).
// When ignoreTTL is true, records are compared by value only, so the same data
// with differing TTLs counts as equal -- useful because each resolver counts its
// TTLs down independently.
func recordsEqual(first, second []Record, ignoreTTL bool) bool {
	// Fast path: if lengths differ, they can't be equal
	if len(first) != len(second) {
		return false
	}

	// Build a frequency map for slice 'first'. This counts how many times each
	// unique record appears. For example, if 'first' contains [X, X, Y], the map
	// will be {X: 2, Y: 1}.
	counts := make(map[recordKey]int)

	for _, r := range first {
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

		counts[key]++
	}

	// Check that slice 'second' has the exact same frequency of each record.
	// For each record in 'second', decrement its count in the map. If we encounter
	// a record that's not in the map or has count 0, the slices aren't equal.
	for _, r := range second {
		key := recordKey{
			value: r.Value,
			ttl:   r.TTL,
		}
		if ignoreTTL {
			key.ttl = 0
		}

		count, exists := counts[key]
		if !exists || count == 0 {
			// Either this record isn't in 'first', or 'second' has more copies of it than 'first' does
			return false
		}

		counts[key]--
	}

	// If we get here, both slices contain the same records with the same frequencies
	return true
}

// ipToRecord wraps an IP literal in a synthetic [Record] (Name and Value both
// set to the IP's string form) so it can be passed through the same [Filter]
// predicates used for resolved records. It reports false for a nil or
// malformed IP.
//
// Classification deliberately uses To4 rather than the slice length:
// net.ParseIP stores IPv4 addresses in 16-byte IPv4-mapped form, so a length
// check would mislabel every parsed IPv4 literal as AAAA.
func ipToRecord(ip net.IP) (Record, bool) {
	if ip.To4() != nil {
		return Record{
			Type:  TypeA,
			Name:  ip.String(),
			Value: ip.String(),
		}, true
	}

	if ip.To16() != nil {
		return Record{
			Type:  TypeAAAA,
			Name:  ip.String(),
			Value: ip.String(),
		}, true
	}

	// nil or a malformed slice length (To4 and To16 both reject those).
	return Record{}, false
}

// parseHostAndPort splits addr ("host:port", standard net package format) and
// parses the port as a numeric uint16. Unlike net.SplitHostPort alone, it
// rejects service names ("http") and out-of-range ports, since callers need a
// concrete port number to rebuild per-IP dial addresses.
func parseHostAndPort(addr string) (host string, port string, err error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", "", fmt.Errorf("invalid address %q: %w", addr, err)
	}

	// SplitHostPort accepts service names ("http") and arbitrary numbers; we
	// need a concrete numeric port in the uint16 range.
	if _, err := strconv.ParseUint(portStr, 10, 16); err != nil {
		return "", "", fmt.Errorf("invalid port %q in address %q: %w", portStr, addr, err)
	}

	return host, portStr, nil
}

// filterIPs narrows ips to the address family implied by network. Unknown
// network strings fall through to the permissive "both families" behavior
// rather than erroring; the dial attempt will reject a truly bogus network.
func filterIPs(ips []net.IP, network string) []net.IP {
	switch network {
	case "tcp4", "udp4":
		// Only use IPv4 addresses, the caller explicitly asked for v4
		return filterIPv4(ips)
	case "tcp6", "udp6":
		// Only use IPv6 addresses, the caller explicitly asked for v6
		return filterIPv6(ips)
	default:
		// For "tcp" and "udp", use all IPs we got. Try IPv4 first for better compatibility,
		// more things support IPv4 than IPv6 in practice.
		return filterAnyIP(ips)
	}
}

// filterIPv4 returns only the IPv4 addresses from ips (including IPv4-mapped
// IPv6 forms, which To4 unwraps). Returns nil when none match.
func filterIPv4(ips []net.IP) []net.IP {
	var filteredIPs []net.IP

	for _, ip := range ips {
		if ip.To4() != nil {
			filteredIPs = append(filteredIPs, ip)
		}
	}

	return filteredIPs
}

// filterIPv6 returns only the genuine IPv6 addresses from ips. The To4 check
// excludes IPv4-mapped addresses that would otherwise satisfy To16, since
// those are not dialable on an IPv6-only network. Returns nil when none match.
func filterIPv6(ips []net.IP) []net.IP {
	var filteredIPs []net.IP

	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() != nil {
			filteredIPs = append(filteredIPs, ip)
		}
	}

	return filteredIPs
}

// filterAnyIP keeps both address families but orders the result IPv4 first,
// then IPv6, dropping anything malformed. Callers try addresses in order, and
// IPv4 succeeds more often in practice (Happy-Eyeballs-lite).
func filterAnyIP(ips []net.IP) []net.IP {
	filteredIPs := make([]net.IP, 0, len(ips))

	// Add IPv4 addresses first
	for _, ip := range ips {
		if ip.To4() != nil {
			filteredIPs = append(filteredIPs, ip)
		}
	}

	// Then add IPv6 addresses
	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() != nil {
			filteredIPs = append(filteredIPs, ip)
		}
	}

	return filteredIPs
}

func getAddrStr(a net.Addr) string {
	if a == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%s (%s)", a.String(), a.Network())
}

package utils //nolint:revive // Established package name for general utilities

import (
	"fmt"
	"net"
)

var privateIPBlocks []*net.IPNet //nolint:gochecknoglobals

func init() {
	// These IP blocks are reserved for private networks and should not be
	// exposed to the public internet. We use these to determine if an IP
	// address is private or not. Private IPs are not useful for tracking
	// clients, so we don't want to clutter the logs with them - if we can
	// avoid doing so.
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	}

	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			// This should never happen unless the above CIDR strings are modified
			panic(fmt.Errorf("parse error on %q: %w", cidr, err))
		}

		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func IsPrivateIPString(s string) (bool, bool) {
	ip := net.ParseIP(s)
	if ip == nil {
		return false, false
	}

	return IsPrivateIPAddress(ip), true
}

func IsPublicIPString(s string) (bool, bool) {
	ip := net.ParseIP(s)
	if ip == nil {
		return false, false
	}

	return IsPublicIPAddress(ip), true
}

func IsPublicIPAddress(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return false
		}
	}

	return true
}

func IsPrivateIPAddress(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

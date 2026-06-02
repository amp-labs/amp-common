package dns

import (
	"net"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

// ipCacheEntry is a cached set of IPs for one host along with the wall-clock
// time at which it should be considered stale.
type ipCacheEntry struct {
	ips       []net.IP
	expiresAt time.Time
}

// isExpired reports whether the entry's TTL has elapsed.
func (e *ipCacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// dnsCache is a TTL-aware, size-bounded cache of resolved IP addresses keyed by
// host. The backing LRU enforces the size bound and a hard maxTTL eviction; the
// per-entry expiry (clamped between minTTL and maxTTL) governs freshness so the
// record's own TTL is honored. A zero-size cache is disabled and every method
// becomes a no-op, letting callers use it unconditionally.
type dnsCache struct {
	ipCache *lru.LRU[string, *ipCacheEntry]
	mu      sync.RWMutex
	enabled bool
	minTTL  time.Duration
	maxTTL  time.Duration
}

// newDNSCache builds a cache holding up to size hosts. A non-positive size
// returns a disabled cache. minTTL and maxTTL bound how long any entry is kept,
// regardless of the TTL reported by the resolver.
func newDNSCache(size int, minTTL, maxTTL time.Duration) *dnsCache {
	if size <= 0 {
		return &dnsCache{enabled: false}
	}

	ipCache := lru.NewLRU[string, *ipCacheEntry](size, nil, maxTTL)

	return &dnsCache{
		ipCache: ipCache,
		enabled: true,
		minTTL:  minTTL,
		maxTTL:  maxTTL,
	}
}

// getIPs returns a copy of the cached IPs for host, or nil if the cache is
// disabled, the host is absent, or its entry has expired. The copy prevents
// callers from mutating the cached slice.
func (c *dnsCache) getIPs(host string) []net.IP {
	if !c.enabled {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.ipCache.Get(host)
	if !ok {
		return nil
	}

	if entry.isExpired() {
		return nil
	}

	ips := make([]net.IP, len(entry.ips))
	copy(ips, entry.ips)

	return ips
}

// setIPs caches ips for host with the given TTL clamped to [minTTL, maxTTL]. It
// is a no-op when the cache is disabled or there are no IPs to store.
func (c *dnsCache) setIPs(host string, ips []net.IP, ttl time.Duration) {
	if !c.enabled || len(ips) == 0 {
		return
	}

	if ttl < c.minTTL {
		ttl = c.minTTL
	}

	if ttl > c.maxTTL {
		ttl = c.maxTTL
	}

	entry := &ipCacheEntry{
		ips:       ips,
		expiresAt: time.Now().Add(ttl),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.ipCache.Add(host, entry)
}

package dns

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSCache_Disabled(t *testing.T) {
	t.Parallel()

	c := newDNSCache(0, 0, 0)
	assert.False(t, c.enabled)

	// Both operations must be safe no-ops on a disabled cache.
	c.setIPs("a.com", []net.IP{net.ParseIP("1.2.3.4")}, time.Minute)
	assert.Nil(t, c.getIPs("a.com"))
}

func TestDNSCache_SetGetRoundTrip(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, 0, time.Hour)
	ips := []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8")}

	c.setIPs("a.com", ips, time.Minute)

	got := c.getIPs("a.com")
	require.Len(t, got, 2)
	assert.Equal(t, "1.2.3.4", got[0].String())
	assert.Equal(t, "5.6.7.8", got[1].String())
}

func TestDNSCache_MissReturnsNil(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, 0, time.Hour)
	assert.Nil(t, c.getIPs("absent.com"))
}

func TestDNSCache_GetReturnsCopy(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, 0, time.Hour)
	c.setIPs("a.com", []net.IP{net.ParseIP("1.2.3.4")}, time.Minute)

	// Mutating the returned slice must not corrupt the cached entry.
	got := c.getIPs("a.com")
	got[0] = net.ParseIP("9.9.9.9")

	fresh := c.getIPs("a.com")
	assert.Equal(t, "1.2.3.4", fresh[0].String())
}

func TestDNSCache_EmptyIPsNotStored(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, 0, time.Hour)
	c.setIPs("a.com", nil, time.Minute)
	assert.Nil(t, c.getIPs("a.com"))
}

func TestDNSCache_Expiry(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, 0, time.Hour)
	c.setIPs("a.com", []net.IP{net.ParseIP("1.2.3.4")}, 10*time.Millisecond)

	assert.NotNil(t, c.getIPs("a.com"))

	time.Sleep(25 * time.Millisecond)

	assert.Nil(t, c.getIPs("a.com"), "an entry must not be returned after its TTL elapses")
}

func TestDNSCache_TTLClampedToMin(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, time.Hour, 24*time.Hour)
	before := time.Now()

	c.setIPs("a.com", []net.IP{net.ParseIP("1.2.3.4")}, time.Second)

	entry, ok := c.ipCache.Get("a.com")
	require.True(t, ok)
	// A 1s TTL is raised to the 1h floor.
	assert.True(t, entry.expiresAt.After(before.Add(30*time.Minute)),
		"a sub-minimum TTL should be raised to minTTL")
}

func TestDNSCache_TTLClampedToMax(t *testing.T) {
	t.Parallel()

	c := newDNSCache(10, time.Second, time.Hour)
	before := time.Now()

	c.setIPs("a.com", []net.IP{net.ParseIP("1.2.3.4")}, 48*time.Hour)

	entry, ok := c.ipCache.Get("a.com")
	require.True(t, ok)
	// A 48h TTL is capped at the 1h ceiling.
	assert.True(t, entry.expiresAt.Before(before.Add(2*time.Hour)),
		"an over-maximum TTL should be capped at maxTTL")
}

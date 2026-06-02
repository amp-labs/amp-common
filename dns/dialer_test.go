package dns

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDialer builds a Dialer wired to the given strategy and cache, with a
// plain net.Dialer and a single placeholder resolver (the fakeStrategy ignores
// the resolver set).
func newTestDialer(strategy Strategy, cache *dnsCache) *Dialer {
	return &Dialer{
		resolvers: []Resolver{&stubResolver{name: "placeholder"}},
		strategy:  strategy,
		timeout:   defaultTimeout,
		poolSize:  defaultPoolSize,
		dialer:    &net.Dialer{},
		cache:     cache,
	}
}

func TestDialer_LookupIPs_ParsesAddresses(t *testing.T) {
	t.Parallel()

	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA:    {aRec("a.com.", "1.2.3.4")},
		TypeAAAA: {{Type: TypeAAAA, Name: "a.com.", Value: "::1", TTL: 300}},
	}}

	d := newTestDialer(strategy, newDNSCache(0, 0, 0))

	ips, err := d.lookupIPs(context.Background(), "a.com")

	require.NoError(t, err)
	require.Len(t, ips, 2)

	// lookup() fans out concurrently, so the union order is not deterministic.
	got := []string{ips[0].String(), ips[1].String()}
	assert.ElementsMatch(t, []string{"1.2.3.4", "::1"}, got)
}

func TestDialer_LookupIPs_ServesFromCache(t *testing.T) {
	t.Parallel()

	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA: {aRec("a.com.", "1.2.3.4")},
	}}

	d := newTestDialer(strategy, newDNSCache(10, time.Second, time.Hour))

	_, err := d.lookupIPs(context.Background(), "a.com")
	require.NoError(t, err)

	// lookup() fans out one query per record type (A, AAAA, CNAME).
	firstCalls := strategy.calls.Load()
	require.Equal(t, int32(3), firstCalls)

	_, err = d.lookupIPs(context.Background(), "a.com")
	require.NoError(t, err)

	assert.Equal(t, firstCalls, strategy.calls.Load(), "a cached lookup must not re-query the resolvers")
}

func TestDialer_LookupIPs_NoAddressesError(t *testing.T) {
	t.Parallel()

	// Only a CNAME comes back, with no terminal A/AAAA record.
	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeCNAME: {cnameRec("a.com.", "b.com.")},
	}}

	d := newTestDialer(strategy, newDNSCache(0, 0, 0))

	_, err := d.lookupIPs(context.Background(), "a.com")
	require.Error(t, err)
}

func TestDialer_DialContext_InvalidAddress(t *testing.T) {
	t.Parallel()

	d := newTestDialer(&fakeStrategy{}, newDNSCache(0, 0, 0))

	_, err := d.DialContext(context.Background(), "tcp", "no-port-here")
	require.Error(t, err)
}

func TestDialer_DialContext_NoSuitableIPForNetwork(t *testing.T) {
	t.Parallel()

	// Resolution yields only IPv4, but the caller demands IPv6.
	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA: {aRec("a.com.", "1.2.3.4")},
	}}

	d := newTestDialer(strategy, newDNSCache(0, 0, 0))

	_, err := d.DialContext(context.Background(), "tcp6", "a.com:80")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable IP")
}

func TestDialer_DialContext_IPHostBypassesResolution(t *testing.T) {
	t.Parallel()

	lc := net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer func() { _ = listener.Close() }()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	// No strategy set: if DialContext attempted DNS resolution it would panic on
	// the nil strategy. A literal IP host must skip resolution entirely.
	d := &Dialer{dialer: &net.Dialer{}, cache: newDNSCache(0, 0, 0)}

	conn, err := d.DialContext(context.Background(), "tcp", listener.Addr().String())
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

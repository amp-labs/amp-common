package dns

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCoordinator builds a LookupCoordinator wired to the given strategy
// and cache, with a single placeholder resolver (the fakeStrategy ignores the
// resolver set) and no filter.
func newTestCoordinator(strategy Strategy, cache *dnsCache) *LookupCoordinator {
	return &LookupCoordinator{
		resolvers: []Resolver{&stubResolver{name: "placeholder"}},
		strategy:  strategy,
		cache:     cache,
	}
}

func TestNewLookupCoordinator_RequiresResolvers(t *testing.T) {
	t.Parallel()

	_, err := NewLookupCoordinator()
	require.ErrorIs(t, err, ErrNoResolvers)
}

func TestNewLookupCoordinator_WithResolvers(t *testing.T) {
	t.Parallel()

	l, err := NewLookupCoordinator(WithResolvers("8.8.8.8", "1.1.1.1"))
	require.NoError(t, err)
	require.NotNil(t, l)
	assert.Len(t, l.resolvers, 2)
}

func TestLookupCoordinator_LookupIPs_ParsesAddresses(t *testing.T) {
	t.Parallel()

	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA:    {aRec("a.com.", "1.2.3.4")},
		TypeAAAA: {{Type: TypeAAAA, Name: "a.com.", Value: "::1", TTL: 300}},
	}}

	l := newTestCoordinator(strategy, newDNSCache(0, 0, 0))

	ips, err := l.lookupIPs(context.Background(), "a.com")

	require.NoError(t, err)
	require.Len(t, ips, 2)

	// lookup() fans out concurrently, so the union order is not deterministic.
	got := []string{ips[0].String(), ips[1].String()}
	assert.ElementsMatch(t, []string{"1.2.3.4", "::1"}, got)
}

func TestLookupCoordinator_LookupIPs_ServesFromCache(t *testing.T) {
	t.Parallel()

	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA: {aRec("a.com.", "1.2.3.4")},
	}}

	l := newTestCoordinator(strategy, newDNSCache(10, time.Second, time.Hour))

	_, err := l.lookupIPs(context.Background(), "a.com")
	require.NoError(t, err)

	// lookup() fans out one query per record type (A, AAAA, CNAME).
	firstCalls := strategy.calls.Load()
	require.Equal(t, int32(3), firstCalls)

	_, err = l.lookupIPs(context.Background(), "a.com")
	require.NoError(t, err)

	assert.Equal(t, firstCalls, strategy.calls.Load(), "a cached lookup must not re-query the resolvers")
}

func TestLookupCoordinator_LookupIPs_NoAddressesError(t *testing.T) {
	t.Parallel()

	// Only a CNAME comes back, with no terminal A/AAAA record.
	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeCNAME: {cnameRec("a.com.", "b.com.")},
	}}

	l := newTestCoordinator(strategy, newDNSCache(0, 0, 0))

	_, err := l.lookupIPs(context.Background(), "a.com")
	require.ErrorIs(t, err, errNoIPAddresses)
}

func TestLookupCoordinator_Lookup_FiltersByNetwork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		network string
		want    []string
	}{
		{name: "tcp4 keeps only IPv4", network: "tcp4", want: []string{"1.2.3.4"}},
		{name: "udp4 keeps only IPv4", network: "udp4", want: []string{"1.2.3.4"}},
		{name: "tcp6 keeps only IPv6", network: "tcp6", want: []string{"::1"}},
		{name: "udp6 keeps only IPv6", network: "udp6", want: []string{"::1"}},
		{name: "tcp keeps both, IPv4 first", network: "tcp", want: []string{"1.2.3.4", "::1"}},
		{name: "udp keeps both, IPv4 first", network: "udp", want: []string{"1.2.3.4", "::1"}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			strategy := &fakeStrategy{byType: map[RecordType][]Record{
				TypeA:    {aRec("a.com.", "1.2.3.4")},
				TypeAAAA: {{Type: TypeAAAA, Name: "a.com.", Value: "::1", TTL: 300}},
			}}

			l := newTestCoordinator(strategy, newDNSCache(0, 0, 0))

			ips, port, err := l.Lookup(context.Background(), testCase.network, "a.com:443")
			require.NoError(t, err)
			assert.Equal(t, uint16(443), port)

			got := make([]string, len(ips))
			for i, ip := range ips {
				got[i] = ip.String()
			}

			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestLookupCoordinator_Lookup_NoSuitableIPForNetwork(t *testing.T) {
	t.Parallel()

	// Resolution yields only IPv4, but the caller demands IPv6.
	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA: {aRec("a.com.", "1.2.3.4")},
	}}

	l := newTestCoordinator(strategy, newDNSCache(0, 0, 0))

	_, _, err := l.Lookup(context.Background(), "tcp6", "a.com:80")
	require.ErrorIs(t, err, errNoSuitableIPs)
}

func TestLookupCoordinator_Lookup_ResolutionFailure(t *testing.T) {
	t.Parallel()

	strategy := &fakeStrategy{err: ErrNoRecords}

	l := newTestCoordinator(strategy, newDNSCache(0, 0, 0))

	_, _, err := l.Lookup(context.Background(), "tcp", "a.com:80")
	require.ErrorIs(t, err, errNoIPAddresses)
	assert.Contains(t, err.Error(), "DNS lookup failed")
}

func TestLookupCoordinator_Lookup_InvalidAddresses(t *testing.T) {
	t.Parallel()

	coordinator := newTestCoordinator(&fakeStrategy{}, newDNSCache(0, 0, 0))

	tests := []struct {
		name string
		addr string
	}{
		{name: "missing port", addr: "no-port-here"},
		{name: "service name port", addr: "a.com:http"},
		{name: "port out of range", addr: "a.com:70000"},
		{name: "empty", addr: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := coordinator.Lookup(context.Background(), "tcp", testCase.addr)
			require.Error(t, err)
		})
	}
}

func TestLookupCoordinator_Lookup_LiteralIPSkipsResolution(t *testing.T) {
	t.Parallel()

	// No strategy and no resolvers: if Lookup attempted DNS resolution it would
	// panic on the nil strategy. A literal IP host must skip resolution entirely.
	l := &LookupCoordinator{cache: newDNSCache(0, 0, 0)}

	ips, port, err := l.Lookup(context.Background(), "tcp", "127.0.0.1:8080")
	require.NoError(t, err)
	require.Len(t, ips, 1)
	assert.Equal(t, "127.0.0.1", ips[0].String())
	assert.Equal(t, uint16(8080), port)
}

func TestLookupCoordinator_Lookup_LiteralIPThroughFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addr     string
		wantType RecordType
		wantIP   string
	}{
		{
			// Regression: net.ParseIP stores IPv4 in 16-byte mapped form, which a
			// length-based classification would mislabel as AAAA.
			name:     "IPv4 literal presented to filter as an A record",
			addr:     "1.2.3.4:443",
			wantType: TypeA,
			wantIP:   "1.2.3.4",
		},
		{
			name:     "IPv6 literal presented to filter as an AAAA record",
			addr:     "[::1]:443",
			wantType: TypeAAAA,
			wantIP:   "::1",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var seen []Record

			l := &LookupCoordinator{
				filter: newFilter(func(_ string, record Record) bool {
					seen = append(seen, record)

					return true
				}),
				cache: newDNSCache(0, 0, 0),
			}

			ips, _, err := l.Lookup(context.Background(), "tcp", testCase.addr)
			require.NoError(t, err)
			require.Len(t, ips, 1)
			assert.Equal(t, testCase.wantIP, ips[0].String())

			require.Len(t, seen, 1)
			assert.Equal(t, testCase.wantType, seen[0].Type)
			assert.Equal(t, testCase.wantIP, seen[0].Value)
		})
	}
}

func TestLookupCoordinator_Lookup_LiteralIPRejectedByFilter(t *testing.T) {
	t.Parallel()

	l := &LookupCoordinator{
		filter: newFilter(func(_ string, _ Record) bool { return false }),
		cache:  newDNSCache(0, 0, 0),
	}

	_, _, err := l.Lookup(context.Background(), "tcp", "10.0.0.1:80")
	require.ErrorIs(t, err, errNoSuitableIPs)
}

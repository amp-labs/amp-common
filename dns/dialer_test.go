package dns

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDialer builds a Dialer whose LookupCoordinator is wired to the given
// strategy and cache, with a plain net.Dialer and a single placeholder
// resolver (the fakeStrategy ignores the resolver set).
func newTestDialer(strategy Strategy, cache *dnsCache) *Dialer {
	return &Dialer{
		lookup: newTestCoordinator(strategy, cache),
		dialer: &net.Dialer{},
	}
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

	listener := newTestListener(t)

	// No strategy set: if DialContext attempted DNS resolution it would panic on
	// the nil strategy. A literal IP host must skip resolution entirely.
	d := &Dialer{
		lookup: &LookupCoordinator{cache: newDNSCache(0, 0, 0)},
		dialer: &net.Dialer{},
	}

	conn, err := d.DialContext(context.Background(), "tcp", listener.Addr().String())
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

func TestDialer_DialContext_ResolvesHostnameAndDials(t *testing.T) {
	t.Parallel()

	listener := newTestListener(t)

	// Resolve the fake hostname to the listener's loopback address, exercising
	// the full lookup-then-dial path.
	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	strategy := &fakeStrategy{byType: map[RecordType][]Record{
		TypeA: {aRec("a.com.", "127.0.0.1")},
	}}

	d := newTestDialer(strategy, newDNSCache(0, 0, 0))

	conn, err := d.DialContext(context.Background(), "tcp", "a.com:"+portStr)
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.Equal(t, listener.Addr().String(), conn.RemoteAddr().String())

	_ = conn.Close()
}

// newTestListener starts a loopback TCP listener that accepts and immediately
// closes a single connection. It is cleaned up when the test ends.
func newTestListener(t *testing.T) net.Listener {
	t.Helper()

	lc := net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	return listener
}

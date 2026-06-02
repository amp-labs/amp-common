package dns

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"codeberg.org/miekg/dns/dnsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordReq identifies a single ResolveType call (canonical host + type).
type recordReq struct {
	host  string
	qtype RecordType
}

type fakeResponse struct {
	records []Record
	trunc   TruncationStatus
	err     error
}

// fakeResolver is a programmable Resolver that records every query it receives,
// so tests can assert both the records returned and that no unnecessary
// follow-up queries were made.
type fakeResolver struct {
	responses map[recordReq]fakeResponse
	mu        sync.Mutex
	calls     []recordReq
}

func newFakeResolver() *fakeResolver {
	return &fakeResolver{responses: make(map[recordReq]fakeResponse)}
}

func (f *fakeResolver) add(host string, qtype RecordType, resp fakeResponse) {
	f.responses[recordReq{dnsutil.Canonical(host), qtype}] = resp
}

func (f *fakeResolver) ResolveType(
	_ context.Context,
	host string,
	qtype RecordType,
) ([]Record, TruncationStatus, error) {
	key := recordReq{dnsutil.Canonical(host), qtype}

	f.mu.Lock()
	f.calls = append(f.calls, key)
	f.mu.Unlock()

	resp, ok := f.responses[key]
	if !ok {
		// Mirror the base resolvers: a name with no records of this type errors.
		return nil, TruncationStatusOK, ErrNoRecords
	}

	return resp.records, resp.trunc, resp.err
}

func (f *fakeResolver) Name() string { return "fake" }

func cnameRec(name, target string) Record {
	return Record{Type: TypeCNAME, Name: name, Value: target, TTL: 300}
}

func aRec(name, ip string) Record {
	return Record{Type: TypeA, Name: name, Value: ip, TTL: 300}
}

// TestCNAMEResolver_DirectAnswer: a name that resolves straight to an A record
// is returned untouched with no extra queries.
func TestCNAMEResolver_DirectAnswer(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("example.com", TypeA, fakeResponse{records: []Record{aRec("example.com.", "1.2.3.4")}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "example.com", TypeA)

	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "1.2.3.4", recs[0].Value)
	assert.Len(t, f.calls, 1, "a direct answer needs no follow-up query")
}

// TestCNAMEResolver_RecursiveChainNoExtraQuery: a recursive resolver returns the
// whole chain in one response, so we must not issue any follow-up query.
func TestCNAMEResolver_RecursiveChainNoExtraQuery(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("www.example.com", TypeA, fakeResponse{records: []Record{
		cnameRec("www.example.com.", "shop.example.com."),
		aRec("shop.example.com.", "1.2.3.4"),
	}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "www.example.com", TypeA)

	require.NoError(t, err)
	assert.Len(t, f.calls, 1, "recursive answer already holds the terminal record")
	assert.True(t, hasRecordOfType(recs, "shop.example.com.", TypeA))
}

// TestCNAMEResolver_NonRecursiveFollows: a non-recursive resolver returns only
// the CNAME, so we must chase the terminal record ourselves.
func TestCNAMEResolver_NonRecursiveFollows(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("www.example.com", TypeA, fakeResponse{records: []Record{
		cnameRec("www.example.com.", "shop.example.com."),
	}})
	f.add("shop.example.com", TypeA, fakeResponse{records: []Record{
		aRec("shop.example.com.", "1.2.3.4"),
	}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "www.example.com", TypeA)

	require.NoError(t, err)
	require.Len(t, f.calls, 2)
	assert.Equal(t, recordReq{"www.example.com.", TypeA}, f.calls[0])
	assert.Equal(t, recordReq{"shop.example.com.", TypeA}, f.calls[1])
	assert.True(t, hasRecordOfType(recs, "shop.example.com.", TypeA))
}

// TestCNAMEResolver_MultiHop: a multi-link non-recursive chain is followed to
// the terminal record, one query per missing link.
func TestCNAMEResolver_MultiHop(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("a.com", TypeA, fakeResponse{records: []Record{cnameRec("a.com.", "b.com.")}})
	f.add("b.com", TypeA, fakeResponse{records: []Record{cnameRec("b.com.", "c.com.")}})
	f.add("c.com", TypeA, fakeResponse{records: []Record{aRec("c.com.", "9.9.9.9")}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "a.com", TypeA)

	require.NoError(t, err)
	assert.Len(t, f.calls, 3)
	assert.True(t, hasRecordOfType(recs, "c.com.", TypeA))
}

// TestCNAMEResolver_SkipsQueryForInlinedCNAME: when a response inlines an
// intermediate CNAME but not the terminal record, we query only for the missing
// terminal -- not for the link we already hold.
func TestCNAMEResolver_SkipsQueryForInlinedCNAME(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("www.example.com", TypeA, fakeResponse{records: []Record{
		cnameRec("www.example.com.", "a.example.com."),
		cnameRec("a.example.com.", "b.example.com."),
	}})
	f.add("b.example.com", TypeA, fakeResponse{records: []Record{
		aRec("b.example.com.", "1.2.3.4"),
	}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "www.example.com", TypeA)

	require.NoError(t, err)
	require.Len(t, f.calls, 2, "the inlined a.example.com link must not be re-queried")
	assert.Equal(t, recordReq{"www.example.com.", TypeA}, f.calls[0])
	assert.Equal(t, recordReq{"b.example.com.", TypeA}, f.calls[1])
	assert.True(t, hasRecordOfType(recs, "b.example.com.", TypeA))
}

// TestCNAMEResolver_LoopDetected: a -> b -> a is reported as a loop.
func TestCNAMEResolver_LoopDetected(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("a.com", TypeA, fakeResponse{records: []Record{
		cnameRec("a.com.", "b.com."),
		cnameRec("b.com.", "a.com."),
	}})

	c := newCNameResolver("8.8.8.8:53", f)

	_, _, err := c.ResolveType(context.Background(), "a.com", TypeA)

	require.ErrorIs(t, err, ErrCNAMELoop)
}

// TestCNAMEResolver_SelfLoop: a -> a is reported as a loop.
func TestCNAMEResolver_SelfLoop(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("a.com", TypeA, fakeResponse{records: []Record{cnameRec("a.com.", "a.com.")}})

	c := newCNameResolver("8.8.8.8:53", f)

	_, _, err := c.ResolveType(context.Background(), "a.com", TypeA)

	require.ErrorIs(t, err, ErrCNAMELoop)
}

// TestCNAMEResolver_ChainTooLong: a chain longer than maxCNAMEDepth is rejected.
func TestCNAMEResolver_ChainTooLong(t *testing.T) {
	t.Parallel()

	chain := make([]Record, 0, maxCNAMEDepth+5)
	for i := 0; i < maxCNAMEDepth+5; i++ {
		chain = append(chain, cnameRec(fmt.Sprintf("n%d.com.", i), fmt.Sprintf("n%d.com.", i+1)))
	}

	f := newFakeResolver()
	f.add("n0.com", TypeA, fakeResponse{records: chain})

	c := newCNameResolver("8.8.8.8:53", f)

	_, _, err := c.ResolveType(context.Background(), "n0.com", TypeA)

	require.ErrorIs(t, err, ErrCNAMEChainTooLong)
}

// TestCNAMEResolver_CanonicalMatching: matching is case-insensitive and tolerant
// of trailing dots on both the host and the record names.
func TestCNAMEResolver_CanonicalMatching(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("www.example.com", TypeA, fakeResponse{records: []Record{
		cnameRec("WWW.Example.COM.", "Shop.Example.com."),
		aRec("shop.example.com.", "1.2.3.4"),
	}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "WWW.EXAMPLE.COM", TypeA)

	require.NoError(t, err)
	assert.Len(t, f.calls, 1)
	assert.True(t, hasRecordOfType(recs, "shop.example.com.", TypeA))
}

// TestCNAMEResolver_CNAMEQueryDoesNotChase: a CNAME-type query is answered by the
// CNAME itself; it must not be flattened to the terminal address.
func TestCNAMEResolver_CNAMEQueryDoesNotChase(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("www.example.com", TypeCNAME, fakeResponse{records: []Record{
		cnameRec("www.example.com.", "shop.example.com."),
	}})
	f.add("shop.example.com", TypeA, fakeResponse{records: []Record{
		aRec("shop.example.com.", "1.2.3.4"),
	}})

	c := newCNameResolver("8.8.8.8:53", f)

	recs, _, err := c.ResolveType(context.Background(), "www.example.com", TypeCNAME)

	require.NoError(t, err)
	require.Len(t, f.calls, 1, "a CNAME query must not chase to the terminal record")
	require.Len(t, recs, 1)
	assert.Equal(t, TypeCNAME, recs[0].Type)
}

// TestCNAMEResolver_InitialErrorPropagates: an error on the first query is
// returned as-is with no follow-up.
func TestCNAMEResolver_InitialErrorPropagates(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")

	f := newFakeResolver()
	f.add("example.com", TypeA, fakeResponse{err: sentinel})

	c := newCNameResolver("8.8.8.8:53", f)

	_, _, err := c.ResolveType(context.Background(), "example.com", TypeA)

	require.ErrorIs(t, err, sentinel)
	assert.Len(t, f.calls, 1)
}

// TestCNAMEResolver_FollowUpErrorPropagates: if the terminal hop has no record,
// the resulting error surfaces (wrapped).
func TestCNAMEResolver_FollowUpErrorPropagates(t *testing.T) {
	t.Parallel()

	f := newFakeResolver()
	f.add("www.example.com", TypeA, fakeResponse{records: []Record{
		cnameRec("www.example.com.", "shop.example.com."),
	}})
	// No entry for shop.example.com -> fake returns ErrNoRecords.

	c := newCNameResolver("8.8.8.8:53", f)

	_, _, err := c.ResolveType(context.Background(), "www.example.com", TypeA)

	require.ErrorIs(t, err, ErrNoRecords)
	assert.Len(t, f.calls, 2)
}

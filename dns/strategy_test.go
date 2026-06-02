package dns

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static sentinel errors for the resolver stubs. errFirst/errLast are distinct
// so tests can assert which one a strategy ultimately surfaces.
var (
	errStub  = errors.New("stub resolver failure")
	errFirst = errors.New("first resolver failure")
	errLast  = errors.New("last resolver failure")
)

func TestRace_FirstSuccessWins(t *testing.T) {
	t.Parallel()

	slow := &stubResolver{
		name:    "slow",
		records: []Record{aRec("a.com.", "5.6.7.8")},
		delay:   time.Second,
	}
	fast := &stubResolver{
		name:    "fast",
		records: []Record{aRec("a.com.", "1.2.3.4")},
	}

	recs, err := Race{}.ResolveType(context.Background(), "a.com", TypeA, []Resolver{slow, fast})

	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "1.2.3.4", recs[0].Value, "the resolver with no delay should win")
}

func TestRace_SkipsErrorsForSuccess(t *testing.T) {
	t.Parallel()

	bad := &stubResolver{name: "bad", err: errStub}
	good := &stubResolver{name: "good", records: []Record{aRec("a.com.", "1.2.3.4")}}

	recs, err := Race{}.ResolveType(context.Background(), "a.com", TypeA, []Resolver{bad, good})

	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "1.2.3.4", recs[0].Value)
}

func TestRace_AllFailReturnsError(t *testing.T) {
	t.Parallel()

	a := &stubResolver{name: "a", err: errStub}
	b := &stubResolver{name: "b", err: errStub}

	_, err := Race{}.ResolveType(context.Background(), "a.com", TypeA, []Resolver{a, b})

	require.ErrorIs(t, err, errStub)
}

func TestFallback_ReturnsFirstSuccessWithoutTryingRest(t *testing.T) {
	t.Parallel()

	first := &stubResolver{name: "first", records: []Record{aRec("a.com.", "1.2.3.4")}}
	second := &stubResolver{name: "second", records: []Record{aRec("a.com.", "5.6.7.8")}}

	recs, err := Fallback{}.ResolveType(context.Background(), "a.com", TypeA, []Resolver{first, second})

	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "1.2.3.4", recs[0].Value)
	assert.Equal(t, int32(1), first.calls.Load())
	assert.Equal(t, int32(0), second.calls.Load(), "the second resolver must not be queried once the first succeeds")
}

func TestFallback_AdvancesPastFailures(t *testing.T) {
	t.Parallel()

	first := &stubResolver{name: "first", err: errStub}
	second := &stubResolver{name: "second", records: []Record{aRec("a.com.", "5.6.7.8")}}

	recs, err := Fallback{}.ResolveType(context.Background(), "a.com", TypeA, []Resolver{first, second})

	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "5.6.7.8", recs[0].Value)
	assert.Equal(t, int32(1), first.calls.Load())
	assert.Equal(t, int32(1), second.calls.Load())
}

func TestFallback_AllFailReturnsLastError(t *testing.T) {
	t.Parallel()

	first := &stubResolver{name: "first", err: errFirst}
	second := &stubResolver{name: "second", err: errLast}

	_, err := Fallback{}.ResolveType(context.Background(), "a.com", TypeA, []Resolver{first, second})

	require.ErrorIs(t, err, errLast)
}

func TestConsensus_MajorityWins(t *testing.T) {
	t.Parallel()

	agreed := []Record{aRec("a.com.", "1.2.3.4")}
	resolvers := []Resolver{
		&stubResolver{name: "a", records: agreed},
		&stubResolver{name: "b", records: agreed},
		&stubResolver{name: "c", records: []Record{aRec("a.com.", "9.9.9.9")}},
	}

	// MinAgreement defaults to a strict majority (2 of 3).
	recs, err := Consensus{}.ResolveType(context.Background(), "a.com", TypeA, resolvers)

	require.NoError(t, err)
	assert.Equal(t, agreed, recs)
}

func TestConsensus_NoAgreementErrors(t *testing.T) {
	t.Parallel()

	resolvers := []Resolver{
		&stubResolver{name: "a", records: []Record{aRec("a.com.", "1.1.1.1")}},
		&stubResolver{name: "b", records: []Record{aRec("a.com.", "2.2.2.2")}},
		&stubResolver{name: "c", records: []Record{aRec("a.com.", "3.3.3.3")}},
	}

	_, err := Consensus{}.ResolveType(context.Background(), "a.com", TypeA, resolvers)

	require.ErrorIs(t, err, ErrNoConsensus)
}

func TestConsensus_ExplicitMinAgreement(t *testing.T) {
	t.Parallel()

	agreed := []Record{aRec("a.com.", "1.2.3.4")}
	resolvers := []Resolver{
		&stubResolver{name: "a", records: agreed},
		&stubResolver{name: "b", records: agreed},
	}

	// Two agreeing resolvers don't meet a required threshold of three.
	_, err := Consensus{MinAgreement: 3}.ResolveType(context.Background(), "a.com", TypeA, resolvers)

	require.ErrorIs(t, err, ErrNoConsensus)
}

func TestConsensus_IgnoresErroringResolvers(t *testing.T) {
	t.Parallel()

	agreed := []Record{aRec("a.com.", "1.2.3.4")}
	resolvers := []Resolver{
		&stubResolver{name: "a", records: agreed},
		&stubResolver{name: "b", records: agreed},
		&stubResolver{name: "c", err: errStub},
	}

	// MinAgreement defaults to majority of 3 = 2; the two healthy resolvers agree.
	recs, err := Consensus{}.ResolveType(context.Background(), "a.com", TypeA, resolvers)

	require.NoError(t, err)
	assert.Equal(t, agreed, recs)
}

func TestConsensus_IgnoreTTLGroupsAcrossTTLs(t *testing.T) {
	t.Parallel()

	resolvers := []Resolver{
		&stubResolver{name: "a", records: []Record{{Type: TypeA, Name: "a.com.", Value: "1.2.3.4", TTL: 300}}},
		&stubResolver{name: "b", records: []Record{{Type: TypeA, Name: "a.com.", Value: "1.2.3.4", TTL: 60}}},
	}

	// Without IgnoreTTL these two answers form separate groups and never reach
	// consensus; with it they group together.
	_, err := Consensus{}.ResolveType(context.Background(), "a.com", TypeA, resolvers)
	require.ErrorIs(t, err, ErrNoConsensus)

	recs, err := Consensus{IgnoreTTL: true}.ResolveType(context.Background(), "a.com", TypeA, resolvers)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "1.2.3.4", recs[0].Value)
}

func TestCompare_NoDiscrepancyDoesNotInvokeCallback(t *testing.T) {
	t.Parallel()

	agreed := []Record{aRec("a.com.", "1.2.3.4")}
	resolvers := []Resolver{
		&stubResolver{name: "a", records: agreed},
		&stubResolver{name: "b", records: agreed},
	}

	called := false
	strategy := Compare{OnDiscrepancy: func(string, RecordType, map[string][]Record) { called = true }}

	recs, err := strategy.ResolveType(context.Background(), "a.com", TypeA, resolvers)

	require.NoError(t, err)
	assert.Equal(t, agreed, recs)
	assert.False(t, called, "matching resolvers must not trigger the discrepancy callback")
}

func TestCompare_DiscrepancyInvokesCallback(t *testing.T) {
	t.Parallel()

	resolvers := []Resolver{
		&stubResolver{name: "a", records: []Record{aRec("a.com.", "1.1.1.1")}},
		&stubResolver{name: "b", records: []Record{aRec("a.com.", "2.2.2.2")}},
	}

	var got map[string][]Record

	strategy := Compare{OnDiscrepancy: func(_ string, _ RecordType, results map[string][]Record) {
		got = results
	}}

	_, err := strategy.ResolveType(context.Background(), "a.com", TypeA, resolvers)

	require.NoError(t, err)
	require.Len(t, got, 2, "the callback should receive every resolver's answer keyed by name")
	assert.Equal(t, "1.1.1.1", got["a"][0].Value)
	assert.Equal(t, "2.2.2.2", got["b"][0].Value)
}

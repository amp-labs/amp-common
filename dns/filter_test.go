package dns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilter_NilAcceptsEverything(t *testing.T) {
	t.Parallel()

	f := newFilter(nil)
	assert.True(t, f.Accept("a.com", aRec("a.com.", "1.2.3.4")))
}

func TestNewFilter_AppliesPredicate(t *testing.T) {
	t.Parallel()

	// Reject anything resolving to the documentation range 1.2.3.4.
	f := newFilter(func(_ string, r Record) bool { return r.Value != "1.2.3.4" })

	assert.False(t, f.Accept("a.com", aRec("a.com.", "1.2.3.4")))
	assert.True(t, f.Accept("a.com", aRec("a.com.", "5.6.7.8")))
}

func TestFilterResolver_KeepsAcceptedRecords(t *testing.T) {
	t.Parallel()

	inner := &stubResolver{
		name: "8.8.8.8:53",
		records: []Record{
			aRec("a.com.", "1.2.3.4"),
			aRec("a.com.", "5.6.7.8"),
		},
		trunc: TruncationStatusOK,
	}
	filter := newFilter(func(_ string, r Record) bool { return r.Value != "1.2.3.4" })

	fr := newFilterResolver("8.8.8.8:53", inner, filter)

	recs, _, err := fr.ResolveType(context.Background(), "a.com", TypeA)

	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "5.6.7.8", recs[0].Value)
}

func TestFilterResolver_AllFilteredOutIsNoRecords(t *testing.T) {
	t.Parallel()

	inner := &stubResolver{
		name:    "8.8.8.8:53",
		records: []Record{aRec("a.com.", "1.2.3.4")},
		trunc:   TruncationStatusOK,
	}
	filter := newFilter(func(_ string, _ Record) bool { return false })

	fr := newFilterResolver("8.8.8.8:53", inner, filter)

	_, _, err := fr.ResolveType(context.Background(), "a.com", TypeA)

	require.ErrorIs(t, err, ErrNoRecords)
}

func TestFilterResolver_PropagatesUnderlyingError(t *testing.T) {
	t.Parallel()

	inner := &stubResolver{name: "8.8.8.8:53", err: ErrNoRecords}
	fr := newFilterResolver("8.8.8.8:53", inner, newFilter(nil))

	_, _, err := fr.ResolveType(context.Background(), "a.com", TypeA)

	require.ErrorIs(t, err, ErrNoRecords)
}

func TestFilterResolver_NameDefaultsPort(t *testing.T) {
	t.Parallel()

	// A bare host gets the standard DNS port appended.
	fr := newFilterResolver("8.8.8.8", &stubResolver{name: "x"}, newFilter(nil))
	assert.Equal(t, "8.8.8.8:53", fr.Name())
}

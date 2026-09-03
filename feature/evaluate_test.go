package feature

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findValues returns one value inside and one outside the given percentage
// for a flag, so tests do not depend on hard-coded hash outputs.
func findValues(t *testing.T, name Name, salt string, dim Dimension, pct float64) (inside, outside string) {
	t.Helper()

	limit := int(pct * percentScale)

	for i := 0; inside == "" || outside == ""; i++ {
		value := "value-" + strconv.Itoa(i)

		if Bucket(nil, name, salt, dim, value) < limit {
			if inside == "" {
				inside = value
			}
		} else if outside == "" {
			outside = value
		}
	}

	return inside, outside
}

func TestEvaluateOrder(t *testing.T) {
	t.Parallel()

	const name Name = "order"

	reg := NewRegistry()
	flag := reg.Define(name, Off().Deny(dimUser, "bad").Allow(dimUser, "bad", "vip").Percent(dimUser, 50))
	ctx := t.Context()

	result := flag.Evaluate(ctx, Subject{dimUser: "bad"})
	assert.False(t, result.Enabled)
	assert.Equal(t, ReasonDenied, result.Reason, "deny wins over allow")
	assert.Equal(t, dimUser, result.Dimension)
	assert.Equal(t, -1, result.Bucket)

	result = flag.Evaluate(ctx, Subject{dimUser: "vip"})
	assert.True(t, result.Enabled)
	assert.Equal(t, ReasonAllowed, result.Reason)

	inside, outside := findValues(t, name, "", dimUser, 50)

	result = flag.Evaluate(ctx, Subject{dimUser: inside})
	assert.True(t, result.Enabled)
	assert.Equal(t, ReasonPercentage, result.Reason)
	assert.Equal(t, dimUser, result.Dimension)
	assert.Less(t, result.Bucket, 5000)

	result = flag.Evaluate(ctx, Subject{dimUser: outside})
	assert.False(t, result.Enabled)
	assert.Equal(t, ReasonDefault, result.Reason)
	assert.GreaterOrEqual(t, result.Bucket, 5000, "bucket is reported even on a miss")

	result = flag.Evaluate(ctx, nil)
	assert.False(t, result.Enabled)
	assert.Equal(t, ReasonDefault, result.Reason, "missing dimension falls through")
	assert.Equal(t, -1, result.Bucket)
	assert.Equal(t, name, result.Flag)

	require.NoError(t, flag.Set(On().Deny(dimUser, "bad")))
	assert.True(t, flag.Enabled(ctx))
	assert.False(t, flag.EnabledFor(ctx, Subject{dimUser: "bad"}))
}

func TestEvaluateEmptyValueIsAbsent(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("empty", Off().Allow(dimUser, "").Percent(dimUser, 100))

	result := flag.Evaluate(t.Context(), Subject{dimUser: ""})
	assert.False(t, result.Enabled)
	assert.Equal(t, ReasonDefault, result.Reason)
}

func TestPercentageDistribution(t *testing.T) {
	t.Parallel()

	const samples = 20_000

	reg := NewRegistry()
	flag := reg.Define("distribution", Off().Percent(dimUser, 25))
	ctx := t.Context()

	enabled := 0

	for i := range samples {
		if flag.EnabledFor(ctx, Subject{dimUser: "user-" + strconv.Itoa(i)}) {
			enabled++
		}
	}

	assert.InDelta(t, 0.25, float64(enabled)/samples, 0.015)
}

func TestPercentageMonotonic(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("monotonic", Off().Percent(dimUser, 10))
	ctx := t.Context()

	var insideAtTen []string

	for i := range 5000 {
		user := "user-" + strconv.Itoa(i)
		if flag.EnabledFor(ctx, Subject{dimUser: user}) {
			insideAtTen = append(insideAtTen, user)
		}
	}

	require.NotEmpty(t, insideAtTen)
	require.NoError(t, flag.Set(Off().Percent(dimUser, 30)))

	for _, user := range insideAtTen {
		assert.True(t, flag.EnabledFor(ctx, Subject{dimUser: user}), "raising the percentage keeps %s inside", user)
	}
}

func TestPercentageIndependentAcrossFlags(t *testing.T) {
	t.Parallel()

	const samples = 4000

	reg := NewRegistry()
	first := reg.Define("flag-a", Off().Percent(dimUser, 50))
	second := reg.Define("flag-b", Off().Percent(dimUser, 50))
	ctx := t.Context()

	agree := 0

	for i := range samples {
		subject := Subject{dimUser: "user-" + strconv.Itoa(i)}
		if first.EnabledFor(ctx, subject) == second.EnabledFor(ctx, subject) {
			agree++
		}
	}

	assert.InDelta(t, 0.5, float64(agree)/samples, 0.05, "two independent 50% rollouts agree about half the time")
}

func TestPercentageSaltReshuffles(t *testing.T) {
	t.Parallel()

	const samples = 4000

	reg := NewRegistry()
	flag := reg.Define("salted", Off().Percent(dimUser, 50))
	ctx := t.Context()

	before := make([]bool, samples)
	for i := range samples {
		before[i] = flag.EnabledFor(ctx, Subject{dimUser: "user-" + strconv.Itoa(i)})
	}

	require.NoError(t, flag.Set(Off().PercentWithSalt(dimUser, 50, "v2")))

	changed := 0

	for i := range samples {
		if before[i] != flag.EnabledFor(ctx, Subject{dimUser: "user-" + strconv.Itoa(i)}) {
			changed++
		}
	}

	assert.InDelta(t, 0.5, float64(changed)/samples, 0.05)
}

func TestPercentageDeterministicAcrossRegistries(t *testing.T) {
	t.Parallel()

	first := NewRegistry().Define("deterministic", Off().Percent(dimUser, 30))
	second := NewRegistry().Define("deterministic", Off().Percent(dimUser, 30))
	ctx := t.Context()

	for i := range 1000 {
		subject := Subject{dimUser: "user-" + strconv.Itoa(i)}
		assert.Equal(t, first.EnabledFor(ctx, subject), second.EnabledFor(ctx, subject))
	}
}

func TestPercentagesAreOred(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("ored", Off().Percent(dimUser, 0).Percent(dimInstallation, 100))
	ctx := t.Context()

	result := flag.Evaluate(ctx, Subject{dimUser: "u", dimInstallation: "i"})
	assert.True(t, result.Enabled)
	assert.Equal(t, dimInstallation, result.Dimension)

	result = flag.Evaluate(ctx, Subject{dimUser: "u"})
	assert.False(t, result.Enabled)
	assert.Equal(t, ReasonDefault, result.Reason)
	assert.GreaterOrEqual(t, result.Bucket, 0, "bucket of the last percentage that applied")
}

func TestBucketMatchesEvaluation(t *testing.T) {
	t.Parallel()

	const name Name = "bucket-check"

	reg := NewRegistry()
	flag := reg.Define(name, Off().PercentWithSalt(dimUser, 33.33, "s"))
	ctx := t.Context()

	for i := range 500 {
		user := "user-" + strconv.Itoa(i)
		bucket := Bucket(nil, name, "s", dimUser, user)
		result := flag.Evaluate(ctx, Subject{dimUser: user})

		assert.Equal(t, bucket, result.Bucket)
		assert.Equal(t, bucket < 3333, result.Enabled)
	}
}

func TestPercentageEdges(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	none := reg.Define("none", Off().Percent(dimUser, 0))
	all := reg.Define("all", Off().Percent(dimUser, 100))
	ctx := t.Context()

	for i := range 200 {
		subject := Subject{dimUser: "user-" + strconv.Itoa(i)}
		assert.False(t, none.EnabledFor(ctx, subject))
		assert.True(t, all.EnabledFor(ctx, subject))
	}
}

func TestCustomHasher(t *testing.T) {
	t.Parallel()

	lowest := NewRegistry(WithHasher(func(string) uint64 { return 0 }))
	highest := NewRegistry(WithHasher(func(string) uint64 { return Buckets - 1 }))
	ctx := t.Context()

	assert.True(t, lowest.Define("h", Off().Percent(dimUser, 0.01)).EnabledFor(ctx, Subject{dimUser: "anyone"}))
	assert.False(t, highest.Define("h", Off().Percent(dimUser, 99.99)).EnabledFor(ctx, Subject{dimUser: "anyone"}))
	assert.True(t, highest.Define("h2", Off().Percent(dimUser, 100)).EnabledFor(ctx, Subject{dimUser: "anyone"}))
}

package dns

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptions_Defaults(t *testing.T) {
	t.Parallel()

	o := newOptions()

	assert.IsType(t, Race{}, o.strategy)
	assert.Equal(t, defaultTimeout, o.timeout)
	assert.Equal(t, defaultPoolSize, o.poolSize)
	assert.NotNil(t, o.dialer)
	require.NotNil(t, o.cache)
	assert.False(t, o.cache.enabled, "caching is disabled by default")
}

func TestNewDialer_NoResolversErrors(t *testing.T) {
	t.Parallel()

	_, err := NewDialer()
	require.ErrorIs(t, err, ErrNoResolvers)
}

func TestNewDialer_BuildsResolverPerAddress(t *testing.T) {
	t.Parallel()

	d, err := NewDialer(WithResolvers("8.8.8.8:53", "1.1.1.1:53"))

	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Len(t, d.resolvers, 2)
}

func TestWithResolvers_Accumulates(t *testing.T) {
	t.Parallel()

	o := newOptions()
	WithResolvers("8.8.8.8:53")(o)
	WithResolvers("1.1.1.1:53", "9.9.9.9:53")(o)

	assert.Equal(t, []string{"8.8.8.8:53", "1.1.1.1:53", "9.9.9.9:53"}, o.resolvers)
}

func TestWithOptions_Apply(t *testing.T) {
	t.Parallel()

	o := newOptions()
	dialer := &net.Dialer{}

	WithStrategy(Fallback{})(o)
	WithTimeout(2 * time.Second)(o)
	WithConnPoolSize(8)(o)
	WithDialer(dialer)(o)
	WithCache(100, time.Second, time.Minute)(o)

	assert.IsType(t, Fallback{}, o.strategy)
	assert.Equal(t, 2*time.Second, o.timeout)
	assert.Equal(t, 8, o.poolSize)
	assert.Same(t, dialer, o.dialer)
	assert.True(t, o.cache.enabled)
}

func TestWithFilter_NilIgnored(t *testing.T) {
	t.Parallel()

	o := newOptions()
	WithFilter(nil)(o)
	assert.Nil(t, o.filter, "a nil predicate must not install a filter")

	WithFilter(func(string, Record) bool { return true })(o)
	assert.NotNil(t, o.filter)
}

func TestWithConnPoolSize_NonPositiveIgnored(t *testing.T) {
	t.Parallel()

	o := newOptions()
	WithConnPoolSize(0)(o)
	WithConnPoolSize(-5)(o)
	assert.Equal(t, defaultPoolSize, o.poolSize, "non-positive pool sizes keep the default")
}

func TestWithCache_NonPositiveSizeDisabled(t *testing.T) {
	t.Parallel()

	o := newOptions()
	WithCache(0, time.Second, time.Minute)(o)
	assert.False(t, o.cache.enabled)
}

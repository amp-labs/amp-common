package diskmap

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueCacheEvictsOldestAndWraps(t *testing.T) {
	t.Parallel()

	cache := newValueCache(4096)
	cached := make(map[loc][]byte)

	// Enough varied entries to wrap the ring many times.
	for i := range 2000 {
		l := loc{seg: uint32(i % 7), off: uint32(i), size: 1} //nolint:gosec // Small test values.
		key := "k" + strconv.Itoa(i)
		value := bytes.Repeat([]byte{byte(i)}, i%300)

		cache.put(l, key, value)
		cached[l] = value

		require.LessOrEqual(t, len(cache.index), cache.maxEntries)
		require.LessOrEqual(t, cache.head-cache.tail, uint64(len(cache.ring)))

		// Every indexed entry must still hold the bytes it was given.
		for k, pos := range cache.index {
			entry := cache.ring[pos%uint64(len(cache.ring)):]
			l := loc{seg: uint32(k >> 32), off: uint32(k)} //nolint:gosec // Decoding locKey.

			keyLen := int(entry[8])
			value, ok := cache.get(l, string(entry[vcHeader:vcHeader+keyLen]))
			require.True(t, ok)
			require.Equal(t, cached[loc{seg: l.seg, off: l.off, size: 1}], value)
		}
	}

	// The newest entry survives; a lookup with the wrong key is a cached miss.
	last := loc{seg: 1999 % 7, off: 1999, size: 1}
	value, ok := cache.get(last, "k1999")
	require.True(t, ok)
	require.Equal(t, cached[last], value)

	value, ok = cache.get(last, "other")
	require.True(t, ok)
	require.Nil(t, value)

	cache.reset()

	_, ok = cache.get(last, "k1999")
	require.False(t, ok)
}

func TestValueCacheAdmitsOnSecondMiss(t *testing.T) {
	t.Parallel()

	cache := newValueCache(1 << 20)
	l := loc{seg: 3, off: 99, size: 100}

	require.False(t, cache.admit(l, 100))
	require.True(t, cache.admit(l, 100))
	require.False(t, cache.admit(l, 100), "the doorkeeper resets after admitting")
	require.False(t, cache.admit(loc{seg: 1, off: 1}, len(cache.ring)), "huge records are never admitted")
}

package diskmap

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/xxh3"
)

// smallOptions force page eviction, segment rotation and compaction with
// little data.
func smallOptions() []Option {
	return []Option{WithMemoryLimit(minMemoryLimit), WithSegmentSize(minSegmentSize)}
}

func openTest(t *testing.T, dir string, opts ...Option) *Map {
	t.Helper()

	m, err := Open(dir, opts...)
	require.NoError(t, err)

	t.Cleanup(func() { _ = m.Close() })

	return m
}

func valueFor(rng *rand.Rand, key string, version int) []byte {
	size := rng.IntN(200)
	if rng.IntN(50) == 0 {
		size = rng.IntN(100_000)
	}

	v := make([]byte, size)
	for i := range v {
		v[i] = byte(i + version)
	}

	return append(v, key...)
}

// checkModel verifies that m holds exactly the entries in model.
func checkModel(t *testing.T, m *Map, model map[string][]byte) {
	t.Helper()

	require.Equal(t, len(model), m.Size())

	for k, want := range model {
		got, found, err := m.Get(k)
		require.NoError(t, err)
		require.True(t, found, "key %q missing", k)
		require.True(t, bytes.Equal(want, got), "key %q has the wrong value", k)
	}

	seen := make(map[string]bool, len(model))

	require.NoError(t, m.Range(func(key string, value []byte) bool {
		require.False(t, seen[key], "key %q visited twice", key)
		seen[key] = true

		require.True(t, bytes.Equal(model[key], value), "Range gave key %q the wrong value", key)

		return true
	}))
	require.Len(t, seen, len(model))
}

func TestRandomOperationsMatchGoMap(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		opts []Option
	}{
		{name: "default hash"},
		{
			// 64 distinct hash values: most keys share their full 64-bit hash
			// with others, so every lookup has to compare stored keys.
			name: "colliding hash",
			opts: []Option{withHash(func(key string, seed uint64) uint64 {
				return (xxh3.HashStringSeed(key, seed)%64 + 1) * 0x9e3779b97f4a7c15
			})},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			opts := append(smallOptions(), tc.opts...)
			m := openTest(t, dir, opts...)
			rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // Deterministic test data.
			model := make(map[string][]byte)
			compacted := int64(0)

			const (
				keySpace = 3000
				ops      = 60_000
			)

			for op := range ops {
				key := "key-" + strconv.Itoa(rng.IntN(keySpace))

				switch r := rng.IntN(10); {
				case r < 5:
					v := valueFor(rng, key, op)
					require.NoError(t, m.Add(key, v))

					model[key] = v
				case r < 7:
					require.NoError(t, m.Remove(key))
					delete(model, key)
				case r < 9:
					got, found, err := m.Get(key)
					require.NoError(t, err)

					want, ok := model[key]
					require.Equal(t, ok, found)
					require.True(t, bytes.Equal(want, got))
				default:
					found, err := m.Contains(key)
					require.NoError(t, err)

					_, ok := model[key]
					require.Equal(t, ok, found)
				}

				if op%20_000 == 19_999 {
					checkModel(t, m, model)

					compacted += m.Stats().CompactedBytes
					require.NoError(t, m.Close())
					m = openTest(t, dir, opts...)
					checkModel(t, m, model)
				}
			}

			stats := m.Stats()
			assert.Positive(t, compacted, "compaction never ran")
			assert.Positive(t, stats.PageMisses, "index never left the cache")
			assert.Greater(t, stats.IndexPages, 1)

			require.NoError(t, m.Compact())
			assert.Zero(t, m.Stats().DeadBytes)
			checkModel(t, m, model)
		})
	}
}

func TestManyKeysWithSmallCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := openTest(t, dir, smallOptions()...)

	const n = 200_000

	for i := range n {
		require.NoError(t, m.Add(strconv.Itoa(i), []byte(strconv.Itoa(i*7))))
	}

	require.Equal(t, n, m.Size())

	for i := range n {
		v, found, err := m.Get(strconv.Itoa(i))
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, strconv.Itoa(i*7), string(v))
	}

	stats := m.Stats()
	assert.Greater(t, stats.IndexPages, stats.CachedPages, "index should not fit in the cache")

	require.NoError(t, m.Close())

	m = openTest(t, dir, smallOptions()...)
	require.Equal(t, n, m.Size())

	v, found, err := m.Get("123456")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, strconv.Itoa(123456*7), string(v))
}

func TestEdgeCaseKeysAndValues(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), smallOptions()...)

	big := bytes.Repeat([]byte("x"), 3*minSegmentSize)
	entries := map[string][]byte{
		"":              []byte("empty key"),
		"empty value":   {},
		"big":           big,
		"after big":     []byte("small"),
		string([]byte{0, 1, 2, 255}): []byte("binary key"),
	}

	for k, v := range entries {
		require.NoError(t, m.Add(k, v))
	}

	checkModel(t, m, entries)

	got, found, err := m.Get("empty value")
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, got)
	require.Empty(t, got)

	require.ErrorIs(t, m.Add(string(make([]byte, MaxKeySize+1)), nil), ErrKeyTooLarge)
}

func TestAppendValueReusesBuffer(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir())
	require.NoError(t, m.Add("a", []byte("alpha")))
	require.NoError(t, m.Add("b", []byte("beta")))

	buf := make([]byte, 0, 64)

	buf, found, err := m.AppendValue(buf[:0], "a")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "alpha", string(buf))

	buf, found, err = m.AppendValue(buf, "b")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "alphabeta", string(buf))

	buf, found, err = m.AppendValue(buf, "missing")
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, "alphabeta", string(buf))
}

func TestOverwritesKeepDiskUsageBounded(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), smallOptions()...)
	value := make([]byte, 1000)

	const keys = 1000 // about 1 MB live

	for round := range 50 {
		for i := range keys {
			value[0] = byte(round)
			require.NoError(t, m.Add(strconv.Itoa(i), value))
		}
	}

	stats := m.Stats()
	live := stats.SegmentBytes - stats.DeadBytes

	// Live data, plus dead data up to the threshold, plus segments that are
	// being filled or compacted.
	assert.Less(t, stats.SegmentBytes, 2*live+3*minSegmentSize, "%+v", stats)
	assert.Equal(t, keys, stats.Entries)
}

func TestReopenRequiresCleanClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	m, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, m.Add("k", []byte("v")))

	// A second Open while the first is still open sees no meta file, the same
	// as after a crash.
	_, err = Open(dir)
	require.ErrorIs(t, err, ErrNotCleanlyClosed)

	require.NoError(t, m.Close())
	require.NoError(t, m.Close(), "closing twice is a no-op")

	_, _, err = m.Get("k")
	require.ErrorIs(t, err, ErrClosed)

	m = openTest(t, dir)

	v, found, err := m.Get("k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "v", string(v))

	// Discarding starts over even though m never closed.
	other := openTest(t, dir, WithDiscardExisting())
	require.Zero(t, other.Size())
}

func TestOpenLeavesOtherFilesAlone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	unrelated := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(unrelated, []byte("keep me"), 0o600))

	m := openTest(t, dir, WithDiscardExisting())
	require.NoError(t, m.Add("k", nil))
	require.NoError(t, m.Clear())
	require.NoError(t, m.Close())

	_ = openTest(t, dir, WithDiscardExisting())

	_, err := os.Stat(unrelated)
	require.NoError(t, err)
}

func TestCorruptRecordIsReported(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := openTest(t, dir)
	require.NoError(t, m.Add("k", []byte("value")))
	require.NoError(t, m.Close())

	path := segmentPath(dir, 0)
	data, err := os.ReadFile(path) //nolint:gosec // Test file.
	require.NoError(t, err)

	data[len(data)-1] ^= 0xff
	require.NoError(t, os.WriteFile(path, data, 0o600))

	m = openTest(t, dir)

	_, _, err = m.Get("k")
	require.ErrorIs(t, err, ErrCorrupt)
}

func TestIdenticalHashesFailCleanly(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), withHash(func(string, uint64) uint64 { return 42 }))

	var err error
	for i := 0; i <= slotsPerPage && err == nil; i++ {
		err = m.Add(strconv.Itoa(i), nil)
	}

	require.ErrorIs(t, err, ErrTooManyCollisions)

	// Every key that was accepted is still readable.
	for i := range slotsPerPage {
		found, err := m.Contains(strconv.Itoa(i))
		require.NoError(t, err)
		require.True(t, found)
	}
}

func TestClear(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), smallOptions()...)
	for i := range 10_000 {
		require.NoError(t, m.Add(strconv.Itoa(i), make([]byte, 300)))
	}

	require.NoError(t, m.Clear())
	require.Zero(t, m.Size())

	found, err := m.Contains("5")
	require.NoError(t, err)
	require.False(t, found)

	require.NoError(t, m.Add("5", []byte("five")))
	checkModel(t, m, map[string][]byte{"5": []byte("five")})
}

func TestRangeWhileModifying(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), smallOptions()...)
	model := make(map[string][]byte)

	for i := range 5000 {
		k := strconv.Itoa(i)
		model[k] = bytes.Repeat([]byte{byte(i)}, 500)
		require.NoError(t, m.Add(k, model[k]))
	}

	seen := make(map[string]bool)

	require.NoError(t, m.Range(func(key string, value []byte) bool {
		require.False(t, seen[key], "key %q visited twice", key)
		seen[key] = true

		require.True(t, bytes.Equal(model[key], value))

		i, err := strconv.Atoi(key)
		require.NoError(t, err)

		// Remove a key that has not been reached yet, and keep writing so
		// the active segment rotates under the iterator.
		if next := strconv.Itoa(i + 1); i%2 == 0 {
			require.NoError(t, m.Remove(next))
			delete(model, next)
		}

		require.NoError(t, m.Add("new-"+key, []byte("x")))
		model["new-"+key] = []byte("x")

		return true
	}))

	for k := range seen {
		_, ok := model[k]
		require.True(t, ok, "visited removed key %q", k)
	}

	for i := 0; i < 5000; i += 2 {
		require.True(t, seen[strconv.Itoa(i)], "missed key %d", i)
	}

	checkModel(t, m, model)

	// Stopping early works.
	calls := 0

	require.NoError(t, m.Range(func(string, []byte) bool {
		calls++

		return calls < 10
	}))
	require.Equal(t, 10, calls)
}

func TestConcurrentUse(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), smallOptions()...)

	const (
		workers = 8
		perKey  = 2000
	)

	var wg sync.WaitGroup

	for w := range workers {
		wg.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(w), 0)) //nolint:gosec // Deterministic test data.

			for i := range perKey {
				// Each worker owns its keys, so it can check its own reads.
				key := fmt.Sprintf("w%d-%d", w, rng.IntN(500))
				value := []byte(fmt.Sprintf("%s=%d", key, i))

				assert.NoError(t, m.Add(key, value))

				got, found, err := m.Get(key)
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, value, got)

				if i%10 == 0 {
					assert.NoError(t, m.Remove(key))
				}
			}
		})
	}

	wg.Wait()
	require.NoError(t, m.Close())
}

func TestInvalidOptions(t *testing.T) {
	t.Parallel()

	for _, opt := range []Option{
		WithMemoryLimit(100),
		WithSegmentSize(10),
		WithSegmentSize(1 << 40),
		WithCompactionThreshold(1),
		WithCompactionThreshold(-0.1),
	} {
		_, err := Open(t.TempDir(), opt)
		require.ErrorIs(t, err, ErrInvalidOption)
	}
}

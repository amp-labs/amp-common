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

	dmap, err := Open(dir, opts...)
	require.NoError(t, err)

	t.Cleanup(func() { _ = dmap.Close() })

	return dmap
}

// noError reports err from a worker goroutine, where require cannot stop the
// test, and says whether the worker should carry on.
func noError(t *testing.T, err error) bool {
	t.Helper()

	if err != nil {
		t.Error(err)

		return false
	}

	return true
}

func valueFor(rng *rand.Rand, key string, version int) []byte {
	size := rng.IntN(200)
	if rng.IntN(50) == 0 {
		size = rng.IntN(100_000)
	}

	value := make([]byte, size, size+len(key))
	for i := range value {
		value[i] = byte(i + version) //nolint:gosec // Wrapping is the point.
	}

	return append(value, key...)
}

// checkModel verifies that dmap holds exactly the entries in model.
func checkModel(t *testing.T, dmap *Map, model map[string][]byte) {
	t.Helper()

	require.Equal(t, len(model), dmap.Size())

	for key, want := range model {
		got, found, err := dmap.Get(key)
		require.NoError(t, err)
		require.True(t, found, "key %q missing", key)
		require.True(t, bytes.Equal(want, got), "key %q has the wrong value", key)
	}

	seen := make(map[string]bool, len(model))

	require.NoError(t, dmap.Range(func(key string, value []byte) bool {
		require.False(t, seen[key], "key %q visited twice", key)
		seen[key] = true

		require.True(t, bytes.Equal(model[key], value), "Range gave key %q the wrong value", key)

		return true
	}))
	require.Len(t, seen, len(model))
}

// randomOp applies one random operation to dmap and model and checks reads
// against the model.
func randomOp(t *testing.T, dmap *Map, rng *rand.Rand, model map[string][]byte, version int) {
	t.Helper()

	const keySpace = 3000

	key := "key-" + strconv.Itoa(rng.IntN(keySpace))

	switch r := rng.IntN(10); {
	case r < 5:
		value := valueFor(rng, key, version)
		require.NoError(t, dmap.Add(key, value))

		model[key] = value
	case r < 7:
		require.NoError(t, dmap.Remove(key))
		delete(model, key)
	case r < 9:
		got, found, err := dmap.Get(key)
		require.NoError(t, err)

		want, ok := model[key]
		require.Equal(t, ok, found)
		require.True(t, bytes.Equal(want, got))
	default:
		found, err := dmap.Contains(key)
		require.NoError(t, err)

		_, ok := model[key]
		require.Equal(t, ok, found)
	}
}

func TestRandomOperationsMatchGoMap(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		opts []Option
	}{
		{name: "default hash"},
		{name: "value cache", opts: []Option{WithMemoryLimit(4 << 20), WithValueCache(1 << 20)}},
		{name: "one shard", opts: []Option{WithShards(1)}},
		{
			// 64 distinct hash values: most keys share their full 64-bit hash
			// with others, so every lookup has to compare stored keys.
			name: "colliding hash",
			opts: []Option{withHash(func(key string, seed uint64) uint64 {
				return (xxh3.HashStringSeed(key, seed)%64 + 1) * fibonacciHash
			})},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			opts := append(smallOptions(), test.opts...)
			dmap := openTest(t, dir, opts...)
			rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // Deterministic test data.
			model := make(map[string][]byte)

			var compacted, cacheHits, pageMisses uint64

			for round := range 3 {
				for op := range 20_000 {
					randomOp(t, dmap, rng, model, round*20_000+op)
				}

				checkModel(t, dmap, model)

				stats := dmap.Stats()
				compacted += uint64(stats.CompactedBytes) //nolint:gosec // Non-negative.
				cacheHits += stats.ValueCacheHits
				pageMisses += stats.PageMisses

				require.NoError(t, dmap.Close())

				dmap = openTest(t, dir, opts...)
				checkModel(t, dmap, model)
			}

			assert.Positive(t, compacted, "compaction never ran")
			assert.Positive(t, pageMisses, "index never left the cache")

			switch test.name {
			case "one shard":
				assert.Greater(t, dmap.Stats().IndexPages, 1, "index never split")
			case "value cache":
				assert.Positive(t, cacheHits, "value cache never hit")
			}

			require.NoError(t, dmap.Compact())
			assert.Zero(t, dmap.Stats().DeadBytes)
			checkModel(t, dmap, model)
		})
	}
}

func TestManyKeysWithSmallCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dmap := openTest(t, dir, smallOptions()...)

	const keyCount = 200_000

	for i := range keyCount {
		require.NoError(t, dmap.Add(strconv.Itoa(i), []byte(strconv.Itoa(i*7))))
	}

	require.Equal(t, keyCount, dmap.Size())

	for i := range keyCount {
		value, found, err := dmap.Get(strconv.Itoa(i))
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, strconv.Itoa(i*7), string(value))
	}

	stats := dmap.Stats()
	assert.Greater(t, stats.IndexPages, stats.CachedPages, "index should not fit in the cache")

	require.NoError(t, dmap.Close())

	dmap = openTest(t, dir, smallOptions()...)
	require.Equal(t, keyCount, dmap.Size())

	value, found, err := dmap.Get("123456")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, strconv.Itoa(123456*7), string(value))
}

func TestEdgeCaseKeysAndValues(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), smallOptions()...)

	entries := map[string][]byte{
		"":                           []byte("empty key"),
		"empty value":                {},
		"big":                        bytes.Repeat([]byte("x"), 3*minSegmentSize),
		"after big":                  []byte("small"),
		string([]byte{0, 1, 2, 255}): []byte("binary key"),
	}

	for key, value := range entries {
		require.NoError(t, dmap.Add(key, value))
	}

	checkModel(t, dmap, entries)

	got, found, err := dmap.Get("empty value")
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, got)
	require.Empty(t, got)

	require.ErrorIs(t, dmap.Add(string(make([]byte, MaxKeySize+1)), nil), ErrKeyTooLarge)
}

func TestAppendValueReusesBuffer(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir())
	require.NoError(t, dmap.Add("a", []byte("alpha")))
	require.NoError(t, dmap.Add("b", []byte("beta")))

	buf := make([]byte, 0, 64)

	buf, found, err := dmap.AppendValue(buf[:0], "a")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "alpha", string(buf))

	buf, found, err = dmap.AppendValue(buf, "b")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "alphabeta", string(buf))

	buf, found, err = dmap.AppendValue(buf, "missing")
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, "alphabeta", string(buf))
}

func TestOverwritesKeepDiskUsageBounded(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), smallOptions()...)
	value := make([]byte, 1000)

	const keys = 1000 // about 1 MB live

	for round := range 50 {
		for i := range keys {
			value[0] = byte(round)
			require.NoError(t, dmap.Add(strconv.Itoa(i), value))
		}
	}

	stats := dmap.Stats()
	live := stats.SegmentBytes - stats.DeadBytes

	// Live data, plus dead data up to the threshold, plus segments that are
	// being filled or compacted.
	assert.Less(t, stats.SegmentBytes, 2*live+3*minSegmentSize, "%+v", stats)
	assert.Equal(t, keys, stats.Entries)
}

func TestReopenRequiresCleanClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	dmap, err := Open(dir, WithShards(4))
	require.NoError(t, err)
	require.NoError(t, dmap.Add("k", []byte("v")))

	// A second Open while the first is still open sees no meta file, the same
	// as after a crash.
	_, err = Open(dir)
	require.ErrorIs(t, err, ErrNotCleanlyClosed)

	require.NoError(t, dmap.Close())
	require.NoError(t, dmap.Close(), "closing twice is a no-op")

	_, _, err = dmap.Get("k")
	require.ErrorIs(t, err, ErrClosed)
	require.ErrorIs(t, dmap.Add("k", nil), ErrClosed)
	require.ErrorIs(t, dmap.Clear(), ErrClosed)

	// The shard count is fixed when the map is created.
	dmap = openTest(t, dir, WithShards(64))
	require.Len(t, dmap.shards, 4)

	value, found, err := dmap.Get("k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "v", string(value))

	// Discarding starts over even though dmap never closed.
	other := openTest(t, dir, WithDiscardExisting())
	require.Zero(t, other.Size())
}

func TestOpenLeavesOtherFilesAlone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	unrelated := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(unrelated, []byte("keep me"), 0o600))

	dmap := openTest(t, dir, WithDiscardExisting())
	require.NoError(t, dmap.Add("k", nil))
	require.NoError(t, dmap.Clear())
	require.NoError(t, dmap.Close())

	_ = openTest(t, dir, WithDiscardExisting())

	_, err := os.Stat(unrelated)
	require.NoError(t, err)
}

func TestCorruptRecordIsReported(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dmap := openTest(t, dir)
	require.NoError(t, dmap.Add("k", []byte("value")))
	require.NoError(t, dmap.Close())

	path := segmentPath(dir, 0)
	data, err := os.ReadFile(path) //nolint:gosec // Path inside the test's temp dir.
	require.NoError(t, err)

	data[len(data)-1] ^= 0xff
	require.NoError(t, os.WriteFile(path, data, 0o600)) //nolint:gosec // Path inside the test's temp dir.

	dmap = openTest(t, dir)

	_, _, err = dmap.Get("k")
	require.ErrorIs(t, err, ErrCorrupt)
}

func TestIdenticalHashesFailCleanly(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), withHash(func(string, uint64) uint64 { return 42 }))

	var err error
	for i := 0; i <= slotsPerPage && err == nil; i++ {
		err = dmap.Add(strconv.Itoa(i), nil)
	}

	require.ErrorIs(t, err, ErrTooManyCollisions)

	// Every key that was accepted is still readable.
	for i := range slotsPerPage {
		found, err := dmap.Contains(strconv.Itoa(i))
		require.NoError(t, err)
		require.True(t, found)
	}
}

func TestClear(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), smallOptions()...)
	for i := range 10_000 {
		require.NoError(t, dmap.Add(strconv.Itoa(i), make([]byte, 300)))
	}

	require.NoError(t, dmap.Clear())
	require.Zero(t, dmap.Size())

	found, err := dmap.Contains("5")
	require.NoError(t, err)
	require.False(t, found)

	require.NoError(t, dmap.Add("5", []byte("five")))
	checkModel(t, dmap, map[string][]byte{"5": []byte("five")})
}

func TestRangeWhileModifying(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), smallOptions()...)
	model := make(map[string][]byte)

	for i := range 5000 {
		key := strconv.Itoa(i)
		model[key] = bytes.Repeat([]byte{byte(i)}, 500) //nolint:gosec // Wrapping is fine.
		require.NoError(t, dmap.Add(key, model[key]))
	}

	seen := make(map[string]bool)

	require.NoError(t, dmap.Range(func(key string, value []byte) bool {
		require.False(t, seen[key], "key %q visited twice", key)
		seen[key] = true

		require.True(t, bytes.Equal(model[key], value))

		i, err := strconv.Atoi(key)
		require.NoError(t, err)

		// Remove a key that has not been reached yet, and keep writing so
		// the active segment rotates under the iterator.
		if next := strconv.Itoa(i + 1); i%2 == 0 {
			require.NoError(t, dmap.Remove(next))
			delete(model, next)
		}

		require.NoError(t, dmap.Add("new-"+key, []byte("x")))
		model["new-"+key] = []byte("x")

		return true
	}))

	for key := range seen {
		_, ok := model[key]
		require.True(t, ok, "visited removed key %q", key)
	}

	for i := 0; i < 5000; i += 2 {
		require.True(t, seen[strconv.Itoa(i)], "missed key %d", i)
	}

	checkModel(t, dmap, model)

	// Stopping early works.
	calls := 0

	require.NoError(t, dmap.Range(func(string, []byte) bool {
		calls++

		return calls < 10
	}))
	require.Equal(t, 10, calls)
}

func TestConcurrentUse(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), smallOptions()...)

	const (
		workers   = 8
		perWorker = 2000
	)

	var group sync.WaitGroup

	for worker := range workers {
		group.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(worker), 0)) //nolint:gosec // Deterministic test data.

			for version := range perWorker {
				// Each worker owns its keys, so it can check its own reads.
				key := fmt.Sprintf("w%d-%d", worker, rng.IntN(500))
				value := fmt.Appendf(nil, "%s=%d", key, version)

				if !noError(t, dmap.Add(key, value)) {
					return
				}

				got, found, err := dmap.Get(key)
				if !noError(t, err) {
					return
				}

				if !found || !bytes.Equal(value, got) {
					t.Errorf("key %s: found %v, value %q, want %q", key, found, got, value)

					return
				}

				if version%10 == 0 && !noError(t, dmap.Remove(key)) {
					return
				}
			}
		})
	}

	group.Wait()
	require.NoError(t, dmap.Close())
}

func TestInvalidOptions(t *testing.T) {
	t.Parallel()

	for _, opt := range []Option{
		WithMemoryLimit(100),
		WithSegmentSize(10),
		WithSegmentSize(1 << 40),
		WithCompactionThreshold(1),
		WithCompactionThreshold(-0.1),
		WithShards(0),
		WithShards(3),
		WithShards(1024),
		WithValueCache(-1),
		WithValueCache(DefaultMemoryLimit),
	} {
		_, err := Open(t.TempDir(), opt)
		require.ErrorIs(t, err, ErrInvalidOption)
	}
}

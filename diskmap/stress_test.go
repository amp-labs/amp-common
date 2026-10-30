package diskmap

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stressValue builds a value that names its own key, so any reader can tell
// whether a value it got belongs to the key it asked for.
func stressValue(rng *rand.Rand, key string, version int) []byte {
	pad := bytes.Repeat([]byte{'.'}, rng.IntN(3000))

	return fmt.Appendf(nil, "%s=%d|%s", key, version, pad)
}

func valueOwner(value []byte) string {
	owner, _, _ := strings.Cut(string(value), "=")

	return owner
}

// TestStressConcurrentOperations runs writers, readers, iterators and
// explicit compactions at once with tiny caches and segments, so page
// eviction, segment rotation and background compaction all overlap with
// foreground work.
func TestStressConcurrentOperations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	opts := append(smallOptions(), WithCompactionThreshold(0.3), WithShards(4))
	m := openTest(t, dir, opts...)

	const (
		writers  = 6
		keysEach = 400
		opsEach  = 6000
	)

	var (
		wg      sync.WaitGroup
		done    atomic.Bool
		models  = make([]map[string][]byte, writers)
		ranges  atomic.Int64
		compact atomic.Int64
	)

	for w := range writers {
		models[w] = make(map[string][]byte)

		wg.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(w), 7)) //nolint:gosec // Deterministic test data.
			model := models[w]

			for op := range opsEach {
				key := fmt.Sprintf("w%d-k%d", w, rng.IntN(keysEach))

				switch r := rng.IntN(10); {
				case r < 5:
					v := stressValue(rng, key, op)
					if !assert.NoError(t, m.Add(key, v)) {
						return
					}

					model[key] = v
				case r < 7:
					if !assert.NoError(t, m.Remove(key)) {
						return
					}

					delete(model, key)
				default:
					got, found, err := m.Get(key)
					if !assert.NoError(t, err) {
						return
					}

					want, ok := model[key]
					assert.Equal(t, ok, found, "key %s", key)
					assert.True(t, bytes.Equal(want, got), "key %s", key)
				}
			}
		})
	}

	var background sync.WaitGroup

	background.Go(func() {
		for !done.Load() {
			seen := make(map[string]bool)

			err := m.Range(func(key string, value []byte) bool {
				assert.False(t, seen[key], "Range visited %s twice", key)
				seen[key] = true

				assert.Equal(t, key, valueOwner(value), "Range gave %s another key's value", key)

				return true
			})
			assert.NoError(t, err)
			ranges.Add(1)
		}
	})

	background.Go(func() {
		for !done.Load() {
			assert.NoError(t, m.Compact())
			compact.Add(1)
			time.Sleep(5 * time.Millisecond)
		}
	})

	wg.Wait()
	done.Store(true)
	background.Wait()

	require.False(t, t.Failed())
	assert.Positive(t, ranges.Load())
	assert.Positive(t, compact.Load())

	all := make(map[string][]byte)
	for _, model := range models {
		for k, v := range model {
			all[k] = v
		}
	}

	checkModel(t, m, all)
	require.NoError(t, m.Compact())
	checkModel(t, m, all)

	require.NoError(t, m.Close())

	m = openTest(t, dir, opts...)
	checkModel(t, m, all)
}

// TestStressClearDuringWrites checks that Clear racing with writers and
// readers never leaves the index pointing at deleted data.
func TestStressClearDuringWrites(t *testing.T) {
	t.Parallel()

	m := openTest(t, t.TempDir(), append(smallOptions(), WithShards(4))...)

	var (
		wg   sync.WaitGroup
		done atomic.Bool
	)

	for w := range 4 {
		wg.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(w), 9)) //nolint:gosec // Deterministic test data.

			for op := 0; !done.Load(); op++ {
				key := "k" + strconv.Itoa(rng.IntN(300))

				if rng.IntN(2) == 0 {
					assert.NoError(t, m.Add(key, stressValue(rng, key, op)))

					continue
				}

				got, found, err := m.Get(key)
				if assert.NoError(t, err) && found {
					assert.Equal(t, key, valueOwner(got))
				}
			}
		})
	}

	for range 20 {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, m.Clear())
	}

	done.Store(true)
	wg.Wait()

	// Whatever survived the last Clear must be readable and consistent.
	require.NoError(t, m.Range(func(key string, value []byte) bool {
		assert.Equal(t, key, valueOwner(value))

		return true
	}))

	count := 0

	require.NoError(t, m.Range(func(string, []byte) bool {
		count++

		return true
	}))
	require.Equal(t, m.Size(), count)
}

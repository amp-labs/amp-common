package diskmap

import (
	"bytes"
	"fmt"
	"maps"
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

// stressWriter adds, removes and reads keys only it writes, checking every
// read against its own model.
func stressWriter(t *testing.T, dmap *Map, worker int, model map[string][]byte) {
	t.Helper()

	const (
		keysEach = 400
		opsEach  = 6000
	)

	rng := rand.New(rand.NewPCG(uint64(worker), 7)) //nolint:gosec // Deterministic test data.

	for version := range opsEach {
		key := fmt.Sprintf("w%d-k%d", worker, rng.IntN(keysEach))

		switch r := rng.IntN(10); {
		case r < 5:
			value := stressValue(rng, key, version)
			if !noError(t, dmap.Add(key, value)) {
				return
			}

			model[key] = value
		case r < 7:
			if !noError(t, dmap.Remove(key)) {
				return
			}

			delete(model, key)
		default:
			got, found, err := dmap.Get(key)
			if !noError(t, err) {
				return
			}

			want, ok := model[key]
			if ok != found || !bytes.Equal(want, got) {
				t.Errorf("key %s: found %v, want %v; values equal %v", key, found, ok, bytes.Equal(want, got))

				return
			}
		}
	}
}

// stressRanger iterates until done, checking that no key is visited twice
// and that every value belongs to its key.
func stressRanger(t *testing.T, dmap *Map, done *atomic.Bool) int {
	t.Helper()

	ranges := 0

	for !done.Load() {
		seen := make(map[string]bool)

		err := dmap.Range(func(key string, value []byte) bool {
			if seen[key] {
				t.Errorf("Range visited %s twice", key)
			}

			seen[key] = true

			if owner := valueOwner(value); owner != key {
				t.Errorf("Range gave %s the value of %s", key, owner)
			}

			return true
		})
		if !noError(t, err) {
			return ranges
		}

		ranges++
	}

	return ranges
}

// TestStressConcurrentOperations runs writers, readers, iterators and
// explicit compactions at once with tiny caches and segments, so page
// eviction, segment rotation and background compaction all overlap with
// foreground work.
func TestStressConcurrentOperations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	opts := append(smallOptions(), WithCompactionThreshold(0.3), WithShards(4),
		WithMemoryLimit(2<<20), WithValueCache(256<<10))
	dmap := openTest(t, dir, opts...)

	const writers = 6

	var (
		group       sync.WaitGroup
		background  sync.WaitGroup
		done        atomic.Bool
		models      = make([]map[string][]byte, writers)
		ranges      int
		compactions int
	)

	for worker := range writers {
		models[worker] = make(map[string][]byte)

		group.Go(func() { stressWriter(t, dmap, worker, models[worker]) })
	}

	background.Go(func() { ranges = stressRanger(t, dmap, &done) })
	background.Go(func() {
		for !done.Load() && noError(t, dmap.Compact()) {
			compactions++

			time.Sleep(5 * time.Millisecond)
		}
	})

	group.Wait()
	done.Store(true)
	background.Wait()

	require.False(t, t.Failed())
	assert.Positive(t, ranges)
	assert.Positive(t, compactions)

	all := make(map[string][]byte)
	for _, model := range models {
		maps.Copy(all, model)
	}

	checkModel(t, dmap, all)
	require.NoError(t, dmap.Compact())
	checkModel(t, dmap, all)

	require.NoError(t, dmap.Close())

	dmap = openTest(t, dir, opts...)
	checkModel(t, dmap, all)
}

// TestStressClearDuringWrites checks that Clear racing with writers and
// readers never leaves the index pointing at deleted data.
func TestStressClearDuringWrites(t *testing.T) {
	t.Parallel()

	dmap := openTest(t, t.TempDir(), append(smallOptions(), WithShards(4))...)

	var (
		group sync.WaitGroup
		done  atomic.Bool
	)

	for worker := range 4 {
		group.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(worker), 9)) //nolint:gosec // Deterministic test data.

			for version := 0; !done.Load(); version++ {
				key := "k" + strconv.Itoa(rng.IntN(300))

				if rng.IntN(2) == 0 {
					if !noError(t, dmap.Add(key, stressValue(rng, key, version))) {
						return
					}

					continue
				}

				got, found, err := dmap.Get(key)
				if !noError(t, err) {
					return
				}

				if owner := valueOwner(got); found && owner != key {
					t.Errorf("Get(%s) returned the value of %s", key, owner)
				}
			}
		})
	}

	for range 20 {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, dmap.Clear())
	}

	done.Store(true)
	group.Wait()

	// Whatever survived the last Clear must be readable and consistent.
	count := 0

	require.NoError(t, dmap.Range(func(key string, value []byte) bool {
		count++

		assert.Equal(t, key, valueOwner(value))

		return true
	}))
	require.Equal(t, dmap.Size(), count)
}

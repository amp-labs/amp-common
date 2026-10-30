package diskmap

import (
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
)

// Benchmarks use benchKeys keys of benchValueSize bytes, about 100 MiB of
// data, so the OS keeps it cached and they measure the map rather than the
// disk. The cmd-style harness in the PR description covers larger data.
const (
	benchKeys      = 100_000
	benchValueSize = 1024
)

func benchKey(i int) string {
	return "user:" + strconv.FormatUint(uint64(i)*fibonacciHash, 16) //nolint:gosec // Non-negative.
}

func loadedMap(b *testing.B, opts ...Option) *Map {
	b.Helper()

	dmap, err := Open(b.TempDir(), opts...)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = dmap.Close() })

	value := make([]byte, benchValueSize)
	for i := range benchKeys {
		err = dmap.Add(benchKey(i), value)
		if err != nil {
			b.Fatal(err)
		}
	}

	return dmap
}

func BenchmarkAdd(b *testing.B) {
	dmap, err := Open(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = dmap.Close() })

	value := make([]byte, benchValueSize)
	b.SetBytes(benchValueSize)

	i := 0
	for b.Loop() {
		err = dmap.Add(benchKey(i), value)
		if err != nil {
			b.Fatal(err)
		}

		i++
	}
}

func BenchmarkOverwrite(b *testing.B) {
	dmap := loadedMap(b)
	value := make([]byte, benchValueSize)
	rng := rand.New(rand.NewPCG(1, 1)) //nolint:gosec // Benchmark data.

	b.SetBytes(benchValueSize)

	for b.Loop() {
		err := dmap.Add(benchKey(rng.IntN(benchKeys)), value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	dmap := loadedMap(b)
	rng := rand.New(rand.NewPCG(1, 1)) //nolint:gosec // Benchmark data.

	b.SetBytes(benchValueSize)

	for b.Loop() {
		_, found, err := dmap.Get(benchKey(rng.IntN(benchKeys)))
		if err != nil || !found {
			b.Fatal(err, found)
		}
	}
}

func BenchmarkAppendValue(b *testing.B) {
	dmap := loadedMap(b)
	rng := rand.New(rand.NewPCG(1, 1)) //nolint:gosec // Benchmark data.
	buf := make([]byte, 0, benchValueSize+64)

	b.SetBytes(benchValueSize)

	for b.Loop() {
		var (
			found bool
			err   error
		)

		buf, found, err = dmap.AppendValue(buf[:0], benchKey(rng.IntN(benchKeys)))
		if err != nil || !found {
			b.Fatal(err, found)
		}
	}
}

func BenchmarkAppendValueSkewed(b *testing.B) {
	for _, test := range []struct {
		name string
		opts []Option
	}{
		{name: "no value cache"},
		{name: "value cache", opts: []Option{WithValueCache(16 << 20)}},
	} {
		b.Run(test.name, func(b *testing.B) {
			dmap := loadedMap(b, test.opts...)
			zipf := rand.NewZipf(rand.New(rand.NewPCG(1, 1)), 1.1, 1, benchKeys-1) //nolint:gosec // Benchmark data.
			buf := make([]byte, 0, benchValueSize+64)

			b.SetBytes(benchValueSize)

			for b.Loop() {
				var (
					found bool
					err   error
				)

				buf, found, err = dmap.AppendValue(buf[:0], benchKey(int(zipf.Uint64()))) //nolint:gosec // Bounded.
				if err != nil || !found {
					b.Fatal(err, found)
				}
			}
		})
	}
}

func BenchmarkAppendValueParallel(b *testing.B) {
	dmap := loadedMap(b)

	b.SetBytes(benchValueSize)
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewPCG(rand.Uint64(), 1)) //nolint:gosec // Benchmark data.
		buf := make([]byte, 0, benchValueSize+64)

		for pb.Next() {
			var (
				found bool
				err   error
			)

			buf, found, err = dmap.AppendValue(buf[:0], benchKey(rng.IntN(benchKeys)))
			if err != nil || !found {
				b.Error(err, found)

				return
			}
		}
	})
}

func BenchmarkMixedParallel(b *testing.B) {
	dmap := loadedMap(b)
	value := make([]byte, benchValueSize)

	b.SetBytes(benchValueSize)
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewPCG(rand.Uint64(), 1)) //nolint:gosec // Benchmark data.
		buf := make([]byte, 0, benchValueSize+64)

		for i := 0; pb.Next(); i++ {
			key := benchKey(rng.IntN(benchKeys))

			var err error
			if i%4 == 0 {
				err = dmap.Add(key, value)
			} else {
				buf, _, err = dmap.AppendValue(buf[:0], key)
			}

			if err != nil {
				b.Error(err)

				return
			}
		}
	})
}

// BenchmarkGoMapGet is a baseline: a plain map behind a mutex, holding
// everything in memory.
func BenchmarkGoMapGet(b *testing.B) {
	var lock sync.RWMutex

	goMap := make(map[string][]byte, benchKeys)
	for i := range benchKeys {
		goMap[benchKey(i)] = make([]byte, benchValueSize)
	}

	rng := rand.New(rand.NewPCG(1, 1)) //nolint:gosec // Benchmark data.
	buf := make([]byte, 0, benchValueSize)

	b.SetBytes(benchValueSize)

	for b.Loop() {
		key := benchKey(rng.IntN(benchKeys))

		lock.RLock()

		buf = append(buf[:0], goMap[key]...)

		lock.RUnlock()
	}
}

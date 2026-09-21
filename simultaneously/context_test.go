package simultaneously

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/amp-labs/amp-common/set"
	"github.com/amp-labs/amp-common/should"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errCtxJobFailed = errors.New("ambient job failed")
	errCtxSentinel  = errors.New("sentinel")
)

// recordingExecutor wraps an Executor and counts how it was used, so tests can
// prove which executor a Do* call actually dispatched onto and whether it took
// ownership by closing it.
type recordingExecutor struct {
	inner      Executor
	goCalls    atomic.Int32
	closeCalls atomic.Int32
}

func newRecordingExecutor(maxConcurrent int) *recordingExecutor {
	return &recordingExecutor{inner: NewDefaultExecutor(maxConcurrent)}
}

func (r *recordingExecutor) GoContext(ctx context.Context, fn func(context.Context) error, done func(error)) {
	r.goCalls.Add(1)

	r.inner.GoContext(ctx, fn, done)
}

func (r *recordingExecutor) Go(fn func(context.Context) error, done func(error)) {
	r.goCalls.Add(1)

	r.inner.Go(fn, done)
}

func (r *recordingExecutor) Close() error {
	r.closeCalls.Add(1)

	return r.inner.Close()
}

// peakTracker records the high-water mark of jobs running at the same time,
// which is how these tests observe the concurrency limit that was actually in
// force rather than the one that was requested.
type peakTracker struct {
	mu      sync.Mutex
	current int
	peak    int
}

func (p *peakTracker) enter() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current++
	if p.current > p.peak {
		p.peak = p.current
	}
}

func (p *peakTracker) exit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current--
}

func (p *peakTracker) observed() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.peak
}

// job returns a Job that holds a concurrency slot for a beat, long enough for
// overlapping jobs to be observed by the tracker.
func (p *peakTracker) job() Job {
	return func(ctx context.Context) error {
		p.enter()
		defer p.exit()

		time.Sleep(50 * time.Millisecond)

		return nil
	}
}

func TestGetExecutor_NilContext(t *testing.T) {
	t.Parallel()

	// A nil context must be reported as "no executor" rather than panicking, so
	// that DoCtx stays callable from not-yet-context-plumbed code paths.
	exec, ok := GetExecutor(nil) //nolint:staticcheck // deliberately passing a nil context

	assert.False(t, ok)
	assert.Nil(t, exec)
}

func TestGetExecutor_NoExecutorOnContext(t *testing.T) {
	t.Parallel()

	exec, ok := GetExecutor(t.Context())

	assert.False(t, ok)
	assert.Nil(t, exec)
}

func TestGetExecutor_RoundTrip(t *testing.T) {
	t.Parallel()

	want := NewDefaultExecutor(2)
	defer should.Close(want, "closing executor")

	got, ok := GetExecutor(WithExecutor(t.Context(), want))

	require.True(t, ok)
	assert.Same(t, want, got)
}

func TestGetExecutor_SurvivesContextDerivation(t *testing.T) {
	t.Parallel()

	want := NewDefaultExecutor(2)
	defer should.Close(want, "closing executor")

	// The executor has to survive the ordinary context plumbing that happens
	// between the call that attaches it and the DoCtx that consumes it.
	ctx := WithExecutor(t.Context(), want)
	ctx = context.WithValue(ctx, contextKey("unrelated"), "value")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, time.Minute)
	defer cancel()

	got, ok := GetExecutor(ctx)

	require.True(t, ok)
	assert.Same(t, want, got)
}

func TestGetExecutor_InnermostWins(t *testing.T) {
	t.Parallel()

	outer := NewDefaultExecutor(2)
	defer should.Close(outer, "closing outer executor")

	inner := NewDefaultExecutor(2)
	defer should.Close(inner, "closing inner executor")

	// Re-attaching shadows the previous executor, which is how a subtree hands
	// its own pool to the work below it.
	ctx := WithExecutor(WithExecutor(t.Context(), outer), inner)

	got, ok := GetExecutor(ctx)

	require.True(t, ok)
	assert.Same(t, inner, got)
}

func TestGetExecutor_KeyIsNotCollidable(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(2)
	defer should.Close(exec, "closing executor")

	ctx := WithExecutor(t.Context(), exec)

	// The private contextKey type is what keeps a same-named key from another
	// package out of our slot, and ours out of theirs.
	//nolint:staticcheck // a raw string key is exactly what must not match
	assert.Nil(t, ctx.Value("executor"))
	assert.NotNil(t, ctx.Value(executorContextKey))
	assert.Nil(t, ctx.Value(contextKey("other")))
}

func TestGetExecutor_NilExecutor(t *testing.T) {
	t.Parallel()

	// Storing a nil Executor is filtered out: contexts.GetValue fails its type
	// assertion on the nil interface, so callers fall back to their own executor
	// instead of dispatching onto nothing.
	exec, ok := GetExecutor(WithExecutor(t.Context(), nil))

	assert.False(t, ok)
	assert.Nil(t, exec)
}

func TestGetExecutor_TypedNilExecutorIsFiltered(t *testing.T) {
	t.Parallel()

	// A typed nil is a non-nil interface holding a nil pointer, so neither the
	// plain `val == nil` comparison nor GetValue's type assertion rejects it --
	// utils.IsNilish is what looks through the interface. Without that, a typed
	// nil would be reported as a usable executor and DoCtx would deadlock on it:
	// GoContext panics on the nil receiver, and that panic unwinds into
	// collector.cleanup, which waits forever on an already-incremented WaitGroup.
	var typedNil *defaultExecutor

	exec, ok := GetExecutor(WithExecutor(t.Context(), typedNil))

	assert.False(t, ok)
	assert.Nil(t, exec)
}

func TestDoCtx_TypedNilAmbientExecutorFallsBack(t *testing.T) {
	t.Parallel()

	// Filtering the typed nil is what makes this survivable: DoCtx sees "no
	// ambient executor" and builds its own instead of dispatching onto nothing.
	var typedNil *defaultExecutor

	var ran atomic.Int32

	count := func(ctx context.Context) error {
		ran.Add(1)

		return nil
	}

	err := DoCtx(WithExecutor(t.Context(), typedNil), 2, count, count)

	require.NoError(t, err)
	assert.Equal(t, int32(2), ran.Load())
}

func TestDoCtx_UsesAmbientExecutor(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(4)
	defer should.Close(rec, "closing recording executor")

	var ran atomic.Int32

	count := func(ctx context.Context) error {
		ran.Add(1)

		return nil
	}

	err := DoCtx(WithExecutor(t.Context(), rec), 4, count, count, count)

	require.NoError(t, err)
	assert.Equal(t, int32(3), ran.Load())
	assert.Equal(t, int32(3), rec.goCalls.Load(), "every job should dispatch on the ambient executor")
}

func TestDoCtx_DoesNotUseAmbientExecutorWhenAbsent(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(4)
	defer should.Close(rec, "closing recording executor")

	// Same executor exists, but it was never attached to this context.
	err := DoCtx(t.Context(), 2,
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	)

	require.NoError(t, err)
	assert.Equal(t, int32(0), rec.goCalls.Load())
}

func TestDoCtx_DoesNotCloseAmbientExecutor(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(2)
	defer should.Close(rec, "closing recording executor")

	ctx := WithExecutor(t.Context(), rec)

	require.NoError(t, DoCtx(ctx, 2, func(ctx context.Context) error { return nil }))
	assert.Equal(t, int32(0), rec.closeCalls.Load(), "ownership stays with the caller that attached it")

	// The executor is still live, which is the whole point of sharing it.
	require.NoError(t, DoCtx(ctx, 2, func(ctx context.Context) error { return nil }))
	assert.Equal(t, int32(2), rec.goCalls.Load())
}

func TestDoCtx_AmbientExecutorOverridesMaxConcurrent(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(1)
	defer should.Close(exec, "closing executor")

	tracker := &peakTracker{}

	// maxConcurrent of 100 is ignored: the ambient executor's single slot wins.
	err := DoCtx(WithExecutor(t.Context(), exec), 100,
		tracker.job(), tracker.job(), tracker.job(),
	)

	require.NoError(t, err)
	assert.Equal(t, 1, tracker.observed())
}

func TestDoCtx_MaxConcurrentHonoredWithoutAmbientExecutor(t *testing.T) {
	t.Parallel()

	tracker := &peakTracker{}

	// Contrast with the test above: with no ambient executor, maxConcurrent is
	// what governs, so these jobs do overlap.
	err := DoCtx(t.Context(), 3, tracker.job(), tracker.job(), tracker.job())

	require.NoError(t, err)
	assert.Greater(t, tracker.observed(), 1)
}

func TestDoCtx_AmbientExecutorNoJobs(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(2)
	defer should.Close(rec, "closing recording executor")

	err := DoCtx(WithExecutor(t.Context(), rec), 2)

	require.NoError(t, err)
	assert.Equal(t, int32(0), rec.goCalls.Load())
}

func TestDoCtx_AmbientExecutorPropagatesError(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(3)
	defer should.Close(exec, "closing executor")

	err := DoCtx(WithExecutor(t.Context(), exec), 3,
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return errCtxJobFailed },
	)

	require.ErrorIs(t, err, errCtxJobFailed)
}

func TestDoCtx_AmbientExecutorCancelsSiblingsOnError(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(3)
	defer should.Close(exec, "closing executor")

	canceled := make(chan struct{})

	err := DoCtx(WithExecutor(t.Context(), exec), 3,
		func(ctx context.Context) error {
			<-ctx.Done()
			close(canceled)

			return ctx.Err()
		},
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)

			return errCtxJobFailed
		},
	)

	require.ErrorIs(t, err, errCtxJobFailed)

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("sibling job was never canceled")
	}
}

func TestDoCtx_AmbientExecutorRecoversPanic(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(2)
	defer should.Close(exec, "closing executor")

	err := DoCtx(WithExecutor(t.Context(), exec), 2,
		func(ctx context.Context) error { panic(errCtxSentinel) },
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errCtxSentinel)
	assert.Contains(t, err.Error(), "recovered from panic")
}

func TestDoCtx_AmbientExecutorAlreadyClosed(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(2)
	require.NoError(t, exec.Close())

	var ran atomic.Int32

	// A closed ambient executor surfaces as an error per job rather than a
	// silent no-op or a fallback to a fresh executor.
	count := func(ctx context.Context) error {
		ran.Add(1)

		return nil
	}

	err := DoCtx(WithExecutor(t.Context(), exec), 2, count, count)

	require.ErrorIs(t, err, ErrExecutorClosed)
	assert.Equal(t, int32(0), ran.Load())
}

func TestDoCtx_NestedCallsShareAmbientExecutor(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(8)
	defer should.Close(rec, "closing recording executor")

	// Jobs receive a context derived from the one DoCtx was given, so the
	// ambient executor reaches nested calls too.
	err := DoCtx(WithExecutor(t.Context(), rec), 8,
		func(ctx context.Context) error {
			return DoCtx(ctx, 8,
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
			)
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(3), rec.goCalls.Load(), "1 outer + 2 inner jobs on one executor")
	assert.Equal(t, int32(0), rec.closeCalls.Load(), "the nested call must not close the shared executor")
}

func TestDoCtx_NestedCallsDeadlockOnSaturatedAmbientExecutor(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(1)
	defer should.Close(exec, "closing executor")

	// The documented hazard: the outer job holds the only slot, and its nested
	// DoCtx queues for a slot that cannot free up until the outer job returns.
	// Nothing but context cancellation breaks the cycle -- hence the deadline.
	ctx, cancel := context.WithTimeout(WithExecutor(t.Context(), exec), 200*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- DoCtx(ctx, 1, func(ctx context.Context) error {
			return DoCtx(ctx, 1, func(ctx context.Context) error { return nil })
		})
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(10 * time.Second):
		t.Fatal("nested DoCtx never returned; the deadline should have broken the deadlock")
	}
}

// mapFamilyCase runs one Map*/FlatMap* entry point over a fixed three-element
// input, invoking work once per element so the test can observe how the call was
// scheduled. Every case must behave identically with respect to the ambient
// executor; that uniformity is the property under test.
type mapFamilyCase struct {
	name string
	run  func(ctx context.Context, maxConcurrent int, work func()) error
}

//nolint:funlen // a flat table of every entry point reads better than clever factoring
func mapFamilyCases(t *testing.T) []mapFamilyCase {
	t.Helper()

	ints := []int{1, 2, 3}

	goMap := map[string]int{"a": 1, "b": 2, "c": 3}

	newHashSet := func() set.Set[hashing.HashableInt] {
		out := set.NewSet[hashing.HashableInt](hashing.Sha256)
		for _, i := range ints {
			require.NoError(t, out.Add(hashing.HashableInt(i)))
		}

		return out
	}

	newOrderedSet := func() set.OrderedSet[hashing.HashableInt] {
		out := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
		for _, i := range ints {
			require.NoError(t, out.Add(hashing.HashableInt(i)))
		}

		return out
	}

	newHashMap := func() maps.Map[maps.Key[string], int] {
		out := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
		for k, v := range goMap {
			require.NoError(t, out.Add(maps.Key[string]{Key: k}, v))
		}

		return out
	}

	newOrderedMap := func() maps.OrderedMap[maps.Key[string], int] {
		out := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
		for k, v := range goMap {
			require.NoError(t, out.Add(maps.Key[string]{Key: k}, v))
		}

		return out
	}

	return []mapFamilyCase{
		{"MapSliceCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := MapSliceCtx(ctx, mc, ints, func(ctx context.Context, v int) (int, error) {
				work()

				return v, nil
			})

			return err
		}},
		{"FlatMapSliceCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := FlatMapSliceCtx(ctx, mc, ints, func(ctx context.Context, v int) ([]int, error) {
				work()

				return []int{v}, nil
			})

			return err
		}},
		{"MapGoMapCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := MapGoMapCtx(ctx, mc, goMap,
				func(ctx context.Context, k string, v int) (string, int, error) {
					work()

					return k, v, nil
				})

			return err
		}},
		{"FlatMapGoMapCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := FlatMapGoMapCtx(ctx, mc, goMap,
				func(ctx context.Context, k string, v int) (map[string]int, error) {
					work()

					return map[string]int{k: v}, nil
				})

			return err
		}},
		{"MapMapCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := MapMapCtx(ctx, mc, newHashMap(),
				func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
					work()

					return k, v, nil
				})

			return err
		}},
		{"FlatMapMapCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := FlatMapMapCtx(ctx, mc, newHashMap(),
				func(ctx context.Context, k maps.Key[string], v int) (maps.Map[maps.Key[string], int], error) {
					work()

					out := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)

					return out, out.Add(k, v)
				})

			return err
		}},
		{"MapSetCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := MapSetCtx(ctx, mc, newHashSet(),
				func(ctx context.Context, e hashing.HashableInt) (hashing.HashableInt, error) {
					work()

					return e, nil
				})

			return err
		}},
		{"FlatMapSetCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := FlatMapSetCtx(ctx, mc, newHashSet(),
				func(ctx context.Context, e hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
					work()

					out := set.NewSet[hashing.HashableInt](hashing.Sha256)

					return out, out.Add(e)
				})

			return err
		}},
		{"MapOrderedMapCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := MapOrderedMapCtx(ctx, mc, newOrderedMap(),
				func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
					work()

					return k, v, nil
				})

			return err
		}},
		{"FlatMapOrderedMapCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := FlatMapOrderedMapCtx(ctx, mc, newOrderedMap(),
				func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
					work()

					out := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)

					return out, out.Add(k, v)
				})

			return err
		}},
		{"MapOrderedSetCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := MapOrderedSetCtx(ctx, mc, newOrderedSet(),
				func(ctx context.Context, e hashing.HashableInt) (hashing.HashableInt, error) {
					work()

					return e, nil
				})

			return err
		}},
		{"FlatMapOrderedSetCtx", func(ctx context.Context, mc int, work func()) error {
			_, err := FlatMapOrderedSetCtx(ctx, mc, newOrderedSet(),
				func(ctx context.Context, e hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
					work()

					out := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)

					return out, out.Add(e)
				})

			return err
		}},
	}
}

func TestMapFamily_UsesAmbientExecutor(t *testing.T) {
	t.Parallel()

	for _, tc := range mapFamilyCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// One slot, so the ambient executor's limit is directly observable:
			// if it is in force nothing can overlap, whatever maxConcurrent says.
			rec := newRecordingExecutor(1)
			defer should.Close(rec, "closing recording executor")

			tracker := &peakTracker{}

			err := tc.run(WithExecutor(t.Context(), rec), 100, func() {
				tracker.enter()
				defer tracker.exit()

				time.Sleep(20 * time.Millisecond)
			})

			require.NoError(t, err)
			assert.Equal(t, int32(3), rec.goCalls.Load(), "each element dispatches on the ambient executor")
			assert.Equal(t, 1, tracker.observed(), "the ambient limit governs, not maxConcurrent")
			assert.Equal(t, int32(0), rec.closeCalls.Load(), "ownership stays with the caller that attached it")
		})
	}
}

func TestMapFamily_MaxConcurrentHonoredWithoutAmbientExecutor(t *testing.T) {
	t.Parallel()

	for _, tc := range mapFamilyCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := &peakTracker{}

			// Contrast with the test above: with no ambient executor, maxConcurrent
			// governs and the three elements overlap.
			err := tc.run(t.Context(), 3, func() {
				tracker.enter()
				defer tracker.exit()

				time.Sleep(50 * time.Millisecond)
			})

			require.NoError(t, err)
			assert.Greater(t, tracker.observed(), 1)
		})
	}
}

func TestMapFamily_AmbientExecutorIsReusable(t *testing.T) {
	t.Parallel()

	for _, tc := range mapFamilyCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := newRecordingExecutor(3)
			defer should.Close(rec, "closing recording executor")

			ctx := WithExecutor(t.Context(), rec)
			noop := func() {}

			// Since the entry point does not close what it did not create, the same
			// executor must still serve a second call.
			require.NoError(t, tc.run(ctx, 3, noop))
			require.NoError(t, tc.run(ctx, 3, noop))

			assert.Equal(t, int32(6), rec.goCalls.Load())
			assert.Equal(t, int32(0), rec.closeCalls.Load())
		})
	}
}

func TestMapSliceCtx_AmbientExecutorAlreadyClosed(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(2)
	require.NoError(t, exec.Close())

	// Same contract as DoCtx: a closed ambient executor is an error, not a silent
	// fallback to a fresh one.
	_, err := MapSliceCtx(WithExecutor(t.Context(), exec), 2, []int{1, 2},
		func(ctx context.Context, v int) (int, error) { return v, nil })

	require.ErrorIs(t, err, ErrExecutorClosed)
}

func TestMapSliceCtx_AmbientExecutorPropagatesError(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(3)
	defer should.Close(exec, "closing executor")

	_, err := MapSliceCtx(WithExecutor(t.Context(), exec), 3, []int{1, 2, 3},
		func(ctx context.Context, v int) (int, error) {
			if v == 2 {
				return 0, errCtxJobFailed
			}

			return v, nil
		})

	require.ErrorIs(t, err, errCtxJobFailed)
}

func TestMapSlice_NeverSeesAmbientExecutor(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(2)
	defer should.Close(rec, "closing recording executor")

	// The context-free variants start from context.Background(), so an ambient
	// executor is unreachable to them by construction.
	_ = WithExecutor(t.Context(), rec)

	out, err := MapSlice(2, []int{1, 2}, func(ctx context.Context, v int) (int, error) { return v * 2, nil })

	require.NoError(t, err)
	assert.Equal(t, []int{2, 4}, out)
	assert.Equal(t, int32(0), rec.goCalls.Load())
}

func TestDo_NeverSeesAmbientExecutor(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(2)
	defer should.Close(rec, "closing recording executor")

	// Do starts from context.Background(), so an executor attached anywhere else
	// is unreachable to it by construction. Recorded here so a future change to
	// Do's context handling trips a test.
	_ = WithExecutor(t.Context(), rec)

	err := Do(2, func(ctx context.Context) error { return nil })

	require.NoError(t, err)
	assert.Equal(t, int32(0), rec.goCalls.Load())
}

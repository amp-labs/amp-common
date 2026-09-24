package simultaneously

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// histogramCount and histogramSum read the sample count and sum of one
// executor's executionDuration child.
func histogramCount(t *testing.T, name string) uint64 {
	t.Helper()

	return histogramMetric(t, name).GetHistogram().GetSampleCount()
}

func histogramSum(t *testing.T, name string) float64 {
	t.Helper()

	return histogramMetric(t, name).GetHistogram().GetSampleSum()
}

func histogramMetric(t *testing.T, name string) *dto.Metric {
	t.Helper()

	obs, err := executionDuration.GetMetricWithLabelValues(name)
	require.NoError(t, err)

	metric := &dto.Metric{}

	hist, ok := obs.(prometheus.Histogram)
	require.True(t, ok)
	require.NoError(t, hist.Write(metric))

	return metric
}

func executions(name, outcome string) float64 {
	return testutil.ToFloat64(executionsTotal.WithLabelValues(name, outcome))
}

var errTestBoom = errors.New("boom")

// runAll runs every job on exec directly, rather than through DoWithExecutor, so
// that a failing job does not cancel the others before they get to run.
func runAll(exec Executor, jobs ...Job) {
	var wg sync.WaitGroup

	wg.Add(len(jobs))

	for _, job := range jobs {
		exec.Go(job, func(error) { wg.Done() })
	}

	wg.Wait()
}

// The metrics are process-wide, so each test uses its own executor name and
// measures deltas, which keeps them correct under t.Parallel and -count.

func TestMetrics_CountsOutcomes(t *testing.T) {
	t.Parallel()

	const name = "test-counts-outcomes"

	successBefore := executions(name, outcomeSuccess)
	errorBefore := executions(name, outcomeError)
	panicBefore := executions(name, outcomePanic)
	observedBefore := histogramCount(t, name)

	exec := NewDefaultExecutor(4, WithName(name))

	runAll(exec,
		func(context.Context) error { return nil },
		func(context.Context) error { return nil },
		func(context.Context) error { return errTestBoom },
		func(context.Context) error { panic("kaboom") },
	)
	require.NoError(t, exec.Close())

	assert.InDelta(t, 2, executions(name, outcomeSuccess)-successBefore, 0)
	assert.InDelta(t, 1, executions(name, outcomeError)-errorBefore, 0)
	assert.InDelta(t, 1, executions(name, outcomePanic)-panicBefore, 0)
	assert.Equal(t, uint64(4), histogramCount(t, name)-observedBefore)
	assert.InDelta(t, 0, testutil.ToFloat64(activeExecutions.WithLabelValues(name)), 0)
}

func TestMetrics_TracksActiveExecutions(t *testing.T) {
	t.Parallel()

	const (
		name    = "test-active-executions"
		workers = 3
	)

	exec := NewDefaultExecutor(workers, WithName(name))

	var started sync.WaitGroup

	started.Add(workers)

	release := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		jobs := make([]Job, workers)
		for i := range jobs {
			jobs[i] = func(context.Context) error {
				started.Done()
				<-release

				return nil
			}
		}

		assert.NoError(t, DoWithExecutor(exec, jobs...))
	}()

	started.Wait()

	gauge := activeExecutions.WithLabelValues(name)
	assert.InDelta(t, workers, testutil.ToFloat64(gauge), 0)

	close(release)
	<-finished
	require.NoError(t, exec.Close())

	assert.InDelta(t, 0, testutil.ToFloat64(gauge), 0)
}

func TestMetrics_AccumulatesTime(t *testing.T) {
	t.Parallel()

	const (
		name  = "test-accumulates-time"
		sleep = 20 * time.Millisecond
	)

	totalBefore := testutil.ToFloat64(executionMillisecondsTotal.WithLabelValues(name))
	sumBefore := histogramSum(t, name)

	exec := NewDefaultExecutor(2, WithName(name))

	nap := func(context.Context) error {
		time.Sleep(sleep)

		return nil
	}

	require.NoError(t, DoWithExecutor(exec, nap, nap))
	require.NoError(t, exec.Close())

	total := testutil.ToFloat64(executionMillisecondsTotal.WithLabelValues(name)) - totalBefore
	assert.GreaterOrEqual(t, total, float64(2*sleep.Milliseconds()))
	assert.InDelta(t, total, (histogramSum(t, name)-sumBefore)*1000, 1e-6)
}

func TestMetrics_EmptyNameUsesDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultExecutorName, executorLabel(""))
	assert.Equal(t, "custom", executorLabel("custom"))
}

func TestMetrics_SkipsCallbacksThatNeverRun(t *testing.T) {
	t.Parallel()

	const name = "test-skips-never-run"

	successBefore := executions(name, outcomeSuccess)
	observedBefore := histogramCount(t, name)

	exec := NewDefaultExecutor(1, WithName(name))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := DoCtxWithExecutor(ctx, exec, func(context.Context) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, exec.Close())

	assert.InDelta(t, successBefore, executions(name, outcomeSuccess), 0)
	assert.Equal(t, observedBefore, histogramCount(t, name))
}

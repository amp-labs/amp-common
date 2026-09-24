package simultaneously

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for monitoring executor behavior. Every metric carries an
// "executor" label whose value is the name given to the executor via WithName,
// or DefaultExecutorName if none was given. The throwaway executors that the
// Do/Map/FlatMap entry points create internally always report under the default.
//
// An execution is one run of a callback. The time a callback spends waiting for
// a free slot is not part of it, and a callback whose context has already ended
// by the time it gets a slot is never run, so it is not counted at all.

// DefaultExecutorName is the "executor" label value reported by executors that
// were not given a name, or were given an empty one.
const DefaultExecutorName = "default"

// Names of the labels on the executor metrics.
const (
	labelExecutor = "executor"
	labelOutcome  = "outcome"
)

// Values of the outcome label on executionsTotal.
const (
	outcomeSuccess = "success" // callback returned nil
	outcomeError   = "error"   // callback returned an error
	outcomePanic   = "panic"   // callback panicked (whether or not it also returned an error)
)

var (
	// activeExecutions is the number of callbacks currently running.
	activeExecutions = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "simultaneously_active_executions",
		Help: "The number of callbacks currently running on the executor",
	}, []string{labelExecutor})

	// executionsTotal counts the callbacks that have finished running, by outcome.
	executionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "simultaneously_executions_total",
		Help: "The total number of callbacks that have finished running on the executor",
	}, []string{labelExecutor, labelOutcome})

	// executionMillisecondsTotal accumulates the time spent running callbacks.
	// It is in milliseconds rather than seconds because most callbacks are short.
	executionMillisecondsTotal = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "simultaneously_execution_milliseconds_total",
		Help: "The total time spent running callbacks on the executor, in milliseconds",
	}, []string{labelExecutor})

	// executionDuration measures how long each callback ran.
	executionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{ //nolint:gochecknoglobals
		Name: "simultaneously_execution_duration_seconds",
		Help: "The time spent running each callback on the executor, in seconds",
		Buckets: []float64{
			0.001, // 1ms
			0.01,  // 10ms
			0.1,   // 100ms
			1,     // 1s
			10,    // 10s
			60,    // 1m
			300,   // 5m
			600,   // 10m
		},
	}, []string{labelExecutor})
)

// executorMetrics holds the metric children for one executor name, resolved
// once at construction so that recording an execution needs no label lookups.
type executorMetrics struct {
	active   prometheus.Gauge
	success  prometheus.Counter
	failure  prometheus.Counter
	panicked prometheus.Counter
	millis   prometheus.Counter
	duration prometheus.Observer
}

func newExecutorMetrics(name string) *executorMetrics {
	name = executorLabel(name)

	return &executorMetrics{
		active:   activeExecutions.WithLabelValues(name),
		success:  executionsTotal.WithLabelValues(name, outcomeSuccess),
		failure:  executionsTotal.WithLabelValues(name, outcomeError),
		panicked: executionsTotal.WithLabelValues(name, outcomePanic),
		millis:   executionMillisecondsTotal.WithLabelValues(name),
		duration: executionDuration.WithLabelValues(name),
	}
}

// executorLabel returns the "executor" label value for an executor named name.
func executorLabel(name string) string {
	if name == "" {
		return DefaultExecutorName
	}

	return name
}

// start records that a callback has begun running and returns its start time.
func (m *executorMetrics) start() time.Time {
	m.active.Inc()

	return time.Now()
}

// finish records that a callback which began at start has stopped running.
func (m *executorMetrics) finish(start time.Time, panicked bool, err error) {
	elapsed := time.Since(start)

	m.active.Dec()
	m.millis.Add(float64(elapsed) / float64(time.Millisecond))
	m.duration.Observe(elapsed.Seconds())

	switch {
	case panicked:
		m.panicked.Inc()
	case err != nil:
		m.failure.Inc()
	default:
		m.success.Inc()
	}
}

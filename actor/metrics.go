package actor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for monitoring actor behavior and performance.
// All metrics with labels support multi-tenancy via "subsystem" and "actor" labels.

var (
	// actorStarted counts the total number of actors started (global counter).
	actorStarted = promauto.NewCounter(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "actor_started",
		Help: "The total number of actors started",
	})

	// actorStopped counts the total number of actors stopped (global counter).
	actorStopped = promauto.NewCounter(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "actor_stopped",
		Help: "The total number of actors stopped",
	})

	// actorIdle tracks the number of actors currently idle (waiting for messages).
	actorIdle = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "actor_idle",
		Help: "The total number of actors that are idle",
	}, []string{"subsystem", "actor"})

	// actorBusy tracks the number of actors currently busy (processing messages).
	actorBusy = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "actor_busy",
		Help: "The total number of actors that are busy",
	}, []string{"subsystem", "actor"})

	// actorPanic counts the number of times an actor recovered from a panic.
	actorPanic = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "actor_panic",
		Help: "The total number of actors that recovered from a panic",
	}, []string{"subsystem", "actor"})

	// aliveActors tracks the number of currently running actors.
	aliveActors = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "actor_alive_actors",
		Help: "The total number of actors alive",
	}, []string{"subsystem", "actor"})

	// enqueuedMessages tracks the current queue depth (number of messages waiting to be processed).
	enqueuedMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "actor_enqueued_messages",
		Help: "The total number of messages enqueued",
	}, []string{"subsystem", "actor"})

	// submitCount counts the total number of messages submitted to actors.
	submitCount = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "actor_submit_count",
		Help: "The total number of messages submitted",
	}, []string{"subsystem", "actor"})

	// submitTime measures the time spent waiting to submit a message to an actor's inbox.
	submitTime = promauto.NewHistogramVec(prometheus.HistogramOpts{ //nolint:gochecknoglobals
		Name: "actor_submit_time",
		Help: "The time spent waiting for a message to be sent",
		Buckets: []float64{
			0.01, // 10ms
			0.1,  // 100ms
			1,    // 1s
			10,   // 10s
			60,   // 1m
			120,  // 2m
			300,  // 5m
			600,  // 10m
		},
	}, []string{"subsystem", "actor"})

	// receiveTime measures the time spent waiting to receive a response from an actor.
	receiveTime = promauto.NewHistogramVec(prometheus.HistogramOpts{ //nolint:gochecknoglobals
		Name: "actor_receive_time",
		Help: "The time spent waiting for a message to be received",
		Buckets: []float64{
			0.01, // 10ms
			0.1,  // 100ms
			1,    // 1s
			10,   // 10s
			60,   // 1m
			120,  // 2m
			300,  // 5m
			600,  // 10m
		},
	}, []string{"subsystem", "actor"})

	// processedMessages counts the total number of messages successfully processed.
	processedMessages = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "actor_processed_messages",
		Help: "The total number of messages processed",
	}, []string{"subsystem", "actor"})

	// processingTime measures the time spent by the processor handling each message.
	processingTime = promauto.NewHistogramVec(prometheus.HistogramOpts{ //nolint:gochecknoglobals
		Name: "actor_processing_time",
		Help: "The time spent processing a message",
		Buckets: []float64{
			0.01, // 10ms
			0.1,  // 100ms
			1,    // 1s
			10,   // 10s
			60,   // 1m
			120,  // 2m
			300,  // 5m
			600,  // 10m
		},
	}, []string{"subsystem", "actor"})
)

package pool

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// labelPool is the name of the label carrying the pool's name on every pool metric.
const labelPool = "pool"

var (
	poolAlive = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_alive",
		Help: "1 if the pool is alive and running",
	}, []string{labelPool})

	poolCreated = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_created_total",
		Help: "The total number of pools created",
	}, []string{labelPool})

	objectsCreated = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_created_total",
		Help: "The total number of objects created",
	}, []string{labelPool})

	creationErrors = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_creation_errors_total",
		Help: "The total number of errors creating objects",
	}, []string{labelPool})

	objectsClosed = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_closed_total",
		Help: "The total number of objects closed",
	}, []string{labelPool})

	objectsClosedErrors = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_closed_errors_total",
		Help: "The total number of errors closing objects",
	}, []string{labelPool})

	poolObjectsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_objects",
		Help: "The total number of objects in the pool",
	}, []string{labelPool})

	poolObjectsInUse = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_in_use",
		Help: "The total number of objects in use",
	}, []string{labelPool})

	poolObjectsIdle = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_idle",
		Help: "The total number of objects idle",
	}, []string{labelPool})
)

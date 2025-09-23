package pool

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	poolAlive = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_alive",
		Help: "1 if the pool is alive and running",
	}, []string{"pool"})

	poolCreated = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_created_total",
		Help: "The total number of pools created",
	}, []string{"pool"})

	objectsCreated = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_created_total",
		Help: "The total number of objects created",
	}, []string{"pool"})

	creationErrors = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_creation_errors_total",
		Help: "The total number of errors creating objects",
	}, []string{"pool"})

	objectsClosed = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_closed_total",
		Help: "The total number of objects closed",
	}, []string{"pool"})

	objectsClosedErrors = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_closed_errors_total",
		Help: "The total number of errors closing objects",
	}, []string{"pool"})

	poolObjectsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_objects",
		Help: "The total number of objects in the pool",
	}, []string{"pool"})

	poolObjectsInUse = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_in_use",
		Help: "The total number of objects in use",
	}, []string{"pool"})

	poolObjectsIdle = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:gochecknoglobals
		Name: "pool_objects_idle",
		Help: "The total number of objects idle",
	}, []string{"pool"})
)

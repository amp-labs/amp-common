package statemachine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metric definitions with appropriate labels.
var (
	// StateVisitsTotal tracks state exit counts by tool, state, provider, and outcome (success/error).
	stateVisitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "statemachine_state_visits_total",
		Help: "Total number of state visits by tool, state, provider, and outcome (success or error)",
	}, []string{"tool", "state", "provider", "outcome", "project_id", "chunk_id"})

	// TransitionTotal tracks state transitions.
	transitionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "statemachine_transitions_total",
		Help: "Total number of state transitions by tool, from_state, to_state, and provider",
	}, []string{"tool", "from_state", "to_state", "provider", "project_id", "chunk_id"})

	// ExecutionDuration tracks end-to-end state machine execution time.
	executionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "statemachine_execution_duration_seconds",
		Help:    "Duration of state machine execution by tool and outcome",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
	}, []string{"tool", "outcome", "project_id", "chunk_id"})

	// ActionDuration tracks individual action execution time.
	actionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "statemachine_action_duration_seconds",
		Help:    "Duration of action execution by tool, action, and state",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30},
	}, []string{"tool", "action", "state", "project_id", "chunk_id"})

	// StateDuration tracks individual state execution time.
	stateDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "statemachine_state_duration_seconds",
		Help:    "Duration of state execution by tool, state, provider, and outcome",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60},
	}, []string{"tool", "state", "provider", "outcome", "project_id", "chunk_id"})

	// PathLength tracks the distribution of execution path lengths.
	pathLength = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "statemachine_path_length",
		Help:    "Number of states visited during execution by tool and outcome",
		Buckets: []float64{1, 2, 3, 5, 10, 15, 20, 30, 50},
	}, []string{"tool", "outcome", "project_id", "chunk_id"})

	// ExecutionsCancelledTotal tracks how often executions are canceled via context.
	executionsCancelledTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "statemachine_executions_cancelled_total",
		Help: "Total number of state machine executions canceled via context",
	}, []string{"tool", "state", "project_id", "chunk_id"})
)

// Helper functions for label sanitization.
func sanitizeProjectID(projectID string) string {
	if projectID == "" {
		return "unknown"
	}

	return projectID
}

func sanitizeChunkID(chunkID string) string {
	if chunkID == "" {
		return "none"
	}

	return chunkID
}

func sanitizeProvider(provider string) string {
	if provider == "" {
		return "none"
	}

	return provider
}

func sanitizeTool(tool string) string {
	if tool == "" {
		return "unknown"
	}

	return tool
}

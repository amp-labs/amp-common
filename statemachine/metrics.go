package statemachine

import (
	"crypto/sha1" //nolint:gosec // SHA1 used for non-cryptographic metric label hashing, not security
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metric definitions with appropriate labels.
var (
	// StateVisitsTotal tracks state exit counts by tool, state, provider, and outcome (success/error).
	stateVisitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "statemachine_state_visits_total",
		Help: "Total number of state visits by tool, state, provider, and outcome (success or error)",
	}, []string{"tool", "state", "provider", "outcome", "project_id_hash", "chunk_id_hash"})

	// TransitionTotal tracks state transitions.
	transitionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "statemachine_transitions_total",
		Help: "Total number of state transitions by tool, from_state, to_state, and provider",
	}, []string{"tool", "from_state", "to_state", "provider", "project_id_hash", "chunk_id_hash"})

	// ExecutionDuration tracks end-to-end state machine execution time.
	executionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "statemachine_execution_duration_seconds",
		Help:    "Duration of state machine execution by tool and outcome",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
	}, []string{"tool", "outcome", "project_id_hash", "chunk_id_hash"})

	// ActionDuration tracks individual action execution time.
	actionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "statemachine_action_duration_seconds",
		Help:    "Duration of action execution by tool, action, and state",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30},
	}, []string{"tool", "action", "state", "project_id_hash", "chunk_id_hash"})
)

// Helper functions for label sanitization.
func sanitizeProjectID(projectID string) string {
	if projectID == "" {
		return "unknown"
	}

	hash := sha1.Sum([]byte(projectID)) //nolint:gosec // SHA1 used for non-cryptographic metric label hashing

	return hex.EncodeToString(hash[:])[:8]
}

func sanitizeChunkID(chunkID string) string {
	if chunkID == "" {
		return "none"
	}

	hash := sha1.Sum([]byte(chunkID)) //nolint:gosec // SHA1 used for non-cryptographic metric label hashing

	return hex.EncodeToString(hash[:])[:8]
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

package statemachine

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// TestStateVisitsMetric verifies that state visit metrics are recorded correctly.
// Note: Cannot use t.Parallel() because this test modifies global Prometheus metrics.
//
//nolint:paralleltest // Test modifies global Prometheus metric state
func TestStateVisitsMetric(t *testing.T) {
	// Reset metrics
	stateVisitsTotal.Reset()

	// Create test context
	smCtx := NewContext("test-session", "test-project")
	smCtx.ToolName = "test_tool"
	smCtx.Provider = "test_provider"
	smCtx.CurrentState = "test_state"

	// Record metric
	stateVisitsTotal.WithLabelValues(
		sanitizeTool(smCtx.ToolName),
		smCtx.CurrentState,
		sanitizeProvider(smCtx.Provider),
		"success",
		sanitizeProjectID(smCtx.ProjectID),
		sanitizeChunkID(smCtx.ContextChunkID),
	).Inc()

	// Verify metric
	count := testutil.CollectAndCount(stateVisitsTotal)
	assert.Equal(t, 1, count)
}

func TestSanitization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		fn       func(string) string
	}{
		{"empty project", "", "unknown", sanitizeProjectID},
		{"empty provider", "", "none", sanitizeProvider},
		{"empty tool", "", "unknown", sanitizeTool},
		{"empty chunk", "", "none", sanitizeChunkID},
		{"short chunk", "short", "8-char-hash", sanitizeChunkID},
		{"long chunk", "very-long-chunk-id-12345", "8-char-hash", sanitizeChunkID},
		{"project id", "test-project-123", "8-char-hash", sanitizeProjectID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.fn(tt.input)
			if tt.expected == "8-char-hash" {
				assert.Len(t, result, 8)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

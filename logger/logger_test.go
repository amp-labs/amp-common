package logger

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) { //nolint:paralleltest
	// Configure logging for JSON output
	ConfigureLoggingWithOptions(Options{
		Subsystem: "test",
		JSON:      true,
	})

	// Just use slog directly, as a point of comparison
	slog.Info("test info")

	// Use logger with no args (will embed subsystem but nothing else)
	Get().Info("should have the default subsystem")

	// Use logger with an embedded customer ID (should have customer ID and default subsystem)
	ctx := WithCustomerId(t.Context(), "1234")
	Get(ctx).Info("should have customer_id and default subsystem")

	// Use logger with an embedded subsystem (should have subsystem but no customer ID)
	ctx = WithSubsystem(t.Context(), "overridden")
	Get(ctx).Info("should have overridden subsystem")

	// Use logger with an embedded subsystem and customer ID (should have both)
	ctx = WithCustomerId(WithSubsystem(t.Context(), "overridden"), "1234")
	Get(ctx).Info("should have overridden subsystem and customer_id")

	// Use logger with an embedded sensitive flag (should have subsystem but no customer ID)
	ctx = WithSensitive(t.Context())
	Get(ctx).Info("should have only the subsystem")

	// Use logger with an embedded sensitive flag and customer ID (should have subsystem but no customer ID)
	ctx = WithSensitive(WithCustomerId(t.Context(), "1234"))
	Get(ctx).Info("should have only the subsystem")

	// Use logger with an embedded sensitive flag and subsystem (should have subsystem but no customer ID)
	ctx = WithSensitive(WithSubsystem(t.Context(), "overridden"))
	Get(ctx).Info("should have only the subsystem (overridden)")

	// Use logger with an embedded sensitive flag, subsystem, and customer ID (should have subsystem but no customer ID)
	ctx = WithSensitive(WithCustomerId(WithSubsystem(t.Context(), "overridden"), "1234"))
	Get(ctx).Info("should have only the subsystem (overridden)")

	// Use logger with an embedded routing to builder (should have log_project and default subsystem)
	ctx = WithRoutingToBuilder(t.Context(), "ampersand-project-id")
	Get(ctx).Info("should have log_project and default subsystem")

	// Use logger with an embedded routing to builder and subsystem (should have both)
	ctx = WithRoutingToBuilder(WithSubsystem(t.Context(), "overridden"), "ampersand-project-id")
	Get(ctx).Info("should have log_project and overridden subsystem")

	// Use logger with an embedded routing to builder, subsystem &
	// sensitive flag (should have subsystem but no log_project)
	ctx = WithSensitive(WithRoutingToBuilder(WithSubsystem(t.Context(), "overridden"), "builder-project-id"))
	Get(ctx).Info("should have only the overridden subsystem")
}

func TestLegacy(t *testing.T) { //nolint:paralleltest
	// Configure logging for JSON output
	ConfigureLoggingWithOptions(Options{
		Subsystem:   "test",
		JSON:        true,
		MinLevel:    slog.LevelDebug,
		LegacyLevel: slog.LevelInfo,
	})

	// Should output JSON
	log.Println("test")

	// Turn off JSON
	ConfigureLoggingWithOptions(Options{
		Subsystem: "test",
		JSON:      false,
	})

	// Should output text (slog text, just not JSON)
	log.Println("test")
}

// TestGetCustomerId tests the GetCustomerId function.
func TestGetCustomerId(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name           string
		ctx            context.Context //nolint:containedctx
		expectedID     string
		expectedExists bool
	}{
		{
			name:           "nil context",
			ctx:            nil,
			expectedID:     "",
			expectedExists: false,
		},
		{
			name:           "context without customer ID",
			ctx:            t.Context(),
			expectedID:     "",
			expectedExists: false,
		},
		{
			name:           "context with customer ID",
			ctx:            WithCustomerId(t.Context(), "customer123"),
			expectedID:     "customer123",
			expectedExists: true,
		},
		{
			name:           "context with empty customer ID",
			ctx:            WithCustomerId(t.Context(), ""),
			expectedID:     "",
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, exists := GetCustomerId(tt.ctx)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedExists, exists)
		})
	}
}

// TestGetRequestId tests the GetRequestId function.
func TestGetRequestId(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name           string
		ctx            context.Context //nolint:containedctx
		expectedID     string
		expectedExists bool
	}{
		{
			name:           "nil context",
			ctx:            nil,
			expectedID:     "",
			expectedExists: false,
		},
		{
			name:           "context without request ID",
			ctx:            t.Context(),
			expectedID:     "",
			expectedExists: false,
		},
		{
			name:           "context with request ID",
			ctx:            WithRequestId(t.Context(), "req-123"),
			expectedID:     "req-123",
			expectedExists: true,
		},
		{
			name:           "context with empty request ID",
			ctx:            WithRequestId(t.Context(), ""),
			expectedID:     "",
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, exists := GetRequestId(tt.ctx)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedExists, exists)
		})
	}
}

// TestGetSubsystem tests the GetSubsystem function.
func TestGetSubsystem(t *testing.T) {
	t.Parallel()

	t.Run("nil context returns non-empty default", func(t *testing.T) {
		t.Parallel()
		// Default subsystem is set by other tests, so we just verify it's not empty
		result := GetSubsystem(nil) //nolint:staticcheck
		assert.NotEmpty(t, result)
	})

	t.Run("context without subsystem returns non-empty default", func(t *testing.T) {
		t.Parallel()
		// Default subsystem is set by other tests, so we just verify it's not empty
		result := GetSubsystem(t.Context())
		assert.NotEmpty(t, result)
	})

	t.Run("context with subsystem override", func(t *testing.T) {
		t.Parallel()

		ctx := WithSubsystem(t.Context(), "override")
		result := GetSubsystem(ctx)
		assert.Equal(t, "override", result)
	})

	t.Run("context with empty subsystem override", func(t *testing.T) {
		t.Parallel()

		ctx := WithSubsystem(t.Context(), "")
		result := GetSubsystem(ctx)
		assert.Equal(t, "", result)
	})
}

// TestGetRoutingToBuilder tests the GetRoutingToBuilder function.
func TestGetRoutingToBuilder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ctx            context.Context //nolint:containedctx
		expectedID     string
		expectedExists bool
	}{
		{
			name:           "nil context",
			ctx:            nil,
			expectedID:     "",
			expectedExists: false,
		},
		{
			name:           "context without routing",
			ctx:            t.Context(),
			expectedID:     "",
			expectedExists: false,
		},
		{
			name:           "context with routing to builder",
			ctx:            WithRoutingToBuilder(t.Context(), "project-123"),
			expectedID:     "project-123",
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, exists := GetRoutingToBuilder(tt.ctx)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedExists, exists)
		})
	}
}

// TestIsSensitiveMessage tests the IsSensitiveMessage function.
func TestIsSensitiveMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context //nolint:containedctx
		expected bool
	}{
		{
			name:     "context without sensitive flag",
			ctx:      t.Context(),
			expected: false,
		},
		{
			name:     "context with sensitive flag",
			ctx:      WithSensitive(t.Context()),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsSensitiveMessage(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetSlackNotification tests the GetSlackNotification function.
func TestGetSlackNotification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context //nolint:containedctx
		expected bool
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: false,
		},
		{
			name:     "context without slack notification",
			ctx:      t.Context(),
			expected: false,
		},
		{
			name:     "context with slack notification",
			ctx:      WithSlackNotification(t.Context()),
			expected: true,
		},
		{
			name:     "context with slack channel (implies notification)",
			ctx:      WithSlackChannel(t.Context(), "alerts"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := GetSlackNotification(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetSlackChannel tests the GetSlackChannel function.
func TestGetSlackChannel(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name           string
		ctx            context.Context //nolint:containedctx
		expectedChan   string
		expectedExists bool
	}{
		{
			name:           "nil context",
			ctx:            nil,
			expectedChan:   "",
			expectedExists: false,
		},
		{
			name:           "context without slack channel",
			ctx:            t.Context(),
			expectedChan:   "",
			expectedExists: false,
		},
		{
			name:           "context with slack channel",
			ctx:            WithSlackChannel(t.Context(), "alerts"),
			expectedChan:   "alerts",
			expectedExists: true,
		},
		{
			name:           "context with empty slack channel",
			ctx:            WithSlackChannel(t.Context(), ""),
			expectedChan:   "",
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			channel, exists := GetSlackChannel(tt.ctx)
			assert.Equal(t, tt.expectedChan, channel)
			assert.Equal(t, tt.expectedExists, exists)
		})
	}
}

// TestGetPodName tests the GetPodName function.
func TestGetPodName(t *testing.T) {
	t.Parallel()

	podName := GetPodName()
	assert.NotEmpty(t, podName)
	// Should be callable multiple times and return the same value
	assert.Equal(t, podName, GetPodName())
}

// TestWith tests the With function for adding key-value pairs.
func TestWith(t *testing.T) { //nolint:paralleltest
	tests := []struct {
		name         string
		ctx          context.Context //nolint:containedctx
		values       []any
		expectedKeys []string
		logMessage   string
	}{
		{
			name:         "add single key-value pair",
			ctx:          t.Context(),
			values:       []any{"key1", "value1"},
			expectedKeys: []string{"key1"},
			logMessage:   "test message with single kv",
		},
		{
			name:         "add multiple key-value pairs",
			ctx:          t.Context(),
			values:       []any{"key1", "value1", "key2", "value2"},
			expectedKeys: []string{"key1", "key2"},
			logMessage:   "test message with multiple kvs",
		},
		{
			name:         "chain multiple With calls",
			ctx:          With(t.Context(), "key1", "value1"),
			values:       []any{"key2", "value2"},
			expectedKeys: []string{"key1", "key2"},
			logMessage:   "test message with chained kvs",
		},
		{
			name:         "empty values on non-nil context",
			ctx:          t.Context(),
			values:       []any{},
			expectedKeys: []string{},
			logMessage:   "test message with no additional kvs",
		},
	}

	for _, tt := range tests { //nolint:paralleltest,varnamelen
		t.Run(tt.name, func(t *testing.T) {
			// Set up logger with custom output for each subtest
			var buf bytes.Buffer

			ConfigureLoggingWithOptions(Options{
				Subsystem: "test",
				JSON:      true,
				Output:    &buf,
			})

			ctx := With(tt.ctx, tt.values...)
			Get(ctx).Info(tt.logMessage)

			output := buf.String()
			for _, key := range tt.expectedKeys {
				assert.Contains(t, output, key, "Expected key %s in output", key)
			}

			assert.Contains(t, output, tt.logMessage)
		})
	}
}

// TestConfigureLoggingWithOptions tests various configuration options.
func TestConfigureLoggingWithOptions(t *testing.T) { //nolint:paralleltest
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "JSON output",
			opts: Options{
				Subsystem: "test",
				JSON:      true,
				MinLevel:  slog.LevelInfo,
			},
		},
		{
			name: "Text output",
			opts: Options{
				Subsystem: "test",
				JSON:      false,
				MinLevel:  slog.LevelDebug,
			},
		},
		{
			name: "Custom output writer",
			opts: Options{
				Subsystem: "test",
				JSON:      true,
				Output:    &bytes.Buffer{},
			},
		},
		{
			name: "Nil output defaults to stdout",
			opts: Options{
				Subsystem: "test",
				JSON:      true,
				Output:    nil,
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			logger := ConfigureLoggingWithOptions(tt.opts)
			assert.NotNil(t, logger)
		})
	}
}

// TestConfigureLoggingConcurrency tests that ConfigureLogging is thread-safe.
func TestConfigureLoggingConcurrency(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup //nolint:varnamelen

	iterations := 10

	for i := 0; i < iterations; i++ { //nolint:intrange
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()
			ConfigureLoggingWithOptions(Options{
				Subsystem: "concurrent-test",
				JSON:      idx%2 == 0,
			})
		}(i)
	}

	wg.Wait() // If we get here without deadlock or panic, the test passes
}

// TestConfigureLogging tests the environment-based configuration.
func TestConfigureLogging(t *testing.T) { //nolint:paralleltest
	// Save original env vars
	originalJSON := os.Getenv("LOG_JSON")
	originalLevel := os.Getenv("LOG_LEVEL")
	originalOutput := os.Getenv("LOG_OUTPUT")

	defer func() {
		t.Setenv("LOG_JSON", originalJSON)
		t.Setenv("LOG_LEVEL", originalLevel)
		t.Setenv("LOG_OUTPUT", originalOutput)
	}()

	tests := []struct {
		name    string
		envVars map[string]string
		appName string
		wantErr bool
	}{
		{
			name:    "default configuration",
			envVars: map[string]string{},
			appName: "test-app",
			wantErr: false,
		},
		{
			name: "JSON logging enabled",
			envVars: map[string]string{
				"LOG_JSON": "true",
			},
			appName: "test-app",
			wantErr: false,
		},
		{
			name: "custom log level",
			envVars: map[string]string{
				"LOG_LEVEL": "DEBUG",
			},
			appName: "test-app",
			wantErr: false,
		},
		{
			name: "stdout output",
			envVars: map[string]string{
				"LOG_OUTPUT": "stdout",
			},
			appName: "test-app",
			wantErr: false,
		},
		{
			name: "stderr output",
			envVars: map[string]string{
				"LOG_OUTPUT": "stderr",
			},
			appName: "test-app",
			wantErr: false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			logger := ConfigureLogging(t.Context(), tt.appName)
			assert.NotNil(t, logger)

			// Clean up
			for k := range tt.envVars {
				os.Unsetenv(k)
			}
		})
	}
}

// TestLoggerIntegration tests the complete flow with all context values.
func TestLoggerIntegration(t *testing.T) { //nolint:paralleltest
	var buf bytes.Buffer

	ConfigureLoggingWithOptions(Options{
		Subsystem: "integration-test",
		JSON:      true,
		Output:    &buf,
	})

	ctx := t.Context()
	ctx = WithCustomerId(ctx, "cust-123")
	ctx = WithRequestId(ctx, "req-456")
	ctx = WithSubsystem(ctx, "api")
	ctx = WithRoutingToBuilder(ctx, "proj-789")
	ctx = With(ctx, "operation", "create", "resource", "account")

	Get(ctx).Info("integration test message")

	output := buf.String()
	assert.Contains(t, output, "cust-123")
	assert.Contains(t, output, "req-456")
	assert.Contains(t, output, "api")
	assert.Contains(t, output, "proj-789")
	assert.Contains(t, output, "operation")
	assert.Contains(t, output, "create")
	assert.Contains(t, output, "integration test message")
}

// TestSlackIntegration tests Slack notification flags in logger output.
func TestSlackIntegration(t *testing.T) { //nolint:paralleltest
	t.Run("slack notification enabled", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		ConfigureLoggingWithOptions(Options{
			Subsystem: "slack-test",
			JSON:      true,
			Output:    &buf,
		})

		ctx := WithSlackNotification(t.Context())
		Get(ctx).Info("slack notification test")

		output := buf.String()
		assert.Contains(t, output, `"slack":"1"`)
	})

	t.Run("slack channel specified", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		ConfigureLoggingWithOptions(Options{
			Subsystem: "slack-test",
			JSON:      true,
			Output:    &buf,
		})

		ctx := WithSlackChannel(t.Context(), "critical-alerts")
		Get(ctx).Info("slack channel test")

		output := buf.String()
		assert.Contains(t, output, `"slack":"1"`)
		assert.Contains(t, output, "critical-alerts")
	})
}

// TestSensitiveLogging tests that sensitive logs don't include customer info.
func TestSensitiveLogging(t *testing.T) { //nolint:paralleltest
	var buf bytes.Buffer

	ConfigureLoggingWithOptions(Options{
		Subsystem: "sensitive-test",
		JSON:      true,
		Output:    &buf,
	})

	ctx := WithCustomerId(t.Context(), "customer-secret")
	ctx = WithRoutingToBuilder(ctx, "project-secret")
	ctx = WithSensitive(ctx)

	Get(ctx).Info("sensitive message")

	output := buf.String()
	// Should NOT contain customer_id or log_project
	assert.NotContains(t, output, "customer-secret")
	assert.NotContains(t, output, "project-secret")
	// Should contain the message itself
	assert.Contains(t, output, "sensitive message")
}

// TestContextChaining tests chaining multiple context operations.
func TestContextChaining(t *testing.T) { //nolint:paralleltest
	// Test that multiple context values can be chained
	ctx := t.Context()
	ctx = WithCustomerId(ctx, "cust1")
	ctx = WithRequestId(ctx, "req1")
	ctx = WithSubsystem(ctx, "subsys1")
	ctx = WithRoutingToBuilder(ctx, "proj1")
	ctx = With(ctx, "key1", "val1")

	custID, ok := GetCustomerId(ctx)
	require.True(t, ok)
	assert.Equal(t, "cust1", custID)

	reqID, ok := GetRequestId(ctx)
	require.True(t, ok)
	assert.Equal(t, "req1", reqID)

	assert.Equal(t, "subsys1", GetSubsystem(ctx))

	projID, ok := GetRoutingToBuilder(ctx)
	require.True(t, ok)
	assert.Equal(t, "proj1", projID)

	// Verify logging includes all values
	var buf bytes.Buffer

	ConfigureLoggingWithOptions(Options{
		Subsystem: "chain-test",
		JSON:      true,
		Output:    &buf,
	})

	Get(ctx).Info("chained context test")

	output := buf.String()
	assert.Contains(t, output, "cust1")
	assert.Contains(t, output, "req1")
	assert.Contains(t, output, "subsys1")
	assert.Contains(t, output, "proj1")
	assert.Contains(t, output, "key1")
}

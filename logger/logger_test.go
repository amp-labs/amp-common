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
		assert.Empty(t, result)
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

	for i := range iterations { //nolint:intrange
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
				_ = os.Unsetenv(k)
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

// TestOtelIntegration tests OpenTelemetry integration.
func TestOtelIntegration(t *testing.T) { //nolint:paralleltest
	t.Run("otel disabled by default", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		handler := CreateLoggerHandler(Options{
			Subsystem: "otel-test",
			JSON:      true,
			Output:    &buf,
			MinLevel:  slog.LevelInfo,
		})

		// Handler should be slogErrorLogger wrapping the base handler
		assert.NotNil(t, handler)
		_, ok := handler.(*slogErrorLogger)
		assert.True(t, ok, "handler should be slogErrorLogger when OTel is disabled")
	})

	t.Run("otel enabled", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		handler := CreateLoggerHandler(Options{
			Subsystem:  "otel-test",
			JSON:       true,
			Output:     &buf,
			MinLevel:   slog.LevelInfo,
			EnableOtel: true,
		})

		// Handler should still be slogErrorLogger
		assert.NotNil(t, handler)
		errorLogger, ok := handler.(*slogErrorLogger)
		assert.True(t, ok, "handler should be slogErrorLogger")

		// The inner handler should be multiHandler
		_, ok = errorLogger.inner.(*multiHandler)
		assert.True(t, ok, "inner handler should be multiHandler when OTel is enabled")
	})

	t.Run("logs work with otel enabled", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		ConfigureLoggingWithOptions(Options{
			Subsystem:  "otel-test",
			JSON:       true,
			Output:     &buf,
			MinLevel:   slog.LevelInfo,
			EnableOtel: true,
		})

		ctx := t.Context()
		ctx = WithCustomerId(ctx, "test-customer")

		Get(ctx).Info("test message with otel")

		output := buf.String()
		// Verify the console output still works
		assert.Contains(t, output, "test message with otel")
		assert.Contains(t, output, "test-customer")
		assert.Contains(t, output, "otel-test")
	})
}

// TestMultiHandler tests the multiHandler implementation.
func TestMultiHandler(t *testing.T) {
	t.Parallel()

	t.Run("enabled returns true if any handler is enabled", func(t *testing.T) {
		t.Parallel()

		var buf1, buf2 bytes.Buffer

		h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
		h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelWarn})

		multi := &multiHandler{handlers: []slog.Handler{h1, h2}}

		ctx := t.Context()
		assert.True(t, multi.Enabled(ctx, slog.LevelInfo), "should be enabled for Info")
		assert.True(t, multi.Enabled(ctx, slog.LevelWarn), "should be enabled for Warn")
		assert.False(t, multi.Enabled(ctx, slog.LevelDebug), "should not be enabled for Debug")
	})

	t.Run("handle forwards to all handlers", func(t *testing.T) {
		t.Parallel()

		var buf1, buf2 bytes.Buffer

		h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
		h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

		multi := &multiHandler{handlers: []slog.Handler{h1, h2}}
		logger := slog.New(multi)

		logger.Info("test message")

		// Both buffers should contain the message
		assert.Contains(t, buf1.String(), "test message")
		assert.Contains(t, buf2.String(), "test message")
	})

	t.Run("with attrs creates new multiHandler", func(t *testing.T) {
		t.Parallel()

		var buf1, buf2 bytes.Buffer

		h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
		h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

		multi := &multiHandler{handlers: []slog.Handler{h1, h2}}
		newMulti := multi.WithAttrs([]slog.Attr{slog.String("key", "value")})

		logger := slog.New(newMulti)
		logger.Info("test")

		// Both handlers should have the attribute
		assert.Contains(t, buf1.String(), "key")
		assert.Contains(t, buf1.String(), "value")
		assert.Contains(t, buf2.String(), "key")
		assert.Contains(t, buf2.String(), "value")
	})

	t.Run("with group creates new multiHandler", func(t *testing.T) {
		t.Parallel()

		var buf1, buf2 bytes.Buffer

		h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
		h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

		multi := &multiHandler{handlers: []slog.Handler{h1, h2}}
		newMulti := multi.WithGroup("testgroup")

		logger := slog.New(newMulti)
		logger.Info("test", "key", "value")

		// Both handlers should have the group
		assert.Contains(t, buf1.String(), "testgroup")
		assert.Contains(t, buf2.String(), "testgroup")
	})
}

// TestOtelSuppressibleHandler tests the otelSuppressibleHandler implementation.
func TestOtelSuppressibleHandler(t *testing.T) {
	t.Parallel()

	t.Run("enabled returns false when otel is suppressed", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		handler := &otelSuppressibleHandler{inner: innerHandler}

		ctx := WithSuppressOtel(t.Context(), true)
		assert.False(t, handler.Enabled(ctx, slog.LevelInfo), "should be disabled when suppressed")

		normalCtx := t.Context()
		assert.True(t, handler.Enabled(normalCtx, slog.LevelInfo), "should be enabled when not suppressed")
	})

	t.Run("handle discards logs when otel is suppressed", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		handler := &otelSuppressibleHandler{inner: innerHandler}
		logger := slog.New(handler)

		// Log with suppression enabled
		ctx := WithSuppressOtel(t.Context(), true)
		logger.InfoContext(ctx, "suppressed message")

		// Buffer should be empty
		assert.Empty(t, buf.String(), "suppressed logs should not reach handler")
	})

	t.Run("handle forwards logs when otel is not suppressed", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		handler := &otelSuppressibleHandler{inner: innerHandler}
		logger := slog.New(handler)

		// Log without suppression
		logger.InfoContext(t.Context(), "normal message")

		// Buffer should contain the message
		assert.Contains(t, buf.String(), "normal message")
	})

	t.Run("with attrs preserves suppressibility", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		handler := &otelSuppressibleHandler{inner: innerHandler}
		newHandler := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})

		logger := slog.New(newHandler)
		ctx := WithSuppressOtel(t.Context(), true)
		logger.InfoContext(ctx, "test")

		// Should still be suppressed
		assert.Empty(t, buf.String(), "suppression should work after WithAttrs")
	})

	t.Run("with group preserves suppressibility", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		handler := &otelSuppressibleHandler{inner: innerHandler}
		newHandler := handler.WithGroup("testgroup")

		logger := slog.New(newHandler)
		ctx := WithSuppressOtel(t.Context(), true)
		logger.InfoContext(ctx, "test")

		// Should still be suppressed
		assert.Empty(t, buf.String(), "suppression should work after WithGroup")
	})
}

// TestOtelSuppressionIntegration tests the full OTel suppression integration.
func TestOtelSuppressionIntegration(t *testing.T) { //nolint:paralleltest
	t.Run("otel enabled but suppressed via context", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		ConfigureLoggingWithOptions(Options{
			Subsystem:  "otel-suppress-test",
			JSON:       true,
			Output:     &buf,
			MinLevel:   slog.LevelInfo,
			EnableOtel: true,
		})

		// Log with OTel suppression
		ctx := WithSuppressOtel(t.Context(), true)
		Get(ctx).InfoContext(ctx, "test message with otel suppressed")

		output := buf.String()
		// Console output should still work
		assert.Contains(t, output, "test message with otel suppressed")
		assert.Contains(t, output, "otel-suppress-test")
	})

	t.Run("otel enabled and not suppressed", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		ConfigureLoggingWithOptions(Options{
			Subsystem:  "otel-normal-test",
			JSON:       true,
			Output:     &buf,
			MinLevel:   slog.LevelInfo,
			EnableOtel: true,
		})

		// Log without OTel suppression
		ctx := t.Context()
		Get(ctx).InfoContext(ctx, "test message with otel enabled")

		output := buf.String()
		// Console output should work
		assert.Contains(t, output, "test message with otel enabled")
		assert.Contains(t, output, "otel-normal-test")
	})

	t.Run("otel not configured - suppress flag has no effect", func(t *testing.T) { //nolint:paralleltest
		var buf bytes.Buffer

		ConfigureLoggingWithOptions(Options{
			Subsystem:  "no-otel-test",
			JSON:       true,
			Output:     &buf,
			MinLevel:   slog.LevelInfo,
			EnableOtel: false,
		})

		// Log with suppress flag (should have no effect since OTel is not enabled)
		ctx := WithSuppressOtel(t.Context(), true)
		Get(ctx).InfoContext(ctx, "test message with otel disabled")

		output := buf.String()
		// Console output should still work
		assert.Contains(t, output, "test message with otel disabled")
		assert.Contains(t, output, "no-otel-test")
	})
}

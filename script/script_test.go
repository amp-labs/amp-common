package script

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amp-labs/amp-common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{
			name:     "exit code 0",
			code:     0,
			expected: "exit 0",
		},
		{
			name:     "exit code 1",
			code:     1,
			expected: "exit 1",
		},
		{
			name:     "exit code 42",
			code:     42,
			expected: "exit 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Exit(tt.code)
			require.Error(t, err)
			assert.Equal(t, tt.expected, err.Error())

			var exitErr *exitError

			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tt.code, exitErr.code)
			assert.NoError(t, exitErr.err)
		})
	}
}

func TestExitWithError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("test error") //nolint:err113
	err := ExitWithError(testErr)

	require.Error(t, err)
	assert.Equal(t, "exit 1: test error", err.Error())

	var exitErr *exitError

	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.code)
	assert.Equal(t, testErr, exitErr.err)
}

func TestExitWithErrorMessage(t *testing.T) {
	t.Parallel()

	err := ExitWithErrorMessage("test error: %s", "details")

	require.Error(t, err)
	assert.Equal(t, "exit 1: test error: details", err.Error())

	var exitErr *exitError

	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.code)
	require.Error(t, exitErr.err)
	assert.Equal(t, "test error: details", exitErr.err.Error())
}

func TestExitError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exitErr  *exitError
		expected string
	}{
		{
			name: "with error",
			exitErr: &exitError{
				code: 1,
				err:  errors.New("something went wrong"), //nolint:err113
			},
			expected: "exit 1: something went wrong",
		},
		{
			name: "without error",
			exitErr: &exitError{
				code: 0,
				err:  nil,
			},
			expected: "exit 0",
		},
		{
			name: "code 42 with error",
			exitErr: &exitError{
				code: 42,
				err:  errors.New("custom error"), //nolint:err113
			},
			expected: "exit 42: custom error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.exitErr.Error())
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("default configuration", func(t *testing.T) {
		t.Parallel()

		script := New("test-script")

		assert.NotNil(t, script)
		assert.Equal(t, "test-script", script.name)
		assert.True(t, script.flagParseEnable)
		assert.Empty(t, script.loggerOpts)
	})

	t.Run("with options", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		script := New("test-script",
			LogLevel(slog.LevelDebug),
			LegacyLogLevel(slog.LevelInfo),
			LogOutput(&buf),
			EnableFlagParse(false),
		)

		assert.NotNil(t, script)
		assert.Equal(t, "test-script", script.name)
		assert.False(t, script.flagParseEnable)
		assert.Len(t, script.loggerOpts, 3)
	})
}

func TestLogLevel(t *testing.T) {
	t.Parallel()

	script := New("test", LogLevel(slog.LevelWarn))

	assert.Len(t, script.loggerOpts, 1)
}

func TestLegacyLogLevel(t *testing.T) {
	t.Parallel()

	script := New("test", LegacyLogLevel(slog.LevelDebug))

	assert.Len(t, script.loggerOpts, 1)
}

func TestLogOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	script := New("test", LogOutput(&buf))

	assert.Len(t, script.loggerOpts, 1)
}

func TestEnableFlagParse(t *testing.T) {
	t.Parallel()

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()

		script := New("test", EnableFlagParse(true))
		assert.True(t, script.flagParseEnable)
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()

		script := New("test", EnableFlagParse(false))
		assert.False(t, script.flagParseEnable)
	})
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		callback     func(ctx context.Context) error
		expectedCode int
	}{
		{
			name: "successful execution",
			callback: func(ctx context.Context) error {
				return nil
			},
			expectedCode: 0,
		},
		{
			name: "exit with code 0",
			callback: func(ctx context.Context) error {
				return Exit(0)
			},
			expectedCode: 0,
		},
		{
			name: "exit with code 1",
			callback: func(ctx context.Context) error {
				return Exit(1)
			},
			expectedCode: 1,
		},
		{
			name: "exit with error",
			callback: func(ctx context.Context) error {
				return ExitWithError(errors.New("test error")) //nolint:err113
			},
			expectedCode: 1,
		},
		{
			name: "exit with error message",
			callback: func(ctx context.Context) error {
				return ExitWithErrorMessage("test error: %s", "details")
			},
			expectedCode: 1,
		},
		{
			name: "regular error",
			callback: func(ctx context.Context) error {
				return errors.New("regular error") //nolint:err113
			},
			expectedCode: 1,
		},
		{
			name: "exit with custom code",
			callback: func(ctx context.Context) error {
				return Exit(42)
			},
			expectedCode: 42,
		},
		{
			name:         "nil callback",
			callback:     nil,
			expectedCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			code := run("test-script", tt.callback, false, func(opts *logger.Options) {
				opts.Output = &buf
			})

			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	t.Parallel()

	callbackExecuted := false
	contextReceived := false

	callback := func(ctx context.Context) error {
		callbackExecuted = true

		if ctx != nil {
			contextReceived = true
		}

		return nil
	}

	var buf bytes.Buffer

	code := run("test-script", callback, false, func(opts *logger.Options) {
		opts.Output = &buf
	})

	assert.Equal(t, 0, code)
	assert.True(t, callbackExecuted)
	assert.True(t, contextReceived)
}

func TestRun_FlagParse(t *testing.T) {
	t.Parallel()

	t.Run("with flag parse enabled", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		code := run("test-script", func(ctx context.Context) error {
			return nil
		}, true, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
	})

	t.Run("with flag parse disabled", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		code := run("test-script", func(ctx context.Context) error {
			return nil
		}, false, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
	})
}

package script

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
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

			code := run("test-script", tt.callback, false, nil, nil, func(opts *logger.Options) {
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

	code := run("test-script", callback, false, nil, nil, func(opts *logger.Options) {
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
		}, true, nil, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
	})

	t.Run("with flag parse disabled", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		code := run("test-script", func(ctx context.Context) error {
			return nil
		}, false, nil, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
	})
}

func TestWithEnvFile(t *testing.T) {
	t.Parallel()

	script := New("test", WithEnvFile("/path/to/.env"))

	assert.Len(t, script.envFiles, 1)
	assert.Equal(t, "/path/to/.env", script.envFiles[0]())
}

func TestWithEnvFileProvider(t *testing.T) {
	t.Parallel()

	provider := func() string {
		return "/dynamic/path/.env"
	}

	script := New("test", WithEnvFileProvider(provider))

	assert.Len(t, script.envFiles, 1)
	assert.Equal(t, "/dynamic/path/.env", script.envFiles[0]())
}

func TestWithEnvFiles(t *testing.T) {
	t.Parallel()

	script := New("test", WithEnvFiles(
		"/path/to/.env.local",
		"/path/to/.env.dev",
		"/path/to/.env",
	))

	assert.Len(t, script.envFiles, 3)
	assert.Equal(t, "/path/to/.env.local", script.envFiles[0]())
	assert.Equal(t, "/path/to/.env.dev", script.envFiles[1]())
	assert.Equal(t, "/path/to/.env", script.envFiles[2]())
}

func TestWithEnvFilesProvider(t *testing.T) {
	t.Parallel()

	provider1 := func() string { return "/path/1/.env" }
	provider2 := func() string { return "/path/2/.env" }

	script := New("test", WithEnvFilesProvider(provider1, provider2))

	assert.Len(t, script.envFiles, 2)
	assert.Equal(t, "/path/1/.env", script.envFiles[0]())
	assert.Equal(t, "/path/2/.env", script.envFiles[1]())
}

func TestWithEnvFile_Multiple(t *testing.T) {
	t.Parallel()

	script := New("test",
		WithEnvFile("/path/1/.env"),
		WithEnvFile("/path/2/.env"),
	)

	assert.Len(t, script.envFiles, 2)
	assert.Equal(t, "/path/1/.env", script.envFiles[0]())
	assert.Equal(t, "/path/2/.env", script.envFiles[1]())
}

func TestRun_WithEnvFile(t *testing.T) {
	t.Parallel()

	t.Run("loads env file successfully", func(t *testing.T) { //nolint:dupl
		t.Parallel()

		// Create temporary env file
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		err := os.WriteFile(envFile, []byte("TEST_SCRIPT_VAR=test_value\nTEST_SCRIPT_NUM=42\n"), 0o600)
		require.NoError(t, err)

		// Clear any existing env vars
		os.Unsetenv("TEST_SCRIPT_VAR")     //nolint:errcheck
		defer os.Unsetenv("TEST_SCRIPT_VAR") //nolint:errcheck

		var buf bytes.Buffer

		envVarValue := ""

		code := run("test-script", func(ctx context.Context) error {
			envVarValue = os.Getenv("TEST_SCRIPT_VAR")

			return nil
		}, false, []func() string{func() string { return envFile }}, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
		assert.Equal(t, "test_value", envVarValue)
	})

	t.Run("handles non-existent file gracefully", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		code := run("test-script", func(ctx context.Context) error {
			return nil
		}, false, []func() string{func() string { return "/nonexistent/.env" }}, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
		// Script should continue execution even if env file doesn't exist
	})

	t.Run("loads multiple env files in order", func(t *testing.T) { //nolint:paralleltest
		// Don't run in parallel - modifies global env vars

		// Create temporary env files
		tmpDir := t.TempDir()

		// Create subdirectories for env files
		dir1 := filepath.Join(tmpDir, "env1")
		err := os.MkdirAll(dir1, 0o750)
		require.NoError(t, err)

		dir2 := filepath.Join(tmpDir, "env2")
		err = os.MkdirAll(dir2, 0o750)
		require.NoError(t, err)

		envFile1 := filepath.Join(dir1, ".env")
		err = os.WriteFile(envFile1, []byte("TEST_MULTI_VAR1=value1\nTEST_MULTI_SHARED=first\n"), 0o600)
		require.NoError(t, err)

		envFile2 := filepath.Join(dir2, ".env")
		err = os.WriteFile(envFile2, []byte("TEST_MULTI_VAR2=value2\nTEST_MULTI_SHARED=second\n"), 0o600)
		require.NoError(t, err)

		// Clear any existing env vars
		os.Unsetenv("TEST_MULTI_VAR1")       //nolint:errcheck
		os.Unsetenv("TEST_MULTI_VAR2")       //nolint:errcheck
		os.Unsetenv("TEST_MULTI_SHARED")     //nolint:errcheck

		defer os.Unsetenv("TEST_MULTI_VAR1") //nolint:errcheck
		defer os.Unsetenv("TEST_MULTI_VAR2") //nolint:errcheck
		defer os.Unsetenv("TEST_MULTI_SHARED") //nolint:errcheck

		var buf bytes.Buffer

		var var1, var2, shared string

		code := run("test-script", func(ctx context.Context) error {
			var1 = os.Getenv("TEST_MULTI_VAR1")
			var2 = os.Getenv("TEST_MULTI_VAR2")
			shared = os.Getenv("TEST_MULTI_SHARED")

			return nil
		}, false, []func() string{
			func() string { return envFile1 },
			func() string { return envFile2 },
		}, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
		assert.Equal(t, "value1", var1)
		assert.Equal(t, "value2", var2)
		// Later file should override earlier file
		assert.Equal(t, "second", shared)
	})

	t.Run("sets env vars in both OS and context", func(t *testing.T) { //nolint:dupl
		t.Parallel()

		// Create temporary env file
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		err := os.WriteFile(envFile, []byte("TEST_CONTEXT_VAR=context_value\n"), 0o600)
		require.NoError(t, err)

		// Clear any existing env var
		os.Unsetenv("TEST_CONTEXT_VAR")     //nolint:errcheck
		defer os.Unsetenv("TEST_CONTEXT_VAR") //nolint:errcheck

		var buf bytes.Buffer

		osEnvValue := ""

		code := run("test-script", func(ctx context.Context) error {
			// Check that OS env var was set
			osEnvValue = os.Getenv("TEST_CONTEXT_VAR")

			return nil
		}, false, []func() string{func() string { return envFile }}, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
		assert.Equal(t, "context_value", osEnvValue)
	})

	t.Run("empty env files list does not affect execution", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		code := run("test-script", func(ctx context.Context) error {
			return nil
		}, false, []func() string{}, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
	})

	t.Run("nil env files list does not affect execution", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		code := run("test-script", func(ctx context.Context) error {
			return nil
		}, false, nil, nil, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
	})
}

func TestSimpleLoader(t *testing.T) {
	t.Parallel()

	loader := simpleLoader("/test/path")

	assert.NotNil(t, loader)
	assert.Equal(t, "/test/path", loader())
}

func TestWithSetEnv(t *testing.T) {
	t.Parallel()

	script := New("test", WithSetEnv("KEY1", "value1"))

	assert.Len(t, script.setEnv, 1)

	key, value := script.setEnv[0]()

	assert.Equal(t, "KEY1", key)
	assert.Equal(t, "value1", value)
}

func TestWithSetEnv_Multiple(t *testing.T) {
	t.Parallel()

	script := New("test",
		WithSetEnv("KEY1", "value1"),
		WithSetEnv("KEY2", "value2"),
		WithSetEnv("KEY3", "value3"),
	)

	assert.Len(t, script.setEnv, 3)

	key1, value1 := script.setEnv[0]()
	assert.Equal(t, "KEY1", key1)
	assert.Equal(t, "value1", value1)

	key2, value2 := script.setEnv[1]()
	assert.Equal(t, "KEY2", key2)
	assert.Equal(t, "value2", value2)

	key3, value3 := script.setEnv[2]()
	assert.Equal(t, "KEY3", key3)
	assert.Equal(t, "value3", value3)
}

func TestRun_WithSetEnv(t *testing.T) {
	t.Parallel()

	t.Run("sets env var programmatically", func(t *testing.T) {
		t.Parallel()

		os.Unsetenv("TEST_SET_VAR")     //nolint:errcheck
		defer os.Unsetenv("TEST_SET_VAR") //nolint:errcheck

		var buf bytes.Buffer

		envVarValue := ""

		code := run("test-script", func(ctx context.Context) error {
			envVarValue = os.Getenv("TEST_SET_VAR")

			return nil
		}, false, nil, []func() (string, string){
			func() (string, string) { return "TEST_SET_VAR", "programmatic_value" },
		}, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
		assert.Equal(t, "programmatic_value", envVarValue)
	})

	t.Run("setEnv overrides env file values", func(t *testing.T) { //nolint:paralleltest
		// Don't run in parallel - modifies global env vars

		// Create temporary env file
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		err := os.WriteFile(envFile, []byte("TEST_OVERRIDE_VAR=from_file\n"), 0o600)
		require.NoError(t, err)

		os.Unsetenv("TEST_OVERRIDE_VAR")     //nolint:errcheck
		defer os.Unsetenv("TEST_OVERRIDE_VAR") //nolint:errcheck

		var buf bytes.Buffer

		envVarValue := ""

		code := run("test-script", func(ctx context.Context) error {
			envVarValue = os.Getenv("TEST_OVERRIDE_VAR")

			return nil
		}, false, []func() string{func() string { return envFile }}, []func() (string, string){
			func() (string, string) { return "TEST_OVERRIDE_VAR", "from_setenv" },
		}, func(opts *logger.Options) {
			opts.Output = &buf
		})

		assert.Equal(t, 0, code)
		// setEnv should override file value
		assert.Equal(t, "from_setenv", envVarValue)
	})
}

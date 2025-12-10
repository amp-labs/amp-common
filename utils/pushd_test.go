package utils //nolint:revive // utils is an appropriate package name for utility functions

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushd(t *testing.T) {
	t.Parallel()

	t.Run("executes function in different directory and returns", func(t *testing.T) { //nolint:paralleltest
		// NOTE: Cannot run in parallel because Pushd modifies global process state (working directory)
		// Get current working directory
		originalWd, err := os.Getwd()
		require.NoError(t, err)

		// Create temp directory
		tempDir := t.TempDir()

		// Track whether function was executed
		var functionExecuted bool

		err = Pushd(tempDir, func() error {
			functionExecuted = true

			return nil
		})

		require.NoError(t, err)
		assert.True(t, functionExecuted, "function should have been executed")

		// Verify we're back in original directory
		currentWd, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalWd, currentWd)
	})

	t.Run("returns to original directory even if function returns error", func(t *testing.T) { //nolint:paralleltest
		// NOTE: Cannot run in parallel because Pushd modifies global process state (working directory)
		originalWd, err := os.Getwd()
		require.NoError(t, err)

		tempDir := t.TempDir()

		testErr := errors.New("test error") //nolint:err113
		err = Pushd(tempDir, func() error {
			return testErr
		})

		require.Error(t, err)
		require.ErrorIs(t, err, testErr)

		// Verify we're back in original directory despite error
		currentWd, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalWd, currentWd)
	})

	t.Run("does not change directory when path is '.'", func(t *testing.T) {
		t.Parallel()

		originalWd, err := os.Getwd()
		require.NoError(t, err)

		executed := false
		err = Pushd(".", func() error {
			executed = true

			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)

		currentWd, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalWd, currentWd)
	})

	t.Run("returns error when directory does not exist", func(t *testing.T) {
		t.Parallel()

		err := Pushd("/nonexistent/directory/path", func() error {
			return nil
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrChdir)
	})
}

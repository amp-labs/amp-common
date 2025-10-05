package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollection_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds non-nil errors", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		err1 := errors.New("error 1") //nolint:err113
		err2 := errors.New("error 2") //nolint:err113

		c.Add(err1)
		c.Add(err2)

		assert.True(t, c.HasError())
		assert.Len(t, c.errors, 2)
	})

	t.Run("ignores nil errors", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}

		c.Add(nil)

		assert.False(t, c.HasError())
		assert.Empty(t, c.errors)
	})

	t.Run("handles mixed nil and non-nil errors", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		err1 := errors.New("error 1") //nolint:err113

		c.Add(err1)
		c.Add(nil)
		c.Add(errors.New("error 2")) //nolint:err113

		assert.True(t, c.HasError())
		assert.Len(t, c.errors, 2)
	})
}

func TestCollection_Clear(t *testing.T) {
	t.Parallel()

	t.Run("clears all errors", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		c.Add(errors.New("error 1")) //nolint:err113
		c.Add(errors.New("error 2")) //nolint:err113

		c.Clear()

		assert.False(t, c.HasError())
		assert.Empty(t, c.errors)
	})

	t.Run("can be called on empty collection", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}

		c.Clear()

		assert.False(t, c.HasError())
		assert.Empty(t, c.errors)
	})
}

func TestCollection_HasError(t *testing.T) {
	t.Parallel()

	t.Run("returns false when empty", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}

		assert.False(t, c.HasError())
	})

	t.Run("returns true when errors exist", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		c.Add(errors.New("error")) //nolint:err113

		assert.True(t, c.HasError())
	})

	t.Run("returns false after clear", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		c.Add(errors.New("error")) //nolint:err113
		c.Clear()

		assert.False(t, c.HasError())
	})
}

func TestCollection_GetError(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when empty", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}

		err := c.GetError()

		assert.NoError(t, err)
	})

	t.Run("returns single error", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		err1 := errors.New("error 1") //nolint:err113
		c.Add(err1)

		err := c.GetError()

		assert.Equal(t, err1, err)
	})

	t.Run("returns joined errors for multiple errors", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		err1 := errors.New("error 1") //nolint:err113
		err2 := errors.New("error 2") //nolint:err113
		err3 := errors.New("error 3") //nolint:err113

		c.Add(err1)
		c.Add(err2)
		c.Add(err3)

		err := c.GetError()

		require.Error(t, err)
		// Verify that the joined error contains all original errors
		require.ErrorIs(t, err, err1)
		require.ErrorIs(t, err, err2)
		require.ErrorIs(t, err, err3)
	})

	t.Run("returns nil after clear", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		c.Add(errors.New("error")) //nolint:err113
		c.Clear()

		err := c.GetError()

		assert.NoError(t, err)
	})
}

func TestCollection_Integration(t *testing.T) {
	t.Parallel()

	t.Run("typical usage pattern", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}
		operationErr := errors.New("operation failed") //nolint:err113

		// Simulate collecting errors from multiple operations
		c.Add(performOperation(true, operationErr)) // Returns error
		c.Add(performOperation(false, nil))         // Returns nil
		c.Add(performOperation(true, operationErr)) // Returns error

		assert.True(t, c.HasError())

		err := c.GetError()
		require.Error(t, err)
		assert.ErrorIs(t, err, operationErr)
	})

	t.Run("reuse collection after clear", func(t *testing.T) {
		t.Parallel()

		c := &Collection{}

		c.Add(errors.New("first batch")) //nolint:err113
		assert.True(t, c.HasError())

		c.Clear()
		assert.False(t, c.HasError())

		c.Add(errors.New("second batch")) //nolint:err113
		assert.True(t, c.HasError())

		err := c.GetError()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "second batch")
	})
}

// Helper function for integration test.
func performOperation(shouldFail bool, err error) error {
	if shouldFail {
		return err
	}

	return nil
}

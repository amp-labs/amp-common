package utils //nolint:revive // utils is an appropriate package name for utility functions

import (
	"hash"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCollectable string

func (t testCollectable) Equals(other testCollectable) bool {
	return t == other
}

func (t testCollectable) UpdateHash(h hash.Hash) error {
	_, err := h.Write([]byte(t))

	return err
}

func TestDeduplicateValues(t *testing.T) {
	t.Parallel()

	t.Run("removes duplicates from slice", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "b", "a", "c", "b"}
		result, err := DeduplicateValues(input)
		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Contains(t, result, testCollectable("a"))
		assert.Contains(t, result, testCollectable("b"))
		assert.Contains(t, result, testCollectable("c"))
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{}
		result, err := DeduplicateValues(input)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns same slice when no duplicates", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "b", "c"}
		result, err := DeduplicateValues(input)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})
}

func TestHasDuplicateValues(t *testing.T) {
	t.Parallel()

	t.Run("returns true when duplicates exist", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "b", "a"}
		result, err := HasDuplicateValues(input)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("returns false when no duplicates", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "b", "c"}
		result, err := HasDuplicateValues(input)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("returns false for empty slice", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{}
		result, err := HasDuplicateValues(input)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("returns false for single element", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a"}
		result, err := HasDuplicateValues(input)
		require.NoError(t, err)
		assert.False(t, result)
	})
}

func TestCountDuplicateValues(t *testing.T) {
	t.Parallel()

	t.Run("counts duplicates correctly", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "a", "b", "b", "b"}
		count, err := CountDuplicateValues(input)
		require.NoError(t, err)
		assert.Equal(t, 3, count) // 1 extra 'a' and 2 extra 'b's
	})

	t.Run("returns 0 for no duplicates", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "b", "c"}
		count, err := CountDuplicateValues(input)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns 0 for empty slice", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{}
		count, err := CountDuplicateValues(input)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("counts all duplicates when all same", func(t *testing.T) {
		t.Parallel()

		input := []testCollectable{"a", "a", "a", "a"}
		count, err := CountDuplicateValues(input)
		require.NoError(t, err)
		assert.Equal(t, 3, count) // 3 duplicates of 'a'
	})
}

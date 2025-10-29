package set

import (
	"errors"
	"fmt"
	"hash"
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errGenerationFailed = errors.New("generation failed")

type testElem struct {
	value string
}

func (e testElem) UpdateHash(h hash.Hash) error {
	_, err := h.Write([]byte(e.value))

	return err
}

func (e testElem) Equals(other testElem) bool {
	return e.value == other.value
}

func TestNewDefaultSet(t *testing.T) {
	t.Parallel()

	t.Run("creates default set with set storage", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		require.NotNil(t, s)
		assert.Equal(t, 0, s.Size())
	})
}

func TestDefaultSetContains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)

		contains, err := s.Contains(elem)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("generates default and returns true for missing element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: elem.value + "-normalized"}, nil
		})

		elem := testElem{value: "test"}
		contains, err := s.Contains(elem)
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, 1, s.Size())

		// Verify normalized value was actually added
		normalized := testElem{value: "test-normalized"}
		contains, err = s.Contains(normalized)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns false when default function returns ErrNoDefaultValue", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, ErrNoDefaultValue
		})

		elem := testElem{value: "missing"}
		contains, err := s.Contains(elem)
		require.NoError(t, err)
		assert.False(t, contains)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("returns error when default function fails", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, errGenerationFailed
		})

		elem := testElem{value: "missing"}
		contains, err := s.Contains(elem)
		require.Error(t, err)
		assert.False(t, contains)
		assert.ErrorIs(t, err, errGenerationFailed)
	})

	t.Run("default function receives correct element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)

		var receivedElem testElem

		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			receivedElem = elem

			return testElem{value: elem.value + "-default"}, nil
		})

		elem := testElem{value: "myelem"}
		contains, err := s.Contains(elem)
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, elem, receivedElem)
	})
}

func TestDefaultSetAdd(t *testing.T) {
	t.Parallel()

	t.Run("adds new element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())

		contains, err := s.Contains(elem)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("no error for duplicate element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)
		err = s.Add(elem)
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("bypasses default function", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		called := false
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			called = true

			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)
		assert.False(t, called)
	})
}

func TestDefaultSetAddAll(t *testing.T) {
	t.Parallel()

	t.Run("adds multiple elements", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		err := s.AddAll(
			testElem{value: "a"},
			testElem{value: "b"},
			testElem{value: "c"},
		)
		require.NoError(t, err)
		assert.Equal(t, 3, s.Size())
	})

	t.Run("bypasses default function", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		called := false
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			called = true

			return testElem{value: "default"}, nil
		})

		err := s.AddAll(testElem{value: "a"}, testElem{value: "b"})
		require.NoError(t, err)
		assert.False(t, called)
	})
}

func TestDefaultSetRemove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)
		err = s.Remove(elem)
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("no-op for non-existent element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "missing"}
		err := s.Remove(elem)
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())
	})
}

func TestDefaultSetClear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		for i := range 10 {
			elem := testElem{value: fmt.Sprintf("elem%d", i)}
			err := s.Add(elem)
			require.NoError(t, err)
		}

		assert.Equal(t, 10, s.Size())
		s.Clear()
		assert.Equal(t, 0, s.Size())
	})
}

func TestDefaultSetSize(t *testing.T) {
	t.Parallel()

	t.Run("returns correct size", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		assert.Equal(t, 0, s.Size())

		for i := range 5 {
			elem := testElem{value: fmt.Sprintf("elem%d", i)}
			err := s.Add(elem)
			require.NoError(t, err)
			assert.Equal(t, i+1, s.Size())
		}
	})
}

func TestDefaultSetEntries(t *testing.T) {
	t.Parallel()

	t.Run("returns all elements", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		expected := []testElem{
			{value: "a"},
			{value: "b"},
			{value: "c"},
		}

		for _, elem := range expected {
			err := s.Add(elem)
			require.NoError(t, err)
		}

		entries := s.Entries()
		assert.Len(t, entries, 3)
		assert.ElementsMatch(t, expected, entries)
	})
}

func TestDefaultSetSeq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		expected := make(map[string]bool)

		for i := range 5 {
			elem := testElem{value: fmt.Sprintf("elem%d", i)}
			err := s.Add(elem)
			require.NoError(t, err)

			expected[elem.value] = true
		}

		count := 0
		for elem := range s.Seq() {
			count++

			assert.True(t, expected[elem.value])
		}

		assert.Equal(t, 5, count)
	})
}

func TestDefaultSetUnion(t *testing.T) {
	t.Parallel()

	t.Run("combines two sets", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s1 := NewDefaultSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		baseSet2 := NewSet[testElem](hashing.Sha256)

		err := s1.Add(testElem{value: "a"})
		require.NoError(t, err)
		err = s1.Add(testElem{value: "b"})
		require.NoError(t, err)

		err = baseSet2.Add(testElem{value: "c"})
		require.NoError(t, err)
		err = baseSet2.Add(testElem{value: "b"})
		require.NoError(t, err)

		result, err := s1.Union(baseSet2)
		require.NoError(t, err)
		assert.Equal(t, 3, result.Size())

		contains, err := result.Contains(testElem{value: "a"})
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(testElem{value: "b"})
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(testElem{value: "c"})
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewSet[testElem](hashing.Sha256)
		s1 := NewDefaultSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: elem.value + "-default"}, nil
		})

		baseSet2 := NewSet[testElem](hashing.Sha256)

		result, err := s1.Union(baseSet2)
		require.NoError(t, err)

		// Test that default function works on result
		contains, err := result.Contains(testElem{value: "missing"})
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, 1, result.Size())

		// Verify the default function was applied
		contains, err = result.Contains(testElem{value: "missing-default"})
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

func TestDefaultSetIntersection(t *testing.T) {
	t.Parallel()

	t.Run("returns only common elements", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s1 := NewDefaultSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		baseSet2 := NewSet[testElem](hashing.Sha256)

		err := s1.Add(testElem{value: "a"})
		require.NoError(t, err)
		err = s1.Add(testElem{value: "b"})
		require.NoError(t, err)
		err = s1.Add(testElem{value: "c"})
		require.NoError(t, err)

		err = baseSet2.Add(testElem{value: "b"})
		require.NoError(t, err)
		err = baseSet2.Add(testElem{value: "c"})
		require.NoError(t, err)
		err = baseSet2.Add(testElem{value: "d"})
		require.NoError(t, err)

		result, err := s1.Intersection(baseSet2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		// Check the entries directly to avoid triggering default value generation
		entries := result.Entries()
		assert.Len(t, entries, 2)
		assert.ElementsMatch(t, []testElem{
			{value: "b"},
			{value: "c"},
		}, entries)
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewSet[testElem](hashing.Sha256)
		s1 := NewDefaultSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: elem.value + "-default"}, nil
		})

		baseSet2 := NewSet[testElem](hashing.Sha256)

		result, err := s1.Intersection(baseSet2)
		require.NoError(t, err)

		// Test that default function works on result
		contains, err := result.Contains(testElem{value: "missing"})
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, 1, result.Size())
	})
}

func TestDefaultSetHashFunction(t *testing.T) {
	t.Parallel()

	t.Run("returns underlying hash function", func(t *testing.T) {
		t.Parallel()

		baseSet := NewSet[testElem](hashing.Sha256)
		s := NewDefaultSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		hashFunc := s.HashFunction()
		assert.NotNil(t, hashFunc)
	})
}

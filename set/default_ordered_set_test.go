package set

import (
	"fmt"
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultOrderedSet(t *testing.T) {
	t.Parallel()

	t.Run("creates default ordered set with ordered set storage", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		require.NotNil(t, s)
		assert.Equal(t, 0, s.Size())
	})
}

func TestDefaultOrderedSetContains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

		baseSet := NewOrderedSet[testElem](hashing.Sha256)

		var receivedElem testElem

		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			receivedElem = elem

			return testElem{value: elem.value + "-default"}, nil
		})

		elem := testElem{value: "myelem"}
		contains, err := s.Contains(elem)
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, elem, receivedElem)
	})

	t.Run("generated default is appended to end of insertion order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: elem.value + "-default"}, nil
		})

		// Add some elements directly
		err := s.Add(testElem{value: "a"})
		require.NoError(t, err)
		err = s.Add(testElem{value: "b"})
		require.NoError(t, err)

		// Trigger default generation
		contains, err := s.Contains(testElem{value: "c"})
		require.NoError(t, err)
		assert.True(t, contains)

		// Check order
		entries := s.Entries()
		assert.Equal(t, []testElem{
			{value: "a"},
			{value: "b"},
			{value: "c-default"},
		}, entries)
	})
}

func TestDefaultOrderedSetAdd(t *testing.T) {
	t.Parallel()

	t.Run("adds new element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

	t.Run("no error for duplicate element and preserves order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)
		err = s.Add(elem)
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("maintains insertion order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{}, nil
		})

		elements := []testElem{
			{value: "first"},
			{value: "second"},
			{value: "third"},
		}

		for _, elem := range elements {
			err := s.Add(elem)
			require.NoError(t, err)
		}

		entries := s.Entries()
		assert.Equal(t, elements, entries)
	})

	t.Run("bypasses default function", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		called := false
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			called = true

			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "test"}
		err := s.Add(elem)
		require.NoError(t, err)
		assert.False(t, called)
	})
}

func TestDefaultOrderedSetAddAll(t *testing.T) {
	t.Parallel()

	t.Run("adds multiple elements in order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		elements := []testElem{
			{value: "a"},
			{value: "b"},
			{value: "c"},
		}

		err := s.AddAll(elements...)
		require.NoError(t, err)
		assert.Equal(t, 3, s.Size())

		entries := s.Entries()
		assert.Equal(t, elements, entries)
	})

	t.Run("bypasses default function", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		called := false
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			called = true

			return testElem{value: "default"}, nil
		})

		err := s.AddAll(testElem{value: "a"}, testElem{value: "b"})
		require.NoError(t, err)
		assert.False(t, called)
	})
}

func TestDefaultOrderedSetRemove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing element and maintains order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		err := s.AddAll(
			testElem{value: "a"},
			testElem{value: "b"},
			testElem{value: "c"},
		)
		require.NoError(t, err)

		err = s.Remove(testElem{value: "b"})
		require.NoError(t, err)
		assert.Equal(t, 2, s.Size())

		entries := s.Entries()
		assert.Equal(t, []testElem{
			{value: "a"},
			{value: "c"},
		}, entries)
	})

	t.Run("no-op for non-existent element", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		elem := testElem{value: "missing"}
		err := s.Remove(elem)
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())
	})
}

func TestDefaultOrderedSetClear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

func TestDefaultOrderedSetSize(t *testing.T) {
	t.Parallel()

	t.Run("returns correct size", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

func TestDefaultOrderedSetEntries(t *testing.T) {
	t.Parallel()

	t.Run("returns all elements in insertion order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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
		assert.Equal(t, expected, entries)
	})
}

func TestDefaultOrderedSetSeq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries in insertion order", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
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

		var result []testElem
		for idx, elem := range s.Seq() {
			assert.Len(t, result, idx)
			result = append(result, elem)
		}

		assert.Equal(t, expected, result)
	})
}

func TestDefaultOrderedSetUnion(t *testing.T) {
	t.Parallel()

	t.Run("combines two sets maintaining order", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewOrderedSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s1 := NewDefaultOrderedSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		baseSet2 := NewOrderedSet[testElem](hashing.Sha256)

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

		// Check order: s1's elements first, then s2's new elements
		entries := result.Entries()
		assert.Equal(t, []testElem{
			{value: "a"},
			{value: "b"},
			{value: "c"},
		}, entries)
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewOrderedSet[testElem](hashing.Sha256)
		s1 := NewDefaultOrderedSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: elem.value + "-default"}, nil
		})

		baseSet2 := NewOrderedSet[testElem](hashing.Sha256)

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

func TestDefaultOrderedSetIntersection(t *testing.T) {
	t.Parallel()

	t.Run("returns only common elements maintaining order from first set", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewOrderedSet[testElem](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		s1 := NewDefaultOrderedSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		baseSet2 := NewOrderedSet[testElem](hashing.Sha256)

		err := s1.AddAll(
			testElem{value: "a"},
			testElem{value: "b"},
			testElem{value: "c"},
		)
		require.NoError(t, err)

		err = baseSet2.AddAll(
			testElem{value: "b"},
			testElem{value: "c"},
			testElem{value: "d"},
		)
		require.NoError(t, err)

		result, err := s1.Intersection(baseSet2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		// Check order preserved from s1
		entries := result.Entries()
		assert.Equal(t, []testElem{
			{value: "b"},
			{value: "c"},
		}, entries)
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseSet1 := NewOrderedSet[testElem](hashing.Sha256)
		s1 := NewDefaultOrderedSet(baseSet1, func(elem testElem) (testElem, error) {
			return testElem{value: elem.value + "-default"}, nil
		})

		baseSet2 := NewOrderedSet[testElem](hashing.Sha256)

		result, err := s1.Intersection(baseSet2)
		require.NoError(t, err)

		// Test that default function works on result
		contains, err := result.Contains(testElem{value: "missing"})
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, 1, result.Size())
	})
}

func TestDefaultOrderedSetHashFunction(t *testing.T) {
	t.Parallel()

	t.Run("returns underlying hash function", func(t *testing.T) {
		t.Parallel()

		baseSet := NewOrderedSet[testElem](hashing.Sha256)
		s := NewDefaultOrderedSet(baseSet, func(elem testElem) (testElem, error) {
			return testElem{value: "default"}, nil
		})

		hashFunc := s.HashFunction()
		assert.NotNil(t, hashFunc)
	})
}

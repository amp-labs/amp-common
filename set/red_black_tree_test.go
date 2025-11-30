package set

import (
	"fmt"
	"testing"

	"github.com/amp-labs/amp-common/sortable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedBlackTreeSet(t *testing.T) {
	t.Parallel()

	t.Run("creates empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		require.NotNil(t, s)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("set is usable immediately", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})
}

func TestRedBlackTreeSet_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("no error for duplicate element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		err = s.Add(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("handles multiple different elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 100 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 100, s.Size())
	})

	t.Run("maintains sorted order", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		// Insert in random order
		elements := []int{5, 2, 8, 1, 9, 3, 7, 4, 6}
		for _, elem := range elements {
			err := s.Add(sortable.Int(elem))
			require.NoError(t, err)
		}

		// Verify sorted iteration
		expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
		i := 0

		for elem := range s.Seq() {
			assert.Equal(t, sortable.Int(expected[i]), elem)

			i++
		}
	})

	t.Run("handles adding to root", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		contains, err := s.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

func TestRedBlackTreeSet_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("adds multiple elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.AddAll(sortable.Int(1), sortable.Int(2), sortable.Int(3))
		require.NoError(t, err)
		assert.Equal(t, 3, s.Size())
	})

	t.Run("handles duplicates in batch", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.AddAll(sortable.Int(1), sortable.Int(2), sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 2, s.Size())
	})

	t.Run("handles empty batch", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.AddAll()
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())
	})
}

func TestRedBlackTreeSet_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		err = s.Remove(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())

		contains, err := s.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("no-op for non-existent element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Remove(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("removes from multiple elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]() //nolint:varnamelen

		for i := range 10 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		for i := 0; i < 10; i += 2 {
			err := s.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 5, s.Size())

		// Verify remaining elements
		for i := 1; i < 10; i += 2 {
			contains, err := s.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, contains)
		}
	})

	t.Run("maintains tree balance after removal", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]() //nolint:varnamelen // Short variable name acceptable in tests

		// Insert many elements
		for i := range 100 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// Remove half of them
		for i := 0; i < 100; i += 2 {
			err := s.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 50, s.Size())

		// Verify all remaining elements are accessible
		for i := 1; i < 100; i += 2 {
			contains, err := s.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, contains)
		}
	})

	t.Run("handles removing root node", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]() //nolint:varnamelen // Short variable name acceptable in tests
		err := s.Add(sortable.Int(5))
		require.NoError(t, err)
		err = s.Add(sortable.Int(3))
		require.NoError(t, err)
		err = s.Add(sortable.Int(7))
		require.NoError(t, err)

		err = s.Remove(sortable.Int(5))
		require.NoError(t, err)
		assert.Equal(t, 2, s.Size())

		contains, err := s.Contains(sortable.Int(3))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = s.Contains(sortable.Int(7))
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

func TestRedBlackTreeSet_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 10 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		s.Clear()
		assert.Equal(t, 0, s.Size())
	})

	t.Run("set is usable after clear", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		s.Clear()

		err = s.Add(sortable.Int(2))
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("clear on empty set is no-op", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		s.Clear()
		assert.Equal(t, 0, s.Size())
	})
}

func TestRedBlackTreeSet_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		contains, err := s.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns false for non-existent element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		contains, err := s.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("returns false after removal", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		err = s.Remove(sortable.Int(1))
		require.NoError(t, err)

		contains, err := s.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("handles multiple lookups correctly", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 10 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		for i := range 10 {
			contains, err := s.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, contains)
		}

		for i := 10; i < 20; i++ {
			contains, err := s.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.False(t, contains)
		}
	})
}

func TestRedBlackTreeSet_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		assert.Equal(t, 0, s.Size())
	})

	t.Run("returns correct size after additions", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 5 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
			assert.Equal(t, i+1, s.Size())
		}
	})

	t.Run("size decreases after removals", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 5 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		for i := range 3 {
			err := s.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 2, s.Size())
	})

	t.Run("duplicate additions do not affect size", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s.Add(sortable.Int(1))
		require.NoError(t, err)

		assert.Equal(t, 1, s.Size())
	})
}

func TestRedBlackTreeSet_Entries(t *testing.T) {
	t.Parallel()

	t.Run("returns all elements in sorted order", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		elements := []int{5, 2, 8, 1, 9, 3, 7, 4, 6}
		for _, elem := range elements {
			err := s.Add(sortable.Int(elem))
			require.NoError(t, err)
		}

		entries := s.Entries()
		expected := []sortable.Int{1, 2, 3, 4, 5, 6, 7, 8, 9}
		assert.Equal(t, expected, entries)
	})

	t.Run("returns nil for empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		entries := s.Entries()
		assert.Nil(t, entries)
	})

	t.Run("returned slice is independent", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)

		entries := s.Entries()
		entries[0] = sortable.Int(999)

		// Original set should be unchanged
		contains, err := s.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

func TestRedBlackTreeSet_Seq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all elements in sorted order", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		elements := []int{5, 2, 8, 1, 9}
		for _, elem := range elements {
			err := s.Add(sortable.Int(elem))
			require.NoError(t, err)
		}

		result := make([]int, 0)
		for elem := range s.Seq() {
			result = append(result, int(elem))
		}

		assert.Equal(t, []int{1, 2, 5, 8, 9}, result)
	})

	t.Run("handles empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		count := 0
		for range s.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("stops early when iteration breaks", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 10 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		count := 0

		for elem := range s.Seq() {
			count++

			if elem == sortable.Int(5) {
				break
			}
		}

		assert.LessOrEqual(t, count, 6)
	})
}

func TestRedBlackTreeSet_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two sets", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]()
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s1.Add(sortable.Int(2))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()
		err = s2.Add(sortable.Int(3))
		require.NoError(t, err)

		result, err := s1.Union(s2)
		require.NoError(t, err)
		assert.Equal(t, 3, result.Size())
	})

	t.Run("handles overlapping elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]()
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s1.Add(sortable.Int(2))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()
		err = s2.Add(sortable.Int(2))
		require.NoError(t, err)
		err = s2.Add(sortable.Int(3))
		require.NoError(t, err)

		result, err := s1.Union(s2)
		require.NoError(t, err)
		assert.Equal(t, 3, result.Size())

		contains, err := result.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(sortable.Int(2))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(sortable.Int(3))
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("original sets are not modified", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]()
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()
		err = s2.Add(sortable.Int(2))
		require.NoError(t, err)

		_, err = s1.Union(s2)
		require.NoError(t, err)

		assert.Equal(t, 1, s1.Size())
		assert.Equal(t, 1, s2.Size())
	})

	t.Run("handles union with empty set", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]()
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()

		result, err := s1.Union(s2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
	})
}

func TestRedBlackTreeSet_Intersection(t *testing.T) {
	t.Parallel()

	t.Run("returns common elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]() //nolint:varnamelen // Short variable name acceptable in tests
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s1.Add(sortable.Int(2))
		require.NoError(t, err)
		err = s1.Add(sortable.Int(3))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()
		err = s2.Add(sortable.Int(2))
		require.NoError(t, err)
		err = s2.Add(sortable.Int(3))
		require.NoError(t, err)
		err = s2.Add(sortable.Int(4))
		require.NoError(t, err)

		result, err := s1.Intersection(s2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		contains, err := result.Contains(sortable.Int(2))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(sortable.Int(3))
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns empty set when no common elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]()
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()
		err = s2.Add(sortable.Int(2))
		require.NoError(t, err)

		result, err := s1.Intersection(s2)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Size())
	})

	t.Run("original sets are not modified", func(t *testing.T) {
		t.Parallel()

		s1 := NewRedBlackTreeSet[sortable.Int]()
		err := s1.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s1.Add(sortable.Int(2))
		require.NoError(t, err)

		s2 := NewRedBlackTreeSet[sortable.Int]()
		err = s2.Add(sortable.Int(2))
		require.NoError(t, err)

		_, err = s1.Intersection(s2)
		require.NoError(t, err)

		assert.Equal(t, 2, s1.Size())
		assert.Equal(t, 1, s2.Size())
	})
}

func TestRedBlackTreeSet_HashFunction(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for red-black tree", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		assert.Nil(t, s.HashFunction())
	})
}

func TestRedBlackTreeSet_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(1))
		require.NoError(t, err)
		err = s.Add(sortable.Int(2))
		require.NoError(t, err)

		cloned := s.Clone()
		assert.Equal(t, s.Size(), cloned.Size())

		err = cloned.Add(sortable.Int(3))
		require.NoError(t, err)

		assert.Equal(t, 2, s.Size())
		assert.Equal(t, 3, cloned.Size())
	})

	t.Run("clones empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		cloned := s.Clone()
		assert.Equal(t, 0, cloned.Size())
	})

	t.Run("cloned set has same elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 10 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		cloned := s.Clone()

		for i := range 10 {
			contains, err := cloned.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, contains)
		}
	})
}

func TestRedBlackTreeSet_StressTest(t *testing.T) {
	t.Parallel()

	t.Run("handles large number of insertions and deletions", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]() //nolint:varnamelen

		// Insert many elements
		for i := range 1000 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 1000, s.Size())

		// Remove every other element
		for i := 0; i < 1000; i += 2 {
			err := s.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 500, s.Size())

		// Verify remaining elements
		for i := 1; i < 1000; i += 2 {
			contains, err := s.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, contains)
		}

		// Verify deleted elements
		for i := 0; i < 1000; i += 2 {
			contains, err := s.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.False(t, contains)
		}
	})

	t.Run("maintains sorted order with many elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		// Insert in reverse order
		for i := 999; i >= 0; i-- {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// Verify sorted iteration
		prev := sortable.Int(-1)
		for elem := range s.Seq() {
			assert.Greater(t, elem, prev)
			prev = elem
		}
	})
}

func TestRedBlackTreeSet_StringElements(t *testing.T) {
	t.Parallel()

	t.Run("works with string elements", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.String]()

		err := s.Add(sortable.String("apple"))
		require.NoError(t, err)
		err = s.Add(sortable.String("banana"))
		require.NoError(t, err)
		err = s.Add(sortable.String("cherry"))
		require.NoError(t, err)

		assert.Equal(t, 3, s.Size())

		// Verify sorted iteration
		expected := []string{"apple", "banana", "cherry"}
		i := 0

		for elem := range s.Seq() {
			assert.Equal(t, sortable.String(expected[i]), elem)

			i++
		}
	})

	t.Run("handles string removal", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.String]()

		words := []string{"zebra", "apple", "mango", "grape", "kiwi"}
		for _, word := range words {
			err := s.Add(sortable.String(word))
			require.NoError(t, err)
		}

		err := s.Remove(sortable.String("mango"))
		require.NoError(t, err)

		assert.Equal(t, 4, s.Size())

		contains, err := s.Contains(sortable.String("mango"))
		require.NoError(t, err)
		assert.False(t, contains)
	})
}

func TestRedBlackTreeSet_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles single element", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()
		err := s.Add(sortable.Int(42))
		require.NoError(t, err)

		assert.Equal(t, 1, s.Size())

		entries := s.Entries()
		assert.Equal(t, []sortable.Int{42}, entries)
	})

	t.Run("handles alternating insertions and deletions", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 100 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)

			if i%2 == 1 {
				err = s.Remove(sortable.Int(i - 1))
				require.NoError(t, err)
			}
		}

		assert.Equal(t, 50, s.Size())
	})

	t.Run("handles removing all elements one by one", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 10 {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		for i := range 10 {
			err := s.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 0, s.Size())
	})
}

func TestRedBlackTreeSet_MultipleOperations(t *testing.T) {
	t.Parallel()

	t.Run("supports complex operation sequences", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]() //nolint:varnamelen // Short variable name acceptable in tests

		// Add some elements
		for i := 1; i <= 5; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// Remove some
		err := s.Remove(sortable.Int(3))
		require.NoError(t, err)

		// Add more
		for i := 6; i <= 10; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// Clone
		cloned := s.Clone()

		// Remove from original
		err = s.Remove(sortable.Int(7))
		require.NoError(t, err)

		// Verify original
		assert.Equal(t, 8, s.Size())
		contains, err := s.Contains(sortable.Int(7))
		require.NoError(t, err)
		assert.False(t, contains)

		// Verify clone still has it
		assert.Equal(t, 9, cloned.Size())
		contains, err = cloned.Contains(sortable.Int(7))
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

func TestRedBlackTreeSet_Filter(t *testing.T) {
	t.Parallel()

	t.Run("Filter maintains sorted order", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		// Add elements in random order
		elements := []int{10, 3, 7, 1, 9, 5, 2, 8, 4, 6}
		for _, elem := range elements {
			err := s.Add(sortable.Int(elem))
			require.NoError(t, err)
		}

		// Filter for even numbers
		filtered := s.Filter(func(item sortable.Int) bool {
			return int(item)%2 == 0
		})

		assert.Equal(t, 5, filtered.Size())

		// Verify filtered set is in sorted order
		expected := []int{2, 4, 6, 8, 10}
		i := 0

		for elem := range filtered.Seq() {
			assert.Equal(t, sortable.Int(expected[i]), elem)

			i++
		}
	})

	t.Run("Filter with no matches", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := 1; i <= 10; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// Filter for numbers > 100 (none match)
		filtered := s.Filter(func(item sortable.Int) bool {
			return int(item) > 100
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("Filter with all matches", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := 1; i <= 5; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// Filter for all numbers (all match)
		filtered := s.Filter(func(item sortable.Int) bool {
			return true
		})

		assert.Equal(t, 5, filtered.Size())

		// Verify same elements in sorted order
		entries := filtered.Entries()
		expected := []sortable.Int{
			sortable.Int(1),
			sortable.Int(2),
			sortable.Int(3),
			sortable.Int(4),
			sortable.Int(5),
		}
		assert.Equal(t, expected, entries)
	})

	t.Run("Filter on empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		filtered := s.Filter(func(item sortable.Int) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
	})

	t.Run("Filter does not modify original set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := 1; i <= 10; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		_ = s.Filter(func(item sortable.Int) bool {
			return int(item)%2 == 0
		})

		// Original set should be unchanged
		assert.Equal(t, 10, s.Size())
	})
}

func TestRedBlackTreeSet_FilterNot(t *testing.T) {
	t.Parallel()

	t.Run("FilterNot maintains sorted order", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		// Add elements in random order
		elements := []int{10, 3, 7, 1, 9, 5, 2, 8, 4, 6}
		for _, elem := range elements {
			err := s.Add(sortable.Int(elem))
			require.NoError(t, err)
		}

		// FilterNot for even numbers (exclude those, keep odd)
		filtered := s.FilterNot(func(item sortable.Int) bool {
			return int(item)%2 == 0
		})

		assert.Equal(t, 5, filtered.Size())

		// Verify filtered set is in sorted order
		expected := []int{1, 3, 5, 7, 9}
		i := 0

		for elem := range filtered.Seq() {
			assert.Equal(t, sortable.Int(expected[i]), elem)

			i++
		}
	})

	t.Run("FilterNot with no matches includes all", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := 1; i <= 5; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// FilterNot for numbers > 100 (none match, so all included)
		filtered := s.FilterNot(func(item sortable.Int) bool {
			return int(item) > 100
		})

		assert.Equal(t, 5, filtered.Size())

		entries := filtered.Entries()
		expected := []sortable.Int{
			sortable.Int(1),
			sortable.Int(2),
			sortable.Int(3),
			sortable.Int(4),
			sortable.Int(5),
		}
		assert.Equal(t, expected, entries)
	})

	t.Run("FilterNot with all matches returns empty", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := 1; i <= 5; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		// FilterNot for all numbers (all match, so none included)
		filtered := s.FilterNot(func(item sortable.Int) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("FilterNot on empty set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		filtered := s.FilterNot(func(item sortable.Int) bool {
			return false
		})

		assert.Equal(t, 0, filtered.Size())
	})

	t.Run("FilterNot does not modify original set", func(t *testing.T) {
		t.Parallel()

		s := NewRedBlackTreeSet[sortable.Int]()

		for i := 1; i <= 10; i++ {
			err := s.Add(sortable.Int(i))
			require.NoError(t, err)
		}

		_ = s.FilterNot(func(item sortable.Int) bool {
			return int(item)%2 == 0
		})

		// Original set should be unchanged
		assert.Equal(t, 10, s.Size())
	})
}

func BenchmarkRedBlackTreeSet_Add(b *testing.B) {
	s := NewRedBlackTreeSet[sortable.Int]()

	b.ResetTimer()

	for i := range b.N {
		_ = s.Add(sortable.Int(i))
	}
}

func BenchmarkRedBlackTreeSet_Contains(b *testing.B) {
	s := NewRedBlackTreeSet[sortable.Int]()

	for i := range 1000 {
		_ = s.Add(sortable.Int(i))
	}

	b.ResetTimer()

	for i := range b.N {
		_, _ = s.Contains(sortable.Int(i % 1000))
	}
}

func BenchmarkRedBlackTreeSet_Remove(b *testing.B) {
	b.StopTimer()

	for range b.N {
		s := NewRedBlackTreeSet[sortable.Int]()

		for i := range 1000 {
			_ = s.Add(sortable.Int(i))
		}

		b.StartTimer()

		for i := range 1000 {
			_ = s.Remove(sortable.Int(i))
		}

		b.StopTimer()
	}
}

func ExampleNewRedBlackTreeSet() {
	s := NewRedBlackTreeSet[sortable.Int]()

	_ = s.Add(sortable.Int(5))
	_ = s.Add(sortable.Int(2))
	_ = s.Add(sortable.Int(8))
	_ = s.Add(sortable.Int(1))

	for elem := range s.Seq() {
		fmt.Println(elem)
	}

	// Output:
	// 1
	// 2
	// 5
	// 8
}

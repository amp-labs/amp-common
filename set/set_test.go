package set

import (
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSet tests the generic Set implementation.
func TestSet(t *testing.T) {
	t.Parallel()

	t.Run("Add and Contains", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		contains, err := s.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = s.Contains(hashing.HashableString("bar"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Add duplicate element", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		// Adding the same element again should not error
		err = s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		assert.Equal(t, 1, s.Size())
	})

	t.Run("AddAll", func(t *testing.T) {
		t.Parallel()

		set := NewSet[hashing.HashableString](hashing.Sha256)

		err := set.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		assert.Equal(t, 3, set.Size())

		contains, err := set.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = set.Contains(hashing.HashableString("bar"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = set.Contains(hashing.HashableString("baz"))
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("Remove", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		err = s.Remove(hashing.HashableString("foo"))
		require.NoError(t, err)

		contains, err := s.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.False(t, contains)

		assert.Equal(t, 0, s.Size())
	})

	t.Run("Remove non-existent element", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		// Removing non-existent element should not error
		err := s.Remove(hashing.HashableString("foo"))
		require.NoError(t, err)
	})

	t.Run("Clear", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		assert.Equal(t, 3, s.Size())

		s.Clear()

		assert.Equal(t, 0, s.Size())

		contains, err := s.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Entries", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		entries := s.Entries()
		assert.Len(t, entries, 3)
		assert.ElementsMatch(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		}, entries)
	})

	t.Run("Union", func(t *testing.T) {
		t.Parallel()

		s1 := NewSet[hashing.HashableString](hashing.Sha256)
		err := s1.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
		)
		require.NoError(t, err)

		s2 := NewSet[hashing.HashableString](hashing.Sha256)
		err = s2.AddAll(
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		union, err := s1.Union(s2)
		require.NoError(t, err)

		assert.Equal(t, 3, union.Size())

		entries := union.Entries()
		assert.ElementsMatch(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		}, entries)
	})

	t.Run("Intersection", func(t *testing.T) {
		t.Parallel()

		set1 := NewSet[hashing.HashableString](hashing.Sha256)
		err := set1.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		set2 := NewSet[hashing.HashableString](hashing.Sha256)
		err = set2.AddAll(
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		)
		require.NoError(t, err)

		intersection, err := set1.Intersection(set2)
		require.NoError(t, err)

		assert.Equal(t, 2, intersection.Size())

		entries := intersection.Entries()
		assert.ElementsMatch(t, []hashing.HashableString{
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		}, entries)
	})

	t.Run("Intersection with no common elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewSet[hashing.HashableString](hashing.Sha256)
		err := s1.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
		)
		require.NoError(t, err)

		s2 := NewSet[hashing.HashableString](hashing.Sha256)
		err = s2.AddAll(
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		)
		require.NoError(t, err)

		intersection, err := s1.Intersection(s2)
		require.NoError(t, err)

		assert.Equal(t, 0, intersection.Size())
	})

	t.Run("Empty set", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		assert.Equal(t, 0, s.Size())
		assert.Empty(t, s.Entries())

		contains, err := s.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Seq iteration", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// Collect all items via Seq
		var items []hashing.HashableString
		for item := range s.Seq() {
			items = append(items, item)
		}

		assert.Len(t, items, 3)
		assert.ElementsMatch(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		}, items)
	})

	t.Run("Seq early termination", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// Stop after first item
		count := 0
		for range s.Seq() {
			count++

			break
		}

		assert.Equal(t, 1, count)
	})

	t.Run("Seq on empty set", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		count := 0
		for range s.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})
}

// TestStringSet tests the StringSet implementation.
func TestStringSet(t *testing.T) {
	t.Parallel()

	t.Run("Add and Contains", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.Add("foo")
		require.NoError(t, err)

		contains, err := s.Contains("foo")
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = s.Contains("bar")
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Add duplicate element", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.Add("foo")
		require.NoError(t, err)

		// Adding the same element again should not error
		err = s.Add("foo")
		require.NoError(t, err)

		assert.Equal(t, 1, s.Size())
	})

	t.Run("AddAll", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		assert.Equal(t, 3, s.Size())

		contains, err := s.Contains("foo")
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = s.Contains("bar")
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = s.Contains("baz")
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("Remove", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.Add("foo")
		require.NoError(t, err)

		err = s.Remove("foo")
		require.NoError(t, err)

		contains, err := s.Contains("foo")
		require.NoError(t, err)
		assert.False(t, contains)

		assert.Equal(t, 0, s.Size())
	})

	t.Run("Remove non-existent element", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		// Removing non-existent element should not error
		err := s.Remove("foo")
		require.NoError(t, err)
	})

	t.Run("Clear", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		assert.Equal(t, 3, s.Size())

		s.Clear()

		assert.Equal(t, 0, s.Size())

		contains, err := s.Contains("foo")
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Entries", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		entries := s.Entries()
		assert.Len(t, entries, 3)
		assert.ElementsMatch(t, []string{"foo", "bar", "baz"}, entries)
	})

	t.Run("SortedEntries", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("zebra", "apple", "banana")
		require.NoError(t, err)

		entries := s.SortedEntries()
		assert.Equal(t, []string{"apple", "banana", "zebra"}, entries)
	})

	t.Run("NaturalSortedEntries", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("file10", "file2", "file1", "file20")
		require.NoError(t, err)

		entries := s.NaturalSortedEntries()
		assert.Equal(t, []string{"file1", "file2", "file10", "file20"}, entries)
	})

	t.Run("Union", func(t *testing.T) {
		t.Parallel()

		s1 := NewStringSet(hashing.Sha256)
		err := s1.AddAll("foo", "bar")
		require.NoError(t, err)

		s2 := NewStringSet(hashing.Sha256)
		err = s2.AddAll("bar", "baz")
		require.NoError(t, err)

		union, err := s1.Union(s2)
		require.NoError(t, err)

		assert.Equal(t, 3, union.Size())

		entries := union.Entries()
		assert.ElementsMatch(t, []string{"foo", "bar", "baz"}, entries)
	})

	t.Run("Intersection", func(t *testing.T) {
		t.Parallel()

		s1 := NewStringSet(hashing.Sha256)
		err := s1.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		s2 := NewStringSet(hashing.Sha256)
		err = s2.AddAll("bar", "baz", "qux")
		require.NoError(t, err)

		intersection, err := s1.Intersection(s2)
		require.NoError(t, err)

		assert.Equal(t, 2, intersection.Size())

		entries := intersection.Entries()
		assert.ElementsMatch(t, []string{"bar", "baz"}, entries)
	})

	t.Run("Intersection with no common elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewStringSet(hashing.Sha256)
		err := s1.AddAll("foo", "bar")
		require.NoError(t, err)

		s2 := NewStringSet(hashing.Sha256)
		err = s2.AddAll("baz", "qux")
		require.NoError(t, err)

		intersection, err := s1.Intersection(s2)
		require.NoError(t, err)

		assert.Equal(t, 0, intersection.Size())
	})

	t.Run("Empty set", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		assert.Equal(t, 0, s.Size())
		assert.Empty(t, s.Entries())
		assert.Empty(t, s.SortedEntries())
		assert.Empty(t, s.NaturalSortedEntries())

		contains, err := s.Contains("foo")
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Multiple hash functions", func(t *testing.T) {
		t.Parallel()

		// Test with different hash functions
		hashFuncs := []hashing.HashFunc{
			hashing.Sha256,
			hashing.Sha1,
			hashing.Md5,
		}

		for _, hashFunc := range hashFuncs {
			s := NewStringSet(hashFunc)

			err := s.AddAll("foo", "bar", "baz")
			require.NoError(t, err)

			assert.Equal(t, 3, s.Size())

			contains, err := s.Contains("foo")
			require.NoError(t, err)
			assert.True(t, contains)
		}
	})

	t.Run("Seq iteration", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		// Collect all items via Seq
		var items []string
		for item := range s.Seq() {
			items = append(items, item)
		}

		assert.Len(t, items, 3)
		assert.ElementsMatch(t, []string{"foo", "bar", "baz"}, items)
	})

	t.Run("Seq early termination", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		err := s.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		// Stop after first item
		count := 0
		for range s.Seq() {
			count++

			break
		}

		assert.Equal(t, 1, count)
	})

	t.Run("Seq on empty set", func(t *testing.T) {
		t.Parallel()

		s := NewStringSet(hashing.Sha256)

		count := 0
		for range s.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})
}

// TestOrderedSet tests the OrderedSet implementation.
func TestOrderedSet(t *testing.T) {
	t.Parallel()

	t.Run("Add and Contains maintains order", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := set.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		err = set.Add(hashing.HashableString("bar"))
		require.NoError(t, err)

		err = set.Add(hashing.HashableString("baz"))
		require.NoError(t, err)

		contains, err := set.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = set.Contains(hashing.HashableString("qux"))
		require.NoError(t, err)
		assert.False(t, contains)

		// Verify order is preserved
		entries := set.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		}, entries)
	})

	t.Run("Add duplicate element preserves original position", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		err = s.Add(hashing.HashableString("bar"))
		require.NoError(t, err)

		// Adding "foo" again should not change its position
		err = s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		assert.Equal(t, 2, s.Size())

		entries := s.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
		}, entries)
	})

	t.Run("AddAll maintains insertion order", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("zebra"),
			hashing.HashableString("apple"),
			hashing.HashableString("banana"),
		)
		require.NoError(t, err)

		assert.Equal(t, 3, s.Size())

		entries := s.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("zebra"),
			hashing.HashableString("apple"),
			hashing.HashableString("banana"),
		}, entries)
	})

	t.Run("Remove maintains order of remaining elements", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := set.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		)
		require.NoError(t, err)

		// Remove middle element
		err = set.Remove(hashing.HashableString("bar"))
		require.NoError(t, err)

		assert.Equal(t, 3, set.Size())

		entries := set.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		}, entries)

		// Remove first element
		err = set.Remove(hashing.HashableString("foo"))
		require.NoError(t, err)

		assert.Equal(t, 2, set.Size())

		entries = set.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		}, entries)
	})

	t.Run("Remove non-existent element", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.Add(hashing.HashableString("foo"))
		require.NoError(t, err)

		// Removing non-existent element should not error
		err = s.Remove(hashing.HashableString("bar"))
		require.NoError(t, err)

		assert.Equal(t, 1, s.Size())
	})

	t.Run("Clear", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := set.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		assert.Equal(t, 3, set.Size())

		set.Clear()

		assert.Equal(t, 0, set.Size())
		assert.Empty(t, set.Entries())

		contains, err := set.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Entries returns copy", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
		)
		require.NoError(t, err)

		entries1 := s.Entries()
		entries2 := s.Entries()

		// Modifying one copy should not affect the other
		entries1[0] = hashing.HashableString("modified")

		assert.NotEqual(t, entries1[0], entries2[0])
		assert.Equal(t, hashing.HashableString("foo"), entries2[0])
	})

	t.Run("Union maintains order", func(t *testing.T) {
		t.Parallel()

		s1 := NewOrderedSet[hashing.HashableString](hashing.Sha256)
		err := s1.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
		)
		require.NoError(t, err)

		s2 := NewOrderedSet[hashing.HashableString](hashing.Sha256)
		err = s2.AddAll(
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		)
		require.NoError(t, err)

		union, err := s1.Union(s2)
		require.NoError(t, err)

		assert.Equal(t, 4, union.Size())

		// Should have s1's elements first, then s2's new elements
		entries := union.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		}, entries)
	})

	t.Run("Intersection maintains order from first set", func(t *testing.T) {
		t.Parallel()

		set1 := NewOrderedSet[hashing.HashableString](hashing.Sha256)
		err := set1.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		)
		require.NoError(t, err)

		set2 := NewOrderedSet[hashing.HashableString](hashing.Sha256)
		err = set2.AddAll(
			hashing.HashableString("qux"),
			hashing.HashableString("bar"),
			hashing.HashableString("extra"),
		)
		require.NoError(t, err)

		intersection, err := set1.Intersection(set2)
		require.NoError(t, err)

		assert.Equal(t, 2, intersection.Size())

		// Should maintain order from set1 (bar, then qux)
		entries := intersection.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("bar"),
			hashing.HashableString("qux"),
		}, entries)
	})

	t.Run("Intersection with no common elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewOrderedSet[hashing.HashableString](hashing.Sha256)
		err := s1.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
		)
		require.NoError(t, err)

		s2 := NewOrderedSet[hashing.HashableString](hashing.Sha256)
		err = s2.AddAll(
			hashing.HashableString("baz"),
			hashing.HashableString("qux"),
		)
		require.NoError(t, err)

		intersection, err := s1.Intersection(s2)
		require.NoError(t, err)

		assert.Equal(t, 0, intersection.Size())
		assert.Empty(t, intersection.Entries())
	})

	t.Run("Empty set", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		assert.Equal(t, 0, s.Size())
		assert.Empty(t, s.Entries())

		contains, err := s.Contains(hashing.HashableString("foo"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Complex ordering scenario", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		// Add elements in specific order
		err := set.AddAll(
			hashing.HashableString("first"),
			hashing.HashableString("second"),
			hashing.HashableString("third"),
			hashing.HashableString("fourth"),
			hashing.HashableString("fifth"),
		)
		require.NoError(t, err)

		// Remove some elements
		err = set.Remove(hashing.HashableString("second"))
		require.NoError(t, err)

		err = set.Remove(hashing.HashableString("fourth"))
		require.NoError(t, err)

		// Try to add duplicate
		err = set.Add(hashing.HashableString("third"))
		require.NoError(t, err)

		// Add new element
		err = set.Add(hashing.HashableString("sixth"))
		require.NoError(t, err)

		// Verify final order
		entries := set.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("first"),
			hashing.HashableString("third"),
			hashing.HashableString("fifth"),
			hashing.HashableString("sixth"),
		}, entries)
	})

	t.Run("Seq iteration maintains order", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := set.AddAll(
			hashing.HashableString("first"),
			hashing.HashableString("second"),
			hashing.HashableString("third"),
		)
		require.NoError(t, err)

		// Collect all items and indices via Seq
		var items []hashing.HashableString

		var indices []int

		for i, item := range set.Seq() {
			indices = append(indices, i)
			items = append(items, item)
		}

		assert.Equal(t, []int{0, 1, 2}, indices)
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("first"),
			hashing.HashableString("second"),
			hashing.HashableString("third"),
		}, items)
	})

	t.Run("Seq early termination", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := set.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// Stop after second item
		count := 0
		for i := range set.Seq() {
			count++

			if i == 1 {
				break
			}
		}

		assert.Equal(t, 2, count)
	})

	t.Run("Seq on empty set", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		count := 0
		for range set.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("Seq after modifications maintains order", func(t *testing.T) {
		t.Parallel()

		set := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := set.AddAll(
			hashing.HashableString("a"),
			hashing.HashableString("b"),
			hashing.HashableString("c"),
			hashing.HashableString("d"),
		)
		require.NoError(t, err)

		// Remove middle element
		err = set.Remove(hashing.HashableString("b"))
		require.NoError(t, err)

		// Verify order via Seq
		var items []hashing.HashableString
		for _, item := range set.Seq() {
			items = append(items, item)
		}

		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("a"),
			hashing.HashableString("c"),
			hashing.HashableString("d"),
		}, items)
	})
}

// TestStringOrderedSet tests the StringOrderedSet implementation.
func TestStringOrderedSet(t *testing.T) {
	t.Parallel()

	t.Run("Add and Contains maintains order", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		err := set.Add("foo")
		require.NoError(t, err)

		err = set.Add("bar")
		require.NoError(t, err)

		err = set.Add("baz")
		require.NoError(t, err)

		contains, err := set.Contains("foo")
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = set.Contains("qux")
		require.NoError(t, err)
		assert.False(t, contains)

		// Verify order is preserved
		entries := set.Entries()
		assert.Equal(t, []string{"foo", "bar", "baz"}, entries)
	})

	t.Run("Add duplicate element preserves original position", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.Add("foo")
		require.NoError(t, err)

		err = s.Add("bar")
		require.NoError(t, err)

		// Adding "foo" again should not change its position
		err = s.Add("foo")
		require.NoError(t, err)

		assert.Equal(t, 2, s.Size())

		entries := s.Entries()
		assert.Equal(t, []string{"foo", "bar"}, entries)
	})

	t.Run("AddAll maintains insertion order", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.AddAll("zebra", "apple", "banana")
		require.NoError(t, err)

		assert.Equal(t, 3, s.Size())

		entries := s.Entries()
		assert.Equal(t, []string{"zebra", "apple", "banana"}, entries)
	})

	t.Run("Remove maintains order of remaining elements", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		err := set.AddAll("foo", "bar", "baz", "qux")
		require.NoError(t, err)

		// Remove middle element
		err = set.Remove("bar")
		require.NoError(t, err)

		assert.Equal(t, 3, set.Size())

		entries := set.Entries()
		assert.Equal(t, []string{"foo", "baz", "qux"}, entries)

		// Remove first element
		err = set.Remove("foo")
		require.NoError(t, err)

		assert.Equal(t, 2, set.Size())

		entries = set.Entries()
		assert.Equal(t, []string{"baz", "qux"}, entries)
	})

	t.Run("Remove non-existent element", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.Add("foo")
		require.NoError(t, err)

		// Removing non-existent element should not error
		err = s.Remove("bar")
		require.NoError(t, err)

		assert.Equal(t, 1, s.Size())
	})

	t.Run("Clear", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		err := set.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		assert.Equal(t, 3, set.Size())

		set.Clear()

		assert.Equal(t, 0, set.Size())
		assert.Empty(t, set.Entries())

		contains, err := set.Contains("foo")
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Entries returns insertion order", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.AddAll("zebra", "apple", "banana")
		require.NoError(t, err)

		entries := s.Entries()
		assert.Equal(t, []string{"zebra", "apple", "banana"}, entries)
	})

	t.Run("SortedEntries returns alphabetically sorted", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.AddAll("zebra", "apple", "banana")
		require.NoError(t, err)

		// Insertion order should still be zebra, apple, banana
		entries := s.Entries()
		assert.Equal(t, []string{"zebra", "apple", "banana"}, entries)

		// But sorted entries should be alphabetical
		sorted := s.SortedEntries()
		assert.Equal(t, []string{"apple", "banana", "zebra"}, sorted)
	})

	t.Run("NaturalSortedEntries returns naturally sorted", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.AddAll("file10", "file2", "file1", "file20")
		require.NoError(t, err)

		// Insertion order
		entries := s.Entries()
		assert.Equal(t, []string{"file10", "file2", "file1", "file20"}, entries)

		// Natural sorted
		sorted := s.NaturalSortedEntries()
		assert.Equal(t, []string{"file1", "file2", "file10", "file20"}, sorted)
	})

	t.Run("Union maintains order", func(t *testing.T) {
		t.Parallel()

		s1 := NewStringOrderedSet(hashing.Sha256)
		err := s1.AddAll("foo", "bar")
		require.NoError(t, err)

		s2 := NewStringOrderedSet(hashing.Sha256)
		err = s2.AddAll("bar", "baz", "qux")
		require.NoError(t, err)

		union, err := s1.Union(s2)
		require.NoError(t, err)

		assert.Equal(t, 4, union.Size())

		// Should have s1's elements first, then s2's new elements
		entries := union.Entries()
		assert.Equal(t, []string{"foo", "bar", "baz", "qux"}, entries)
	})

	t.Run("Intersection maintains order from first set", func(t *testing.T) {
		t.Parallel()

		set1 := NewStringOrderedSet(hashing.Sha256)
		err := set1.AddAll("foo", "bar", "baz", "qux")
		require.NoError(t, err)

		set2 := NewStringOrderedSet(hashing.Sha256)
		err = set2.AddAll("qux", "bar", "extra")
		require.NoError(t, err)

		intersection, err := set1.Intersection(set2)
		require.NoError(t, err)

		assert.Equal(t, 2, intersection.Size())

		// Should maintain order from set1 (bar, then qux)
		entries := intersection.Entries()
		assert.Equal(t, []string{"bar", "qux"}, entries)
	})

	t.Run("Intersection with no common elements", func(t *testing.T) {
		t.Parallel()

		s1 := NewStringOrderedSet(hashing.Sha256)
		err := s1.AddAll("foo", "bar")
		require.NoError(t, err)

		s2 := NewStringOrderedSet(hashing.Sha256)
		err = s2.AddAll("baz", "qux")
		require.NoError(t, err)

		intersection, err := s1.Intersection(s2)
		require.NoError(t, err)

		assert.Equal(t, 0, intersection.Size())
		assert.Empty(t, intersection.Entries())
	})

	t.Run("Empty set", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		assert.Equal(t, 0, s.Size())
		assert.Empty(t, s.Entries())
		assert.Empty(t, s.SortedEntries())
		assert.Empty(t, s.NaturalSortedEntries())

		contains, err := s.Contains("foo")
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Multiple hash functions", func(t *testing.T) {
		t.Parallel()

		// Test with different hash functions
		hashFuncs := []hashing.HashFunc{
			hashing.Sha256,
			hashing.Sha1,
			hashing.Md5,
		}

		for _, hashFunc := range hashFuncs {
			s := NewStringOrderedSet(hashFunc)

			err := s.AddAll("foo", "bar", "baz")
			require.NoError(t, err)

			assert.Equal(t, 3, s.Size())

			// Verify order is maintained regardless of hash function
			entries := s.Entries()
			assert.Equal(t, []string{"foo", "bar", "baz"}, entries)

			contains, err := s.Contains("foo")
			require.NoError(t, err)
			assert.True(t, contains)
		}
	})

	t.Run("Complex ordering scenario", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		// Add elements in specific order
		err := set.AddAll("first", "second", "third", "fourth", "fifth")
		require.NoError(t, err)

		// Remove some elements
		err = set.Remove("second")
		require.NoError(t, err)

		err = set.Remove("fourth")
		require.NoError(t, err)

		// Try to add duplicate
		err = set.Add("third")
		require.NoError(t, err)

		// Add new element
		err = set.Add("sixth")
		require.NoError(t, err)

		// Verify final order
		entries := set.Entries()
		assert.Equal(t, []string{"first", "third", "fifth", "sixth"}, entries)

		// Verify sorted order is independent
		sorted := set.SortedEntries()
		assert.Equal(t, []string{"fifth", "first", "sixth", "third"}, sorted)
	})

	t.Run("Sorted methods don't affect insertion order", func(t *testing.T) {
		t.Parallel()

		s := NewStringOrderedSet(hashing.Sha256)

		err := s.AddAll("zebra", "apple", "banana", "file10", "file2")
		require.NoError(t, err)

		// Call sorted methods
		_ = s.SortedEntries()
		_ = s.NaturalSortedEntries()

		// Verify insertion order is still preserved
		entries := s.Entries()
		assert.Equal(t, []string{"zebra", "apple", "banana", "file10", "file2"}, entries)
	})

	t.Run("Seq iteration maintains order", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		err := set.AddAll("first", "second", "third")
		require.NoError(t, err)

		// Collect all items and indices via Seq
		var items []string

		var indices []int

		for i, item := range set.Seq() {
			indices = append(indices, i)
			items = append(items, item)
		}

		assert.Equal(t, []int{0, 1, 2}, indices)
		assert.Equal(t, []string{"first", "second", "third"}, items)
	})

	t.Run("Seq early termination", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		err := set.AddAll("foo", "bar", "baz")
		require.NoError(t, err)

		// Stop after second item
		count := 0
		for i := range set.Seq() {
			count++

			if i == 1 {
				break
			}
		}

		assert.Equal(t, 2, count)
	})

	t.Run("Seq on empty set", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		count := 0
		for range set.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("Seq after modifications maintains order", func(t *testing.T) {
		t.Parallel()

		set := NewStringOrderedSet(hashing.Sha256)

		err := set.AddAll("a", "b", "c", "d")
		require.NoError(t, err)

		// Remove middle element
		err = set.Remove("b")
		require.NoError(t, err)

		// Verify order via Seq
		var items []string
		for _, item := range set.Seq() {
			items = append(items, item)
		}

		assert.Equal(t, []string{"a", "c", "d"}, items)
	})
}

// TestSetFilter tests the Filter method on Set implementations.
func TestSetFilter(t *testing.T) {
	t.Parallel()

	t.Run("Filter with some matches", func(t *testing.T) { //nolint:dupl // Test duplication is acceptable
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("apple"),
			hashing.HashableString("banana"),
			hashing.HashableString("cherry"),
			hashing.HashableString("apricot"),
		)
		require.NoError(t, err)

		// Filter for strings starting with 'a'
		filtered := s.Filter(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'a'
		})

		assert.Equal(t, 2, filtered.Size())

		contains, err := filtered.Contains(hashing.HashableString("apple"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = filtered.Contains(hashing.HashableString("apricot"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = filtered.Contains(hashing.HashableString("banana"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("Filter with no matches", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// Filter for strings starting with 'x' (none match)
		filtered := s.Filter(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'x'
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("Filter with all matches", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// Filter for all strings (all match)
		filtered := s.Filter(func(item hashing.HashableString) bool {
			return true
		})

		assert.Equal(t, 3, filtered.Size())
		assert.ElementsMatch(t, s.Entries(), filtered.Entries())
	})

	t.Run("Filter on empty set", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		filtered := s.Filter(func(item hashing.HashableString) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("Filter does not modify original set", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		_ = s.Filter(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'b'
		})

		// Original set should be unchanged
		assert.Equal(t, 3, s.Size())
	})
}

// TestSetFilterNot tests the FilterNot method on Set implementations.
func TestSetFilterNot(t *testing.T) {
	t.Parallel()

	t.Run("FilterNot with some matches", func(t *testing.T) { //nolint:dupl // Test duplication is acceptable
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("apple"),
			hashing.HashableString("banana"),
			hashing.HashableString("cherry"),
			hashing.HashableString("apricot"),
		)
		require.NoError(t, err)

		// FilterNot for strings starting with 'a' (exclude those)
		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'a'
		})

		assert.Equal(t, 2, filtered.Size())

		contains, err := filtered.Contains(hashing.HashableString("banana"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = filtered.Contains(hashing.HashableString("cherry"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = filtered.Contains(hashing.HashableString("apple"))
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("FilterNot with no matches", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// FilterNot for strings starting with 'x' (none match, so all included)
		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'x'
		})

		assert.Equal(t, 3, filtered.Size())
		assert.ElementsMatch(t, s.Entries(), filtered.Entries())
	})

	t.Run("FilterNot with all matches", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		// FilterNot for all strings (all match, so none included)
		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("FilterNot on empty set", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("FilterNot does not modify original set", func(t *testing.T) {
		t.Parallel()

		s := NewSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		_ = s.FilterNot(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'b'
		})

		// Original set should be unchanged
		assert.Equal(t, 3, s.Size())
	})
}

// TestOrderedSetFilter tests the Filter method on OrderedSet implementations.
func TestOrderedSetFilter(t *testing.T) {
	t.Parallel()

	t.Run("Filter maintains insertion order", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("apple"),
			hashing.HashableString("banana"),
			hashing.HashableString("apricot"),
			hashing.HashableString("cherry"),
			hashing.HashableString("avocado"),
		)
		require.NoError(t, err)

		// Filter for strings starting with 'a'
		filtered := s.Filter(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'a'
		})

		assert.Equal(t, 3, filtered.Size())

		// Order should be preserved from original set
		entries := filtered.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("apple"),
			hashing.HashableString("apricot"),
			hashing.HashableString("avocado"),
		}, entries)
	})

	t.Run("Filter with no matches preserves empty order", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		filtered := s.Filter(func(item hashing.HashableString) bool {
			return false
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("Filter on empty OrderedSet", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		filtered := s.Filter(func(item hashing.HashableString) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
	})
}

// TestOrderedSetFilterNot tests the FilterNot method on OrderedSet implementations.
func TestOrderedSetFilterNot(t *testing.T) {
	t.Parallel()

	t.Run("FilterNot maintains insertion order", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("apple"),
			hashing.HashableString("banana"),
			hashing.HashableString("apricot"),
			hashing.HashableString("cherry"),
			hashing.HashableString("avocado"),
		)
		require.NoError(t, err)

		// FilterNot for strings starting with 'a' (exclude those)
		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return len(item) > 0 && item[0] == 'a'
		})

		assert.Equal(t, 2, filtered.Size())

		// Order should be preserved from original set
		entries := filtered.Entries()
		assert.Equal(t, []hashing.HashableString{
			hashing.HashableString("banana"),
			hashing.HashableString("cherry"),
		}, entries)
	})

	t.Run("FilterNot with all matches returns empty in order", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		err := s.AddAll(
			hashing.HashableString("foo"),
			hashing.HashableString("bar"),
			hashing.HashableString("baz"),
		)
		require.NoError(t, err)

		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return true
		})

		assert.Equal(t, 0, filtered.Size())
		assert.Empty(t, filtered.Entries())
	})

	t.Run("FilterNot on empty OrderedSet", func(t *testing.T) {
		t.Parallel()

		s := NewOrderedSet[hashing.HashableString](hashing.Sha256)

		filtered := s.FilterNot(func(item hashing.HashableString) bool {
			return false
		})

		assert.Equal(t, 0, filtered.Size())
	})
}

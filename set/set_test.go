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
}

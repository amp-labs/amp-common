package maps_test

import (
	"testing"

	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
)

func TestNewCaseInsensitiveMap(t *testing.T) {
	t.Parallel()

	t.Run("creates empty map with nil input", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		assert.NotNil(t, m)
		assert.True(t, m.IsEmpty())
		assert.Equal(t, 0, m.Size())
	})

	t.Run("creates map with initial values", func(t *testing.T) {
		t.Parallel()

		initial := map[string]string{
			"Content-Type": "application/json",
			"Accept":       "text/html",
		}
		m := maps.NewCaseInsensitiveMap(initial)

		assert.Equal(t, 2, m.Size())
		assert.False(t, m.IsEmpty())
	})
}

func TestCaseInsensitiveMap_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		assert.Equal(t, 1, m.Size())
		_, val, ok := m.Get("Content-Type", true)
		assert.True(t, ok)
		assert.Equal(t, "application/json", val)
	})

	t.Run("preserves original key casing", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		key, val, ok := m.Get("content-type", false)
		assert.True(t, ok)
		assert.Equal(t, "Content-Type", key)
		assert.Equal(t, "application/json", val)
	})

	t.Run("replaces existing value for case-insensitive match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")
		m.Add("content-type", "text/html")

		assert.Equal(t, 1, m.Size())
		_, val, ok := m.Get("CONTENT-TYPE", false)
		assert.True(t, ok)
		assert.Equal(t, "text/html", val)
	})
}

func TestCaseInsensitiveMap_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("adds multiple key-value pairs", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[int](nil)
		m.AddAll(map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		})

		assert.Equal(t, 3, m.Size())
	})

	t.Run("works with nil map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.AddAll(nil)

		assert.Equal(t, 0, m.Size())
	})
}

func TestCaseInsensitiveMap_Get(t *testing.T) {
	t.Parallel()

	t.Run("case-sensitive lookup returns exact match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		key, val, ok := m.Get("Content-Type", true)
		assert.True(t, ok)
		assert.Equal(t, "Content-Type", key)
		assert.Equal(t, "application/json", val)
	})

	t.Run("case-sensitive lookup returns false for different casing", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		key, val, ok := m.Get("content-type", true)
		assert.False(t, ok)
		assert.Equal(t, "content-type", key)
		assert.Equal(t, "", val)
	})

	t.Run("case-insensitive lookup returns original key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		key, val, ok := m.Get("content-type", false)
		assert.True(t, ok)
		assert.Equal(t, "Content-Type", key)
		assert.Equal(t, "application/json", val)
	})

	t.Run("case-insensitive lookup works with various casings", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		testCases := []string{"CONTENT-TYPE", "content-TYPE", "CoNtEnT-tYpE"}
		for _, tc := range testCases {
			key, val, ok := m.Get(tc, false)
			assert.True(t, ok)
			assert.Equal(t, "Content-Type", key)
			assert.Equal(t, "application/json", val)
		}
	})

	t.Run("returns zero value for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[int](nil)
		key, val, ok := m.Get("missing", false)
		assert.False(t, ok)
		assert.Equal(t, "missing", key)
		assert.Equal(t, 0, val)
	})
}

func TestCaseInsensitiveMap_GetAll(t *testing.T) {
	t.Parallel()

	t.Run("returns all key-value pairs", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")
		m.Add("Accept", "text/html")

		all := m.GetAll()
		assert.Len(t, all, 2)
		assert.Equal(t, "application/json", all["Content-Type"])
		assert.Equal(t, "text/html", all["Accept"])
	})

	t.Run("returns nil for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		all := m.GetAll()
		assert.Nil(t, all)
	})
}

func TestCaseInsensitiveMap_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes key-value pair case-insensitively", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")
		m.Remove("content-type")

		assert.Equal(t, 0, m.Size())
		_, _, ok := m.Get("Content-Type", false)
		assert.False(t, ok)
	})

	t.Run("no-op for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")
		m.Remove("missing")

		assert.Equal(t, 1, m.Size())
	})
}

func TestCaseInsensitiveMap_RemoveAll(t *testing.T) {
	t.Parallel()

	t.Run("removes multiple keys", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.AddAll(map[string]string{
			"one":   "1",
			"two":   "2",
			"three": "3",
		})
		m.RemoveAll("ONE", "two")

		assert.Equal(t, 1, m.Size())
		_, _, ok := m.Get("three", false)
		assert.True(t, ok)
	})
}

func TestCaseInsensitiveMap_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all key-value pairs", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.AddAll(map[string]string{
			"one": "1",
			"two": "2",
		})
		m.Clear()

		assert.Equal(t, 0, m.Size())
		assert.True(t, m.IsEmpty())
	})

	t.Run("map is still usable after clear", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("key", "value")
		m.Clear()
		m.Add("new", "value")

		assert.Equal(t, 1, m.Size())
	})
}

func TestCaseInsensitiveMap_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns correct size", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		assert.Equal(t, 0, m.Size())

		m.Add("one", "1")
		assert.Equal(t, 1, m.Size())

		m.Add("two", "2")
		assert.Equal(t, 2, m.Size())

		m.Remove("one")
		assert.Equal(t, 1, m.Size())
	})
}

func TestCaseInsensitiveMap_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns all keys with original casing", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")
		m.Add("Accept", "text/html")

		keys := m.Keys()
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "Content-Type")
		assert.Contains(t, keys, "Accept")
	})

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		keys := m.Keys()
		assert.NotNil(t, keys)
		assert.Empty(t, keys)
	})
}

func TestCaseInsensitiveMap_Values(t *testing.T) {
	t.Parallel()

	t.Run("returns all values", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("one", "1")
		m.Add("two", "2")

		values := m.Values()
		assert.Len(t, values, 2)
		assert.Contains(t, values, "1")
		assert.Contains(t, values, "2")
	})

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[int](nil)
		values := m.Values()
		assert.NotNil(t, values)
		assert.Empty(t, values)
	})
}

func TestCaseInsensitiveMap_ContainsKey(t *testing.T) {
	t.Parallel()

	t.Run("case-sensitive lookup", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		exists, key := m.ContainsKey("Content-Type", true)
		assert.True(t, exists)
		assert.Equal(t, "Content-Type", key)

		exists, key = m.ContainsKey("content-type", true)
		assert.False(t, exists)
		assert.Equal(t, "content-type", key)
	})

	t.Run("case-insensitive lookup", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")

		exists, key := m.ContainsKey("content-type", false)
		assert.True(t, exists)
		assert.Equal(t, "Content-Type", key)

		exists, key = m.ContainsKey("CONTENT-TYPE", false)
		assert.True(t, exists)
		assert.Equal(t, "Content-Type", key)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		exists, key := m.ContainsKey("missing", false)
		assert.False(t, exists)
		assert.Equal(t, "missing", key)
	})
}

func TestCaseInsensitiveMap_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("returns true for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		assert.True(t, m.IsEmpty())
	})

	t.Run("returns false for non-empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("key", "value")
		assert.False(t, m.IsEmpty())
	})
}

func TestCaseInsensitiveMap_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates deep copy", func(t *testing.T) {
		t.Parallel()

		original := maps.NewCaseInsensitiveMap[string](nil)
		original.Add("Content-Type", "application/json")
		original.Add("Accept", "text/html")

		clone := original.Clone()
		assert.Equal(t, original.Size(), clone.Size())

		// Verify clone has same values
		_, val1, ok1 := clone.Get("Content-Type", true)
		assert.True(t, ok1)
		assert.Equal(t, "application/json", val1)

		// Modify original
		original.Add("New-Key", "new-value")

		// Clone should not be affected
		assert.Equal(t, 2, clone.Size())
		assert.Equal(t, 3, original.Size())
	})

	t.Run("returns nil for nil receiver", func(t *testing.T) {
		t.Parallel()

		var m *maps.CaseInsensitiveMap[string]
		clone := m.Clone()
		assert.Nil(t, clone)
	})

	t.Run("returns empty map for empty receiver", func(t *testing.T) {
		t.Parallel()

		emptyMap := &maps.CaseInsensitiveMap[string]{}
		clone := emptyMap.Clone()
		assert.NotNil(t, clone)
		assert.True(t, clone.IsEmpty())
	})
}

func TestCaseInsensitiveMap_Merge(t *testing.T) {
	t.Parallel()

	t.Run("merges another map", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewCaseInsensitiveMap[string](nil)
		m1.Add("one", "1")
		m1.Add("two", "2")

		m2 := maps.NewCaseInsensitiveMap[string](nil)
		m2.Add("three", "3")
		m2.Add("four", "4")

		m1.Merge(m2)
		assert.Equal(t, 4, m1.Size())
	})

	t.Run("overwrites existing keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewCaseInsensitiveMap[string](nil)
		m1.Add("key", "value1")

		m2 := maps.NewCaseInsensitiveMap[string](nil)
		m2.Add("KEY", "value2")

		m1.Merge(m2)
		assert.Equal(t, 1, m1.Size())
		_, val, ok := m1.Get("key", false)
		assert.True(t, ok)
		assert.Equal(t, "value2", val)
	})
}

func TestCaseInsensitiveMap_MergeAll(t *testing.T) {
	t.Parallel()

	t.Run("merges multiple maps", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewCaseInsensitiveMap[string](nil)
		m1.Add("one", "1")

		m2 := maps.NewCaseInsensitiveMap[string](nil)
		m2.Add("two", "2")

		m3 := maps.NewCaseInsensitiveMap[string](nil)
		m3.Add("three", "3")

		m1.MergeAll(m2, m3)
		assert.Equal(t, 3, m1.Size())
	})

	t.Run("later maps overwrite earlier ones", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewCaseInsensitiveMap[string](nil)
		m1.Add("key", "original")

		m2 := maps.NewCaseInsensitiveMap[string](nil)
		m2.Add("key", "second")

		m3 := maps.NewCaseInsensitiveMap[string](nil)
		m3.Add("key", "third")

		m1.MergeAll(m2, m3)
		_, val, ok := m1.Get("key", false)
		assert.True(t, ok)
		assert.Equal(t, "third", val)
	})
}

func TestCaseInsensitiveMap_Filter(t *testing.T) {
	t.Parallel()

	t.Run("filters based on predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[int](nil)
		m.Add("one", 1)
		m.Add("two", 2)
		m.Add("three", 3)
		m.Add("four", 4)

		filtered := m.Filter(func(key string, val int) bool {
			return val%2 == 0
		})

		assert.Equal(t, 2, filtered.Size())
		_, val, ok := filtered.Get("two", false)
		assert.True(t, ok)
		assert.Equal(t, 2, val)
		_, val, ok = filtered.Get("four", false)
		assert.True(t, ok)
		assert.Equal(t, 4, val)
	})

	t.Run("preserves original key casing", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[string](nil)
		m.Add("Content-Type", "application/json")
		m.Add("Accept", "text/html")

		filtered := m.Filter(func(key string, val string) bool {
			return key == "Content-Type"
		})

		assert.Equal(t, 1, filtered.Size())
		key, _, ok := filtered.Get("content-type", false)
		assert.True(t, ok)
		assert.Equal(t, "Content-Type", key)
	})

	t.Run("returns empty map when no items match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[int](nil)
		m.Add("one", 1)

		filtered := m.Filter(func(key string, val int) bool {
			return val > 10
		})

		assert.Equal(t, 0, filtered.Size())
	})
}

func TestCaseInsensitiveMap_GenericTypes(t *testing.T) {
	t.Parallel()

	t.Run("works with struct values", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name string
			Age  int
		}

		m := maps.NewCaseInsensitiveMap[Person](nil)
		m.Add("John", Person{Name: "John Doe", Age: 30})

		_, person, ok := m.Get("JOHN", false)
		assert.True(t, ok)
		assert.Equal(t, "John Doe", person.Name)
		assert.Equal(t, 30, person.Age)
	})

	t.Run("works with pointer values", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[*string](nil)
		value := "test"
		m.Add("key", &value)

		_, ptr, ok := m.Get("KEY", false)
		assert.True(t, ok)
		assert.Equal(t, "test", *ptr)
	})

	t.Run("works with slice values", func(t *testing.T) {
		t.Parallel()

		m := maps.NewCaseInsensitiveMap[[]int](nil)
		m.Add("numbers", []int{1, 2, 3})

		_, slice, ok := m.Get("NUMBERS", false)
		assert.True(t, ok)
		assert.Equal(t, []int{1, 2, 3}, slice)
	})
}

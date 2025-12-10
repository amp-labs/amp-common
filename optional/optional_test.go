package optional

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSome(t *testing.T) {
	t.Parallel()

	opt := Some(42)
	assert.True(t, opt.NonEmpty())
	assert.False(t, opt.Empty())

	val, ok := opt.Get()
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestNone(t *testing.T) {
	t.Parallel()

	opt := None[int]()
	assert.False(t, opt.NonEmpty())
	assert.True(t, opt.Empty())

	val, ok := opt.Get()
	assert.False(t, ok)
	assert.Equal(t, 0, val) // zero value
}

func TestGet(t *testing.T) {
	t.Parallel()

	some := Some("hello")
	val, ok := some.Get()
	assert.True(t, ok)
	assert.Equal(t, "hello", val)

	none := None[string]()
	val, ok = none.Get()
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestGetOrPanic(t *testing.T) {
	t.Parallel()

	t.Run("Some", func(t *testing.T) {
		t.Parallel()

		opt := Some(42)
		assert.Equal(t, 42, opt.GetOrPanic())
	})

	t.Run("None", func(t *testing.T) {
		t.Parallel()

		opt := None[int]()

		assert.Panics(t, func() {
			opt.GetOrPanic()
		})
	})
}

func TestGetOrElse(t *testing.T) {
	t.Parallel()

	some := Some(42)
	assert.Equal(t, 42, some.GetOrElse(99))

	none := None[int]()
	assert.Equal(t, 99, none.GetOrElse(99))
}

func TestGetOrElseFunc(t *testing.T) {
	t.Parallel()

	some := Some(42)
	called := false
	result := some.GetOrElseFunc(func() int {
		called = true

		return 99
	})
	assert.Equal(t, 42, result)
	assert.False(t, called, "function should not be called for Some")

	none := None[int]()
	result = none.GetOrElseFunc(func() int {
		called = true

		return 99
	})
	assert.Equal(t, 99, result)
	assert.True(t, called, "function should be called for None")
}

func TestOrElse(t *testing.T) {
	t.Parallel()

	some := Some(42)
	alternative := Some(99)
	result := some.OrElse(alternative)
	assert.Equal(t, 42, result.GetOrPanic())

	none := None[int]()
	result = none.OrElse(alternative)
	assert.Equal(t, 99, result.GetOrPanic())

	result = none.OrElse(None[int]())
	assert.True(t, result.Empty())
}

func TestOrElseFunc(t *testing.T) {
	t.Parallel()

	some := Some(42)
	called := false
	result := some.OrElseFunc(func() Value[int] {
		called = true

		return Some(99)
	})
	assert.Equal(t, 42, result.GetOrPanic())
	assert.False(t, called, "function should not be called for Some")

	none := None[int]()
	result = none.OrElseFunc(func() Value[int] {
		called = true

		return Some(99)
	})
	assert.Equal(t, 99, result.GetOrPanic())
	assert.True(t, called, "function should be called for None")
}

func TestEquals(t *testing.T) {
	t.Parallel()

	eq := func(a, b int) bool { return a == b }

	some42 := Some(42)
	some99 := Some(99)
	none := None[int]()

	assert.True(t, some42.Equals(Some(42), eq))
	assert.False(t, some42.Equals(some99, eq))
	assert.False(t, some42.Equals(none, eq))
	assert.False(t, none.Equals(some42, eq))
	assert.True(t, none.Equals(None[int](), eq))
}

func TestFilter(t *testing.T) {
	t.Parallel()

	isEven := func(n int) bool { return n%2 == 0 }

	some := Some(42)
	result := some.Filter(isEven)
	assert.True(t, result.NonEmpty())
	assert.Equal(t, 42, result.GetOrPanic())

	some = Some(43)
	result = some.Filter(isEven)
	assert.True(t, result.Empty())

	none := None[int]()
	result = none.Filter(isEven)
	assert.True(t, result.Empty())
}

func TestSize(t *testing.T) {
	t.Parallel()

	some := Some(42)
	none := None[int]()

	assert.Equal(t, 1, some.Size())
	assert.Equal(t, 0, none.Size())
}

func TestString(t *testing.T) {
	t.Parallel()

	some := Some(42)
	none := None[int]()
	someStr := Some("hello")

	assert.Equal(t, "Some(42)", some.String())
	assert.Equal(t, "None", none.String())
	assert.Equal(t, "Some(hello)", someStr.String())
}

func TestAll(t *testing.T) {
	t.Parallel()

	t.Run("Some", func(t *testing.T) {
		t.Parallel()

		opt := Some(42)
		values := []int{}

		for v := range opt.All() {
			values = append(values, v)
		}

		assert.Equal(t, []int{42}, values)
	})

	t.Run("None", func(t *testing.T) {
		t.Parallel()

		opt := None[int]()
		values := []int{}

		for v := range opt.All() {
			values = append(values, v)
		}

		assert.Empty(t, values)
	})
}

func TestForEach(t *testing.T) {
	t.Parallel()

	t.Run("Some", func(t *testing.T) {
		t.Parallel()

		opt := Some(42)
		called := false

		var value int

		opt.ForEach(func(v int) {
			called = true
			value = v
		})
		assert.True(t, called)
		assert.Equal(t, 42, value)
	})

	t.Run("None", func(t *testing.T) {
		t.Parallel()

		opt := None[int]()
		called := false

		opt.ForEach(func(v int) {
			called = true
		})
		assert.False(t, called)
	})
}

func TestMap(t *testing.T) {
	t.Parallel()

	double := func(n int) int { return n * 2 }

	some := Some(21)
	result := Map(some, double)
	assert.Equal(t, 42, result.GetOrPanic())

	none := None[int]()
	result = Map(none, double)
	assert.True(t, result.Empty())
}

func TestMapTypeChange(t *testing.T) {
	t.Parallel()

	toString := func(n int) string { return string(rune(n + '0')) }

	some := Some(5)
	result := Map(some, toString)
	assert.Equal(t, "5", result.GetOrPanic())

	none := None[int]()
	strResult := Map(none, toString)
	assert.True(t, strResult.Empty())
}

func TestFlatMap(t *testing.T) {
	t.Parallel()

	safeDivide := func(n int) Value[int] {
		if n == 0 {
			return None[int]()
		}

		return Some(100 / n)
	}

	some := Some(10)
	result := FlatMap(some, safeDivide)
	assert.Equal(t, 10, result.GetOrPanic())

	someZero := Some(0)
	result = FlatMap(someZero, safeDivide)
	assert.True(t, result.Empty())

	none := None[int]()
	result = FlatMap(none, safeDivide)
	assert.True(t, result.Empty())
}

func TestFlatMapTypeChange(t *testing.T) {
	t.Parallel()

	parsePositive := func(n int) Value[string] {
		if n > 0 {
			return Some("positive")
		}

		return None[string]()
	}

	some := Some(42)
	result := FlatMap(some, parsePositive)
	assert.Equal(t, "positive", result.GetOrPanic())

	someNeg := Some(-5)
	result = FlatMap(someNeg, parsePositive)
	assert.True(t, result.Empty())

	none := None[int]()
	result = FlatMap(none, parsePositive)
	assert.True(t, result.Empty())
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("Some", func(t *testing.T) {
		t.Parallel()

		opt := Some(42)

		data, err := json.Marshal(opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.JSONEq(t, `{"value":42}`, string(data))
	})

	t.Run("None", func(t *testing.T) {
		t.Parallel()

		opt := None[int]()

		data, err := json.Marshal(opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "null", string(data))
	})

	t.Run("Some with string", func(t *testing.T) {
		t.Parallel()

		opt := Some("hello")

		data, err := json.Marshal(opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.JSONEq(t, `{"value":"hello"}`, string(data))
	})

	t.Run("Some with struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		opt := Some(testStruct{Name: "Alice", Age: 30})

		data, err := json.Marshal(opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.JSONEq(t, `{"value":{"name":"Alice","age":30}}`, string(data))
	})
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("Some", func(t *testing.T) {
		t.Parallel()

		var opt Value[int]

		err := json.Unmarshal([]byte(`{"value":42}`), &opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, opt.NonEmpty())
		assert.Equal(t, 42, opt.GetOrPanic())
	})

	t.Run("None", func(t *testing.T) {
		t.Parallel()

		var opt Value[int]

		err := json.Unmarshal([]byte(`null`), &opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, opt.Empty())
	})

	t.Run("Some with string", func(t *testing.T) {
		t.Parallel()

		var opt Value[string]

		err := json.Unmarshal([]byte(`{"value":"hello"}`), &opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, opt.NonEmpty())
		assert.Equal(t, "hello", opt.GetOrPanic())
	})

	t.Run("Some with struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		var opt Value[testStruct]

		err := json.Unmarshal([]byte(`{"value":{"name":"Alice","age":30}}`), &opt)
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, opt.NonEmpty())
		result := opt.GetOrPanic()
		assert.Equal(t, "Alice", result.Name)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("Missing value field", func(t *testing.T) {
		t.Parallel()

		var opt Value[int]

		err := json.Unmarshal([]byte(`{"other":42}`), &opt)
		if err == nil {
			t.Fatal("expected error but got nil")
		}

		assert.Contains(t, err.Error(), "missing 'value' field")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		t.Parallel()

		var opt Value[int]

		err := json.Unmarshal([]byte(`{invalid}`), &opt)
		require.Error(t, err)
	})
}

func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("Some", func(t *testing.T) {
		t.Parallel()

		original := Some(42)

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}

		var result Value[int]

		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, original, result)
	})

	t.Run("None", func(t *testing.T) {
		t.Parallel()

		original := None[int]()

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}

		var result Value[int]

		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, result.Empty())
	})
}

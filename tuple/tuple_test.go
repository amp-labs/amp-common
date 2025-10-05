package tuple

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTuple2(t *testing.T) {
	t.Parallel()

	tuple := NewTuple2("hello", 42)

	assert.Equal(t, "hello", tuple.First())
	assert.Equal(t, 42, tuple.Second())
}

func TestTuple3(t *testing.T) {
	t.Parallel()

	tuple := NewTuple3("hello", 42, true)

	assert.Equal(t, "hello", tuple.First())
	assert.Equal(t, 42, tuple.Second())
	assert.True(t, tuple.Third())
}

func TestTuple4(t *testing.T) {
	t.Parallel()

	tuple := NewTuple4("hello", 42, true, 3.14)

	assert.Equal(t, "hello", tuple.First())
	assert.Equal(t, 42, tuple.Second())
	assert.True(t, tuple.Third())
	assert.InEpsilon(t, 3.14, tuple.Fourth(), 0.0001)
}

func TestTuple5(t *testing.T) {
	t.Parallel()

	tuple := NewTuple5("hello", 42, true, 3.14, 'x')

	assert.Equal(t, "hello", tuple.First())
	assert.Equal(t, 42, tuple.Second())
	assert.True(t, tuple.Third())
	assert.InEpsilon(t, 3.14, tuple.Fourth(), 0.0001)
	assert.Equal(t, 'x', tuple.Fifth())
}

func TestTuple6(t *testing.T) {
	t.Parallel()

	tuple := NewTuple6("hello", 42, true, 3.14, 'x', []int{1, 2, 3})

	assert.Equal(t, "hello", tuple.First())
	assert.Equal(t, 42, tuple.Second())
	assert.True(t, tuple.Third())
	assert.InEpsilon(t, 3.14, tuple.Fourth(), 0.0001)
	assert.Equal(t, 'x', tuple.Fifth())
	assert.Equal(t, []int{1, 2, 3}, tuple.Sixth())
}

func TestTupleWithComplexTypes(t *testing.T) {
	t.Parallel()

	type User struct {
		Name string
		Age  int
	}

	user := User{Name: "Alice", Age: 30}
	tuple := NewTuple2(user, map[string]int{"score": 100})

	assert.Equal(t, user, tuple.First())
	assert.Equal(t, map[string]int{"score": 100}, tuple.Second())
}

func TestTupleWithNilValues(t *testing.T) {
	t.Parallel()

	tuple := NewTuple2[*string, *int](nil, nil)

	assert.Nil(t, tuple.First())
	assert.Nil(t, tuple.Second())
}

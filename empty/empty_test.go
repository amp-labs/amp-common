package empty_test

import (
	"testing"

	"github.com/amp-labs/amp-common/empty"
	"github.com/stretchr/testify/assert"
)

func TestT(t *testing.T) {
	t.Parallel()

	var x empty.T

	assert.Equal(t, empty.T{}, x)
}

func TestV(t *testing.T) {
	t.Parallel()

	assert.Equal(t, empty.T{}, empty.V)
}

func TestP(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, empty.P)
	assert.Equal(t, &empty.V, empty.P)
}

func TestSlice(t *testing.T) {
	t.Parallel()

	result := empty.Slice[string]()

	assert.NotNil(t, result)
	assert.Empty(t, result)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, 0, cap(result))
}

func TestSlictPtr(t *testing.T) {
	t.Parallel()

	result := empty.SlictPtr[int]()

	assert.NotNil(t, result)
	assert.NotNil(t, *result)
	assert.Empty(t, *result)
	assert.Equal(t, 0, len(*result))
}

func TestMap(t *testing.T) {
	t.Parallel()

	result := empty.Map[string, int]()

	assert.NotNil(t, result)
	assert.Empty(t, result)
	assert.Equal(t, 0, len(result))
}

func TestMapPtr(t *testing.T) {
	t.Parallel()

	result := empty.MapPtr[string, bool]()

	assert.NotNil(t, result)
	assert.NotNil(t, *result)
	assert.Empty(t, *result)
	assert.Equal(t, 0, len(*result))
}

func TestChan(t *testing.T) {
	t.Parallel()

	result := empty.Chan[int]()

	assert.NotNil(t, result)

	// Channel should be closed
	val, ok := <-result
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestChanPtr(t *testing.T) {
	t.Parallel()

	result := empty.ChanPtr[string]()

	assert.NotNil(t, result)
	assert.NotNil(t, *result)

	// Channel should be closed
	val, ok := <-*result
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestValue(t *testing.T) {
	t.Parallel()

	// Test with various types
	intVal := empty.Value[int]()
	assert.Equal(t, 0, intVal)

	strVal := empty.Value[string]()
	assert.Equal(t, "", strVal)

	boolVal := empty.Value[bool]()
	assert.Equal(t, false, boolVal)

	sliceVal := empty.Value[[]string]()
	assert.Nil(t, sliceVal)

	mapVal := empty.Value[map[string]int]()
	assert.Nil(t, mapVal)
}

func TestFunc(t *testing.T) {
	t.Parallel()

	// Should not panic
	assert.NotPanics(t, func() {
		empty.Func()
	})
}

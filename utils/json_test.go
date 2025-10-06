package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSONMap(t *testing.T) {
	t.Parallel()

	t.Run("converts struct to JSON map", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string `json:"name"`
			Age   int    `json:"age"`
			Email string `json:"email"`
		}

		input := testStruct{
			Name:  "John Doe",
			Age:   30,
			Email: "john@example.com",
		}

		result, err := ToJSONMap(input)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", result["name"])
		assert.InDelta(t, 30, result["age"], 0) // JSON unmarshal converts numbers to float64
		assert.Equal(t, "john@example.com", result["email"])
	})

	t.Run("converts nested struct to JSON map", func(t *testing.T) {
		t.Parallel()

		type address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type person struct {
			Name    string  `json:"name"`
			Address address `json:"address"`
		}

		input := person{
			Name: "Jane",
			Address: address{
				Street: "123 Main St",
				City:   "NYC",
			},
		}

		result, err := ToJSONMap(input)
		require.NoError(t, err)
		assert.Equal(t, "Jane", result["name"])

		addr, ok := result["address"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "123 Main St", addr["street"])
		assert.Equal(t, "NYC", addr["city"])
	})

	t.Run("handles empty struct", func(t *testing.T) {
		t.Parallel()

		type emptyStruct struct{}

		input := emptyStruct{}
		result, err := ToJSONMap(input)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handles struct with omitempty", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string `json:"name"`
			Age   int    `json:"age,omitempty"`
			Email string `json:"email,omitempty"`
		}

		input := testStruct{
			Name: "John",
		}

		result, err := ToJSONMap(input)
		require.NoError(t, err)
		assert.Equal(t, "John", result["name"])
		assert.NotContains(t, result, "email")
	})

	t.Run("handles map input", func(t *testing.T) {
		t.Parallel()

		input := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		result, err := ToJSONMap(input)
		require.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
	})

	t.Run("handles slice in struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Tags []string `json:"tags"`
		}

		input := testStruct{
			Tags: []string{"go", "testing", "utils"},
		}

		result, err := ToJSONMap(input)
		require.NoError(t, err)

		tags, ok := result["tags"].([]interface{})
		require.True(t, ok)
		assert.Len(t, tags, 3)
	})
}

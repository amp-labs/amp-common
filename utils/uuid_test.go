package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkParseUUIDs(t *testing.T) {
	t.Parallel()

	t.Run("parses valid UUIDs with required fields", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
			Value2 string
		}

		type UUIDs struct {
			Value1 uuid.UUID
			Value2 uuid.UUID
		}

		inputs := &StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
			Value2: "5d77349b-df0c-42d7-b351-b30b499b73b9",
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.NoError(t, err)

		expectedValue1, _ := uuid.Parse("798f9d41-f89b-4b90-a3ae-c560c3c99203")
		expectedValue2, _ := uuid.Parse("5d77349b-df0c-42d7-b351-b30b499b73b9")

		assert.Equal(t, expectedValue1, outputs.Value1)
		assert.Equal(t, expectedValue2, outputs.Value2)
	})

	t.Run("parses valid UUIDs with optional fields", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 *string
			Value2 *string
		}

		type UUIDs struct {
			Value1 *uuid.UUID
			Value2 *uuid.UUID
		}

		val1 := "798f9d41-f89b-4b90-a3ae-c560c3c99203" //nolint:goconst // Test data, not a constant
		val2 := "5d77349b-df0c-42d7-b351-b30b499b73b9"

		inputs := &StringUUIDs{
			Value1: &val1,
			Value2: &val2,
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.NoError(t, err)

		expectedValue1, _ := uuid.Parse("798f9d41-f89b-4b90-a3ae-c560c3c99203")
		expectedValue2, _ := uuid.Parse("5d77349b-df0c-42d7-b351-b30b499b73b9")

		require.NotNil(t, outputs.Value1)
		require.NotNil(t, outputs.Value2)
		assert.Equal(t, expectedValue1, *outputs.Value1)
		assert.Equal(t, expectedValue2, *outputs.Value2)
	})

	t.Run("handles nil pointer fields", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 *string
			Value2 *string
		}

		type UUIDs struct {
			Value1 *uuid.UUID
			Value2 *uuid.UUID
		}

		val1 := "798f9d41-f89b-4b90-a3ae-c560c3c99203"

		inputs := &StringUUIDs{
			Value1: &val1,
			Value2: nil,
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.NoError(t, err)

		expectedValue1, _ := uuid.Parse("798f9d41-f89b-4b90-a3ae-c560c3c99203")

		require.NotNil(t, outputs.Value1)
		assert.Equal(t, expectedValue1, *outputs.Value1)
		assert.Nil(t, outputs.Value2)
	})

	t.Run("handles mixed required and optional fields", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Required string
			Optional *string
		}

		type UUIDs struct {
			Required uuid.UUID
			Optional *uuid.UUID
		}

		optVal := "5d77349b-df0c-42d7-b351-b30b499b73b9"

		inputs := &StringUUIDs{
			Required: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
			Optional: &optVal,
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.NoError(t, err)

		expectedRequired, _ := uuid.Parse("798f9d41-f89b-4b90-a3ae-c560c3c99203")
		expectedOptional, _ := uuid.Parse("5d77349b-df0c-42d7-b351-b30b499b73b9")

		assert.Equal(t, expectedRequired, outputs.Required)
		require.NotNil(t, outputs.Optional)
		assert.Equal(t, expectedOptional, *outputs.Optional)
	})

	t.Run("skips unexported fields", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Exported   string
			unexported string //nolint:unused
		}

		type UUIDs struct {
			Exported uuid.UUID
		}

		inputs := &StringUUIDs{
			Exported:   "798f9d41-f89b-4b90-a3ae-c560c3c99203",
			unexported: "should-be-skipped",
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.NoError(t, err)

		expectedExported, _ := uuid.Parse("798f9d41-f89b-4b90-a3ae-c560c3c99203")
		assert.Equal(t, expectedExported, outputs.Exported)
	})

	t.Run("returns error for invalid UUID string", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
		}

		type UUIDs struct {
			Value1 uuid.UUID
		}

		inputs := &StringUUIDs{
			Value1: "not-a-valid-uuid",
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing UUID")
		assert.Contains(t, err.Error(), "Value1")
	})

	t.Run("returns error when inputs is not a pointer", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
		}

		type UUIDs struct {
			Value1 uuid.UUID
		}

		inputs := StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inputs must be a pointer to a struct")
	})

	t.Run("returns error when outputs is not a pointer", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
		}

		type UUIDs struct {
			Value1 uuid.UUID
		}

		inputs := &StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
		}

		outputs := UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "outputs must be a pointer to a struct")
	})

	t.Run("returns error when inputs is not a struct", func(t *testing.T) {
		t.Parallel()

		type UUIDs struct {
			Value1 uuid.UUID
		}

		inputs := new(string)
		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inputs must be a pointer to a struct")
	})

	t.Run("returns error when outputs is not a struct", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
		}

		inputs := &StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
		}

		outputs := new(string)

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "outputs must be a pointer to a struct")
	})

	t.Run("returns error when field not found in output struct", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
			Value2 string
		}

		type UUIDs struct {
			Value1 uuid.UUID
			// Value2 is missing
		}

		inputs := &StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
			Value2: "5d77349b-df0c-42d7-b351-b30b499b73b9",
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field Value2 not found in output struct")
	})

	t.Run("returns error for pointer mismatch - input pointer, output not", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 *string
		}

		type UUIDs struct {
			Value1 uuid.UUID
		}

		val1 := "798f9d41-f89b-4b90-a3ae-c560c3c99203"

		inputs := &StringUUIDs{
			Value1: &val1,
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pointer mismatch")
		assert.Contains(t, err.Error(), "Value1")
	})

	t.Run("returns error for pointer mismatch - input not pointer, output is", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
		}

		type UUIDs struct {
			Value1 *uuid.UUID
		}

		inputs := &StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pointer mismatch")
		assert.Contains(t, err.Error(), "Value1")
	})

	t.Run("returns error when input field is not string", func(t *testing.T) {
		t.Parallel()

		type InvalidInputs struct {
			Value1 int
		}

		type UUIDs struct {
			Value1 uuid.UUID
		}

		inputs := &InvalidInputs{
			Value1: 123,
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be string in input struct")
		assert.Contains(t, err.Error(), "Value1")
	})

	t.Run("returns error when input pointer field is not *string", func(t *testing.T) {
		t.Parallel()

		type InvalidInputs struct {
			Value1 *int
		}

		type UUIDs struct {
			Value1 *uuid.UUID
		}

		val := 123
		inputs := &InvalidInputs{
			Value1: &val,
		}

		outputs := &UUIDs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be *string in input struct")
		assert.Contains(t, err.Error(), "Value1")
	})

	t.Run("returns error when output field is not uuid.UUID", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 string
		}

		type InvalidOutputs struct {
			Value1 string
		}

		inputs := &StringUUIDs{
			Value1: "798f9d41-f89b-4b90-a3ae-c560c3c99203",
		}

		outputs := &InvalidOutputs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be uuid.UUID in output struct")
		assert.Contains(t, err.Error(), "Value1")
	})

	t.Run("returns error when output pointer field is not *uuid.UUID", func(t *testing.T) {
		t.Parallel()

		type StringUUIDs struct {
			Value1 *string
		}

		type InvalidOutputs struct {
			Value1 *string
		}

		val := "798f9d41-f89b-4b90-a3ae-c560c3c99203"
		inputs := &StringUUIDs{
			Value1: &val,
		}

		outputs := &InvalidOutputs{}

		err := BulkParseUUIDs(inputs, outputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be *uuid.UUID in output struct")
		assert.Contains(t, err.Error(), "Value1")
	})
}

func TestUUIDBulkParser(t *testing.T) {
	t.Parallel()

	t.Run("parses valid UUIDs", func(t *testing.T) {
		t.Parallel()

		parser := NewUUIDParser()
		parser.Add("id1", "798f9d41-f89b-4b90-a3ae-c560c3c99203")
		parser.Add("id2", "5d77349b-df0c-42d7-b351-b30b499b73b9")

		result, err := parser.Parse()
		require.NoError(t, err)

		expectedID1, _ := uuid.Parse("798f9d41-f89b-4b90-a3ae-c560c3c99203")
		expectedID2, _ := uuid.Parse("5d77349b-df0c-42d7-b351-b30b499b73b9")

		assert.Equal(t, expectedID1, result["id1"])
		assert.Equal(t, expectedID2, result["id2"])
	})

	t.Run("returns error for invalid UUID", func(t *testing.T) {
		t.Parallel()

		parser := NewUUIDParser()
		parser.Add("id1", "not-a-valid-uuid")

		result, err := parser.Parse()
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error parsing UUID")
		assert.Contains(t, err.Error(), "id1")
	})

	t.Run("handles empty parser", func(t *testing.T) {
		t.Parallel()

		parser := NewUUIDParser()

		result, err := parser.Parse()
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

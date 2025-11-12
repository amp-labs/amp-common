package utils

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// UUIDBulkParser collects and validates multiple UUID strings in bulk.
type UUIDBulkParser struct {
	idMap map[string]string
}

// NewUUIDParser creates a new UUIDBulkParser for collecting UUID strings.
func NewUUIDParser() *UUIDBulkParser {
	return &UUIDBulkParser{
		idMap: make(map[string]string),
	}
}

// Add adds a key-value pair to the parser for later validation.
func (up *UUIDBulkParser) Add(key, value string) {
	up.idMap[key] = value
}

// Parse validates all collected UUID strings and returns them as uuid.UUID values.
// Returns an error if any UUID string is invalid.
func (up *UUIDBulkParser) Parse() (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(up.idMap))

	for key, val := range up.idMap {
		id, err := uuid.Parse(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing UUID %s for %s: %w", val, key, err)
		}

		ids[key] = id
	}

	return ids, nil
}

// BulkParseUUIDs converts a struct with string fields to a struct with uuid.UUID fields.
// Both inputs and outputs must be pointers to structs. Fields are matched by name.
// Input fields must be string or *string, output fields must be uuid.UUID or *uuid.UUID.
// If a field is a pointer in the input, it must also be a pointer in the output (and vice versa).
// The "name" struct tag can be used to customize the field name in error messages (e.g., `name:"customName"`).
//
//nolint:err113,nestif // Validation errors need dynamic context; nested structure is clearer than alternatives
func BulkParseUUIDs(inputs, outputs interface{}) error {
	inputVal := reflect.ValueOf(inputs)
	outputVal := reflect.ValueOf(outputs)

	// Validate that both are pointers
	if inputVal.Kind() != reflect.Ptr {
		return fmt.Errorf("inputs must be a pointer to a struct, got %T", inputs)
	}

	if outputVal.Kind() != reflect.Ptr {
		return fmt.Errorf("outputs must be a pointer to a struct, got %T", outputs)
	}

	// Dereference pointers
	inputVal = inputVal.Elem()
	outputVal = outputVal.Elem()

	// Validate that both are structs
	if inputVal.Kind() != reflect.Struct {
		return fmt.Errorf("inputs must be a pointer to a struct, got pointer to %s", inputVal.Kind())
	}

	if outputVal.Kind() != reflect.Struct {
		return fmt.Errorf("outputs must be a pointer to a struct, got pointer to %s", outputVal.Kind())
	}

	inputType := inputVal.Type()

	// Iterate through input fields
	for i := range inputVal.NumField() {
		inputField := inputVal.Field(i)
		inputFieldType := inputType.Field(i)
		structFieldName := inputFieldType.Name

		// Get display name from "name" tag if present, otherwise use reflected field name
		displayName := inputFieldType.Tag.Get("name")
		if displayName == "" {
			displayName = structFieldName
		}

		// Skip unexported fields
		if !inputField.CanInterface() {
			continue
		}

		// Find corresponding output field (always use actual struct field name)
		outputField := outputVal.FieldByName(structFieldName)
		if !outputField.IsValid() {
			return fmt.Errorf("field %s not found in output struct", displayName)
		}

		if !outputField.CanSet() {
			return fmt.Errorf("field %s in output struct cannot be set (unexported?)", displayName)
		}

		// Handle pointer vs non-pointer field types
		inputIsPtr := inputField.Kind() == reflect.Ptr
		outputIsPtr := outputField.Kind() == reflect.Ptr

		if inputIsPtr != outputIsPtr {
			return fmt.Errorf(
				"field %s pointer mismatch: input is pointer=%v, output is pointer=%v",
				displayName, inputIsPtr, outputIsPtr,
			)
		}

		// Handle pointer fields (optional values)
		if inputIsPtr {
			if inputField.IsNil() {
				// Input is nil, set output to nil
				outputField.Set(reflect.Zero(outputField.Type()))

				continue
			}

			// Dereference pointers for validation and parsing
			inputFieldDeref := inputField.Elem()
			if inputFieldDeref.Kind() != reflect.String {
				return fmt.Errorf("field %s must be *string in input struct, got *%s", displayName, inputFieldDeref.Kind())
			}

			// Validate output is *uuid.UUID
			if outputField.Type() != reflect.TypeOf((*uuid.UUID)(nil)) {
				return fmt.Errorf("field %s must be *uuid.UUID in output struct, got %s", displayName, outputField.Type())
			}

			// Parse UUID
			uuidStr := inputFieldDeref.String()

			parsedUUID, err := uuid.Parse(uuidStr)
			if err != nil {
				return fmt.Errorf("error parsing UUID %s for field %s: %w", uuidStr, displayName, err)
			}

			// Create new pointer and set the value
			newUUID := reflect.New(reflect.TypeOf(uuid.UUID{}))
			newUUID.Elem().Set(reflect.ValueOf(parsedUUID))
			outputField.Set(newUUID)
		} else {
			// Handle non-pointer fields (required values)
			if inputField.Kind() != reflect.String {
				return fmt.Errorf("field %s must be string in input struct, got %s", displayName, inputField.Kind())
			}

			// Validate output is uuid.UUID
			if outputField.Type() != reflect.TypeOf(uuid.UUID{}) {
				return fmt.Errorf("field %s must be uuid.UUID in output struct, got %s", displayName, outputField.Type())
			}

			// Parse UUID
			uuidStr := inputField.String()

			parsedUUID, err := uuid.Parse(uuidStr)
			if err != nil {
				return fmt.Errorf("error parsing UUID %s for field %s: %w", uuidStr, displayName, err)
			}

			outputField.Set(reflect.ValueOf(parsedUUID))
		}
	}

	return nil
}

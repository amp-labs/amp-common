package utils

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// Error variables for BulkParseUUIDs validation failures.
var (
	// ErrInputsNotPointer is returned when the inputs parameter is not a pointer.
	ErrInputsNotPointer = errors.New("inputs must be a pointer to a struct")

	// ErrOutputsNotPointer is returned when the outputs parameter is not a pointer.
	ErrOutputsNotPointer = errors.New("outputs must be a pointer to a struct")

	// ErrInputsNotStruct is returned when inputs is a pointer but not to a struct.
	ErrInputsNotStruct = errors.New("inputs must be a pointer to a struct")

	// ErrOutputsNotStruct is returned when outputs is a pointer but not to a struct.
	ErrOutputsNotStruct = errors.New("outputs must be a pointer to a struct")

	// ErrFieldNotFound is returned when a field from inputs is not found in outputs.
	ErrFieldNotFound = errors.New("field not found in output struct")

	// ErrFieldNotSettable is returned when a field in outputs cannot be set (e.g., unexported).
	ErrFieldNotSettable = errors.New("field in output struct cannot be set")

	// ErrPointerMismatch is returned when input and output fields have mismatched pointer types.
	ErrPointerMismatch = errors.New("field pointer mismatch between input and output")

	// ErrInvalidInputFieldType is returned when an input field is not string or *string.
	ErrInvalidInputFieldType = errors.New("input field must be string or *string")

	// ErrInvalidOutputFieldType is returned when an output field is not uuid.UUID or *uuid.UUID.
	ErrInvalidOutputFieldType = errors.New("output field must be uuid.UUID or *uuid.UUID")

	// ErrInvalidUUID is returned when a UUID string cannot be parsed.
	ErrInvalidUUID = errors.New("invalid UUID")

	// ErrMultipleInvalidUUIDs is returned when multiple UUID parsing errors occur.
	ErrMultipleInvalidUUIDs = errors.New("multiple invalid UUIDs")
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

	var errs []error

	for key, val := range up.idMap {
		id, err := uuid.Parse(val)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w %q for %s: %w", ErrInvalidUUID, val, key, err))

			continue
		}

		ids[key] = id
	}

	if len(errs) > 0 {
		return nil, collectParsingErrors(errs)
	}

	return ids, nil
}

// validateStructPointers validates that both inputs and outputs are pointers to structs.
// Returns the dereferenced values if valid.
func validateStructPointers(inputs, outputs interface{}) (reflect.Value, reflect.Value, error) {
	inputVal := reflect.ValueOf(inputs)
	outputVal := reflect.ValueOf(outputs)

	if inputVal.Kind() != reflect.Ptr {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("%w, got %T", ErrInputsNotPointer, inputs)
	}

	if outputVal.Kind() != reflect.Ptr {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("%w, got %T", ErrOutputsNotPointer, outputs)
	}

	inputVal = inputVal.Elem()
	outputVal = outputVal.Elem()

	if inputVal.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{},
			fmt.Errorf("%w, got pointer to %s", ErrInputsNotStruct, inputVal.Kind())
	}

	if outputVal.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{},
			fmt.Errorf("%w, got pointer to %s", ErrOutputsNotStruct, outputVal.Kind())
	}

	return inputVal, outputVal, nil
}

// getFieldDisplayName returns the display name for a field, using the "name" tag if present.
func getFieldDisplayName(field reflect.StructField) string {
	displayName := field.Tag.Get("name")
	if displayName == "" {
		displayName = field.Name
	}

	return displayName
}

// validateFieldCompatibility validates that input and output fields are compatible.
func validateFieldCompatibility(inputField, outputField reflect.Value, displayName string) error {
	if !outputField.IsValid() {
		return fmt.Errorf("%w: field %s", ErrFieldNotFound, displayName)
	}

	if !outputField.CanSet() {
		return fmt.Errorf("%w: field %s (unexported?)", ErrFieldNotSettable, displayName)
	}

	inputIsPtr := inputField.Kind() == reflect.Ptr
	outputIsPtr := outputField.Kind() == reflect.Ptr

	if inputIsPtr != outputIsPtr {
		return fmt.Errorf(
			"%w: field %s (input is pointer=%v, output is pointer=%v)",
			ErrPointerMismatch, displayName, inputIsPtr, outputIsPtr,
		)
	}

	return nil
}

// processPointerField handles parsing and setting pointer UUID fields.
func processPointerField(inputField, outputField reflect.Value, displayName string) error {
	if inputField.IsNil() {
		outputField.Set(reflect.Zero(outputField.Type()))

		return nil
	}

	inputFieldDeref := inputField.Elem()
	if inputFieldDeref.Kind() != reflect.String {
		return fmt.Errorf("%w: field %s (got *%s)", ErrInvalidInputFieldType, displayName, inputFieldDeref.Kind())
	}

	if outputField.Type() != reflect.TypeOf((*uuid.UUID)(nil)) {
		return fmt.Errorf("%w: field %s (got %s)", ErrInvalidOutputFieldType, displayName, outputField.Type())
	}

	uuidStr := inputFieldDeref.String()

	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return fmt.Errorf("%w %q for field %s: %w", ErrInvalidUUID, uuidStr, displayName, err)
	}

	newUUID := reflect.New(reflect.TypeOf(uuid.UUID{}))
	newUUID.Elem().Set(reflect.ValueOf(parsedUUID))
	outputField.Set(newUUID)

	return nil
}

// processNonPointerField handles parsing and setting non-pointer UUID fields.
func processNonPointerField(inputField, outputField reflect.Value, displayName string) error {
	if inputField.Kind() != reflect.String {
		return fmt.Errorf("%w: field %s (got %s)", ErrInvalidInputFieldType, displayName, inputField.Kind())
	}

	if outputField.Type() != reflect.TypeOf(uuid.UUID{}) {
		return fmt.Errorf("%w: field %s (got %s)", ErrInvalidOutputFieldType, displayName, outputField.Type())
	}

	uuidStr := inputField.String()

	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return fmt.Errorf("%w %q for field %s: %w", ErrInvalidUUID, uuidStr, displayName, err)
	}

	outputField.Set(reflect.ValueOf(parsedUUID))

	return nil
}

// collectParsingErrors returns a combined error from multiple parsing errors.
func collectParsingErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return fmt.Errorf("%w (found %d): %w", ErrMultipleInvalidUUIDs, len(errs), errors.Join(errs...))
}

// BulkParseUUIDs converts a struct with string fields to a struct with uuid.UUID fields.
// Both inputs and outputs must be pointers to structs. Fields are matched by name.
// Input fields must be string or *string, output fields must be uuid.UUID or *uuid.UUID.
// If a field is a pointer in the input, it must also be a pointer in the output (and vice versa).
// The "name" struct tag can be used to customize the field name in error messages (e.g., `name:"customName"`).
func BulkParseUUIDs(inputs, outputs interface{}) error {
	inputVal, outputVal, err := validateStructPointers(inputs, outputs)
	if err != nil {
		return err
	}

	inputType := inputVal.Type()

	var errs []error

	for i := range inputVal.NumField() {
		inputField := inputVal.Field(i)
		inputFieldType := inputType.Field(i)
		structFieldName := inputFieldType.Name

		if !inputField.CanInterface() {
			continue
		}

		displayName := getFieldDisplayName(inputFieldType)
		outputField := outputVal.FieldByName(structFieldName)

		if err := validateFieldCompatibility(inputField, outputField, displayName); err != nil {
			return err
		}

		var fieldErr error
		if inputField.Kind() == reflect.Ptr {
			fieldErr = processPointerField(inputField, outputField, displayName)
		} else {
			fieldErr = processNonPointerField(inputField, outputField, displayName)
		}

		if fieldErr != nil {
			errs = append(errs, fieldErr)
		}
	}

	return collectParsingErrors(errs)
}

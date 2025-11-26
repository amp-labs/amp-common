// nolint
package jsonpath

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPathDeeplyNested = "$['user']['profile']['address']['street']"

func TestParsePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		wantLen   int
		wantKeys  []string
		wantErr   bool
		errString string
	}{
		{
			name:     "simple path",
			path:     "$['user']",
			wantLen:  1,
			wantKeys: []string{"user"},
		},
		{
			name:     "nested path",
			path:     "$['user']['name']",
			wantLen:  2,
			wantKeys: []string{"user", "name"},
		},
		{
			name:     "deeply nested path",
			path:     testPathDeeplyNested,
			wantLen:  4,
			wantKeys: []string{"user", "profile", "address", "street"},
		},
		{
			name:     "field with underscore",
			path:     "$['first_name']",
			wantLen:  1,
			wantKeys: []string{"first_name"},
		},
		{
			name:     "field with hyphen",
			path:     "$['last-name']",
			wantLen:  1,
			wantKeys: []string{"last-name"},
		},
		{
			name:     "field with dot",
			path:     "$['email.primary']",
			wantLen:  1,
			wantKeys: []string{"email.primary"},
		},
		{
			name:      "empty path",
			path:      "",
			wantErr:   true,
			errString: "path cannot be empty",
		},
		{
			name:      "missing dollar sign",
			path:      "['user']",
			wantErr:   true,
			errString: "path must start with $[",
		},
		{
			name:      "empty segment",
			path:      "$['']",
			wantErr:   true,
			errString: "segment 0",
		},
		{
			name:      "empty segment in middle",
			path:      "$['user']['']['name']",
			wantErr:   true,
			errString: "segment 1",
		},
		{
			name:      "invalid syntax - no brackets",
			path:      "$user",
			wantErr:   true,
			errString: "path must start with $[",
		},
		{
			name:      "invalid syntax - missing quotes",
			path:      "$[user]",
			wantErr:   true,
			errString: "no valid segments found",
		},
		{
			name:      "invalid syntax - single quotes mismatch",
			path:      "$['user\"]",
			wantErr:   true,
			errString: "no valid segments found",
		},
		{
			name:      "invalid syntax - extra characters",
			path:      "$['user']extra",
			wantErr:   true,
			errString: "invalid bracket notation syntax",
		},
		{
			name:     "field with spaces",
			path:     "$['full name']",
			wantLen:  1,
			wantKeys: []string{"full name"},
		},
		{
			name:     "field with numbers",
			path:     "$['field123']",
			wantLen:  1,
			wantKeys: []string{"field123"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			segments, err := parsePath(testCase.path)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("parsePath() expected error containing %q, got nil", testCase.errString)

					return
				}

				if testCase.errString != "" && !strings.Contains(err.Error(), testCase.errString) {
					t.Errorf("parsePath() error = %v, want error containing %q", err, testCase.errString)
				}

				return
			}

			if err != nil {
				t.Errorf("parsePath() unexpected error = %v", err)

				return
			}

			if len(segments) != testCase.wantLen {
				t.Errorf("parsePath() got %d segments, want %d", len(segments), testCase.wantLen)

				return
			}

			for idx, wantKey := range testCase.wantKeys {
				if segments[idx].key != wantKey {
					t.Errorf("parsePath() segment[%d].key = %q, want %q", idx, segments[idx].key, wantKey)
				}
			}
		})
	}
}

func TestGetValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     map[string]any
		path      string
		want      any
		wantErr   bool
		errString string
	}{
		{
			name: "simple field",
			input: map[string]any{
				"name": "John",
			},
			path: "$['name']",
			want: "John",
		},
		{
			name: "nested field",
			input: map[string]any{
				"user": map[string]any{
					"name": "Jane",
				},
			},
			path: "$['user']['name']",
			want: "Jane",
		},
		{
			name: "deeply nested field",
			input: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"address": map[string]any{
							"street": "123 Main St",
						},
					},
				},
			},
			path: testPathDeeplyNested,
			want: "123 Main St",
		},
		{
			name: "null value - key exists",
			input: map[string]any{
				"user": map[string]any{
					"middleName": nil,
				},
			},
			path: "$['user']['middleName']",
			want: nil,
		},
		{
			name: "key not found - top level",
			input: map[string]any{
				"foo": "bar",
			},
			path:      "$['missing']",
			wantErr:   true,
			errString: "key 'missing'",
		},
		{
			name: "key not found - nested",
			input: map[string]any{
				"user": map[string]any{
					"name": "John",
				},
			},
			path:      "$['user']['missing']",
			wantErr:   true,
			errString: "key 'missing'",
		},
		{
			name: "null in middle of path",
			input: map[string]any{
				"user": nil,
			},
			path:      "$['user']['name']",
			wantErr:   true,
			errString: "parent is null",
		},
		{
			name: "type mismatch - string not map",
			input: map[string]any{
				"user": "not a map",
			},
			path:      "$['user']['name']",
			wantErr:   true,
			errString: "cannot be traversed",
		},
		{
			name: "case sensitive - exact match",
			input: map[string]any{
				"Name": "John",
			},
			path: "$['Name']",
			want: "John",
		},
		{
			name: "case sensitive - no match",
			input: map[string]any{
				"Name": "John",
			},
			path:      "$['name']",
			wantErr:   true,
			errString: "key 'name'",
		},
		{
			name: "number value",
			input: map[string]any{
				"age": 42,
			},
			path: "$['age']",
			want: 42,
		},
		{
			name: "boolean value",
			input: map[string]any{
				"active": true,
			},
			path: "$['active']",
			want: true,
		},
		{
			name: "array value",
			input: map[string]any{
				"tags": []string{"a", "b", "c"},
			},
			path: "$['tags']",
			want: []string{"a", "b", "c"},
		},
		{
			name: "map value",
			input: map[string]any{
				"config": map[string]any{
					"enabled": true,
				},
			},
			path: "$['config']",
			want: map[string]any{"enabled": true},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetValue(testCase.input, testCase.path, false)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("GetValue() expected error containing %q, got nil", testCase.errString)

					return
				}

				if testCase.errString != "" && !strings.Contains(err.Error(), testCase.errString) {
					t.Errorf("GetValue() error = %v, want error containing %q", err, testCase.errString)
				}

				return
			}

			if err != nil {
				t.Errorf("GetValue() unexpected error = %v", err)

				return
			}

			if !deepEqual(got, testCase.want) {
				t.Errorf("GetValue() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGetValue_CaseInsensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     map[string]any
		path      string
		want      any
		wantErr   bool
		errString string
	}{
		{
			name: "case insensitive - lowercase path matches uppercase key",
			input: map[string]any{
				"Name": "John",
			},
			path: "$['name']",
			want: "John",
		},
		{
			name: "case insensitive - uppercase path matches lowercase key",
			input: map[string]any{
				"name": "John",
			},
			path: "$['NAME']",
			want: "John",
		},
		{
			name: "case insensitive - nested paths",
			input: map[string]any{
				"User": map[string]any{
					"Profile": map[string]any{
						"Email": "john@example.com",
					},
				},
			},
			path: "$['user']['profile']['email']",
			want: "john@example.com",
		},
		{
			name: "case insensitive - mixed case nested",
			input: map[string]any{
				"MailingAddress": map[string]any{
					"Zip": "12345",
				},
			},
			path: "$['mailingaddress']['zip']",
			want: "12345",
		},
		{
			name: "case insensitive - still fails on missing key",
			input: map[string]any{
				"Name": "John",
			},
			path:      "$['missing']",
			wantErr:   true,
			errString: "key 'missing'",
		},
		{
			name: "case insensitive - exact match preferred",
			input: map[string]any{
				"name": "lowercase",
				"Name": "uppercase",
			},
			path: "$['name']",
			want: "lowercase", // Exact match takes precedence in lookupKey
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetValue(testCase.input, testCase.path, true)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("GetValue() expected error containing %q, got nil", testCase.errString)

					return
				}

				if testCase.errString != "" && !strings.Contains(err.Error(), testCase.errString) {
					t.Errorf("GetValue() error = %v, want error containing %q", err, testCase.errString)
				}

				return
			}

			if err != nil {
				t.Errorf("GetValue() unexpected error = %v", err)

				return
			}

			if !deepEqual(got, testCase.want) {
				t.Errorf("GetValue() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestAddPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     map[string]any
		path      string
		value     any
		want      map[string]any
		wantErr   bool
		errString string
	}{
		{
			name:  "simple field",
			input: map[string]any{},
			path:  "$['name']",
			value: "John",
			want: map[string]any{
				"name": "John",
			},
		},
		{
			name:  "nested field - creates intermediate",
			input: map[string]any{},
			path:  "$['user']['name']",
			value: "Jane",
			want: map[string]any{
				"user": map[string]any{
					"name": "Jane",
				},
			},
		},
		{
			name:  "deeply nested - creates all intermediates",
			input: map[string]any{},
			path:  testPathDeeplyNested,
			value: "123 Main St",
			want: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"address": map[string]any{
							"street": "123 Main St",
						},
					},
				},
			},
		},
		{
			name: "nested field - uses existing intermediate",
			input: map[string]any{
				"user": map[string]any{
					"id": 123,
				},
			},
			path:  "$['user']['name']",
			value: "Jane",
			want: map[string]any{
				"user": map[string]any{
					"id":   123,
					"name": "Jane",
				},
			},
		},
		{
			name: "overwrite existing value",
			input: map[string]any{
				"name": "John",
			},
			path:  "$['name']",
			value: "Jane",
			want: map[string]any{
				"name": "Jane",
			},
		},
		{
			name: "set null value",
			input: map[string]any{
				"name": "John",
			},
			path:  "$['name']",
			value: nil,
			want: map[string]any{
				"name": nil,
			},
		},
		{
			name: "conflict - intermediate is not map",
			input: map[string]any{
				"user": "not a map",
			},
			path:      "$['user']['name']",
			value:     "Jane",
			wantErr:   true,
			errString: "exists but is type string",
		},
		{
			name:  "add number value",
			input: map[string]any{},
			path:  "$['age']",
			value: 42,
			want: map[string]any{
				"age": 42,
			},
		},
		{
			name:  "add boolean value",
			input: map[string]any{},
			path:  "$['active']",
			value: true,
			want: map[string]any{
				"active": true,
			},
		},
		{
			name:  "add array value",
			input: map[string]any{},
			path:  "$['tags']",
			value: []string{"a", "b"},
			want: map[string]any{
				"tags": []string{"a", "b"},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Make a deep copy of input to avoid modifying test data
			inputCopy := deepCopyMap(testCase.input)

			err := AddPath(inputCopy, testCase.path, testCase.value)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("AddPath() expected error containing %q, got nil", testCase.errString)

					return
				}

				if testCase.errString != "" && !strings.Contains(err.Error(), testCase.errString) {
					t.Errorf("AddPath() error = %v, want error containing %q", err, testCase.errString)
				}

				return
			}

			if err != nil {
				t.Errorf("AddPath() unexpected error = %v", err)

				return
			}

			if !deepEqual(inputCopy, testCase.want) {
				t.Errorf("AddPath() result = %v, want %v", inputCopy, testCase.want)
			}
		})
	}
}

func TestIsValidPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		wantErr   bool
		errString string
	}{
		{
			name: "valid simple path",
			path: "$['name']",
		},
		{
			name: "valid nested path",
			path: "$['user']['name']",
		},
		{
			name: "valid deeply nested",
			path: "$['a']['b']['c']['d']",
		},
		{
			name:      "empty path",
			path:      "",
			wantErr:   true,
			errString: "path cannot be empty",
		},
		{
			name:      "missing dollar sign",
			path:      "['user']",
			wantErr:   true,
			errString: "path must start with $[",
		},
		{
			name:      "invalid syntax",
			path:      "$user",
			wantErr:   true,
			errString: "path must start with $[",
		},
		{
			name:      "empty segment",
			path:      "$['']",
			wantErr:   true,
			errString: "segment 0",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := IsValidPath(testCase.path)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("IsValidPath() expected error containing %q, got nil", testCase.errString)

					return
				}

				if testCase.errString != "" && !strings.Contains(err.Error(), testCase.errString) {
					t.Errorf("IsValidPath() error = %v, want error containing %q", err, testCase.errString)
				}

				return
			}

			if err != nil {
				t.Errorf("IsValidPath() unexpected error = %v", err)
			}
		})
	}
}

func TestIsNestedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		{
			name:      "valid nested path",
			fieldName: "$['user']['name']",
			want:      true,
		},
		{
			name:      "simple nested path",
			fieldName: "$['name']",
			want:      true,
		},
		{
			name:      "flat field name",
			fieldName: "name",
			want:      false,
		},
		{
			name:      "field with dollar in name",
			fieldName: "price$",
			want:      false,
		},
		{
			name:      "empty string",
			fieldName: "",
			want:      false,
		},
		{
			name:      "just dollar sign",
			fieldName: "$",
			want:      false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := IsNestedPath(testCase.fieldName)
			if got != testCase.want {
				t.Errorf("IsNestedPath(%q) = %v, want %v", testCase.fieldName, got, testCase.want)
			}
		})
	}
}

func TestExtractRootField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
		want      string
	}{
		{
			name:      "nested path - single level",
			fieldName: "$['address']",
			want:      "address",
		},
		{
			name:      "nested path - two levels",
			fieldName: "$['address']['zip']",
			want:      "address",
		},
		{
			name:      "nested path - three levels",
			fieldName: "$['user']['profile']['email']",
			want:      "user",
		},
		{
			name:      "nested path - deeply nested",
			fieldName: "$['MailingAddress']['city']['zipCode']['extended']",
			want:      "MailingAddress",
		},
		{
			name:      "simple field name",
			fieldName: "email",
			want:      "email",
		},
		{
			name:      "simple field name with underscore",
			fieldName: "first_name",
			want:      "first_name",
		},
		{
			name:      "simple field name with dot",
			fieldName: "user.email",
			want:      "user.email",
		},
		{
			name:      "field with case variation",
			fieldName: "$['BillingAddress']['Zip']",
			want:      "BillingAddress",
		},
		{
			name:      "empty string",
			fieldName: "",
			want:      "",
		},
		{
			name:      "invalid path - fallback to original",
			fieldName: "$['']",
			want:      "$['']",
		},
		{
			name:      "invalid path - no brackets - fallback to original",
			fieldName: "$user",
			want:      "$user",
		},
		{
			name:      "path starting with $ but not bracket",
			fieldName: "$email",
			want:      "$email",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ExtractRootField(testCase.fieldName)
			if got != testCase.want {
				t.Errorf("ExtractRootField(%q) = %q, want %q", testCase.fieldName, got, testCase.want)
			}
		})
	}
}

// Helper functions

func deepCopyMap(original map[string]any) map[string]any {
	if original == nil {
		return nil
	}

	copied := make(map[string]any, len(original))

	for key, value := range original {
		switch val := value.(type) {
		case map[string]any:
			// Recursively deep copy nested maps
			copied[key] = deepCopyMap(val)
		default:
			// For other types, shallow copy is sufficient for our test cases
			copied[key] = value
		}
	}

	return copied
}

func deepEqual(left, right any) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil || right == nil {
		return false
	}

	switch leftVal := left.(type) {
	case map[string]any:
		rightVal, ok := right.(map[string]any)
		if !ok {
			return false
		}

		if len(leftVal) != len(rightVal) {
			return false
		}

		for key, val := range leftVal {
			rightV, exists := rightVal[key]
			if !exists || !deepEqual(val, rightV) {
				return false
			}
		}

		return true
	case []string:
		rightVal, ok := right.([]string)
		if !ok {
			return false
		}

		if len(leftVal) != len(rightVal) {
			return false
		}

		for idx := range leftVal {
			if leftVal[idx] != rightVal[idx] {
				return false
			}
		}

		return true
	default:
		return left == right
	}
}

// Benchmarks

func BenchmarkGetValue(b *testing.B) {
	input := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "123 Main St",
				},
			},
		},
	}

	b.ResetTimer()

	for range b.N {
		_, _ = GetValue(input, testPathDeeplyNested, false)
	}
}

func BenchmarkAddPath(b *testing.B) {
	value := "123 Main St"

	b.ResetTimer()

	for range b.N {
		input := make(map[string]any)
		_ = AddPath(input, testPathDeeplyNested, value)
	}
}

func BenchmarkParsePath(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_, _ = parsePath(testPathDeeplyNested)
	}
}

func TestRemovePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		data            map[string]any
		path            string
		expectedResult  map[string]any
		expectedRemoved bool
		expectError     bool
	}{
		{
			name: "remove leaf, parent has other fields",
			data: map[string]any{
				"address": map[string]any{
					"city": "NYC",
					"zip":  "10001",
				},
			},
			path: "$['address']['city']",
			expectedResult: map[string]any{
				"address": map[string]any{
					"zip": "10001",
				},
			},
			expectedRemoved: true,
		},
		{
			name: "remove leaf, parent becomes empty - cleanup parent",
			data: map[string]any{
				"address": map[string]any{
					"city": "NYC",
				},
			},
			path:            "$['address']['city']",
			expectedResult:  map[string]any{},
			expectedRemoved: true,
		},
		{
			name: "deeply nested, all parents become empty",
			data: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"email": "x@y.com",
					},
				},
			},
			path:            "$['user']['profile']['email']",
			expectedResult:  map[string]any{},
			expectedRemoved: true,
		},
		{
			name: "deeply nested, some parents remain",
			data: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"email": "x@y.com",
						"name":  "John",
					},
					"settings": map[string]any{
						"theme": "dark",
					},
				},
			},
			path: "$['user']['profile']['email']",
			expectedResult: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"name": "John",
					},
					"settings": map[string]any{
						"theme": "dark",
					},
				},
			},
			expectedRemoved: true,
		},
		{
			name: "path doesn't exist - no error, not removed",
			data: map[string]any{
				"address": map[string]any{
					"zip": "10001",
				},
			},
			path: "$['address']['city']",
			expectedResult: map[string]any{
				"address": map[string]any{
					"zip": "10001",
				},
			},
			expectedRemoved: false,
		},
		{
			name: "root path doesn't exist",
			data: map[string]any{
				"other": "value",
			},
			path: "$['address']['city']",
			expectedResult: map[string]any{
				"other": "value",
			},
			expectedRemoved: false,
		},
		{
			name: "path traverses non-map value",
			data: map[string]any{
				"address": "simple string, not a map",
			},
			path: "$['address']['city']",
			expectedResult: map[string]any{
				"address": "simple string, not a map",
			},
			expectedRemoved: false,
		},
		{
			name: "single segment path",
			data: map[string]any{
				"email": "x@y.com",
				"name":  "John",
			},
			path: "$['email']",
			expectedResult: map[string]any{
				"name": "John",
			},
			expectedRemoved: true,
		},
		{
			name: "single segment path, becomes empty",
			data: map[string]any{
				"email": "x@y.com",
			},
			path:            "$['email']",
			expectedResult:  map[string]any{},
			expectedRemoved: true,
		},
		{
			name:           "nil map",
			data:           nil,
			path:           "$['address']['city']",
			expectedResult: nil,
			expectError:    true,
		},
		{
			name: "empty path",
			data: map[string]any{
				"address": map[string]any{"city": "NYC"},
			},
			path:        "",
			expectError: true,
		},
		{
			name: "invalid path syntax",
			data: map[string]any{
				"address": map[string]any{"city": "NYC"},
			},
			path:        "address.city",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy of data to avoid test pollution
			dataCopy := copyMap(tc.data)

			removed, err := RemovePath(dataCopy, tc.path)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedRemoved, removed, "removed flag mismatch")
			assert.Equal(t, tc.expectedResult, dataCopy, "result data mismatch")
		})
	}
}

func TestUpdateValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       map[string]any
		path        string
		value       any
		expected    map[string]any
		expectError bool
	}{
		{
			name: "update simple field",
			input: map[string]any{
				"status": "ACTIVE",
			},
			path:  "$['status']",
			value: "INACTIVE",
			expected: map[string]any{
				"status": "INACTIVE",
			},
		},
		{
			name: "update nested field",
			input: map[string]any{
				"user": map[string]any{
					"status": "ACTIVE",
				},
			},
			path:  "$['user']['status']",
			value: "INACTIVE",
			expected: map[string]any{
				"user": map[string]any{
					"status": "INACTIVE",
				},
			},
		},
		{
			name: "update deeply nested field",
			input: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"settings": map[string]any{
							"theme": "light",
						},
					},
				},
			},
			path:  "$['user']['profile']['settings']['theme']",
			value: "dark",
			expected: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"settings": map[string]any{
							"theme": "dark",
						},
					},
				},
			},
		},
		{
			name: "preserve original key casing - exact match",
			input: map[string]any{
				"User": map[string]any{
					"Status": "ACTIVE",
				},
			},
			path:  "$['User']['Status']",
			value: "INACTIVE",
			expected: map[string]any{
				"User": map[string]any{
					"Status": "INACTIVE",
				},
			},
		},
		{
			name: "preserve original key casing - case insensitive match",
			input: map[string]any{
				"User": map[string]any{
					"Status": "ACTIVE",
				},
			},
			path:  "$['user']['status']",
			value: "INACTIVE",
			expected: map[string]any{
				"User": map[string]any{
					"Status": "INACTIVE",
				},
			},
		},
		{
			name: "update field with sibling fields unchanged",
			input: map[string]any{
				"user": map[string]any{
					"name":   "John",
					"status": "ACTIVE",
					"email":  "john@example.com",
				},
			},
			path:  "$['user']['status']",
			value: "INACTIVE",
			expected: map[string]any{
				"user": map[string]any{
					"name":   "John",
					"status": "INACTIVE",
					"email":  "john@example.com",
				},
			},
		},
		{
			name: "error - path does not exist",
			input: map[string]any{
				"user": map[string]any{
					"name": "John",
				},
			},
			path:        "$['user']['status']",
			value:       "ACTIVE",
			expectError: true,
		},
		{
			name: "error - intermediate path is not a map",
			input: map[string]any{
				"user": "string value",
			},
			path:        "$['user']['status']",
			value:       "ACTIVE",
			expectError: true,
		},
		{
			name: "error - top level field does not exist",
			input: map[string]any{
				"user": map[string]any{
					"name": "John",
				},
			},
			path:        "$['contact']",
			value:       "test",
			expectError: true,
		},
		{
			name:        "error - nil map",
			input:       nil,
			path:        "$['status']",
			value:       "ACTIVE",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy of input to avoid modifying test data
			inputCopy := copyMap(tc.input)

			err := UpdateValue(inputCopy, tc.path, tc.value)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, inputCopy, "result data mismatch")
		})
	}
}

// Helper function to deep copy a map for testing
func copyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any)

	for k, v := range src {
		if nestedMap, ok := v.(map[string]any); ok {
			dst[k] = copyMap(nestedMap)
		} else {
			dst[k] = v
		}
	}

	return dst
}

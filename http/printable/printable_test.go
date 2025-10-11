package printable_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/amp-labs/amp-common/http/printable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Payload methods

func TestPayload_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected string
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: "<nil>",
		},
		{
			name: "empty payload",
			payload: &printable.Payload{
				Content: "",
				Length:  0,
			},
			expected: "",
		},
		{
			name: "text content",
			payload: &printable.Payload{
				Content: "hello world",
				Length:  11,
			},
			expected: "hello world",
		},
		{
			name: "base64 content",
			payload: &printable.Payload{
				Base64:  true,
				Content: "aGVsbG8=",
				Length:  5,
			},
			expected: "aGVsbG8=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayload_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected bool
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: true,
		},
		{
			name: "empty content and zero length",
			payload: &printable.Payload{
				Content: "",
				Length:  0,
			},
			expected: true,
		},
		{
			name: "has content",
			payload: &printable.Payload{
				Content: "data",
				Length:  4,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.IsEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayload_IsBase64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected bool
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: false,
		},
		{
			name: "not base64",
			payload: &printable.Payload{
				Base64:  false,
				Content: "text",
			},
			expected: false,
		},
		{
			name: "is base64",
			payload: &printable.Payload{
				Base64:  true,
				Content: "aGVsbG8=",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.IsBase64()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayload_IsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payload     *printable.Payload
		expectedOK  bool
		expectError bool
	}{
		{
			name:        "nil payload",
			payload:     nil,
			expectedOK:  false,
			expectError: false,
		},
		{
			name: "valid JSON object",
			payload: &printable.Payload{
				Content: `{"key":"value"}`,
				Length:  15,
			},
			expectedOK:  true,
			expectError: false,
		},
		{
			name: "valid JSON array",
			payload: &printable.Payload{
				Content: `[1,2,3]`,
				Length:  7,
			},
			expectedOK:  true,
			expectError: false,
		},
		{
			name: "invalid JSON",
			payload: &printable.Payload{
				Content: `{invalid}`,
				Length:  9,
			},
			expectedOK:  false,
			expectError: false,
		},
		{
			name: "plain text",
			payload: &printable.Payload{
				Content: "hello world",
				Length:  11,
			},
			expectedOK:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			isJSON, err := tt.payload.IsJSON()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedOK, isJSON)
		})
	}
}

func TestPayload_GetContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected string
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: "",
		},
		{
			name: "has content",
			payload: &printable.Payload{
				Content: "test data",
				Length:  9,
			},
			expected: "test data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.GetContent()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayload_GetContentBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payload     *printable.Payload
		expected    []byte
		expectError bool
	}{
		{
			name:        "nil payload",
			payload:     nil,
			expected:    nil,
			expectError: false,
		},
		{
			name: "plain text",
			payload: &printable.Payload{
				Content: "hello",
				Length:  5,
			},
			expected:    []byte("hello"),
			expectError: false,
		},
		{
			name: "base64 encoded",
			payload: &printable.Payload{
				Base64:  true,
				Content: base64.StdEncoding.EncodeToString([]byte("hello")),
				Length:  5,
			},
			expected:    []byte("hello"),
			expectError: false,
		},
		{
			name: "invalid base64",
			payload: &printable.Payload{
				Base64:  true,
				Content: "not-valid-base64!!!",
				Length:  5,
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.payload.GetContentBytes()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPayload_GetLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected int64
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: 0,
		},
		{
			name: "has length",
			payload: &printable.Payload{
				Content: "data",
				Length:  1024,
			},
			expected: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.GetLength()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayload_IsTruncated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected bool
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: false,
		},
		{
			name: "not truncated",
			payload: &printable.Payload{
				Content: "data",
				Length:  4,
			},
			expected: false,
		},
		{
			name: "truncated",
			payload: &printable.Payload{
				Content:         "da",
				Length:          4,
				TruncatedLength: 2,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.IsTruncated()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayload_Clone(t *testing.T) {
	t.Parallel()

	t.Run("nil payload", func(t *testing.T) {
		t.Parallel()

		var payload *printable.Payload
		cloned := payload.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clones payload", func(t *testing.T) {
		t.Parallel()

		original := &printable.Payload{
			Base64:          true,
			Content:         "test",
			Length:          100,
			TruncatedLength: 50,
		}

		cloned := original.Clone()
		require.NotNil(t, cloned)
		assert.NotSame(t, original, cloned)
		assert.Equal(t, original.Base64, cloned.Base64)
		assert.Equal(t, original.Content, cloned.Content)
		assert.Equal(t, original.Length, cloned.Length)
		assert.Equal(t, original.TruncatedLength, cloned.TruncatedLength)

		// Modify clone should not affect original
		cloned.Content = "modified"

		assert.Equal(t, "test", original.Content)
	})
}

func TestPayload_Truncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		payload           *printable.Payload
		size              int64
		expectNil         bool
		expectError       bool
		expectedContent   string
		expectedTruncated int64
	}{
		{
			name:        "nil payload",
			payload:     nil,
			size:        10,
			expectNil:   true,
			expectError: false,
		},
		{
			name: "negative size",
			payload: &printable.Payload{
				Content: "hello",
				Length:  5,
			},
			size:        -1,
			expectNil:   true,
			expectError: false,
		},
		{
			name: "no truncation needed - size larger than content",
			payload: &printable.Payload{
				Content: "hello",
				Length:  5,
			},
			size:              10,
			expectNil:         false,
			expectError:       false,
			expectedContent:   "hello",
			expectedTruncated: 0,
		},
		{
			name: "truncate plain text",
			payload: &printable.Payload{
				Content: "hello world",
				Length:  11,
			},
			size:              5,
			expectNil:         false,
			expectError:       false,
			expectedContent:   "hello",
			expectedTruncated: 5,
		},
		{
			name: "truncate base64",
			payload: &printable.Payload{
				Base64:  true,
				Content: base64.StdEncoding.EncodeToString([]byte("hello world")),
				Length:  11,
			},
			size:              5,
			expectNil:         false,
			expectError:       false,
			expectedContent:   base64.StdEncoding.EncodeToString([]byte("hello")),
			expectedTruncated: 5,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := testCase.payload.Truncate(testCase.size)
			if testCase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if testCase.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, testCase.expectedContent, result.Content)
				assert.Equal(t, testCase.expectedTruncated, result.TruncatedLength)
			}
		})
	}
}

func TestPayload_GetTruncatedLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  *printable.Payload
		expected int64
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: 0,
		},
		{
			name: "not truncated - returns full length",
			payload: &printable.Payload{
				Content: "hello",
				Length:  5,
			},
			expected: 5,
		},
		{
			name: "truncated - returns truncated length",
			payload: &printable.Payload{
				Content:         "he",
				Length:          5,
				TruncatedLength: 2,
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.payload.GetTruncatedLength()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test Request and Response functions

func TestRequest_JSONContent(t *testing.T) {
	t.Parallel()

	jsonBody := `{"key":"value","number":123}`
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader(jsonBody)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.JSONEq(t, jsonBody, payload.GetContent())
	assert.Equal(t, int64(len(jsonBody)), payload.GetLength())

	isJSON, err := payload.IsJSON()
	require.NoError(t, err)
	assert.True(t, isJSON)
}

func TestRequest_TextContent(t *testing.T) {
	t.Parallel()

	textBody := "hello world"
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader(textBody)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "text/plain")

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, textBody, payload.GetContent())
	assert.Equal(t, int64(len(textBody)), payload.GetLength())
}

func TestRequest_BinaryContent(t *testing.T) {
	t.Parallel()

	// Binary data (PNG header)
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://example.com", io.NopCloser(bytes.NewReader(binaryData)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "image/png")

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.True(t, payload.IsBase64())
	assert.Equal(t, int64(len(binaryData)), payload.GetLength())

	// Decode and verify
	decoded, err := payload.GetContentBytes()
	require.NoError(t, err)
	assert.Equal(t, binaryData, decoded)
}

func TestRequest_WithPrereadBody(t *testing.T) {
	t.Parallel()

	bodyText := "preread body"
	bodyBytes := []byte(bodyText)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com", nil)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "text/plain")

	payload, err := printable.Request(req, bodyBytes)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, bodyText, payload.GetContent())
}

func TestRequest_EmptyBody(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	assert.Nil(t, payload)
}

func TestResponse_JSONContent(t *testing.T) {
	t.Parallel()

	jsonBody := `{"status":"ok","data":[1,2,3]}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(jsonBody)),
	}
	resp.Header.Set("Content-Type", "application/json")

	payload, err := printable.Response(resp, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.JSONEq(t, jsonBody, payload.GetContent())
	assert.Equal(t, int64(len(jsonBody)), payload.GetLength())

	isJSON, err := payload.IsJSON()
	require.NoError(t, err)
	assert.True(t, isJSON)
}

func TestResponse_TextContent(t *testing.T) {
	t.Parallel()

	textBody := "response body"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(textBody)),
	}
	resp.Header.Set("Content-Type", "text/html")

	payload, err := printable.Response(resp, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, textBody, payload.GetContent())
}

func TestResponse_WithPrereadBody(t *testing.T) {
	t.Parallel()

	bodyText := "preread response"
	bodyBytes := []byte(bodyText)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       nil,
	}
	resp.Header.Set("Content-Type", "text/plain")

	payload, err := printable.Response(resp, bodyBytes)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, bodyText, payload.GetContent())
}

func TestResponse_EmptyBody(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	payload, err := printable.Response(resp, nil)
	require.NoError(t, err)
	assert.Nil(t, payload)
}

func TestRequest_XMLContent(t *testing.T) {
	t.Parallel()

	xmlBody := `<?xml version="1.0"?><root><item>value</item></root>`
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader(xmlBody)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/xml")

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, xmlBody, payload.GetContent())
}

func TestRequest_FormURLEncoded(t *testing.T) {
	t.Parallel()

	formBody := "key1=value1&key2=value2"
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader(formBody)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, formBody, payload.GetContent())
}

func TestPayload_LogValue(t *testing.T) {
	t.Parallel()

	// Test with JSON content
	jsonPayload := &printable.Payload{
		Content: `{"key":"value"}`,
		Length:  15,
	}

	logValue := jsonPayload.LogValue()
	assert.NotNil(t, logValue)

	// Test with non-JSON content
	textPayload := &printable.Payload{
		Content: "plain text",
		Length:  10,
	}

	logValue = textPayload.LogValue()
	assert.NotNil(t, logValue)

	// Test with base64 content
	base64Payload := &printable.Payload{
		Base64:  true,
		Content: "aGVsbG8=",
		Length:  5,
	}

	logValue = base64Payload.LogValue()
	assert.NotNil(t, logValue)
}

func TestRequest_CharsetConversion(t *testing.T) {
	t.Parallel()

	// UTF-8 text
	utf8Text := "Hello, 世界"
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader(utf8Text)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	payload, err := printable.Request(req, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)

	assert.False(t, payload.IsBase64())
	assert.Equal(t, utf8Text, payload.GetContent())
}

func TestPayload_TruncatePreservesTruncationInfo(t *testing.T) {
	t.Parallel()

	original := &printable.Payload{
		Content: "hello world this is a long string",
		Length:  34,
	}

	// First truncation
	truncated1, err := original.Truncate(20)
	require.NoError(t, err)
	require.NotNil(t, truncated1)

	assert.Equal(t, int64(34), truncated1.Length) // Original length preserved
	assert.Equal(t, int64(20), truncated1.TruncatedLength)
	assert.True(t, truncated1.IsTruncated())

	// Second truncation should work on already truncated payload
	truncated2, err := truncated1.Truncate(10)
	require.NoError(t, err)
	require.NotNil(t, truncated2)

	assert.Equal(t, int64(34), truncated2.Length) // Original length still preserved
	assert.Equal(t, int64(10), truncated2.TruncatedLength)
	assert.True(t, truncated2.IsTruncated())
}

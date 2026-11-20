package redact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/amp-labs/amp-common/http/printable"
)

// ErrUnstructuredBody is returned by BodyFields for a body that is neither a JSON object nor
// form fields, so it cannot be redacted one field at a time. Redact the whole body instead.
var ErrUnstructuredBody = errors.New("body is not a JSON object or form fields")

// maxFormFieldNameLength is the longest form field name BodyFields accepts.
const maxFormFieldNameLength = 64

// BodyFields returns a copy of body in which each field's value has been passed through redact,
// as Headers does for each header. It handles bodies that are a JSON object or form fields
// (application/x-www-form-urlencoded), such as OAuth token requests and responses. Anything
// else, including a binary (base64) payload, returns ErrUnstructuredBody, and the caller should
// redact the whole body.
//
// In a JSON object, redact sees every string value, nested ones included, under the name of
// the field that holds it: the innermost object key, or for an array element, the array's
// name. Numbers, booleans and nulls are kept. The result is encoded again, so its fields are
// sorted by name and its whitespace may differ from the original.
//
// A form body is accepted only when every "&"-separated part has an "=" and every field name
// is at most 64 lower-case letters, digits or "_-.[]" characters. That keeps a body that is
// really a bare secret, such as a token in a plain-text response, from being taken for a field
// name and logged as one. Kept fields are logged as they were sent (still percent-encoded),
// and replaced values are not encoded, so markers such as "[redacted]" stay readable.
//
// The returned payload keeps the original Length, so logs show the body's real size.
func BodyFields(ctx context.Context, body *printable.Payload, redact Func) (*printable.Payload, error) {
	if body == nil {
		return nil, nil
	}

	if redact == nil {
		return body.Clone(), nil
	}

	if body.IsBase64() {
		return nil, ErrUnstructuredBody
	}

	content := strings.TrimSpace(body.Content)
	if content == "" {
		return body.Clone(), nil
	}

	var (
		out string
		err error
	)

	if strings.HasPrefix(content, "{") {
		out, err = jsonFields(ctx, content, redact)
	} else {
		out, err = formFields(ctx, content, redact)
	}

	if err != nil {
		return nil, err
	}

	return replacedPayload(body, out), nil
}

// jsonFields redacts the fields of a JSON object.
func jsonFields(ctx context.Context, content string, redact Func) (string, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()

	var object map[string]any

	err := decoder.Decode(&object)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnstructuredBody, err)
	}

	if decoder.More() {
		return "", fmt.Errorf("%w: trailing data after the JSON object", ErrUnstructuredBody)
	}

	redactJSONObject(ctx, object, redact)

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(object)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnstructuredBody, err)
	}

	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// redactJSONObject redacts the string values of object in place.
func redactJSONObject(ctx context.Context, object map[string]any, redact Func) {
	for key, value := range object {
		out, keep := redactJSONValue(ctx, key, value, redact)
		if keep {
			object[key] = out
		} else {
			delete(object, key)
		}
	}
}

// redactJSONValue redacts value, which is held by the field named key, and returns false if
// it should be left out.
func redactJSONValue(ctx context.Context, key string, value any, redact Func) (any, bool) {
	switch typed := value.(type) {
	case string:
		action, partialLen := redact(ctx, key, typed)

		return redactValue(ctx, action, partialLen, typed)
	case map[string]any:
		redactJSONObject(ctx, typed, redact)

		return typed, true
	case []any:
		kept := typed[:0]

		for _, element := range typed {
			out, keep := redactJSONValue(ctx, key, element, redact)
			if keep {
				kept = append(kept, out)
			}
		}

		return kept, true
	default:
		// Numbers, booleans and nulls.
		return value, true
	}
}

// formFields redacts the fields of a form body.
func formFields(ctx context.Context, content string, redact Func) (string, error) {
	parts := strings.Split(content, "&")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}

		rawName, rawValue, found := strings.Cut(part, "=")
		if !found {
			return "", fmt.Errorf("%w: a form part has no '='", ErrUnstructuredBody)
		}

		name, err := url.QueryUnescape(rawName)
		if err != nil || !isFormFieldName(name) {
			return "", fmt.Errorf("%w: a form part has an unexpected field name", ErrUnstructuredBody)
		}

		value, err := url.QueryUnescape(rawValue)
		if err != nil {
			return "", fmt.Errorf("%w: a form value is not percent-encoded correctly", ErrUnstructuredBody)
		}

		action, partialLen := redact(ctx, name, value)
		if action == ActionKeep {
			out = append(out, part)

			continue
		}

		if replaced, keep := redactValue(ctx, action, partialLen, value); keep {
			out = append(out, rawName+"="+replaced)
		}
	}

	return strings.Join(out, "&"), nil
}

// isFormFieldName reports whether name looks like a form field name rather than data.
func isFormFieldName(name string) bool {
	if name == "" || len(name) > maxFormFieldNameLength {
		return false
	}

	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
		case char == '_', char == '-', char == '.', char == '[', char == ']':
		default:
			return false
		}
	}

	return true
}

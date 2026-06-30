// Package jsonpath provides utilities for working with JSONPath bracket notation.
// This package supports a subset of JSONPath specifically designed for field mapping:
// - Bracket notation: $['field']['nestedField']
// - Configurable case-sensitive or case-insensitive key matching
// - Path validation
//
//nolint:godoclint // Package comment is correctly formatted
package jsonpath

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Sentinel errors for path validation and traversal.
var (
	ErrPathEmpty               = errors.New("path cannot be empty")
	ErrPathMustStartWithDollar = errors.New("path must start with $[")
	ErrPathEmptySegment        = errors.New("path contains empty segment")
	ErrPathNoValidSegments     = errors.New("no valid segments found in path")
	ErrPathInvalidSyntax       = errors.New("invalid bracket notation syntax")
	ErrPathMustHaveOneSegment  = errors.New("path must have at least one segment")
	ErrMapNil                  = errors.New("map cannot be nil")
	ErrPathMustContainOneKey   = errors.New("path must contain at least one key")
	ErrPathSegmentNotFound     = errors.New("path segment not found")
	ErrPathCannotTraverse      = errors.New("path segment cannot be traversed")
	ErrPathKeyNotFound         = errors.New("key not found at path segment")
	ErrPathCannotCreateNested  = errors.New("cannot create nested path")
	ErrAddPathNotSupported     = errors.New("path does not support AddPath with wild '*' key")
)

// ArrayWildIndex represents a wildcard index in JSONPath notation.
//
// Normally, a concrete numeric index is used to access a specific array item.
// For our connectors, we often need to match all items in an array, so `*` is used instead.
//
// Example:
//
//	"$['line_items']['data'][*]['description']"
//	==> line_items[data][0,1,2, ... ][description]
const ArrayWildIndex = "*"

type PathSegment struct {
	// Key is the key name of the segment.
	// Example: for $['address']['city'], the segments are "address" and "city".
	Key string
}

func (s PathSegment) IsWildKey() bool {
	return s.Key == ArrayWildIndex
}

// ParsePath parses a JSONPath-like bracket notation into a slice of PathSegment.
//
// The accepted form begins with "$" and uses bracket notation for keys and wildcards,
// for example: "$['mailingaddress']['street']" or "$['items'][*]['id']".
//
// Behavior and error cases:
//   - Returns ErrPathEmpty when the input string is empty.
//   - Returns ErrPathMustStartWithDollar when the path does not start with "$[".
//   - Returns ErrPathEmptySegment when any segment is empty (for example "[”]").
//     The returned error includes the zero-based segment index when possible.
//   - Returns ErrPathNoValidSegments when no valid segments can be extracted.
//   - Returns ErrPathInvalidSyntax when the reconstructed segments do not exactly
//     match the original input (this catches malformed syntax such as extra characters).
//
// On success, returns a slice of PathSegment in the order they appear in the path.
// Example:
//
//	ParsePath("$['mailingaddress']['street']")
//
// returns
//
//	[]PathSegment{{Key: "mailingaddress"}, {Key: "street"}}, nil
func ParsePath(path string) ([]PathSegment, error) {
	if path == "" {
		return nil, ErrPathEmpty
	}

	if !strings.HasPrefix(path, "$[") {
		return nil, fmt.Errorf("%w, got: %s", ErrPathMustStartWithDollar, path)
	}

	// Check for empty segments FIRST
	emptySegmentPattern := `\[''\]`

	emptySegmentRe := regexp.MustCompile(emptySegmentPattern)
	if emptySegmentRe.MatchString(path) {
		// Find position of empty segment for better error message
		allSegmentPattern := `\[`
		allSegmentRe := regexp.MustCompile(allSegmentPattern)
		allMatches := allSegmentRe.FindAllStringIndex(path, -1)

		emptyMatches := emptySegmentRe.FindAllStringIndex(path, -1)
		if len(emptyMatches) > 0 {
			emptyPos := emptyMatches[0][0]
			segmentNum := 0

			for _, match := range allMatches {
				if match[0] < emptyPos {
					segmentNum++
				}
			}

			return nil, fmt.Errorf("%w: segment %d", ErrPathEmptySegment, segmentNum)
		}

		return nil, ErrPathEmptySegment
	}

	// Extract segments using regex
	segmentPattern := `\['([^']+)'\]|\[(\*)\]`
	segmentRe := regexp.MustCompile(segmentPattern)
	matches := segmentRe.FindAllStringSubmatch(path, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrPathNoValidSegments, path)
	}

	// A match is an array of tuples [][]string.
	// Here is an example of some tuples:
	// [0]: "['line_items']"
	// [1]: "line_items"  // captured key
	// [2]: ""            // wildcard capture empty
	//
	// [0]: "[*]"
	// [1]: ""            // key capture empty
	// [2]: "*"           // wildcard capture
	keys := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 && m[1] != "" {
			keys = append(keys, m[1])
		} else if len(m) > 2 && m[2] == ArrayWildIndex {
			keys = append(keys, ArrayWildIndex)
		}
	}

	segments := make([]PathSegment, len(keys))
	for idx, key := range keys {
		segments[idx] = PathSegment{Key: key}
	}

	// Validate path by reconstructing it. This ensures exact syntax match and
	// rejects inputs that contain valid-looking segments plus extra characters.
	if newPath(segments) != path {
		return nil, fmt.Errorf("%w: %s", ErrPathInvalidSyntax, path)
	}

	return segments, nil
}

// GetValue retrieves the value located at the JSON-like path inside the input map.
//
// The path uses dot/bracket style parsed by ParsePath and may include wildcards ([*]).
// If the final value exists and is JSON null, GetValue returns (nil, nil).
// If any key along the path is missing, any traversal fails due to unexpected types,
// or path parsing fails, an error is returned describing the failing segment.
//
// caseInsensitive controls key lookup behavior: when true, lookupKey will match keys
// case-insensitively but prefers exact-case matches when available.
//
// Examples of returned values:
// - a scalar (string, number, bool) if the path resolves to a leaf
// - map[string]any when the path resolves to an object
// - []any when the path resolves to an array or when a wildcard collects values
// - []any{[]any{}, []any{}} is possible when querying nested arrays.
func GetValue(input map[string]any, path string, caseInsensitive bool) (any, error) {
	return getValue(input, path, caseInsensitive, 0)
}

// getValue is the recursive implementation behind GetValue.
//
// offset is the number of path segments that have already been processed by an outer call.
// It is used only for producing stable, meaningful error messages that refer to the
// absolute segment index within the original path.
func getValue(input map[string]any, path string, caseInsensitive bool, offset int) (any, error) {
	segments, err := ParsePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	current := any(input)

	for segmentIndex, segment := range segments {
		// globalIndex refers to the position in the original path (for error messages).
		globalIndex := segmentIndex + offset

		if segment.IsWildKey() {
			// Wildcard expects an array/slice at this position and will iterate its items.
			array, ok := asArray(current)
			if !ok {
				return nil, fmt.Errorf(
					"%w: segment %d ('%s'), parent is type %T but expected []any",
					ErrPathCannotTraverse, globalIndex, segment.Key, current,
				)
			}

			// If wildcard is last segment, return the array as-is.
			if segmentIndex+1 == len(segments) {
				return array, nil
			}

			// Otherwise, collect results from each array element by recursing into the
			// remaining subpath. If any element cannot be traversed as an object,
			// return an error. The returned slice contains values (including nil) from
			// each element's resolution.
			items := make([]any, 0)

			for arrayIndex, item := range array {
				mapping, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf(
						"%w: segment %d ('%s', index[%d]), parent is type %T but expected map[string]any",
						ErrPathCannotTraverse, globalIndex, segment.Key, arrayIndex, item,
					)
				}

				value, err := getValue(mapping, newPath(segments[segmentIndex+1:]), caseInsensitive, globalIndex+1)
				if err != nil {
					return nil, err
				}

				items = append(items, value)
			}

			return items, nil
		}

		// Non-wildcard: expect current to be an object/map.
		currentMap, ok := current.(map[string]any)
		if !ok {
			if current == nil {
				// Parent is null while more segments remain.
				return nil, fmt.Errorf(
					"%w: segment %d ('%s'), parent is null",
					ErrPathSegmentNotFound, globalIndex, segment.Key,
				)
			}

			return nil, fmt.Errorf(
				"%w: segment %d ('%s'), parent is type %T",
				ErrPathCannotTraverse, globalIndex, segment.Key, current,
			)
		}

		// Lookup with configurable case sensitivity. lookupKey returns the value and
		// whether the key was found according to the caseInsensitive policy.
		value, exists := lookupKey(currentMap, segment.Key, caseInsensitive)
		if !exists {
			return nil, fmt.Errorf("%w: key '%s' at segment %d", ErrPathKeyNotFound, segment.Key, globalIndex)
		}

		// If this value is nil while further segments remain, that's a missing path.
		if value == nil && segmentIndex < len(segments)-1 {
			return nil, fmt.Errorf(
				"%w: segment %d ('%s'), parent is null",
				ErrPathSegmentNotFound, globalIndex+1, segments[globalIndex+1].Key,
			)
		}

		current = value
	}

	return current, nil
}

// newPath reconstructs a string path from the given PathSegment slice.
// The returned path begins with '$' and uses "['key']" for normal segments,
// "[*]" for wildcard segments. This is intended for error messages and recursion.
func newPath(segments []PathSegment) string {
	var reconstructedSb86 strings.Builder

	for _, segment := range segments {
		value := fmt.Sprintf("['%s']", segment.Key)
		if segment.IsWildKey() {
			value = "[*]"
		}

		reconstructedSb86.WriteString(value)
	}

	return "$" + reconstructedSb86.String()
}

// asArray normalizes different array-like types to []any.
//
// It accepts []map[string]any (a common internal representation) and []any.
// Returns the normalized slice and true when the input is an array-like type.
func asArray(object any) ([]any, bool) {
	if slice, ok := object.([]map[string]any); ok {
		result := make([]any, len(slice))
		for i, item := range slice {
			result[i] = item
		}

		return result, true
	}

	slice, ok := object.([]any)

	return slice, ok
}

// lookupKey performs key lookup with optional case-insensitive matching.
// When caseInsensitive is true, exact matches are preferred over case-insensitive matches.
func lookupKey(m map[string]any, key string, caseInsensitive bool) (any, bool) {
	// Always try exact match first
	if value, exists := m[key]; exists {
		return value, exists
	}

	// If case-insensitive mode, try case-insensitive match
	if caseInsensitive {
		lowerKey := strings.ToLower(key)
		for k, v := range m {
			if strings.ToLower(k) == lowerKey {
				return v, true
			}
		}
	}

	return nil, false
}

// AddPath sets a value at a nested path, creating intermediate objects as needed.
// Modifies the input map in place.
func AddPath(input map[string]any, path string, value any) error {
	segments, err := ParsePath(path)
	if err != nil {
		return fmt.Errorf("failed to parse path: %w", err)
	}

	if len(segments) == 0 {
		return ErrPathMustHaveOneSegment
	}

	for _, segment := range segments {
		if segment.IsWildKey() {
			return ErrAddPathNotSupported
		}
	}

	// Navigate to parent, creating maps as needed
	current := input

	for idx := range len(segments) - 1 {
		segment := segments[idx]

		if existing, exists := current[segment.Key]; exists {
			if existingMap, ok := existing.(map[string]any); ok {
				current = existingMap
			} else {
				return fmt.Errorf("%w: segment '%s' exists but is type %T", ErrPathCannotCreateNested, segment.Key, existing)
			}
		} else {
			newMap := make(map[string]any)
			current[segment.Key] = newMap
			current = newMap
		}
	}

	// Set final value
	finalSegment := segments[len(segments)-1]
	current[finalSegment.Key] = value

	return nil
}

// UpdateValue updates an existing value at the given JSONPath, preserving key casing.
// Unlike AddPath which creates missing intermediate maps, UpdateValue only succeeds if the path exists.
// This is useful for value transformations where you want to preserve the original structure.
// Uses case-insensitive matching to find keys.
func UpdateValue(data map[string]any, path string, value any) error {
	if data == nil {
		return ErrMapNil
	}

	segments, err := ParsePath(path)
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		return ErrPathMustHaveOneSegment
	}

	// Navigate to parent using case-insensitive lookup
	current := data

	for idx := range len(segments) - 1 {
		segment := segments[idx]

		// Use case-insensitive lookup
		nextValue, exists := lookupKey(current, segment.Key, true)
		if !exists {
			return fmt.Errorf("%w: segment '%s' does not exist", ErrPathSegmentNotFound, segment.Key)
		}

		nextMap, ok := nextValue.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: segment '%s' is not a map", ErrPathCannotTraverse, segment.Key)
		}

		current = nextMap
	}

	// Update the final value using case-insensitive key matching
	finalSegment := segments[len(segments)-1]

	// Try exact match first
	if _, exists := current[finalSegment.Key]; exists {
		current[finalSegment.Key] = value

		return nil
	}

	// Try case-insensitive match
	lowerKey := strings.ToLower(finalSegment.Key)
	for k := range current {
		if strings.ToLower(k) == lowerKey {
			current[k] = value // Update using original key casing

			return nil
		}
	}

	return fmt.Errorf("%w: key '%s'", ErrPathKeyNotFound, finalSegment.Key)
}

// ValidatePath validates that a string is valid JSONPath bracket notation.
func ValidatePath(path string) error {
	if path == "" {
		return ErrPathEmpty
	}

	if !strings.HasPrefix(path, "$[") {
		return ErrPathMustStartWithDollar
	}

	segments, err := ParsePath(path)
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		return ErrPathMustContainOneKey
	}

	return nil
}

// IsNestedPath checks if a field name is a JSONPath bracket notation path.
func IsNestedPath(fieldName string) bool {
	return strings.HasPrefix(fieldName, "$[")
}

// ExtractRootField extracts the root field name from a JSONPath bracket notation.
// For simple field names (non-nested), returns the field unchanged.
//
// Examples:
//   - ExtractRootField("$['address']['zip']") returns "address"
//   - ExtractRootField("$['user']['profile']['email']") returns "user"
//   - ExtractRootField("email") returns "email"
//   - ExtractRootField("firstName") returns "firstName"
//
// This is critical for the connector interface: connectors expect simple field names,
// not bracket notation paths. When building ReadParams.Fields, we must extract root
// fields so connectors can use them in provider API queries (e.g., SOQL for Salesforce).
func ExtractRootField(fieldName string) string {
	// If not a nested path, return as-is
	if !IsNestedPath(fieldName) {
		return fieldName
	}

	// Parse the path to extract segments
	segments, err := ParsePath(fieldName)
	if err != nil || len(segments) == 0 {
		// Fallback to original if parsing fails
		return fieldName
	}

	// Return the first segment (root field)
	return segments[0].Key
}

// RemovePath removes a value at a nested path from a map, cleaning up empty parent objects.
// If removing the path results in empty parent objects, those are also removed.
//
// Examples:
//
//   - RemovePath({"address": {"city": "NYC", "zip": "10001"}}, "$['address']['city']")
//     Results in: {"address": {"zip": "10001"}} (parent still has other fields)
//
//   - RemovePath({"address": {"city": "NYC"}}, "$['address']['city']")
//     Results in: {} (parent becomes empty, so entire "address" key is removed)
//
//   - RemovePath({"user": {"profile": {"email": "x@y.com"}}}, "$['user']['profile']['email']")
//     Results in: {} (all empty parents removed)
//
// Returns:
//   - true if the path was found and removed
//   - false if the path didn't exist (not an error)
//   - error if path is invalid or cannot be traversed
func RemovePath(data map[string]any, path string) (bool, error) {
	if data == nil {
		return false, ErrMapNil
	}

	segments, err := ParsePath(path)
	if err != nil {
		return false, err
	}

	if len(segments) == 0 {
		return false, ErrPathMustHaveOneSegment
	}

	// Helper function to recursively remove and clean up
	var removeRecursive func(current map[string]any, segmentIdx int) bool

	removeRecursive = func(current map[string]any, segmentIdx int) bool {
		if segmentIdx >= len(segments) {
			return false
		}

		segment := segments[segmentIdx]
		key := segment.Key

		// If this is the final segment, remove the key (case-insensitive)
		if segmentIdx == len(segments)-1 {
			// Try exact match first
			if _, exists := current[key]; exists {
				delete(current, key)

				return true
			}

			// Try case-insensitive match
			lowerKey := strings.ToLower(key)
			for k := range current {
				if strings.ToLower(k) == lowerKey {
					delete(current, k)

					return true
				}
			}

			return false
		}

		// Not the final segment - traverse deeper (case-insensitive)
		value, exists := lookupKey(current, key, true)
		if !exists {
			return false // Path doesn't exist
		}

		// Must be a map to traverse
		nestedMap, ok := value.(map[string]any)
		if !ok {
			return false // Can't traverse non-map
		}

		// Recursively remove from nested map
		removed := removeRecursive(nestedMap, segmentIdx+1)

		// If we removed something and the nested map is now empty, remove this key too
		if removed && len(nestedMap) == 0 {
			// Need to find the actual key to delete (case-insensitive)
			if _, exists := current[key]; exists {
				delete(current, key)
			} else {
				// Find case-insensitive match
				lowerKey := strings.ToLower(key)
				for k := range current {
					if strings.ToLower(k) == lowerKey {
						delete(current, k)

						break
					}
				}
			}
		}

		return removed
	}

	removed := removeRecursive(data, 0)

	return removed, nil
}

// ToNestedPath converts path keys into JSONPath bracket notation.
// Supports any depth of nesting.
//
// Examples:
//   - ToNestedPath("address") -> "$['address']"
//   - ToNestedPath("address", "city") -> "$['address']['city']"
//   - ToNestedPath("user", "profile", "email") -> "$['user']['profile']['email']"
func ToNestedPath(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("$")

	for _, key := range keys {
		b.WriteString(fmt.Sprintf("['%s']", key))
	}

	return b.String()
}

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
)

type pathSegment struct {
	key string
}

// ParsePath parses a JSONPath bracket notation string into segments.
// Example: ParsePath("$['mailingaddress']['street']") returns
// []pathSegment{{key: "mailingaddress"}, {key: "street"}}, nil.
func ParsePath(path string) ([]pathSegment, error) {
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
	segmentPattern := `\['([^']+)'\]`
	segmentRe := regexp.MustCompile(segmentPattern)
	matches := segmentRe.FindAllStringSubmatch(path, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrPathNoValidSegments, path)
	}

	// Validate path by reconstructing it
	reconstructed := "$"

	var reconstructedSb86 strings.Builder
	for _, match := range matches {
		reconstructedSb86.WriteString(fmt.Sprintf("['%s']", match[1]))
	}

	reconstructed += reconstructedSb86.String()

	if reconstructed != path {
		return nil, fmt.Errorf("%w: %s", ErrPathInvalidSyntax, path)
	}

	segments := make([]pathSegment, len(matches))
	for idx, match := range matches {
		segments[idx] = pathSegment{key: match[1]}
	}

	return segments, nil
}

// GetValue retrieves a value from a nested path.
// Returns nil, nil if the value at the path exists but is null.
// Returns error if a key in the path is not found or type mismatch occurs.
// If caseInsensitive is true, key matching is case-insensitive with exact matches preferred.
func GetValue(input map[string]any, path string, caseInsensitive bool) (any, error) {
	segments, err := ParsePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	current := any(input)

	for segmentIndex, segment := range segments {
		currentMap, ok := current.(map[string]any)
		if !ok {
			if current == nil {
				return nil, fmt.Errorf(
					"%w: segment %d ('%s'), parent is null",
					ErrPathSegmentNotFound, segmentIndex, segment.key,
				)
			}

			return nil, fmt.Errorf(
				"%w: segment %d ('%s'), parent is type %T",
				ErrPathCannotTraverse, segmentIndex, segment.key, current,
			)
		}

		// Lookup with configurable case sensitivity
		value, exists := lookupKey(currentMap, segment.key, caseInsensitive)
		if !exists {
			return nil, fmt.Errorf("%w: key '%s' at segment %d", ErrPathKeyNotFound, segment.key, segmentIndex)
		}

		// Handle null in middle of path
		if value == nil && segmentIndex < len(segments)-1 {
			return nil, fmt.Errorf(
				"%w: segment %d ('%s'), parent is null",
				ErrPathSegmentNotFound, segmentIndex+1, segments[segmentIndex+1].key,
			)
		}

		current = value
	}

	return current, nil
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

	// Navigate to parent, creating maps as needed
	current := input

	for idx := range len(segments) - 1 {
		segment := segments[idx]

		if existing, exists := current[segment.key]; exists {
			if existingMap, ok := existing.(map[string]any); ok {
				current = existingMap
			} else {
				return fmt.Errorf("%w: segment '%s' exists but is type %T", ErrPathCannotCreateNested, segment.key, existing)
			}
		} else {
			newMap := make(map[string]any)
			current[segment.key] = newMap
			current = newMap
		}
	}

	// Set final value
	finalSegment := segments[len(segments)-1]
	current[finalSegment.key] = value

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
		nextValue, exists := lookupKey(current, segment.key, true)
		if !exists {
			return fmt.Errorf("%w: segment '%s' does not exist", ErrPathSegmentNotFound, segment.key)
		}

		nextMap, ok := nextValue.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: segment '%s' is not a map", ErrPathCannotTraverse, segment.key)
		}

		current = nextMap
	}

	// Update the final value using case-insensitive key matching
	finalSegment := segments[len(segments)-1]

	// Try exact match first
	if _, exists := current[finalSegment.key]; exists {
		current[finalSegment.key] = value

		return nil
	}

	// Try case-insensitive match
	lowerKey := strings.ToLower(finalSegment.key)
	for k := range current {
		if strings.ToLower(k) == lowerKey {
			current[k] = value // Update using original key casing

			return nil
		}
	}

	return fmt.Errorf("%w: key '%s'", ErrPathKeyNotFound, finalSegment.key)
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
	return segments[0].key
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
		key := segment.key

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

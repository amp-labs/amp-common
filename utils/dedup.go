// Package utils provides miscellaneous utility functions for channels, context, JSON, sleep, dedup, and more.
package utils

import (
	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/errors"
	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/set"
)

// DeduplicateValues removes duplicate values from a slice using SHA-256 hashing.
// Returns a new slice containing only unique values in arbitrary order.
// Returns an error if hashing fails or hash collision is detected.
func DeduplicateValues[T collectable.Collectable[T]](values []T) ([]T, error) {
	s := set.NewSet[T](hashing.Sha256) // big hash space, low chance of collision

	err := s.AddAll(values...)
	if err != nil {
		return nil, err
	}

	return s.Entries(), nil
}

// HasDuplicateValues checks whether a slice contains any duplicate values.
// Returns true if duplicates exist, false otherwise.
// Returns an error if hashing fails or hash collision is detected.
func HasDuplicateValues[T collectable.Collectable[T]](values []T) (bool, error) {
	count, err := CountDuplicateValues(values)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CountDuplicateValues counts the total number of duplicate occurrences in a slice.
// For example, [A, A, B, B] returns 3 (one extra A and two extra B's).
// Uses SHA-256 hashing for comparison and detects hash collisions.
// Returns an error if hashing fails or hash collision is detected.
func CountDuplicateValues[T collectable.Collectable[T]](values []T) (int, error) {
	seen := make(map[string]T)
	counts := make(map[string]int)

	for _, val := range values {
		sha, err := hashing.Sha256(val)
		if err != nil {
			return 0, err
		}

		count, ok := counts[sha]
		if !ok {
			count = 0
		}

		counts[sha] = count + 1

		prev, ok := seen[sha]
		if ok {
			if prev.Equals(val) {
				continue
			} else {
				return 0, errors.ErrHashCollision
			}
		}

		seen[sha] = val
	}

	dupes := 0

	for _, count := range counts {
		if count > 1 {
			dupes += count - 1
		}
	}

	return dupes, nil
}

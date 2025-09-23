package utils

import (
	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/set"
)

func DeduplicateValues[T set.Collectable[T]](values []T) ([]T, error) {
	s := set.NewSet[T](hashing.Sha256) // big hash space, low chance of collision
	if err := s.AddAll(values...); err != nil {
		return nil, err
	}

	return s.Entries(), nil
}

func HasDuplicateValues[T set.Collectable[T]](values []T) (bool, error) {
	count, err := CountDuplicateValues(values)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func CountDuplicateValues[T set.Collectable[T]](values []T) (int, error) {
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
				return 0, set.ErrHashCollision
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

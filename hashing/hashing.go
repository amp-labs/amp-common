package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

// HashFunc is a function that takes a Hashable object
// and returns a string representation of its hashing.
// As an example, the Sha256 function is a HashFunc.
// This lets us talk about hashing functions in a generic way.
type HashFunc func(hashable Hashable) (string, error)

// Hashable is an interface that allows an object to update
// a hash.Hash with its contents. This is useful for hashing
// objects so that they can be easily compared.
type Hashable interface {
	UpdateHash(h hash.Hash) error
}

// Sha256 returns the SHA256 hashing of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hashing, an error is returned.
func Sha256(hashable Hashable) (string, error) {
	h := sha256.New()

	if err := hashable.UpdateHash(h); err != nil {
		return "", err
	}

	bts := h.Sum(nil)

	return hex.EncodeToString(bts), nil
}

type HashableString string

func (s HashableString) String() string {
	return string(s)
}

func (s HashableString) UpdateHash(h hash.Hash) error {
	_, err := h.Write([]byte(s))
	if err != nil {
		return err
	}

	return nil
}

func (s HashableString) Equals(other HashableString) bool {
	return s == other
}

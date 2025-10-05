package hashing

import (
	"bytes"
	"crypto/md5"  //nolint:gosec
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
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

// Md5 returns the MD5 hash of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hash, an error is returned.
func Md5(hashable Hashable) (string, error) {
	return HashHex(hashable, md5.New()) //nolint:gosec
}

// Sha1 returns the SHA1 hash of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hash, an error is returned.
func Sha1(hashable Hashable) (string, error) {
	return HashHex(hashable, sha1.New()) //nolint:gosec
}

// Sha256 returns the SHA256 hashing of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hashing, an error is returned.
func Sha256(hashable Hashable) (string, error) {
	return HashHex(hashable, sha256.New())
}

// Sha512 returns the SHA512 hash of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hash, an error is returned.
func Sha512(hashable Hashable) (string, error) {
	return HashHex(hashable, sha512.New())
}

// HashHex is a helper function that applies a hash function to a Hashable
// and returns the result as a hex-encoded string.
func HashHex(hashable Hashable, h hash.Hash) (string, error) {
	if err := hashable.UpdateHash(h); err != nil {
		return "", err
	}

	bts := h.Sum(nil)

	return hex.EncodeToString(bts), nil
}

// HashBase64 is a helper function that applies a hash function to a Hashable
// and returns the result as a base64-encoded string.
func HashBase64(hashable Hashable, h hash.Hash) (string, error) {
	if err := hashable.UpdateHash(h); err != nil {
		return "", err
	}

	bts := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(bts), nil
}

// HashableString is a string type that implements the Hashable interface.
type HashableString string

// String returns the string value.
func (s HashableString) String() string {
	return string(s)
}

// UpdateHash writes the string bytes to the hash.
func (s HashableString) UpdateHash(h hash.Hash) error {
	_, err := h.Write([]byte(s))
	if err != nil {
		return err
	}

	return nil
}

// Equals returns true if the two HashableStrings are equal.
func (s HashableString) Equals(other HashableString) bool {
	return s == other
}

// HashableBytes is a byte slice type that implements the Hashable interface.
type HashableBytes []byte

// UpdateHash writes the bytes to the hash.
func (b HashableBytes) UpdateHash(h hash.Hash) error {
	_, err := h.Write(b)
	if err != nil {
		return err
	}

	return nil
}

// Equals returns true if the two HashableBytes are equal.
func (b HashableBytes) Equals(other HashableBytes) bool {
	return bytes.Equal(b, other)
}

// Package hashing provides cryptographic hash utilities and hashable types for use with Map and Set collections.
package hashing

import (
	"bytes"
	"crypto/md5" //nolint:gosec
	"crypto/rand"
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"math"

	"github.com/OneOfOne/xxhash"
	"github.com/google/uuid"
	"github.com/zeebo/xxh3"
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

// XxHash32 returns the xxHash 32-bit hash of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hash, an error is returned.
func XxHash32(hashable Hashable) (string, error) {
	return HashHex(hashable, xxhash.NewHash32())
}

// XxHash64 returns the xxHash 64-bit hash of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hash, an error is returned.
func XxHash64(hashable Hashable) (string, error) {
	return HashHex(hashable, xxhash.NewHash64())
}

// Xxh3 returns the xxHash3 hash of the given Hashable
// as a hex-encoded string. If the Hashable fails to
// update the hash, an error is returned.
func Xxh3(hashable Hashable) (string, error) {
	return HashHex(hashable, xxh3.New())
}

// HashHex is a helper function that applies a hash function to a Hashable
// and returns the result as a hex-encoded string.
func HashHex(hashable Hashable, h hash.Hash) (string, error) {
	err := hashable.UpdateHash(h)
	if err != nil {
		return "", err
	}

	bts := h.Sum(nil)

	return hex.EncodeToString(bts), nil
}

// HashBase64 is a helper function that applies a hash function to a Hashable
// and returns the result as a base64-encoded string.
func HashBase64(hashable Hashable, h hash.Hash) (string, error) {
	err := hashable.UpdateHash(h)
	if err != nil {
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

// HashableInt is an int type that implements the Hashable and Comparable interfaces.
type HashableInt int

// UpdateHash writes the int value to the hash using little-endian encoding.
func (i HashableInt) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 8)                        //nolint:mnd
	binary.LittleEndian.PutUint64(buf, uint64(i)) //nolint:gosec
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableInt values are equal.
func (i HashableInt) Equals(other HashableInt) bool {
	return int(i) == int(other)
}

// HashableInt8 is an int8 type that implements the Hashable and Comparable interfaces.
type HashableInt8 int8

// UpdateHash writes the int8 value to the hash as a single byte.
func (i HashableInt8) UpdateHash(h hash.Hash) error {
	buf := []byte{byte(i)}
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableInt8 values are equal.
func (i HashableInt8) Equals(other HashableInt8) bool {
	return int8(i) == int8(other)
}

// HashableInt16 is an int16 type that implements the Hashable and Comparable interfaces.
type HashableInt16 int16

// UpdateHash writes the int16 value to the hash using little-endian encoding.
func (i HashableInt16) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 2)                        //nolint:mnd
	binary.LittleEndian.PutUint16(buf, uint16(i)) //nolint:gosec
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableInt16 values are equal.
func (i HashableInt16) Equals(other HashableInt16) bool {
	return int16(i) == int16(other)
}

// HashableInt32 is an int32 type that implements the Hashable and Comparable interfaces.
type HashableInt32 int32

// UpdateHash writes the int32 value to the hash using little-endian encoding.
func (i HashableInt32) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 4)                        //nolint:mnd
	binary.LittleEndian.PutUint32(buf, uint32(i)) //nolint:gosec
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableInt32 values are equal.
func (i HashableInt32) Equals(other HashableInt32) bool {
	return int32(i) == int32(other)
}

// HashableInt64 is an int64 type that implements the Hashable and Comparable interfaces.
type HashableInt64 int64

// UpdateHash writes the int64 value to the hash using little-endian encoding.
func (i HashableInt64) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 8)                        //nolint:mnd
	binary.LittleEndian.PutUint64(buf, uint64(i)) //nolint:gosec
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableInt64 values are equal.
func (i HashableInt64) Equals(other HashableInt64) bool {
	return int64(i) == int64(other)
}

// HashableUint is a uint type that implements the Hashable and Comparable interfaces.
type HashableUint uint

// UpdateHash writes the uint value to the hash using little-endian encoding.
func (u HashableUint) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 8) //nolint:mnd
	binary.LittleEndian.PutUint64(buf, uint64(u))
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableUint values are equal.
func (u HashableUint) Equals(other HashableUint) bool {
	return uint(u) == uint(other)
}

// HashableUint8 is a uint8 type that implements the Hashable and Comparable interfaces.
type HashableUint8 uint8

// UpdateHash writes the uint8 value to the hash as a single byte.
func (u HashableUint8) UpdateHash(h hash.Hash) error {
	buf := []byte{byte(u)}
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableUint8 values are equal.
func (u HashableUint8) Equals(other HashableUint8) bool {
	return uint8(u) == uint8(other)
}

// HashableUint16 is a uint16 type that implements the Hashable and Comparable interfaces.
type HashableUint16 uint16

// UpdateHash writes the uint16 value to the hash using little-endian encoding.
func (u HashableUint16) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 2) //nolint:mnd
	binary.LittleEndian.PutUint16(buf, uint16(u))
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableUint16 values are equal.
func (u HashableUint16) Equals(other HashableUint16) bool {
	return uint16(u) == uint16(other)
}

// HashableUint32 is a uint32 type that implements the Hashable and Comparable interfaces.
type HashableUint32 uint32

// UpdateHash writes the uint32 value to the hash using little-endian encoding.
func (u HashableUint32) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 4) //nolint:mnd
	binary.LittleEndian.PutUint32(buf, uint32(u))
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableUint32 values are equal.
func (u HashableUint32) Equals(other HashableUint32) bool {
	return uint32(u) == uint32(other)
}

// HashableUint64 is a uint64 type that implements the Hashable and Comparable interfaces.
type HashableUint64 uint64

// UpdateHash writes the uint64 value to the hash using little-endian encoding.
func (u HashableUint64) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 8) //nolint:mnd
	binary.LittleEndian.PutUint64(buf, uint64(u))
	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableUint64 values are equal.
func (u HashableUint64) Equals(other HashableUint64) bool {
	return uint64(u) == uint64(other)
}

// HashableFloat32 is a float32 type that implements the Hashable and Comparable interfaces.
type HashableFloat32 float32

// UpdateHash writes the float32 value to the hash using its IEEE 754 binary representation in little-endian encoding.
// If the value is NaN, random bytes are written to ensure NaN values produce different hashes.
func (f HashableFloat32) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 4) //nolint:mnd

	if math.IsNaN(float64(f)) {
		// For NaN values, write random bytes to ensure different hashes
		_, err := rand.Read(buf)
		if err != nil {
			return fmt.Errorf("failed to generate random bytes for NaN: %w", err)
		}
	} else {
		binary.LittleEndian.PutUint32(buf, math.Float32bits(float32(f)))
	}

	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableFloat32 values are equal.
// Returns false if either value is NaN, as NaN is never equal to anything (including itself).
func (f HashableFloat32) Equals(other HashableFloat32) bool {
	// NaN is never equal to anything, including itself
	if math.IsNaN(float64(f)) || math.IsNaN(float64(other)) {
		return false
	}

	return float32(f) == float32(other)
}

// HashableFloat64 is a float64 type that implements the Hashable and Comparable interfaces.
type HashableFloat64 float64

// UpdateHash writes the float64 value to the hash using its IEEE 754 binary representation in little-endian encoding.
// If the value is NaN, random bytes are written to ensure NaN values produce different hashes.
func (f HashableFloat64) UpdateHash(h hash.Hash) error {
	buf := make([]byte, 8) //nolint:mnd

	if math.IsNaN(float64(f)) {
		// For NaN values, write random bytes to ensure different hashes
		_, err := rand.Read(buf)
		if err != nil {
			return fmt.Errorf("failed to generate random bytes for NaN: %w", err)
		}
	} else {
		binary.LittleEndian.PutUint64(buf, math.Float64bits(float64(f)))
	}

	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableFloat64 values are equal.
// Returns false if either value is NaN, as NaN is never equal to anything (including itself).
func (f HashableFloat64) Equals(other HashableFloat64) bool {
	// NaN is never equal to anything, including itself
	if math.IsNaN(float64(f)) || math.IsNaN(float64(other)) {
		return false
	}

	return float64(f) == float64(other)
}

// HashableBool is a bool type that implements the Hashable and Comparable interfaces.
type HashableBool bool

// UpdateHash writes the bool value to the hash as a single byte (0 for false, 1 for true).
func (b HashableBool) UpdateHash(h hash.Hash) error {
	var buf []byte
	if b {
		buf = []byte{1}
	} else {
		buf = []byte{0}
	}

	_, err := h.Write(buf)

	return err
}

// Equals returns true if the two HashableBool values are equal.
func (b HashableBool) Equals(other HashableBool) bool {
	return bool(b) == bool(other)
}

// HashableUUID is a UUID type that implements the Hashable and Comparable interfaces.
type HashableUUID uuid.UUID

// UpdateHash writes the UUID bytes to the hash.
func (u HashableUUID) UpdateHash(h hash.Hash) error {
	_, err := h.Write(u[:])

	return err
}

// Equals returns true if the two HashableUUID values are equal.
func (u HashableUUID) Equals(other HashableUUID) bool {
	return bytes.Equal(u[:], other[:])
}

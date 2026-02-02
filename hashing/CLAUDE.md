# Package: hashing

Cryptographic hash utilities and Hashable interface for use with Map and Set collections.

## Usage

```go
// Hash a value
type MyData struct { Value string }
func (m MyData) UpdateHash(h hash.Hash) error {
    _, err := h.Write([]byte(m.Value))
    return err
}

hash, err := hashing.Sha256(myData)  // Returns hex string
```

## Common Patterns

- Implement `Hashable` interface for custom types
- Built-in hashable types: HashableString, HashableInt, HashableBytes, etc.
- Hash functions: Sha256, Sha512, Md5, Sha1, XxHash32/64, Xxh3

## Gotchas

- MD5 and SHA1 are available but not cryptographically secure
- Use XxHash for fast non-cryptographic hashing
- Returns hex-encoded strings by default (use HashBase64 for base64)

## Related

- `collectable` - Uses Hashable for Map/Set keys

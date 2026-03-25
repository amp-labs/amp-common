# Package: collectable

Interface combining Hashable and Comparable for use in Map and Set data structures.

## Usage

```go
// Use primitive types
key := collectable.FromComparable("mykey")

// Custom type
type MyKey struct { ID string }
func (k MyKey) UpdateHash(h hash.Hash) error { /* ... */ }
func (k MyKey) Equals(other MyKey) bool { /* ... */ }
```

## Common Patterns

- `FromComparable()` wraps primitives (int, string, bool, etc.)
- Implement `Collectable[T]` for custom hashable types
- Used with hash-based maps/sets for O(1) operations

## Gotchas

- `FromComparable()` supports standard types; custom types need full implementation
- Use `Collectable` for hash-based, `Sortable` for ordered collections

## Related

- `hashing` - Hashable interface and hash functions
- `compare` - Comparable interface

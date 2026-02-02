# Package: sortable

Wrapper types (Int, Byte, String) for primitives that implement Sortable interface for use in sorted collections.

## Usage

```go
// Create sorted set
intSet := set.NewRedBlackTreeSet[sortable.Int]()
intSet.Add(sortable.Int(42))
intSet.Add(sortable.Int(10))

// Elements are returned sorted: 10, 42
for val := range intSet.Seq() {
    fmt.Println(int(val))
}
```

## Common Patterns

- Use with red-black tree-based maps and sets
- Provides O(log n) operations with sorted order
- Implement `Sortable` for custom types (needs `Equals` + `LessThan`)

## Gotchas

- Use `Sortable` for ordered collections, `Collectable` for hash-based
- Collections using sortable types need external synchronization

## Related

- `compare` - Base Comparable interface
- `maps`, `set` - Collections using Sortable

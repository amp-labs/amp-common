# Package: maps

Generic map implementations including hash maps, red-black trees, ordered maps, and thread-safe variants.

## Usage

```go
// Red-black tree map (sorted)
m := maps.NewRedBlackTreeMap[sortable.String, int]()
m.Put(sortable.String("key"), 42)

// Hash map (O(1) lookup)
hashMap := maps.NewHashMap[collectable.String, int](hashing.Sha256)

// Case-insensitive map
ciMap := maps.NewCaseInsensitiveMap(map[string]string{
    "Content-Type": "application/json",
})
key, val, ok := ciMap.Get("content-type", false)  // Case-insensitive
```

## Common Patterns

- `NewRedBlackTreeMap()` - Sorted map (O(log n))
- `NewHashMap()` - Hash-based map (O(1) avg)
- `NewCaseInsensitiveMap()` - Case-insensitive string keys
- `NewThreadSafeMap()` - Thread-safe wrapper
- Ordered variants maintain insertion order

## Gotchas

- Red-black tree requires `Sortable` keys
- Hash map requires `Collectable` keys
- Thread-safe wrappers add locking overhead

## Related

- `sortable`, `collectable` - Key types
- `set` - Set implementations
